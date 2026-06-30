package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/OrlandoHdz/kubo/internal/db"
	"github.com/OrlandoHdz/kubo/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PedidosHandler struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewPedidosHandler(q *db.Queries, p *pgxpool.Pool) *PedidosHandler {
	return &PedidosHandler{queries: q, pool: p}
}

type PedidoDetalleInput struct {
	VarianteID int32   `json:"variante_id" binding:"required"`
	Cantidad   int32   `json:"cantidad" binding:"required"`
	Precio     float64 `json:"precio" binding:"required"`
}

type AgregarProductosInput struct {
	Items []PedidoDetalleInput `json:"items" binding:"required,min=1"`
}

type CrearPedidoInput struct {
	ClienteID  int32                `json:"cliente_id" binding:"required"`
	UsuarioID  int32                `json:"usuario_id" binding:"required"`
	MetodoPago string               `json:"metodo_pago" binding:"required"` // 'Tarjeta' o 'Crédito'
	Subtotal   float64              `json:"subtotal" binding:"required"`
	Iva        float64              `json:"iva" binding:"required"`
	TotalOrden float64              `json:"total_orden" binding:"required"`
	Items      []PedidoDetalleInput `json:"items" binding:"required,min=1"`
}

func (h *PedidosHandler) validarVentanaModificacion(c *gin.Context, pedidoID int32) bool {
	rol, _ := c.Get("userRol")
	if rol == "Admin" {
		return true
	}

	dentro, err := h.queries.PedidoDentroVentanaModificacion(c.Request.Context(), pedidoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al validar ventana de modificación: " + err.Error()})
		return false
	}
	if !dentro {
		c.JSON(http.StatusForbidden, gin.H{"error": "El plazo para modificar este pedido ha vencido"})
		return false
	}
	return true
}

