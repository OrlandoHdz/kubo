package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/OrlandoHdz/kubo/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AjustesInventarioHandler struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewAjustesInventarioHandler(q *db.Queries, pool *pgxpool.Pool) *AjustesInventarioHandler {
	return &AjustesInventarioHandler{queries: q, pool: pool}
}

func (h *AjustesInventarioHandler) CrearAjuste(c *gin.Context) {
	var input struct {
		VarianteID int32  `json:"variante_id" binding:"required"`
		Tipo       string `json:"tipo" binding:"required"`
		Cantidad   int32  `json:"cantidad" binding:"required"`
		Comentario string `json:"comentario" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
		return
	}
	if input.Tipo != "SUMAR" && input.Tipo != "RESTAR" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tipo debe ser SUMAR o RESTAR"})
		return
	}
	if input.Cantidad <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cantidad debe ser mayor a 0"})
		return
	}
	if input.Comentario == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Comentario es obligatorio"})
		return
	}

	// Obtener usuario autenticado
	var usuarioID int32
	var usuarioNombre string
	if uid, ok := c.Get("userID"); ok {
		if v, ok2 := uid.(int32); ok2 {
			usuarioID = v
		} else if v, ok2 := uid.(int); ok2 {
			usuarioID = int32(v)
		}
	}
	if email, ok := c.Get("userEmail"); ok {
		if v, ok2 := email.(string); ok2 {
			usuarioNombre = v
		}
	}
	if usuarioID == 0 {
		// Fallback a user_id del contexto o 1
		if uid, ok := c.Get("userID"); ok {
			if v, ok2 := uid.(float64); ok2 {
				usuarioID = int32(v)
			}
		}
	}
	// Si no hay usuario, usar 1 como fallback para no bloquear
	if usuarioNombre == "" {
		if u, ok := c.Get("userEmail"); ok {
			if s, ok2 := u.(string); ok2 {
				usuarioNombre = s
			}
		}
	}

	// Transacción: leer stock actual, actualizar stock, crear ajuste
	tx, err := h.pool.Begin(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error iniciando transacción: " + err.Error()})
		return
	}
	defer tx.Rollback(c.Request.Context())
	qtx := h.queries.WithTx(tx)

	variante, err := qtx.GetVarianteRecompensa(c.Request.Context(), input.VarianteID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Variante no encontrada"})
		return
	}

	stockAnterior := variante.StockActual
	var stockNuevo int32
	if input.Tipo == "SUMAR" {
		stockNuevo = stockAnterior + input.Cantidad
	} else {
		stockNuevo = stockAnterior - input.Cantidad
		if stockNuevo < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Stock insuficiente para restar. Stock actual: " + strconv.Itoa(int(stockAnterior))})
			return
		}
	}

	// Actualizar stock
	err = qtx.ActualizarStockVarianteRecompensa(c.Request.Context(), db.ActualizarStockVarianteRecompensaParams{
		ID:          input.VarianteID,
		StockActual: stockNuevo,
		UpdatedBy:   pgtype.Int4{Int32: usuarioID, Valid: usuarioID != 0},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error actualizando stock: " + err.Error()})
		return
	}

	// Crear registro de ajuste
	ajuste, err := qtx.CrearAjusteInventario(c.Request.Context(), db.CrearAjusteInventarioParams{
		VarianteID:    input.VarianteID,
		Tipo:          input.Tipo,
		Cantidad:      input.Cantidad,
		Comentario:    input.Comentario,
		StockAnterior: stockAnterior,
		StockNuevo:    stockNuevo,
		UsuarioID:     pgtype.Int4{Int32: usuarioID, Valid: usuarioID != 0},
		UsuarioNombre: pgtype.Text{String: usuarioNombre, Valid: usuarioNombre != ""},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error registrando ajuste: " + err.Error()})
		return
	}

	if err := tx.Commit(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error confirmando ajuste: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, ajuste)
}

func (h *AjustesInventarioHandler) ListarAjustes(c *gin.Context) {
	varianteIDStr := c.Query("variante_id")
	fechaInicioStr := c.Query("fecha_inicio")
	fechaFinStr := c.Query("fecha_fin")

	var varianteID pgtype.Int4
	if varianteIDStr != "" {
		if v, err := strconv.Atoi(varianteIDStr); err == nil {
			varianteID = pgtype.Int4{Int32: int32(v), Valid: true}
		}
	}

	var fechaInicio pgtype.Timestamp
	if fechaInicioStr != "" {
		if t, err := time.Parse("2006-01-02", fechaInicioStr); err == nil {
			fechaInicio = pgtype.Timestamp{Time: t, Valid: true}
		} else if t, err := time.Parse(time.RFC3339, fechaInicioStr); err == nil {
			fechaInicio = pgtype.Timestamp{Time: t, Valid: true}
		}
	}

	var fechaFin pgtype.Timestamp
	if fechaFinStr != "" {
		if t, err := time.Parse("2006-01-02", fechaFinStr); err == nil {
			// Incluir todo el día hasta 23:59:59
			t = t.Add(24*time.Hour - time.Second)
			fechaFin = pgtype.Timestamp{Time: t, Valid: true}
		} else if t, err := time.Parse(time.RFC3339, fechaFinStr); err == nil {
			fechaFin = pgtype.Timestamp{Time: t, Valid: true}
		}
	}

	ajustes, err := h.queries.ListarAjustesInventario(c.Request.Context(), db.ListarAjustesInventarioParams{
		VarianteID:  varianteID,
		FechaInicio: fechaInicio,
		FechaFin:    fechaFin,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error listando ajustes: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, ajustes)
}

func (h *AjustesInventarioHandler) ListarVariantes(c *gin.Context) {
	variantes, err := h.queries.ListarVariantesRecompensaConProducto(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error listando variantes: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, variantes)
}
