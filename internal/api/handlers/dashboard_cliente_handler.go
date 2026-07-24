package handlers

import (
	"log"
	"net/http"

	"github.com/OrlandoHdz/kubo/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type DashboardClienteHandler struct {
	queries *db.Queries
}

func NewDashboardClienteHandler(q *db.Queries) *DashboardClienteHandler {
	return &DashboardClienteHandler{queries: q}
}

type DashboardClienteResponse struct {
	// Financiero
	DiasCredito        int32   `json:"dias_credito"`
	LimiteCredito      float64 `json:"limite_credito"`
	SaldoPendienteTotal float64 `json:"saldo_pendiente_total"`
	SaldoVencido       float64 `json:"saldo_vencido"`
	CreditoDisponible  float64 `json:"credito_disponible"`
	PorcentajeUsado    float64 `json:"porcentaje_usado"`

	// Operativo
	PedidosActivos       int64 `json:"pedidos_activos"`
	BackordersActivos    int64 `json:"backorders_activos"`
	DevolucionesEnProceso int64 `json:"devoluciones_en_proceso"`
}

func (h *DashboardClienteHandler) ObtenerDashboard(c *gin.Context) {
	// Hardcoded para testing: simula usuario con id=3, cliente_id=3
	usuarioID := int32(3)
	clienteID := int32(3)

	log.Printf("Dashboard: usuario_id=%d, cliente_id=%d", usuarioID, clienteID)

	cveCte := clienteIDToCveCte(clienteID)

	// Consultar datos financieros desde SAI (usa cve_cte)
	financiero, err := h.queries.ObtenerDatosFinancierosDashboard(c.Request.Context(), cveCte)
	if err != nil {
		log.Printf("Dashboard: error al obtener datos financieros: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener datos financieros: " + err.Error()})
		return
	}

	// Consultar datos operativos desde tablas locales (usa cliente_id)
	clienteIDParam := pgtype.Int4{Int32: clienteID, Valid: true}

	pedidosActivos, err := h.queries.ObtenerPedidosActivosCount(c.Request.Context(), clienteIDParam)
	if err != nil {
		log.Printf("Dashboard: error al contar pedidos activos: %v", err)
		pedidosActivos = 0
	}

	backordersActivos, err := h.queries.ObtenerBackordersActivosCount(c.Request.Context(), clienteIDParam)
	if err != nil {
		log.Printf("Dashboard: error al contar backorders activos: %v", err)
		backordersActivos = 0
	}

	devolucionesEnProceso, err := h.queries.ObtenerDevolucionesEnProcesoCount(c.Request.Context(), clienteIDParam)
	if err != nil {
		log.Printf("Dashboard: error al contar devoluciones en proceso: %v", err)
		devolucionesEnProceso = 0
	}

	// Parsear valores financieros
	diasCredito := int32(0)
	if financiero.DiasCredito.Valid {
		diasCredito = financiero.DiasCredito.Int32
	}

	limiteCredito := float64(0)
	if lc, err := financiero.LimiteCredito.Float64Value(); err == nil {
		limiteCredito = lc.Float64
	}

	saldoPendienteTotal := float64(0)
	if sp, err := financiero.SaldoPendienteTotal.Float64Value(); err == nil {
		saldoPendienteTotal = sp.Float64
	}

	saldoVencido := float64(0)
	if sv, err := financiero.SaldoVencido.Float64Value(); err == nil {
		saldoVencido = sv.Float64
	}

	creditoDisponible := limiteCredito - saldoPendienteTotal
	if creditoDisponible < 0 {
		creditoDisponible = 0
	}

	porcentajeUsado := float64(0)
	if limiteCredito > 0 {
		porcentajeUsado = (saldoPendienteTotal / limiteCredito) * 100
		if porcentajeUsado > 100 {
			porcentajeUsado = 100
		}
	}

	resp := DashboardClienteResponse{
		DiasCredito:          diasCredito,
		LimiteCredito:        limiteCredito,
		SaldoPendienteTotal:  saldoPendienteTotal,
		SaldoVencido:         saldoVencido,
		CreditoDisponible:    creditoDisponible,
		PorcentajeUsado:      porcentajeUsado,
		PedidosActivos:       pedidosActivos,
		BackordersActivos:    backordersActivos,
		DevolucionesEnProceso: devolucionesEnProceso,
	}

	c.JSON(http.StatusOK, gin.H{
		"usuario_id": usuarioID,
		"cliente_id": clienteID,
		"dashboard":  resp,
	})
}
