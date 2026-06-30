package handlers

import (
	"net/http"
	"strconv"

	"github.com/OrlandoHdz/kubo/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type PagosTarjetaHandler struct {
	queries *db.Queries
}

func NewPagosTarjetaHandler(q *db.Queries) *PagosTarjetaHandler {
	return &PagosTarjetaHandler{queries: q}
}

func (h *PagosTarjetaHandler) Listar(c *gin.Context) {
	transacciones, err := h.queries.ListarTransaccionesBanregio(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al listar transacciones: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, transacciones)
}

func (h *PagosTarjetaHandler) Obtener(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	transaccion, err := h.queries.GetTransaccionBanregio(c.Request.Context(), int32(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transacción no encontrada"})
		return
	}
	c.JSON(http.StatusOK, transaccion)
}

func (h *PagosTarjetaHandler) Crear(c *gin.Context) {
	var params db.CrearTransaccionBanregioParams
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
		return
	}

	if params.CreatedBy.Valid == false {
		params.CreatedBy = pgtype.Int4{Valid: false}
	}

	transaccion, err := h.queries.CrearTransaccionBanregio(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear transacción: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, transaccion)
}

func (h *PagosTarjetaHandler) ListarPorCliente(c *gin.Context) {
	clienteIDStr := c.Param("cliente_id")
	clienteID, err := strconv.Atoi(clienteIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de cliente inválido"})
		return
	}

	transacciones, err := h.queries.ListarTransaccionesPorCliente(c.Request.Context(), pgtype.Int4{Int32: int32(clienteID), Valid: true})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al listar transacciones del cliente: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, transacciones)
}

func (h *PagosTarjetaHandler) Eliminar(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	err = h.queries.SoftDeleteTransaccionBanregio(c.Request.Context(), db.SoftDeleteTransaccionBanregioParams{
		ID:        int32(id),
		DeletedBy: pgtype.Int4{Valid: false},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al eliminar transacción: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Transacción eliminada correctamente"})
}
