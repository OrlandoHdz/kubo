package handlers

import (
	"context"
	"log"
	"math/big"
	"net/http"
	"strconv"
	"time"

	"github.com/OrlandoHdz/kubo/internal/db"
	"github.com/OrlandoHdz/kubo/internal/recompensas"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type RecompensasHandler struct {
	queries *db.Queries
	service *recompensas.RecompensasService
}

func NewRecompensasHandler(q *db.Queries) *RecompensasHandler {
	return &RecompensasHandler{
		queries: q,
		service: recompensas.NewRecompensasService(q),
	}
}

// ===== PARÁMETROS =====

func (h *RecompensasHandler) GetParametros(c *gin.Context) {
	params, err := h.queries.GetParametrosRecompensa(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error obteniendo parámetros: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, params)
}

func (h *RecompensasHandler) UpdateParametros(c *gin.Context) {
	var input struct {
		MontoCompraPorPunto float64 `json:"monto_compra_por_punto" binding:"required"`
		ValorPuntoPesos     float64 `json:"valor_punto_pesos" binding:"required"`
		UpdatedBy           int32   `json:"updated_by"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
		return
	}

	params, err := h.queries.GetParametrosRecompensa(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error obteniendo parámetros: " + err.Error()})
		return
	}

	actualizado, err := h.queries.UpdateParametrosRecompensa(c.Request.Context(), db.UpdateParametrosRecompensaParams{
		MontoCompraPorPunto: toNumericForce(input.MontoCompraPorPunto),
		ValorPuntoPesos:     toNumericForce(input.ValorPuntoPesos),
		UpdatedBy:           pgtype.Int4{Int32: input.UpdatedBy, Valid: input.UpdatedBy != 0},
		ID:                  params.ID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error actualizando parámetros: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, actualizado)
}

// ===== NIVELES DE RECOMPENSA =====

func (h *RecompensasHandler) ListarNiveles(c *gin.Context) {
	niveles, err := h.queries.ListNivelesRecompensa(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error listando niveles: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, niveles)
}

func (h *RecompensasHandler) CrearNivel(c *gin.Context) {
	var input struct {
		Nombre                   string `json:"nombre" binding:"required"`
		NumeroNivel              *int32 `json:"numero_nivel"`
		PromedioCompraTrimestral float64 `json:"promedio_compra_trimestral"`
		PorcentajeDescuento      float64 `json:"porcentaje_descuento"`
		ImagenInsigniaURL        string  `json:"imagen_insignia_url"`
		CreatedBy                int32   `json:"created_by"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
		return
	}
	if input.NumeroNivel == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "numero_nivel es requerido"})
		return
	}

	nivel, err := h.queries.CreateNivelRecompensa(c.Request.Context(), db.CreateNivelRecompensaParams{
		Nombre:                   input.Nombre,
		NumeroNivel:              *input.NumeroNivel,
		PromedioCompraTrimestral: toNumericForce(input.PromedioCompraTrimestral),
		PorcentajeDescuento:      toNumericForce(input.PorcentajeDescuento),
		ImagenInsigniaUrl:        pgtype.Text{String: input.ImagenInsigniaURL, Valid: input.ImagenInsigniaURL != ""},
		CreatedBy:                pgtype.Int4{Int32: input.CreatedBy, Valid: input.CreatedBy != 0},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creando nivel: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, nivel)
}

func (h *RecompensasHandler) ActualizarNivel(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	var input struct {
		Nombre                   string `json:"nombre" binding:"required"`
		NumeroNivel              *int32 `json:"numero_nivel"`
		PromedioCompraTrimestral float64 `json:"promedio_compra_trimestral"`
		PorcentajeDescuento      float64 `json:"porcentaje_descuento"`
		ImagenInsigniaURL        string  `json:"imagen_insignia_url"`
		UpdatedBy                int32   `json:"updated_by"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
		return
	}
	if input.NumeroNivel == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "numero_nivel es requerido"})
		return
	}

	nivel, err := h.queries.UpdateNivelRecompensa(c.Request.Context(), db.UpdateNivelRecompensaParams{
		ID:                       int32(id),
		Nombre:                   input.Nombre,
		NumeroNivel:              *input.NumeroNivel,
		PromedioCompraTrimestral: toNumericForce(input.PromedioCompraTrimestral),
		PorcentajeDescuento:      toNumericForce(input.PorcentajeDescuento),
		ImagenInsigniaUrl:        pgtype.Text{String: input.ImagenInsigniaURL, Valid: input.ImagenInsigniaURL != ""},
		UpdatedBy:                pgtype.Int4{Int32: input.UpdatedBy, Valid: input.UpdatedBy != 0},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error actualizando nivel: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, nivel)
}

func (h *RecompensasHandler) EliminarNivel(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	var input struct{ DeletedBy int32 }
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	err = h.queries.SoftDeleteNivelRecompensa(c.Request.Context(), db.SoftDeleteNivelRecompensaParams{
		ID: int32(id), DeletedBy: pgtype.Int4{Int32: input.DeletedBy, Valid: input.DeletedBy != 0},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error eliminando nivel: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Nivel eliminado"})
}

// ===== TRANSACCIONES DE PUNTOS =====

// ForzarEvaluacionNiveles ejecuta manualmente la evaluación de niveles (solo para pruebas)
func (h *RecompensasHandler) ForzarEvaluacionNiveles(c *gin.Context) {
	log.Println("[Recompensas] Evaluación de niveles forzada manualmente por API")
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Minute)
	defer cancel()

	h.service.EvaluarNiveles(ctx)

	c.JSON(http.StatusOK, gin.H{
		"message": "Evaluación de niveles ejecutada correctamente",
	})
}

func (h *RecompensasHandler) ListarTransaccionesCliente(c *gin.Context) {
	clienteIDStr := c.Param("cliente_id")
	clienteID, err := strconv.Atoi(clienteIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de cliente inválido"})
		return
	}

	transacciones, err := h.queries.ListTransaccionesByCliente(c.Request.Context(), int32(clienteID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error listando transacciones: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, transacciones)
}

// ===== INFO RECOMPENSAS CLIENTE =====

func (h *RecompensasHandler) GetInfoRecompensasCliente(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de cliente inválido"})
		return
	}

	info, err := h.queries.GetClienteConNivel(c.Request.Context(), int32(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cliente no encontrado"})
		return
	}
	c.JSON(http.StatusOK, info)
}

func (h *RecompensasHandler) AsignarNivelManual(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de cliente inválido"})
		return
	}

	var input struct {
		NivelRecompensaID int32 `json:"nivel_recompensa_id" binding:"required"`
		UpdatedBy         int32 `json:"updated_by"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	cliente, err := h.queries.UpdateClienteNivelManual(c.Request.Context(), db.UpdateClienteNivelManualParams{
		ID:                int32(id),
		NivelRecompensaID: pgtype.Int4{Int32: input.NivelRecompensaID, Valid: true},
		UpdatedBy:         pgtype.Int4{Int32: input.UpdatedBy, Valid: input.UpdatedBy != 0},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error asignando nivel: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, cliente)
}

// ===== Utilidades =====

func toNumericForce(v float64) pgtype.Numeric {
	n := pgtype.Numeric{}
	n.Int = intToBig(v)
	n.Exp = -2
	n.Valid = true
	return n
}

func intToBig(f float64) *big.Int {
	return big.NewInt(int64(f * 100))
}
