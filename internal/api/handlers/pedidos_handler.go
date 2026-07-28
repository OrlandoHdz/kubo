package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/OrlandoHdz/kubo/internal/db"
	"github.com/OrlandoHdz/kubo/pkg/email"
	"github.com/OrlandoHdz/kubo/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PedidosHandler struct {
	queries  *db.Queries
	pool     *pgxpool.Pool
	emailCfg *email.Config
}

func NewPedidosHandler(q *db.Queries, p *pgxpool.Pool, e *email.Config) *PedidosHandler {
	return &PedidosHandler{queries: q, pool: p, emailCfg: e}
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

	tx, err := h.pool.Begin(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al iniciar transacción: " + err.Error()})
		return
	}
	defer tx.Rollback(c.Request.Context())

	qtx := h.queries.WithTx(tx)

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

	folio := fmt.Sprintf("ARW-%d", time.Now().Unix())

	pedido, err := qtx.CrearPedido(c.Request.Context(), db.CrearPedidoParams{
		Folio:      folio,
		ClienteID:  pgtype.Int4{Int32: input.ClienteID, Valid: true},
		UsuarioID:  pgtype.Int4{Int32: input.UsuarioID, Valid: true},
		Estado:     "Pendiente",
		MetodoPago: input.MetodoPago,
		Subtotal:   utils.ToNumeric(input.Subtotal),
		Iva:        utils.ToNumeric(input.Iva),
		TotalOrden: utils.ToNumeric(input.TotalOrden),
		CreatedBy:  pgtype.Int4{Int32: input.UsuarioID, Valid: true},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear encabezado de pedido: " + err.Error()})
		return
	}

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
	}

	if err := tx.Commit(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al confirmar pedido: " + err.Error()})
		return
	}

	// Enviar notificación por correo en segundo plano
	log.Printf("Email: emailCfg es %v", h.emailCfg != nil)
	if h.emailCfg != nil {
		log.Printf("Email: lanzando notificación en goroutine para pedido %d", pedido.ID)
		go h.notificarNuevoPedido(context.Background(), pedido.ID)
	} else {
		log.Printf("Email: emailCfg es nil, no se enviará notificación")
	}

	c.JSON(http.StatusCreated, pedido)
}

func (h *PedidosHandler) notificarNuevoPedido(ctx context.Context, pedidoID int32) {
	log.Printf("Email: notificarNuevoPedido iniciado para pedido %d", pedidoID)

	pedido, err := h.queries.GetPedido(ctx, pedidoID)
	if err != nil {
		log.Printf("Email: error al obtener pedido %d: %v", pedidoID, err)
		return
	}
	log.Printf("Email: pedido %s obtenido, cliente_id valid=%v", pedido.Folio, pedido.ClienteID.Valid)

	if !pedido.ClienteID.Valid {
		log.Printf("Email: pedido %d sin cliente_id", pedidoID)
		return
	}

	cliente, err := h.queries.GetCliente(ctx, pedido.ClienteID.Int32)
	if err != nil {
		log.Printf("Email: error al obtener cliente %d: %v", pedido.ClienteID.Int32, err)
		return
	}
	log.Printf("Email: cliente %s obtenido", cliente.NombreComercial)

	detalles, err := h.queries.ListarPedidosDetalle(ctx, pgtype.Int4{Int32: pedidoID, Valid: true})
	if err != nil {
		log.Printf("Email: error al obtener detalles del pedido %d: %v", pedidoID, err)
		return
	}
	log.Printf("Email: %d detalles obtenidos", len(detalles))

	usuarios, err := h.queries.ListarUsuariosPorCliente(ctx, pedido.ClienteID)
	if err != nil || len(usuarios) == 0 {
		log.Printf("Email: cliente %d sin usuarios registrados (err=%v)", pedido.ClienteID.Int32, err)
		return
	}
	log.Printf("Email: %d usuarios encontrados, email cliente=%s", len(usuarios), usuarios[0].Email)

	subtotal, _ := utils.NumericToFloat64(pedido.Subtotal)
	iva, _ := utils.NumericToFloat64(pedido.Iva)
	total, _ := utils.NumericToFloat64(pedido.TotalOrden)

	var orderItems []email.OrderItemData
	for _, d := range detalles {
		precio, _ := utils.NumericToFloat64(d.PrecioUnitarioAplicado)
		importe := precio * float64(d.Cantidad)
		orderItems = append(orderItems, email.OrderItemData{
			SKU:         d.VarianteSku.String,
			Descripcion: d.PadreDescripcion.String,
			Cantidad:    d.Cantidad,
			Precio:      formatPrice(precio),
			Importe:     formatPrice(importe),
		})
	}

	orderData := email.OrderData{
		Folio:       pedido.Folio,
		Fecha:       pedido.CreatedAt.Time.Format("02/01/2006 15:04"),
		MetodoPago:  pedido.MetodoPago,
		Subtotal:    formatPrice(subtotal),
		Iva:         formatPrice(iva),
		Total:       formatPrice(total),
		ClienteName: cliente.NombreComercial,
		ClienteID:   pedido.ClienteID.Int32,
		Items:       orderItems,
	}

	h.emailCfg.SendOrderNotification(orderData, usuarios[0].Email)
}

