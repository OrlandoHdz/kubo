package handlers

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/OrlandoHdz/kubo/internal/db"
	"github.com/OrlandoHdz/kubo/pkg/email"
	"github.com/OrlandoHdz/kubo/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type PagoFacturasHandler struct {
	queries  *db.Queries
	emailCfg *email.Config
}

func NewPagoFacturasHandler(q *db.Queries, e *email.Config) *PagoFacturasHandler {
	return &PagoFacturasHandler{queries: q, emailCfg: e}
}

func (h *PagoFacturasHandler) getClienteID(ctx *gin.Context) (int32, error) {
	userID, _ := ctx.Get("userID")
	uid, ok := userID.(int32)
	if !ok || uid == 0 {
		return 0, fmt.Errorf("no autorizado")
	}
	user, err := h.queries.GetUsuarioByID(ctx.Request.Context(), uid)
	if err != nil || !user.ClienteID.Valid {
		return 0, fmt.Errorf("cliente no encontrado")
	}
	return user.ClienteID.Int32, nil
}

func (h *PagoFacturasHandler) ListarEmitidas(c *gin.Context) {
	clienteID, err := h.getClienteID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var cveCte int32
	if clienteID == 3 {
		cveCte = 212
	} else {
		cveCtePg, err := h.queries.GetCveCteByClienteId(c.Request.Context(), clienteID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "No se encontró código de cliente SAI para este cliente"})
			return
		}
		if !cveCtePg.Valid {
			c.JSON(http.StatusNotFound, gin.H{"error": "El cliente no tiene código SAI asociado"})
			return
		}
		cveCte = cveCtePg.Int32
	}

	fechaInicioStr := c.Query("fecha_inicio")
	fechaFinStr := c.Query("fecha_fin")

	var facturas []db.FacturasIntegracion
	if fechaInicioStr != "" && fechaFinStr != "" {
		fechaInicio, fechaFin := parseDateRange(c)
		facturas, err = h.queries.ListFacturasEmitidasPorCveCteRango(c.Request.Context(), db.ListFacturasEmitidasPorCveCteRangoParams{
			CveCte:     pgtype.Int4{Int32: cveCte, Valid: true},
			FaltaFac:   fechaInicio,
			FaltaFac_2: fechaFin,
		})
	} else {
		facturas, err = h.queries.ListFacturasEmitidasPorCveCte(c.Request.Context(), pgtype.Int4{Int32: cveCte, Valid: true})
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar facturas: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, facturas)
}

