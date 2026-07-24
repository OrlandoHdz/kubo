package handlers

import (
	"net/http"

	"github.com/OrlandoHdz/kubo/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type EstadoCuentaHandler struct {
	queries *db.Queries
}

func NewEstadoCuentaHandler(q *db.Queries) *EstadoCuentaHandler {
	return &EstadoCuentaHandler{queries: q}
}

// clienteIDToCveCte mapea cliente_id del sistema local a cve_cte del SAI.
// Para testing: cliente_id=3 → cve_cte="212"
func clienteIDToCveCte(clienteID int32) pgtype.Int4 {
	switch clienteID {
	case 3:
		return pgtype.Int4{Int32: 212, Valid: true}
	}
	return pgtype.Int4{Valid: false}
}

type FacturaItem struct {
	FacturaID        int32   `json:"factura_id"`
	Folio            string  `json:"folio"`
	FechaEmision     string  `json:"fecha_emision"`
	FechaVencimiento string  `json:"fecha_vencimiento"`
	MontoTotal       float64 `json:"monto_total"`
	SaldoPendiente   float64 `json:"saldo_pendiente"`
	Estatus          string  `json:"estatus"`
}

type EstadoCuentaResponse struct {
	CveCte         int32         `json:"cve_cte"`
	Cliente        string        `json:"cliente"`
	DiasCredito    int32         `json:"dias_credito"`
	LimiteCredito  float64       `json:"limite_credito"`
	Facturas       []FacturaItem `json:"facturas"`
	TotalFacturas  int           `json:"total_facturas"`
	SaldoTotal     float64       `json:"saldo_total"`
	MontoTotal     float64       `json:"monto_total"`
}

func (h *EstadoCuentaHandler) ObtenerEstadoCuenta(c *gin.Context) {
	// Hardcoded para testing: simula usuario con id=3, cliente_id=3
	usuarioID := int32(3)
	clienteID := int32(3)

	cveCte := clienteIDToCveCte(clienteID)
	if !cveCte.Valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No hay mapeo definido para el cliente_id"})
		return
	}

	rows, err := h.queries.ObtenerEstadoCuentaPorCveCte(c.Request.Context(), cveCte)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener estado de cuenta: " + err.Error()})
		return
	}

	if len(rows) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"usuario_id":     usuarioID,
			"cliente_id":     clienteID,
			"cve_cte":        cveCte.Int32,
			"mensaje":        "No se encontraron facturas para este cliente",
		})
		return
	}

	var facturas []FacturaItem
	var saldoTotal, montoTotal float64

	for _, r := range rows {
		montoF8, _ := r.MontoTotal.Float64Value()
		saldoF8, _ := r.SaldoPendiente.Float64Value()

		fe := ""
		if r.FechaEmision.Valid {
			fe = r.FechaEmision.Time.Format("02/01/2006")
		}
		fv := ""
		if r.FechaVencimiento.Valid {
			fv = r.FechaVencimiento.Time.Format("02/01/2006")
		}

		facturas = append(facturas, FacturaItem{
			FacturaID:        r.FacturaID,
			Folio:            r.Folio,
			FechaEmision:     fe,
			FechaVencimiento: fv,
			MontoTotal:       montoF8.Float64,
			SaldoPendiente:   saldoF8.Float64,
			Estatus:          r.Estatus.String,
		})
		saldoTotal += saldoF8.Float64
		montoTotal += montoF8.Float64
	}

	diasCredito := int32(0)
	if len(rows) > 0 && rows[0].DiasCredito.Valid {
		diasCredito = rows[0].DiasCredito.Int32
	}

	limiteCredito := float64(0)
	if len(rows) > 0 {
		lc, _ := rows[0].LimiteCredito.Float64Value()
		limiteCredito = lc.Float64
	}

	resp := EstadoCuentaResponse{
		CveCte:        cveCte.Int32,
		Cliente:       rows[0].NomCte.String,
		DiasCredito:   diasCredito,
		LimiteCredito: limiteCredito,
		Facturas:      facturas,
		TotalFacturas: len(facturas),
		SaldoTotal:    saldoTotal,
		MontoTotal:    montoTotal,
	}

	c.JSON(http.StatusOK, gin.H{
		"usuario_id":    usuarioID,
		"cliente_id":    clienteID,
		"estado_cuenta": resp,
	})
}