func formatPrice(v float64) string {
	return fmt.Sprintf("%.2f", v)
}

func (h *PedidosHandler) Listar(c *gin.Context) {
	fechaInicio, fechaFin := parseDateRange(c)
	pedidos, err := h.queries.ListarPedidosPorRango(c.Request.Context(), db.ListarPedidosPorRangoParams{
		CreatedAt:   fechaInicio,
		CreatedAt_2: fechaFin,
	})
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

	fechaInicio, fechaFin := parseDateRange(c)
	pedidos, err := h.queries.ListarPedidosPorClienteRango(c.Request.Context(), db.ListarPedidosPorClienteRangoParams{
		ClienteID:   pgtype.Int4{Int32: int32(clienteID), Valid: true},
		CreatedAt:   fechaInicio,
		CreatedAt_2: fechaFin,
	})
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

	// Notificar al cliente cuando el estado cambie a En Tránsito o Entregado
	if h.emailCfg != nil && (input.Estado == "En Transito" || input.Estado == "Entregado" || input.Estado == "Cancelado") {
		go h.notificarEstadoPedido(context.Background(), int32(id), input.Estado, input.NotasAdmin, input.Guia)
	}

	c.JSON(http.StatusOK, pedidoActualizado)
}

func (h *PedidosHandler) notificarEstadoPedido(ctx context.Context, pedidoID int32, estado, notasAdmin, guia string) {
	log.Printf("Email: notificarEstadoPedido iniciado para pedido %d, estado=%s", pedidoID, estado)

	pedido, err := h.queries.GetPedido(ctx, pedidoID)
	if err != nil {
		log.Printf("Email: error al obtener pedido %d: %v", pedidoID, err)
		return
	}

	if !pedido.ClienteID.Valid {
		log.Printf("Email: pedido %d sin cliente_id", pedidoID)
		return
	}

	cliente, err := h.queries.GetCliente(ctx, pedido.ClienteID.Int32)
	if err != nil {
		log.Printf("Email: error al obtener cliente %d: %v", pedido.ClienteID.Int32, err)
		return
	}

	detalles, err := h.queries.ListarPedidosDetalle(ctx, pgtype.Int4{Int32: pedidoID, Valid: true})
	if err != nil {
		log.Printf("Email: error al obtener detalles del pedido %d: %v", pedidoID, err)
		return
	}

	usuarios, err := h.queries.ListarUsuariosPorCliente(ctx, pedido.ClienteID)
	if err != nil || len(usuarios) == 0 {
		log.Printf("Email: cliente %d sin usuarios registrados (err=%v)", pedido.ClienteID.Int32, err)
		return
	}

	subtotal, _ := utils.NumericToFloat64(pedido.Subtotal)
	iva, _ := utils.NumericToFloat64(pedido.Iva)
	total, _ := utils.NumericToFloat64(pedido.TotalOrden)

	var orderItems []email.OrderItemData
	for _, d := range detalles {
		precio, _ := utils.NumericToFloat64(d.PrecioUnitarioAplicado)
		importe := precio * float64(d.Cantidad)
		orderItems = append(orderItems, email.OrderItemData{
			SKU:         d.VarianteSku.String,
			Descripcion: d.PadreDescripcion.String,
			Cantidad:    d.Cantidad,
			Precio:      formatPrice(precio),
			Importe:     formatPrice(importe),
		})
	}

	if guia == "" {
		guia = "—"
	}

	orderData := email.OrderData{
		Folio:       pedido.Folio,
		Fecha:       pedido.CreatedAt.Time.Format("02/01/2006 15:04"),
		MetodoPago:  pedido.MetodoPago,
		Subtotal:    formatPrice(subtotal),
		Iva:         formatPrice(iva),
		Total:       formatPrice(total),
		ClienteName: cliente.NombreComercial,
		ClienteID:   pedido.ClienteID.Int32,
		Status:      estado,
		NotasAdmin:  notasAdmin,
		Guia:        guia,
		Items:       orderItems,
	}

	h.emailCfg.SendOrderStatusNotification(orderData, usuarios[0].Email)
}

