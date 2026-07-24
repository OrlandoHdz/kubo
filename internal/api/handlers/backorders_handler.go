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

type BackordersHandler struct {
	queries  *db.Queries
	pool     *pgxpool.Pool
	emailCfg *email.Config
}

func NewBackordersHandler(q *db.Queries, p *pgxpool.Pool, e *email.Config) *BackordersHandler {
	return &BackordersHandler{queries: q, pool: p, emailCfg: e}
}

type CrearBackorderInput struct {
	ClienteID         int32                `json:"cliente_id" binding:"required"`
	UsuarioID         int32                `json:"usuario_id" binding:"required"`
	MetodoPago        string               `json:"metodo_pago" binding:"required"`
	Subtotal          float64              `json:"subtotal" binding:"required"`
	Iva               float64              `json:"iva" binding:"required"`
	TotalOrden        float64              `json:"total_orden" binding:"required"`
	Items             []PedidoDetalleInput `json:"items" binding:"required,min=1"`
	PedidoOrigenFolio string               `json:"pedido_origen_folio"`
}

func (h *BackordersHandler) Listar(c *gin.Context) {
	fechaInicio, fechaFin := parseDateRange(c)
	backorders, err := h.queries.ListarBackordersPorRango(c.Request.Context(), db.ListarBackordersPorRangoParams{
		CreatedAt:   fechaInicio,
		CreatedAt_2: fechaFin,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al listar backorders: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, backorders)
}

func (h *BackordersHandler) ListarPorCliente(c *gin.Context) {
	clienteIDStr := c.Param("cliente_id")
	clienteID, err := strconv.Atoi(clienteIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de cliente inválido"})
		return
	}

	fechaInicio, fechaFin := parseDateRange(c)
	backorders, err := h.queries.ListarBackordersPorClienteRango(c.Request.Context(), db.ListarBackordersPorClienteRangoParams{
		ClienteID:   pgtype.Int4{Int32: int32(clienteID), Valid: true},
		CreatedAt:   fechaInicio,
		CreatedAt_2: fechaFin,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al listar backorders del cliente: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, backorders)
}

func (h *BackordersHandler) Obtener(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de backorder inválido"})
		return
	}

	backorder, err := h.queries.GetBackorder(c.Request.Context(), int32(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Backorder no encontrado"})
		return
	}

	detalles, err := h.queries.GetBackorderDetalles(c.Request.Context(), pgtype.Int4{Int32: int32(id), Valid: true})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener detalles del backorder: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"backorder": backorder,
		"detalles":  detalles,
	})
}