func (h *PedidosHandler) Crear(c *gin.Context) {
	var input CrearPedidoInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos de pedido inválidos: " + err.Error()})
		return
	}

	// 1. Iniciar Transacción
	tx, err := h.pool.Begin(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al iniciar transacción: " + err.Error()})
		return
	}
	defer tx.Rollback(c.Request.Context())

	qtx := h.queries.WithTx(tx)

	// 2. Validar Crédito (si aplica)
	if input.MetodoPago == "Crédito" {
		cliente, err := qtx.GetCliente(c.Request.Context(), input.ClienteID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Cliente no encontrado"})
			return
		}

		if !cliente.PermitirPagoCredito {
			c.JSON(http.StatusForbidden, gin.H{"error": "El cliente no tiene habilitado el pago a crédito"})
			return
		}

		// Calcular crédito disponible
		totalUtilizado, _ := utils.NumericToFloat64(cliente.LineaCreditoUtilizada)
		totalLinea, _ := utils.NumericToFloat64(cliente.LineaCreditoTotal)
		disponible := totalLinea - totalUtilizado

		if disponible < input.TotalOrden {
			c.JSON(http.StatusForbidden, gin.H{
				"error":      "Crédito insuficiente",
				"disponible": disponible,
				"requerido":  input.TotalOrden,
			})
			return
		}

		// Actualizar crédito utilizado (el SQL ya suma el valor proporcionado)
		err = qtx.ActualizarSaldoCredito(c.Request.Context(), db.ActualizarSaldoCreditoParams{
			ID:                    input.ClienteID,
			LineaCreditoUtilizada: utils.ToNumeric(input.TotalOrden),
			UpdatedBy:             pgtype.Int4{Int32: input.UsuarioID, Valid: true},
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar saldo de crédito: " + err.Error()})
			return
		}
	}

	// 3. Validar Disponibilidad de Stock (tomando en cuenta pedidos en proceso)
	type itemStockInfo struct {
		VarianteID int32
		Cantidad   int32
		Precio     float64
		Disponible int32
	}

	var itemsStock []itemStockInfo
	esBackorder := false
	for _, item := range input.Items {
		stockReal, err := qtx.GetStockRealVariante(c.Request.Context(), item.VarianteID)
		if err != nil {
			stockReal = pgtype.Numeric{}
		}
		stockRealFloat, _ := utils.NumericToFloat64(stockReal)

		committed, err := qtx.GetCommittedStock(c.Request.Context(), pgtype.Int4{Int32: item.VarianteID, Valid: true})
		if err != nil {
			committed = 0
		}

		disponible := int32(stockRealFloat) - committed
		if disponible < item.Cantidad {
			esBackorder = true
		}
		itemsStock = append(itemsStock, itemStockInfo{
			VarianteID: item.VarianteID,
			Cantidad:   item.Cantidad,
			Precio:     item.Precio,
			Disponible: disponible,
		})
	}

	// 4. Generar Folio
	folio := fmt.Sprintf("ARW-%d", time.Now().Unix())

	// 5. Crear Encabezado de Pedido
	estadoBackorder := ""
	if esBackorder {
		estadoBackorder = "Pendiente"
	}
	pedido, err := qtx.CrearPedido(c.Request.Context(), db.CrearPedidoParams{
		Folio:               folio,
		ClienteID:           pgtype.Int4{Int32: input.ClienteID, Valid: true},
		UsuarioID:           pgtype.Int4{Int32: input.UsuarioID, Valid: true},
		Estado:              "Pendiente",
		MetodoPago:          input.MetodoPago,
		Subtotal:            utils.ToNumeric(input.Subtotal),
		Iva:                 utils.ToNumeric(input.Iva),
		TotalOrden:          utils.ToNumeric(input.TotalOrden),
		EsBackorder:         pgtype.Bool{Bool: esBackorder, Valid: true},
		GuiaBackorder:       pgtype.Text{String: "", Valid: false},
		NotasAdminBackorder: pgtype.Text{String: "", Valid: false},
		EstadoBackorder:     estadoBackorder,
		CreatedBy:           pgtype.Int4{Int32: input.UsuarioID, Valid: true},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear encabezado de pedido: " + err.Error()})
		return
	}

	// 5. Crear Detalles (splitteando en Normal + Backorder según stock disponible)
	for _, is := range itemsStock {
		if is.Disponible >= is.Cantidad {
			// Stock suficiente: un solo detalle Normal
			_, err := qtx.CrearPedidoDetalle(c.Request.Context(), db.CrearPedidoDetalleParams{
				PedidoID:               pgtype.Int4{Int32: pedido.ID, Valid: true},
				VarianteID:             pgtype.Int4{Int32: is.VarianteID, Valid: true},
				Cantidad:               is.Cantidad,
				PrecioUnitarioAplicado: utils.ToNumeric(is.Precio),
				TipoRegistro:           "Normal",
				CreatedBy:              pgtype.Int4{Int32: input.UsuarioID, Valid: true},
			})
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error al crear detalle para variante %d: %s", is.VarianteID, err.Error())})
				return
			}
		} else {
			// Stock insuficiente: dividir en Normal (disponible) + Backorder (resto)
			if is.Disponible > 0 {
				_, err := qtx.CrearPedidoDetalle(c.Request.Context(), db.CrearPedidoDetalleParams{
					PedidoID:               pgtype.Int4{Int32: pedido.ID, Valid: true},
					VarianteID:             pgtype.Int4{Int32: is.VarianteID, Valid: true},
					Cantidad:               is.Disponible,
					PrecioUnitarioAplicado: utils.ToNumeric(is.Precio),
					TipoRegistro:           "Normal",
					CreatedBy:              pgtype.Int4{Int32: input.UsuarioID, Valid: true},
				})
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error al crear detalle Normal para variante %d: %s", is.VarianteID, err.Error())})
					return
				}
			}
			_, err := qtx.CrearPedidoDetalle(c.Request.Context(), db.CrearPedidoDetalleParams{
				PedidoID:               pgtype.Int4{Int32: pedido.ID, Valid: true},
				VarianteID:             pgtype.Int4{Int32: is.VarianteID, Valid: true},
				Cantidad:               is.Cantidad - is.Disponible,
				PrecioUnitarioAplicado: utils.ToNumeric(is.Precio),
				TipoRegistro:           "Backorder",
				CreatedBy:              pgtype.Int4{Int32: input.UsuarioID, Valid: true},
			})
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error al crear detalle Backorder para variante %d: %s", is.VarianteID, err.Error())})
				return
			}
		}
	}

	// 6. Confirmar Transacción
	if err := tx.Commit(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al confirmar pedido: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, pedido)
}