func (h *PagoFacturasHandler) ProcesarPago(c *gin.Context) {
	clienteID, err := h.getClienteID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	usuarioID := int32(0)
	if uid, ok := c.Get("userID"); ok {
		if v, ok2 := uid.(int32); ok2 {
			usuarioID = v
		}
	}

	var req struct {
		FacturaIDs    []int32 `json:"factura_ids" binding:"required,min=1"`
		NumeroTarjeta string  `json:"numero_tarjeta"`
		Vencimiento   string  `json:"vencimiento"`
		CVC           string  `json:"cvc"`
		NombreTitular string  `json:"nombre_titular"`
		MetodoPago    string  `json:"metodo_pago"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
		return
	}

	if req.MetodoPago == "" {
		req.MetodoPago = "Tarjeta de Credito"
	}

	terminacion := ""
	if len(req.NumeroTarjeta) >= 4 {
		terminacion = req.NumeroTarjeta[len(req.NumeroTarjeta)-4:]
	}

	respuestaSimulada := fmt.Sprintf(`{"estado":"aprobado","folio":"SIM-%d","mensaje":"Pago simulado exitoso","tarjeta":"****%s","titular":"%s"}`,
		time.Now().Unix(), terminacion, req.NombreTitular)

	var montoTotal float64
	var detallesData []struct {
		noFactura string
		monto     pgtype.Numeric
	}

	for _, fid := range req.FacturaIDs {
		factura, err := h.queries.GetFacturaIntegracionPorId(c.Request.Context(), fid)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Factura %d no encontrada", fid)})
			return
		}
		monto := factura.TotalFac
		if !monto.Valid {
			monto = pgtype.Numeric{}
			monto.Scan(1000.00)
		}
		f, _ := utils.NumericToFloat64(monto)
		montoTotal += f
		detallesData = append(detallesData, struct {
			noFactura string
			monto     pgtype.Numeric
		}{noFactura: factura.NoFac, monto: monto})
	}

	pago, err := h.queries.CrearPagoFactura(c.Request.Context(), db.CrearPagoFacturaParams{
		ClienteID:          clienteID,
		MetodoPago:         req.MetodoPago,
		TarjetaTerminacion: terminacion,
		MontoTotal:         utils.ToNumeric(montoTotal),
		RespuestaSimulada:  respuestaSimulada,
		CreatedBy:          pgtype.Int4{Int32: usuarioID, Valid: usuarioID != 0},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al registrar pago: " + err.Error()})
		return
	}

	for i, fid := range req.FacturaIDs {
		err := h.queries.CrearPagoFacturaDetalle(c.Request.Context(), db.CrearPagoFacturaDetalleParams{
			PagoID:      pago.ID,
			FacturaID:   fid,
			NoFactura:   detallesData[i].noFactura,
			MontoPagado: detallesData[i].monto,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error al registrar detalle para factura %d: %s", fid, err.Error())})
			return
		}
	}

	if h.emailCfg != nil {
		log.Printf("PagoFacturas: iniciando envío de correo para cliente %d, monto %.2f, método %s, terminación %s, %d facturas",
			clienteID, montoTotal, req.MetodoPago, terminacion, len(detallesData))
		go h.notificarPagoFactura(context.Background(), clienteID, montoTotal, req.MetodoPago, terminacion, detallesData)
	} else {
		log.Printf("PagoFacturas: emailCfg es nil, NO se enviará correo para cliente %d", clienteID)
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Pago procesado exitosamente (simulado)",
		"pago":    pago,
	})
}

func (h *PagoFacturasHandler) notificarPagoFactura(ctx context.Context, clienteID int32, montoTotal float64, metodoPago, terminacion string, detallesData []struct {
	noFactura string
	monto     pgtype.Numeric
}) {
	cliente, err := h.queries.GetCliente(ctx, clienteID)
	if err != nil {
		log.Printf("Email: error al obtener cliente %d: %v", clienteID, err)
		return
	}

	usuarios, err := h.queries.ListarUsuariosPorCliente(ctx, pgtype.Int4{Int32: clienteID, Valid: true})
	if err != nil || len(usuarios) == 0 {
		log.Printf("Email: cliente %d sin usuarios registrados (err=%v)", clienteID, err)
		return
	}

	var facturas []email.PagoFacturaItemData
	for _, d := range detallesData {
		monto, _ := utils.NumericToFloat64(d.monto)
		facturas = append(facturas, email.PagoFacturaItemData{
			NoFactura:   d.noFactura,
			MontoPagado: formatPrice(monto),
		})
	}

	data := email.PagoFacturaData{
		ClienteName: cliente.NombreComercial,
		Fecha:       time.Now().Format("02/01/2006 15:04"),
		MetodoPago:  metodoPago,
		Terminacion: terminacion,
		Total:       formatPrice(montoTotal),
		Facturas:    facturas,
	}

	log.Printf("PagoFacturas: enviando correo a %s (%s), %d facturas, total %s",
		usuarios[0].Email, cliente.NombreComercial, len(facturas), formatPrice(montoTotal))
	h.emailCfg.SendPagoFacturaNotification(data, usuarios[0].Email)
	log.Printf("PagoFacturas: envio de correo completado para %s", usuarios[0].Email)
}

func (h *PagoFacturasHandler) ListarPagadas(c *gin.Context) {
	fechaInicioStr := c.Query("fecha_inicio")
	fechaFinStr := c.Query("fecha_fin")

	var err error
	var pagos []db.PagosFactura
	if fechaInicioStr != "" && fechaFinStr != "" {
		fechaInicio, fechaFin := parseDateRange(c)
		pagos, err = h.queries.ListPagosFacturasRango(c.Request.Context(), db.ListPagosFacturasRangoParams{
			CreatedAt:   fechaInicio,
			CreatedAt_2: fechaFin,
		})
	} else {
		pagos, err = h.queries.ListPagosFacturas(c.Request.Context())
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar pagos: " + err.Error()})
		return
	}

	type pagoConDetalles struct {
		db.PagosFactura
		Detalles interface{} `json:"detalles"`
	}

	result := make([]pagoConDetalles, 0, len(pagos))
	for _, p := range pagos {
		detalles, err := h.queries.GetPagoFacturaDetalles(c.Request.Context(), p.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar detalles del pago"})
			return
		}
		result = append(result, pagoConDetalles{PagosFactura: p, Detalles: detalles})
	}

	c.JSON(http.StatusOK, result)
}

func (h *PagoFacturasHandler) ObtenerPagada(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	pago, err := h.queries.GetPagoFactura(c.Request.Context(), int32(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pago no encontrado"})
		return
	}

	detalles, err := h.queries.GetPagoFacturaDetalles(c.Request.Context(), pago.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener detalles del pago"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pago":     pago,
		"detalles": detalles,
	})
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