func (h *BackordersHandler) Crear(c *gin.Context) {
	var input CrearBackorderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos de backorder inválidos: " + err.Error()})
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

	var pedidoOrigenID pgtype.Int4
	if input.PedidoOrigenFolio != "" {
		pedidoOrigen, err := qtx.GetPedidoByFolio(c.Request.Context(), input.PedidoOrigenFolio)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Pedido origen no encontrado para el folio proporcionado"})
			return
		}
		pedidoOrigenID = pgtype.Int4{Int32: pedidoOrigen.ID, Valid: true}
	}

	folio := fmt.Sprintf("BKO-%d", time.Now().Unix())

	backorder, err := qtx.CrearBackorder(c.Request.Context(), db.CrearBackorderParams{
		Folio:               folio,
		ClienteID:           pgtype.Int4{Int32: input.ClienteID, Valid: true},
		UsuarioID:           pgtype.Int4{Int32: input.UsuarioID, Valid: true},
		EstadoBackorder:     "Pendiente",
		MetodoPago:          input.MetodoPago,
		Subtotal:            utils.ToNumeric(input.Subtotal),
		Iva:                 utils.ToNumeric(input.Iva),
		TotalOrden:          utils.ToNumeric(input.TotalOrden),
		GuiaBackorder:       pgtype.Text{String: "", Valid: false},
		NotasAdminBackorder: pgtype.Text{String: "", Valid: false},
		PedidoOrigenID:      pedidoOrigenID,
		CreatedBy:           pgtype.Int4{Int32: input.UsuarioID, Valid: true},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear backorder: " + err.Error()})
		return
	}

	for _, item := range input.Items {
		_, err := qtx.CrearBackorderDetalle(c.Request.Context(), db.CrearBackorderDetalleParams{
			BackorderID:            pgtype.Int4{Int32: backorder.ID, Valid: true},
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al confirmar backorder: " + err.Error()})
		return
	}

	// Enviar notificación por correo en segundo plano
	log.Printf("Email: emailCfg es %v para backorder", h.emailCfg != nil)
	if h.emailCfg != nil {
		log.Printf("Email: lanzando notificación en goroutine para backorder %d", backorder.ID)
		go h.notificarNuevoBackorder(context.Background(), backorder.ID)
	} else {
		log.Printf("Email: emailCfg es nil, no se enviará notificación de backorder")
	}

	c.JSON(http.StatusCreated, backorder)
}

func (h *BackordersHandler) ActualizarEstado(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de backorder inválido"})
		return
	}

	var input struct {
		EstadoBackorder     string `json:"estado_backorder" binding:"required"`
		GuiaBackorder       string `json:"guia_backorder"`
		NotasAdminBackorder string `json:"notas_admin_backorder"`
		UpdatedBy           int32  `json:"updated_by"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
		return
	}

	backorderActualizado, err := h.queries.ActualizarEstadoBackorder(c.Request.Context(), db.ActualizarEstadoBackorderParams{
		ID:                  int32(id),
		EstadoBackorder:     input.EstadoBackorder,
		UpdatedBy:           pgtype.Int4{Int32: input.UpdatedBy, Valid: input.UpdatedBy != 0},
		GuiaBackorder:       pgtype.Text{String: input.GuiaBackorder, Valid: input.GuiaBackorder != ""},
		NotasAdminBackorder: pgtype.Text{String: input.NotasAdminBackorder, Valid: input.NotasAdminBackorder != ""},
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar estado del backorder: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, backorderActualizado)
}

func (h *BackordersHandler) MarcarDetalleDisponible(c *gin.Context) {
	idStr := c.Param("id")
	detalleIDStr := c.Param("detalle_id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de backorder inválido"})
		return
	}
	detalleID, err := strconv.Atoi(detalleIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de detalle inválido"})
		return
	}

	usuarioID := int32(0)
	if uid, ok := c.Get("userID"); ok {
		if v, ok2 := uid.(int32); ok2 {
			usuarioID = v
		}
	}

	detalleActualizado, err := h.queries.MarcarDisponibleBackorderDetalle(c.Request.Context(), db.MarcarDisponibleBackorderDetalleParams{
		ID:            int32(detalleID),
		BackorderID:   pgtype.Int4{Int32: int32(id), Valid: true},
		UpdatedBy:     pgtype.Int4{Int32: usuarioID, Valid: usuarioID != 0},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al marcar detalle como disponible: " + err.Error()})
		return
	}

	// Enviar notificación por correo en segundo plano
	if h.emailCfg != nil && h.emailCfg.Password != "" {
		go h.notificarDisponibilidad(context.Background(), int32(id), int32(detalleID))
	}

	c.JSON(http.StatusOK, detalleActualizado)
}

func (h *BackordersHandler) notificarNuevoBackorder(ctx context.Context, backorderID int32) {
	log.Printf("Email: notificarNuevoBackorder iniciado para backorder %d", backorderID)

	backorder, err := h.queries.GetBackorder(ctx, backorderID)
	if err != nil {
		log.Printf("Email: error al obtener backorder %d: %v", backorderID, err)
		return
	}
	log.Printf("Email: backorder %s obtenido, cliente_id valid=%v", backorder.Folio, backorder.ClienteID.Valid)

	if !backorder.ClienteID.Valid {
		log.Printf("Email: backorder %d sin cliente_id", backorderID)
		return
	}

	cliente, err := h.queries.GetCliente(ctx, backorder.ClienteID.Int32)
	if err != nil {
		log.Printf("Email: error al obtener cliente %d: %v", backorder.ClienteID.Int32, err)
		return
	}
	log.Printf("Email: cliente %s obtenido", cliente.NombreComercial)

	detalles, err := h.queries.GetBackorderDetalles(ctx, pgtype.Int4{Int32: backorderID, Valid: true})
	if err != nil {
		log.Printf("Email: error al obtener detalles del backorder %d: %v", backorderID, err)
		return
	}
	log.Printf("Email: %d detalles obtenidos", len(detalles))

	usuarios, err := h.queries.ListarUsuariosPorCliente(ctx, backorder.ClienteID)
	if err != nil || len(usuarios) == 0 {
		log.Printf("Email: cliente %d sin usuarios registrados (err=%v)", backorder.ClienteID.Int32, err)
		return
	}
	log.Printf("Email: %d usuarios encontrados, email cliente=%s", len(usuarios), usuarios[0].Email)

	subtotal, _ := utils.NumericToFloat64(backorder.Subtotal)
	iva, _ := utils.NumericToFloat64(backorder.Iva)
	total, _ := utils.NumericToFloat64(backorder.TotalOrden)

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

	fecha := backorder.CreatedAt.Time
	if backorder.FechaBackorder.Valid {
		fecha = backorder.FechaBackorder.Time
	}

	boData := email.BackorderData{
		Folio:       backorder.Folio,
		Fecha:       fecha.Format("02/01/2006 15:04"),
		MetodoPago:  backorder.MetodoPago,
		Subtotal:    formatPrice(subtotal),
		Iva:         formatPrice(iva),
		Total:       formatPrice(total),
		ClienteName: cliente.NombreComercial,
		ClienteID:   backorder.ClienteID.Int32,
		Items:       orderItems,
	}

	h.emailCfg.SendBackorderNotification(boData, usuarios[0].Email)
}

func (h *BackordersHandler) notificarDisponibilidad(ctx context.Context, backorderID, detalleID int32) {
	backorder, err := h.queries.GetBackorder(ctx, backorderID)
	if err != nil {
		log.Printf("Email: error al obtener backorder %d: %v", backorderID, err)
		return
	}

	if !backorder.ClienteID.Valid {
		log.Printf("Email: backorder %d sin cliente_id", backorderID)
		return
	}

	detalles, err := h.queries.GetBackorderDetalles(ctx, pgtype.Int4{Int32: backorderID, Valid: true})
	if err != nil {
		log.Printf("Email: error al obtener detalles del backorder %d: %v", backorderID, err)
		return
	}

	var detalleDesc, varianteSKU string
	for _, d := range detalles {
		if d.ID == detalleID {
			detalleDesc = d.PadreDescripcion.String
			varianteSKU = d.VarianteSku.String
			break
		}
	}
	if detalleDesc == "" {
		detalleDesc = fmt.Sprintf("Detalle #%d", detalleID)
	}

	usuarios, err := h.queries.ListarUsuariosPorCliente(ctx, backorder.ClienteID)
	if err != nil {
		log.Printf("Email: error al listar usuarios del cliente %d: %v", backorder.ClienteID.Int32, err)
		return
	}
	if len(usuarios) == 0 {
		log.Printf("Email: cliente %d sin usuarios registrados", backorder.ClienteID.Int32)
		return
	}

	to := usuarios[0].Email
	subject := fmt.Sprintf("Stock Disponible — Backorder %s", backorder.Folio)
	body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: Arial, sans-serif; margin: 0; padding: 0; background: #f5f5f5;">
  <table width="100%%" cellpadding="0" cellspacing="0" style="background: #f5f5f5; padding: 20px;">
    <tr>
      <td align="left">
        <table width="600" cellpadding="0" cellspacing="0" style="background: #ffffff; border-radius: 8px; overflow: hidden; box-shadow: 0 2px 10px rgba(0,0,0,0.05);">
          <tr>
            <td style="background: linear-gradient(135deg, #c9540e, #f57c1a); padding: 25px; text-align: center;">
              <img src="https://kubo-producto.oss-us-east-1.aliyuncs.com/logos/arsenal_logo1t.png" alt="Arsenal Welds" style="max-height: 55px; margin-bottom: 10px;">
              <h1 style="color: #ffffff; margin: 10px 0 0; font-size: 22px;">¡Stock Disponible!</h1>
            </td>
          </tr>
          <tr>
            <td style="padding: 30px;">
              <p style="font-size: 16px; color: #333;">Estimado cliente,</p>
              <p style="font-size: 15px; color: #555; line-height: 1.6;">
                Nos complace informarle que el siguiente producto de su backorder 
                <strong style="color: #c9540e;">%s</strong> ya tiene stock disponible:
              </p>

              <table width="100%%" cellpadding="0" cellspacing="0" style="margin: 20px 0; background: #fef8f4; border: 1px solid #f5e0d0; border-radius: 6px;">
                <tr>
                  <td style="padding: 12px; width: 40%%; font-weight: bold; color: #333; border-bottom: 1px solid #f5e0d0;">Producto</td>
                  <td style="padding: 12px; color: #333; border-bottom: 1px solid #f5e0d0;">%s</td>
                </tr>
                <tr>
                  <td style="padding: 12px; width: 40%%; font-weight: bold; color: #333; border-bottom: 1px solid #f5e0d0;">SKU</td>
                  <td style="padding: 12px; color: #c9540e; font-weight: bold; border-bottom: 1px solid #f5e0d0;">%s</td>
                </tr>
                <tr>
                  <td style="padding: 12px; width: 40%%; font-weight: bold; color: #333; border-bottom: 1px solid #f5e0d0;">Backorder</td>
                  <td style="padding: 12px; color: #333; border-bottom: 1px solid #f5e0d0;">%s</td>
                </tr>
              </table>

              <p style="font-size: 15px; color: #555; line-height: 1.6;">
                Para completar su compra, ingrese a nuestro portal y genere su pedido:
              </p>

              <table width="100%%" cellpadding="0" cellspacing="0" style="margin: 20px 0;">
                <tr>
                    <td align="left">
                    <a href="https://www.arsenalwelds.com" style="display: inline-block; background: linear-gradient(135deg, #c9540e, #f57c1a); color: #ffffff; text-decoration: none; padding: 14px 40px; border-radius: 6px; font-size: 16px; font-weight: bold;">www.arsenalwelds.com →</a>
                  </td>
                </tr>
              </table>

              <p style="font-size: 15px; color: #555; line-height: 1.6;">
                Atentamente,<br>
                <strong>Arsenal Welds</strong>
              </p>
            </td>
          </tr>
          <tr>
            <td style="background: #f0f0f0; padding: 15px; text-align: center; font-size: 12px; color: #999;">
              Arsenal Welds — ventas@arsenalwelds.com
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`, backorder.Folio, detalleDesc, varianteSKU, backorder.Folio)

	recipients := []string{to}
	recipients = append(recipients, h.emailCfg.DefaultRecipients...)

	if err := h.emailCfg.SendToMultiple(recipients, subject, body); err != nil {
		log.Printf("Email: error al enviar notificación de disponibilidad a %s: %v", to, err)
		return
	}
	log.Printf("Email: notificación de disponibilidad enviada a %s y %d copias (backorder %s, detalle %d)", to, len(h.emailCfg.DefaultRecipients), backorder.Folio, detalleID)
}

func (h *BackordersHandler) ConvertirAPedido(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de backorder inválido"})
		return
	}

	usuarioID := int32(0)
	if uid, ok := c.Get("userID"); ok {
		if v, ok2 := uid.(int32); ok2 {
			usuarioID = v
		}
	}

	backorder, err := h.queries.GetBackorder(c.Request.Context(), int32(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Backorder no encontrado"})
		return
	}

	if backorder.EstadoBackorder != "Stock Disponible" {
		c.JSON(http.StatusForbidden, gin.H{"error": "El backorder debe estar en estado 'Stock Disponible' para convertirlo a pedido"})
		return
	}

	tx, err := h.pool.Begin(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al iniciar transacción: " + err.Error()})
		return
	}
	defer tx.Rollback(c.Request.Context())

	qtx := h.queries.WithTx(tx)

	detalles, err := qtx.GetBackorderDetalles(c.Request.Context(), pgtype.Int4{Int32: backorder.ID, Valid: true})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener detalles del backorder: " + err.Error()})
		return
	}

	for _, d := range detalles {
		stockReal, err := qtx.GetStockRealVariante(c.Request.Context(), d.VarianteID.Int32)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error al verificar stock de variante %d: %s", d.VarianteID.Int32, err.Error())})
			return
		}
		stockRealFloat, _ := utils.NumericToFloat64(stockReal)
		disponible := int32(stockRealFloat)
		if disponible < d.Cantidad {
			c.JSON(http.StatusForbidden, gin.H{
				"error":      fmt.Sprintf("Stock insuficiente para variante %d. Stock real: %d, Requerido: %d", d.VarianteID.Int32, disponible, d.Cantidad),
				"variante":   d.VarianteID.Int32,
				"disponible": disponible,
				"requerido":  d.Cantidad,
			})
			return
		}
	}

	folio := fmt.Sprintf("ARW-%d", time.Now().Unix())

	subtotalFloat, _ := utils.NumericToFloat64(backorder.Subtotal)
	ivaFloat, _ := utils.NumericToFloat64(backorder.Iva)
	totalFloat, _ := utils.NumericToFloat64(backorder.TotalOrden)

	pedido, err := qtx.CrearPedido(c.Request.Context(), db.CrearPedidoParams{
		Folio:      folio,
		ClienteID:  backorder.ClienteID,
		UsuarioID:  pgtype.Int4{Int32: usuarioID, Valid: usuarioID != 0},
		Estado:     "Pendiente",
		MetodoPago: backorder.MetodoPago,
		Subtotal:   utils.ToNumeric(subtotalFloat),
		Iva:        utils.ToNumeric(ivaFloat),
		TotalOrden: utils.ToNumeric(totalFloat),
		CreatedBy:  pgtype.Int4{Int32: usuarioID, Valid: usuarioID != 0},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear pedido desde backorder: " + err.Error()})
		return
	}

	for _, d := range detalles {
		precioFloat, _ := utils.NumericToFloat64(d.PrecioUnitarioAplicado)
		_, err := qtx.CrearPedidoDetalle(c.Request.Context(), db.CrearPedidoDetalleParams{
			PedidoID:               pgtype.Int4{Int32: pedido.ID, Valid: true},
			VarianteID:             d.VarianteID,
			Cantidad:               d.Cantidad,
			PrecioUnitarioAplicado: utils.ToNumeric(precioFloat),
			CreatedBy:              pgtype.Int4{Int32: usuarioID, Valid: usuarioID != 0},
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error al crear detalle del pedido: %s", err.Error())})
			return
		}
	}

	_, err = qtx.SetPedidoOrigenBackorder(c.Request.Context(), db.SetPedidoOrigenBackorderParams{
		ID:        backorder.ID,
		PedidoOrigenID: pgtype.Int4{Int32: pedido.ID, Valid: true},
		UpdatedBy: pgtype.Int4{Int32: usuarioID, Valid: usuarioID != 0},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar backorder con pedido origen: " + err.Error()})
		return
	}

	if err := tx.Commit(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al confirmar conversión: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Backorder convertido a pedido exitosamente",
		"pedido":   pedido,
	})
}