func (h *PedidosHandler) AgregarProductos(c *gin.Context) {
	pedidoIDStr := c.Param("id")
	pedidoID, err := strconv.Atoi(pedidoIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de pedido inválido"})
		return
	}

	var input AgregarProductosInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
		return
	}

	pedido, err := h.queries.GetPedido(c.Request.Context(), int32(pedidoID))
	if err != nil || pedido.Estado != "Pendiente" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Solo se pueden agregar productos a pedidos pendientes"})
		return
	}

	if !h.validarVentanaModificacion(c, int32(pedidoID)) {
		return
	}

	tx, err := h.pool.Begin(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al iniciar transacción: " + err.Error()})
		return
	}
	defer tx.Rollback(c.Request.Context())
	qtx := h.queries.WithTx(tx)

	var addedSubtotal float64
	for _, item := range input.Items {
		_, err := qtx.CrearPedidoDetalle(c.Request.Context(), db.CrearPedidoDetalleParams{
			PedidoID:               pgtype.Int4{Int32: int32(pedidoID), Valid: true},
			VarianteID:             pgtype.Int4{Int32: item.VarianteID, Valid: true},
			Cantidad:               item.Cantidad,
			PrecioUnitarioAplicado: utils.ToNumeric(item.Precio),
			CreatedBy:              pgtype.Int4{Int32: int32(0), Valid: false},
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error al crear detalle: %s", err.Error())})
			return
		}
		addedSubtotal += item.Precio * float64(item.Cantidad)
	}

	_, err = tx.Exec(c.Request.Context(), "UPDATE pedidos SET subtotal = subtotal + $1, total_orden = total_orden + $1 WHERE id = $2", utils.ToNumeric(addedSubtotal), int32(pedidoID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar totales del pedido: " + err.Error()})
		return
	}

	if err := tx.Commit(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al confirmar transacción: " + err.Error()})
		return
	}

	updatedPedido, err := h.queries.GetPedido(c.Request.Context(), int32(pedidoID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener pedido actualizado"})
		return
	}
	c.JSON(http.StatusOK, updatedPedido)
}

type ShipItemInput struct {
	DetalleID       int32 `json:"detalle_id" binding:"required"`
	ShippedQuantity int32 `json:"shipped_quantity" binding:"required"`
}

type ShipOrderInput struct {
	Items     []ShipItemInput `json:"items" binding:"required,min=1"`
	Guia      string          `json:"guia"`
	NotasAdmin string         `json:"notas_admin"`
}

type ShipItemResult struct {
	DetalleID         int32  `json:"detalle_id"`
	Producto          string `json:"producto"`
	SKU               string `json:"sku"`
	CantidadOriginal  int32  `json:"cantidad_original"`
	CantidadEnviada   int32  `json:"cantidad_enviada"`
	Backorder         int32  `json:"backorder"`
}

func (h *PedidosHandler) ShipOrder(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de pedido inválido"})
		return
	}

	var input ShipOrderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos de envío inválidos: " + err.Error()})
		return
	}

	userID := int32(0)
	if uid, ok := c.Get("userID"); ok {
		if v, ok2 := uid.(int32); ok2 {
			userID = v
		}
	}

	tx, err := h.pool.Begin(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al iniciar transacción: " + err.Error()})
		return
	}
	defer tx.Rollback(c.Request.Context())

	qtx := h.queries.WithTx(tx)

	pedido, err := qtx.GetPedido(c.Request.Context(), int32(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pedido no encontrado"})
		return
	}

	if pedido.Estado == "Entregado" || pedido.Estado == "Cancelado" {
		c.JSON(http.StatusForbidden, gin.H{"error": "No se puede embarcar un pedido " + pedido.Estado})
		return
	}

	detalles, err := qtx.ListarPedidosDetalle(c.Request.Context(), pgtype.Int4{Int32: int32(id), Valid: true})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener detalles del pedido: " + err.Error()})
		return
	}

	detalleMap := make(map[int32]db.ListarPedidosDetalleRow)
	for _, d := range detalles {
		detalleMap[d.ID] = d
	}

	hasBackorder := false
	allShipped := true
	var results []ShipItemResult

	for _, item := range input.Items {
		detalle, exists := detalleMap[item.DetalleID]
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Detalle %d no pertenece a este pedido", item.DetalleID)})
			return
		}

		if item.ShippedQuantity < 0 || item.ShippedQuantity > detalle.Cantidad {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf(
					"Cantidad inválida para el producto %s. Enviado: %d, Original: %d",
					detalle.VarianteSku.String, item.ShippedQuantity, detalle.Cantidad,
				),
			})
			return
		}

		backorderQty := detalle.Cantidad - item.ShippedQuantity

		_, err := qtx.ActualizarEnvioPedidoDetalle(c.Request.Context(), db.ActualizarEnvioPedidoDetalleParams{
			ID:                item.DetalleID,
			ShippedQuantity:   item.ShippedQuantity,
			BackorderQuantity: backorderQty,
			UpdatedBy:         pgtype.Int4{Int32: userID, Valid: userID != 0},
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar detalle: " + err.Error()})
			return
		}

		_, err = qtx.CrearModificacionPedido(c.Request.Context(), db.CrearModificacionPedidoParams{
			OrderID:           pgtype.Int4{Int32: int32(id), Valid: true},
			UserID:            pgtype.Int4{Int32: userID, Valid: userID != 0},
			ItemID:            pgtype.Int4{Int32: item.DetalleID, Valid: true},
			OriginalQuantity:  detalle.Cantidad,
			ShippedQuantity:   item.ShippedQuantity,
			BackorderQuantity: backorderQty,
			Notes:             pgtype.Text{String: input.NotasAdmin, Valid: input.NotasAdmin != ""},
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al registrar auditoría: " + err.Error()})
			return
		}

		if backorderQty > 0 {
			hasBackorder = true
		}
		if item.ShippedQuantity == 0 {
			allShipped = false
		}

		results = append(results, ShipItemResult{
			DetalleID:        item.DetalleID,
			Producto:         detalle.PadreDescripcion.String,
			SKU:              detalle.VarianteSku.String,
			CantidadOriginal: detalle.Cantidad,
			CantidadEnviada:  item.ShippedQuantity,
			Backorder:        backorderQty,
		})
	}

	nuevoEstado := "En Transito"
	if hasBackorder {
		nuevoEstado = "En Transito"
	}
	if hasBackorder && !allShipped {
		nuevoEstado = "En Transito"
	}

	_, err = qtx.ActualizarHasBackorderPedido(c.Request.Context(), db.ActualizarHasBackorderPedidoParams{
		ID:           int32(id),
		HasBackorder: hasBackorder,
		Estado:       nuevoEstado,
		UpdatedBy:    pgtype.Int4{Int32: userID, Valid: userID != 0},
		Guia:         pgtype.Text{String: input.Guia, Valid: input.Guia != ""},
		NotasAdmin:   pgtype.Text{String: input.NotasAdmin, Valid: input.NotasAdmin != ""},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar pedido: " + err.Error()})
		return
	}

	if err := tx.Commit(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al confirmar envío: " + err.Error()})
		return
	}

	if h.emailCfg != nil {
		go h.notificarEstadoPedido(context.Background(), int32(id), nuevoEstado, input.NotasAdmin, input.Guia)
	}

	resumen := gin.H{
		"message":      "Envío registrado exitosamente",
		"pedido_id":    id,
		"folio":        pedido.Folio,
		"estado":       nuevoEstado,
		"has_backorder": hasBackorder,
		"items":        results,
	}

	c.JSON(http.StatusOK, resumen)
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

func parseDateRange(c *gin.Context) (pgtype.Timestamp, pgtype.Timestamp) {
	fechaInicioStr := c.Query("fecha_inicio")
	fechaFinStr := c.Query("fecha_fin")

	now := time.Now()

	inicio := now.AddDate(0, -1, 0)
	if fechaInicioStr != "" {
		if t, err := time.Parse(time.RFC3339, fechaInicioStr); err == nil {
			inicio = t
		} else if t, err := time.Parse("2006-01-02", fechaInicioStr); err == nil {
			inicio = t
		}
	}

	fin := now
	if fechaFinStr != "" {
		if t, err := time.Parse(time.RFC3339, fechaFinStr); err == nil {
			fin = t
		} else if t, err := time.Parse("2006-01-02", fechaFinStr); err == nil {
			fin = t.Add(24*time.Hour - time.Second)
		}
	}

	return pgtype.Timestamp{Time: inicio, Valid: true}, pgtype.Timestamp{Time: fin, Valid: true}
}