func (h *PedidosHandler) Listar(c *gin.Context) {
	pedidos, err := h.queries.ListarPedidos(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al listar pedidos: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, pedidos)
}

func (h *PedidosHandler) ListarPorCliente(c *gin.Context) {
	clienteIDStr := c.Param("cliente_id")
	clienteID, err := strconv.Atoi(clienteIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de cliente inválido"})
		return
	}

	pedidos, err := h.queries.ListarPedidosPorCliente(c.Request.Context(), pgtype.Int4{Int32: int32(clienteID), Valid: true})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al listar pedidos del cliente: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, pedidos)
}

func (h *PedidosHandler) Obtener(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de pedido inválido"})
		return
	}

	pedido, err := h.queries.GetPedido(c.Request.Context(), int32(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pedido no encontrado"})
		return
	}

	detalles, err := h.queries.ListarPedidosDetalle(c.Request.Context(), pgtype.Int4{Int32: int32(id), Valid: true})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener detalles del pedido: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pedido":   pedido,
		"detalles": detalles,
	})
}

func (h *PedidosHandler) ActualizarEstado(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de pedido inválido"})
		return
	}

	var input struct {
		Estado      string `json:"estado" binding:"required"`
		Guia        string `json:"guia"`
		NotasAdmin  string `json:"notas_admin"`
		UpdatedBy   int32  `json:"updated_by"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos de estado inválidos: " + err.Error()})
		return
	}

	if !h.validarVentanaModificacion(c, int32(id)) {
		return
	}

	pedidoActualizado, err := h.queries.ActualizarEstadoPedido(c.Request.Context(), db.ActualizarEstadoPedidoParams{
		ID:         int32(id),
		Estado:     input.Estado,
		Guia:       pgtype.Text{String: input.Guia, Valid: input.Guia != ""},
		NotasAdmin: pgtype.Text{String: input.NotasAdmin, Valid: input.NotasAdmin != ""},
		UpdatedBy:  pgtype.Int4{Int32: input.UpdatedBy, Valid: input.UpdatedBy != 0},
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar estado: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, pedidoActualizado)
}

func (h *PedidosHandler) ActualizarBackorder(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de pedido inválido"})
		return
	}

	var input struct {
		GuiaBackorder       string `json:"guia_backorder"`
		NotasAdminBackorder string `json:"notas_admin_backorder"`
		EstadoBackorder     string `json:"estado_backorder" binding:"required"`
		UpdatedBy           int32  `json:"updated_by"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos de backorder inválidos: " + err.Error()})
		return
	}

	pedidoActualizado, err := h.queries.ActualizarBackorderPedido(c.Request.Context(), db.ActualizarBackorderPedidoParams{
		ID:                  int32(id),
		GuiaBackorder:       pgtype.Text{String: input.GuiaBackorder, Valid: input.GuiaBackorder != ""},
		NotasAdminBackorder: pgtype.Text{String: input.NotasAdminBackorder, Valid: input.NotasAdminBackorder != ""},
		EstadoBackorder:     input.EstadoBackorder,
		UpdatedBy:           pgtype.Int4{Int32: input.UpdatedBy, Valid: input.UpdatedBy != 0},
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar backorder: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, pedidoActualizado)
}

