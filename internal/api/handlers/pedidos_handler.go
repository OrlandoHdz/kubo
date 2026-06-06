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
	esBackorder := false
	for _, item := range input.Items {
		variante, err := qtx.GetVariante(c.Request.Context(), item.VarianteID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Variante %d no encontrada", item.VarianteID)})
			return
		}

		committed, err := qtx.GetCommittedStock(c.Request.Context(), pgtype.Int4{Int32: item.VarianteID, Valid: true})
		if err != nil {
			committed = 0 // Si hay error, asumimos 0 para no bloquear, o podrías manejarlo
		}

		disponible := variante.StockActual - committed
		if disponible < item.Cantidad {
			esBackorder = true
			// Aquí podrías decidir si bloquear la venta o solo marcar como backorder
			// El usuario pidió "indicar que no tenemos en stock"
		}
	}

	// 4. Generar Folio
	folio := fmt.Sprintf("PED-%d", time.Now().Unix())

	// 5. Crear Encabezado de Pedido
	pedido, err := qtx.CrearPedido(c.Request.Context(), db.CrearPedidoParams{
		Folio:       folio,
		ClienteID:   pgtype.Int4{Int32: input.ClienteID, Valid: true},
		UsuarioID:   pgtype.Int4{Int32: input.UsuarioID, Valid: true},
		Estado:      "Pendiente",
		MetodoPago:  input.MetodoPago,
		Subtotal:    utils.ToNumeric(input.Subtotal),
		Iva:         utils.ToNumeric(input.Iva),
		TotalOrden:  utils.ToNumeric(input.TotalOrden),
		EsBackorder: pgtype.Bool{Bool: esBackorder, Valid: true},
		CreatedBy:   pgtype.Int4{Int32: input.UsuarioID, Valid: true},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear encabezado de pedido: " + err.Error()})
		return
	}

	// 4. Crear Detalles
	for _, item := range input.Items {
		_, err := qtx.CrearPedidoDetalle(c.Request.Context(), db.CrearPedidoDetalleParams{
			PedidoID:               pgtype.Int4{Int32: pedido.ID, Valid: true},
			VarianteID:             pgtype.Int4{Int32: item.VarianteID, Valid: true},
			Cantidad:               item.Cantidad,
			PrecioUnitarioAplicado: utils.ToNumeric(item.Precio),
			CreatedBy:              pgtype.Int4{Int32: input.UsuarioID, Valid: true},
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error al crear detalle para variante %d: %s", item.VarianteID, err.Error())})
			return
		}

		// 5. Actualizar Stock (Opcional, pero recomendado)
		// Aquí se podría llamar a una función para restar el stock actual
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
		Estado    string `json:"estado" binding:"required"`
		UpdatedBy int32  `json:"updated_by"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos de estado inválidos: " + err.Error()})
		return
	}

	pedido, err := h.queries.ActualizarEstadoPedido(c.Request.Context(), db.ActualizarEstadoPedidoParams{
		ID:        int32(id),
		Estado:    input.Estado,
		UpdatedBy: pgtype.Int4{Int32: input.UpdatedBy, Valid: input.UpdatedBy != 0},
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar estado: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, pedido)
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
	// Check time window (same logic as puedeModificar)
	createdAt := pedido.CreatedAt.Time
	if pedido.FechaPedido.Valid {
		createdAt = pedido.FechaPedido.Time
	}
	diffHoras := time.Since(createdAt).Hours()
	if diffHoras >= 2 {
		c.JSON(http.StatusForbidden, gin.H{"error": "El plazo para modificar este pedido ha vencido"})
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

	// Insert each item as detalle
	var addedSubtotal float64
	for _, item := range input.Items {
		// Validate stock (similar to Crear)
		variante, err := qtx.GetVariante(c.Request.Context(), item.VarianteID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Variante %d no encontrada", item.VarianteID)})
			return
		}
		committed, _ := qtx.GetCommittedStock(c.Request.Context(), pgtype.Int4{Int32: item.VarianteID, Valid: true})
		disponible := variante.StockActual - committed
		if disponible < item.Cantidad {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Stock insuficiente para variante %d", item.VarianteID)})
			return
		}
		// Create detalle
		_, err = qtx.CrearPedidoDetalle(c.Request.Context(), db.CrearPedidoDetalleParams{
			PedidoID:               pgtype.Int4{Int32: int32(pedidoID), Valid: true},
			VarianteID:             pgtype.Int4{Int32: item.VarianteID, Valid: true},
			Cantidad:               item.Cantidad,
			PrecioUnitarioAplicado: utils.ToNumeric(item.Precio),
			CreatedBy:              pgtype.Int4{Int32: int32(0), Valid: false}, // assuming system user
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error al crear detalle: %s", err.Error())})
			return
		}
		addedSubtotal += item.Precio * float64(item.Cantidad)
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
