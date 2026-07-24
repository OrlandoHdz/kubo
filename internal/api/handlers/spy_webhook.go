package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/OrlandoHdz/kubo/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type SpyWebhookHandler struct {
	queries *db.Queries
}

func NewSpyWebhookHandler(q *db.Queries) *SpyWebhookHandler {
	return &SpyWebhookHandler{queries: q}
}

func (h *SpyWebhookHandler) SpyWebhook(c *gin.Context) {
	log.Println("=== SPY WEBHOOK (BanRegio) ===")
	log.Println("Method:", c.Request.Method)
	log.Println("URL:", c.Request.URL.String())

	bnrgMontoTrans := c.Query("BNRG_MONTO_TRANS")
	bnrgIDAfiliacion := c.Query("BNRG_ID_AFILIACION")
	bnrgFechaLocal := c.Query("BNRG_FECHA_LOCAL")
	bnrgCodigoAut := c.Query("BNRG_CODIGO_AUT")
	bnrgFolio := c.Query("BNRG_FOLIO")
	bnrgTexto := c.Query("BNRG_TEXTO")
	bnrgReferencia := c.Query("BNRG_REFERENCIA")
	bnrgIDMedio := c.Query("BNRG_ID_MEDIO")
	bnrgCodigoProc := c.Query("BNRG_CODIGO_PROC")
	bnrgHoraLocal := c.Query("BNRG_HORA_LOCAL")
	bnrgCodigoEmisor := c.Query("BNRG_CODIGO_EMISOR")

	log.Printf("**** BNRG_FOLIO=%s BNRG_MONTO_TRANS=%s BNRG_CODIGO_AUT=%s", bnrgFolio, bnrgMontoTrans, bnrgCodigoAut)

	if bnrgFolio == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "BNRG_FOLIO es requerido"})
		return
	}

	// pedido, err := h.queries.GetPedidoByFolio(c.Request.Context(), bnrgFolio)
	// if err != nil {
	// 	log.Printf("Error al buscar pedido por folio %s: %v", bnrgFolio, err)
	// 	c.JSON(http.StatusNotFound, gin.H{"error": "Pedido no encontrado con el folio proporcionado"})
	// 	return
	// }

	var montoTrans pgtype.Numeric
	if bnrgMontoTrans != "" {
		montoTrans.Scan(bnrgMontoTrans)
	}

	var fechaLocal pgtype.Date
	if bnrgFechaLocal != "" {
		parsed, err := time.Parse("2006-01-02", bnrgFechaLocal)
		if err == nil {
			fechaLocal = pgtype.Date{Time: parsed, Valid: true}
		}
	}

	var horaLocal pgtype.Time
	if bnrgHoraLocal != "" {
		parts := strings.Split(bnrgHoraLocal, ":")
		if len(parts) == 3 {
			hour, _ := strconv.Atoi(parts[0])
			min, _ := strconv.Atoi(parts[1])
			sec, _ := strconv.Atoi(parts[2])
			microseconds := int64((hour*3600 + min*60 + sec) * 1000000)
			horaLocal = pgtype.Time{Microseconds: microseconds, Valid: true}
		}
	}

	params := db.CrearTransaccionBanregioParams{
		BnrgMontoTrans:   montoTrans,
		BnrgIDAfiliacion: pgtype.Text{String: bnrgIDAfiliacion, Valid: bnrgIDAfiliacion != ""},
		BnrgFechaLocal:   fechaLocal,
		BnrgCodigoAut:    pgtype.Text{String: bnrgCodigoAut, Valid: bnrgCodigoAut != ""},
		BnrgFolio:        pgtype.Text{String: bnrgFolio, Valid: bnrgFolio != ""},
		BnrgTexto:        pgtype.Text{String: bnrgTexto, Valid: bnrgTexto != ""},
		BnrgReferencia:   pgtype.Text{String: bnrgReferencia, Valid: bnrgReferencia != ""},
		BnrgIDMediop:     pgtype.Text{String: bnrgIDMedio, Valid: bnrgIDMedio != ""},
		BnrgCodigoProc:   pgtype.Text{String: bnrgCodigoProc, Valid: bnrgCodigoProc != ""},
		BnrgHoraLocal:    horaLocal,
		BnrgCodigoEmisor: pgtype.Text{String: bnrgCodigoEmisor, Valid: bnrgCodigoEmisor != ""},
		// PedidoID:         pgtype.Int4{Int32: pedido.ID, Valid: true},
		CreatedBy: pgtype.Int4{Valid: false},
	}

	transaccion, err := h.queries.CrearTransaccionBanregio(c.Request.Context(), params)
	if err != nil {
		log.Printf("Error al guardar transacción BanRegio: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al guardar la transacción"})
		return
	}

	log.Printf("Transacción BanRegio guardada: ID=%d, Folio=%s", transaccion.ID, transaccion.BnrgFolio.String)

	c.JSON(http.StatusOK, gin.H{
		"status":         "ok",
		"transaccion_id": transaccion.ID,
	})
}