func (h *PedidosHandler) AgregarProductos(c *gin.Context) {
	// Parse order ID from URL
	pedidoIDStr := c.Param("id")
	pedidoID, err := strconv.Atoi(pedidoIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de pedido inválido"})
		return
	}

	// Bind input JSON
	var input AgregarProductosInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
		return
	}

	// Verify order exists and is pending and within 2h window
	pedido, err := h.queries.GetPedido(c.Request.Context(), int32(pedidoID))
	if err != nil || pedido.Estado != "Pendiente" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Solo se pueden agregar productos a pedidos pendientes"})
		return
	}

	if !h.validarVentanaModificacion(c, int32(pedidoID)) {
		return
	}

	// Start transaction
	tx, err := h.pool.Begin(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al iniciar transacción: " + err.Error()})
		return
	}
	defer tx.Rollback(c.Request.Context())
	qtx := h.queries.WithTx(tx)

	// Pre-calcular stock disponible por item
	type itemStockInfo struct {
		VarianteID int32
		Cantidad   int32
		Precio     float64
		Disponible int32
	}

	var itemsStock []itemStockInfo
	esBackorder := false
	for _, item := range input.Items {
		stockReal, err := qtx.GetStockRealVariante(c.Request.Context(), item.VarianteID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error al obtener stock real para variante %d: %s", item.VarianteID, err.Error())})
			return
		}
		stockRealFloat, _ := utils.NumericToFloat64(stockReal)
		committed, _ := qtx.GetCommittedStock(c.Request.Context(), pgtype.Int4{Int32: item.VarianteID, Valid: true})
		disponible := int32(stockRealFloat) - committed
		if disponible < item.Cantidad {
			esBackorder = true
		}
		itemsStock = append(itemsStock, itemStockInfo{
			VarianteID: item.VarianteID,
			Cantidad:   item.Cantidad,
			Precio:     item.Precio,
			Disponible: disponible,
		})
	}

	// Si hay backorder y el pedido no lo estaba, marcar como backorder
	if esBackorder && !pedido.EsBackorder.Bool {
		_, err := qtx.ActualizarBackorderPedido(c.Request.Context(), db.ActualizarBackorderPedidoParams{
			ID:                  int32(pedidoID),
			GuiaBackorder:       pgtype.Text{String: "", Valid: false},
			NotasAdminBackorder: pgtype.Text{String: "", Valid: false},
			EstadoBackorder:     "Pendiente",
			UpdatedBy:           pgtype.Int4{Int32: int32(0), Valid: false},
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar backorder del pedido: " + err.Error()})
			return
		}
		// También actualizar es_backorder a true directamente
		_, err = tx.Exec(c.Request.Context(), "UPDATE pedidos SET es_backorder = true WHERE id = $1", int32(pedidoID))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al marcar pedido como backorder: " + err.Error()})
			return
		}
	}

	// Crear detalles splitteando según stock disponible
	var addedSubtotal float64
	for _, is := range itemsStock {
		if is.Disponible >= is.Cantidad {
			// Stock suficiente: un solo detalle Normal
			_, err := qtx.CrearPedidoDetalle(c.Request.Context(), db.CrearPedidoDetalleParams{
				PedidoID:               pgtype.Int4{Int32: int32(pedidoID), Valid: true},
				VarianteID:             pgtype.Int4{Int32: is.VarianteID, Valid: true},
				Cantidad:               is.Cantidad,
				PrecioUnitarioAplicado: utils.ToNumeric(is.Precio),
				TipoRegistro:           "Normal",
				CreatedBy:              pgtype.Int4{Int32: int32(0), Valid: false},
			})
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error al crear detalle: %s", err.Error())})
				return
			}
			addedSubtotal += is.Precio * float64(is.Cantidad)
		} else {
			// Stock insuficiente: dividir en Normal (disponible) + Backorder (resto)
			if is.Disponible > 0 {
				_, err := qtx.CrearPedidoDetalle(c.Request.Context(), db.CrearPedidoDetalleParams{
					PedidoID:               pgtype.Int4{Int32: int32(pedidoID), Valid: true},
					VarianteID:             pgtype.Int4{Int32: is.VarianteID, Valid: true},
					Cantidad:               is.Disponible,
					PrecioUnitarioAplicado: utils.ToNumeric(is.Precio),
					TipoRegistro:           "Normal",
					CreatedBy:              pgtype.Int4{Int32: int32(0), Valid: false},
				})
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error al crear detalle Normal: %s", err.Error())})
					return
				}
			}
			_, err := qtx.CrearPedidoDetalle(c.Request.Context(), db.CrearPedidoDetalleParams{
				PedidoID:               pgtype.Int4{Int32: int32(pedidoID), Valid: true},
				VarianteID:             pgtype.Int4{Int32: is.VarianteID, Valid: true},
				Cantidad:               is.Cantidad - is.Disponible,
				PrecioUnitarioAplicado: utils.ToNumeric(is.Precio),
				TipoRegistro:           "Backorder",
				CreatedBy:              pgtype.Int4{Int32: int32(0), Valid: false},
			})
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error al crear detalle Backorder: %s", err.Error())})
				return
			}
			addedSubtotal += is.Precio * float64(is.Cantidad)
		}
	}

	// Update order totals
	_, err = tx.Exec(c.Request.Context(), "UPDATE pedidos SET subtotal = subtotal + $1, total_orden = total_orden + $1 WHERE id = $2", utils.ToNumeric(addedSubtotal), int32(pedidoID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar totales del pedido: " + err.Error()})
		return
	}

	// Commit transaction
	if err := tx.Commit(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al confirmar transacción: " + err.Error()})
		return
	}

	// Return updated pedido
	updatedPedido, err := h.queries.GetPedido(c.Request.Context(), int32(pedidoID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener pedido actualizado"})
		return
	}
	c.JSON(http.StatusOK, updatedPedido)
}

func (h *PedidosHandler) CancelarDetalle(c *gin.Context) {
	// Obtener IDs del pedido y del detalle
	pedidoIDStr := c.Param("id")
	detalleIDStr := c.Param("detalle_id")
	pedidoID, err := strconv.Atoi(pedidoIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de pedido inválido"})
		return
	}
	detalleID, err := strconv.Atoi(detalleIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de detalle inválido"})
		return
	}

	// Verificar que el pedido esté pendiente y dentro del plazo de 2 h
	pedido, err := h.queries.GetPedido(c.Request.Context(), int32(pedidoID))
	if err != nil || pedido.Estado != "Pendiente" {
		c.JSON(http.StatusForbidden, gin.H{"error": "No se puede cancelar el detalle de este pedido"})
		return
	}

	if !h.validarVentanaModificacion(c, int32(pedidoID)) {
		return
	}

	// Obtener ID del usuario autenticado (si está disponible)
	userID := int32(0)
	if uid, ok := c.Get("userID"); ok {
		if v, ok2 := uid.(int32); ok2 {
			userID = v
		}
	}

	// Cancelar detalle (soft‑delete)
	if err := h.queries.CancelarDetallePedido(c.Request.Context(), db.CancelarDetallePedidoParams{
		ID:        int32(detalleID),
		DeletedBy: pgtype.Int4{Int32: userID, Valid: userID != 0},
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al cancelar detalle: " + err.Error()})
		return
	}

	// Contar los detalles activos restantes
	detalles, _ := h.queries.ListarPedidosDetalle(c.Request.Context(), pgtype.Int4{Int32: int32(pedidoID), Valid: true})
	activos := 0
	for _, d := range detalles {
		if !d.DeletedAt.Valid {
			activos++
		}
	}

	// Si queda 0 o 1 detalle activo, cancelar el pedido completo
	if activos <= 1 {
		_, err := h.queries.ActualizarEstadoPedido(c.Request.Context(), db.ActualizarEstadoPedidoParams{
			ID:        int32(pedidoID),
			Estado:    "Cancelado",
			UpdatedBy: pgtype.Int4{Int32: userID, Valid: userID != 0},
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al cancelar pedido completo: " + err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Detalle cancelado"})
}
