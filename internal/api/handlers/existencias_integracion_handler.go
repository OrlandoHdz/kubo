package handlers

import (
    "net/http"
    "strconv"

    "github.com/OrlandoHdz/kubo/internal/db"
    "github.com/gin-gonic/gin"
)

type ExistenciasIntegracionHandler struct {
    queries *db.Queries
}

func NewExistenciasIntegracionHandler(q *db.Queries) *ExistenciasIntegracionHandler {
    return &ExistenciasIntegracionHandler{queries: q}
}

// Listar devuelve todas las existencias de integración
func (h *ExistenciasIntegracionHandler) Listar(c *gin.Context) {
    existencias, err := h.queries.ObtenerTodasLasExistencias(c.Request.Context())
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener existencias: " + err.Error()})
        return
    }
    c.JSON(http.StatusOK, existencias)
}

// Obtener devuelve una existencia por ID (no implementado, placeholder)
func (h *ExistenciasIntegracionHandler) Obtener(c *gin.Context) {
    idStr := c.Param("id")
    _, err := strconv.Atoi(idStr) // validate numeric, but no query by ID exists yet
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
        return
    }
    c.JSON(http.StatusNotImplemented, gin.H{"error": "Obtener por ID no implementado"})
}

// Crear crea una nueva existencia de integración
func (h *ExistenciasIntegracionHandler) Crear(c *gin.Context) {
    var params db.CrearExistenciaIntegracionParams
    if err := c.ShouldBindJSON(&params); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
        return
    }
    existencia, err := h.queries.CrearExistenciaIntegracion(c.Request.Context(), params)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear existencia: " + err.Error()})
        return
    }
    c.JSON(http.StatusCreated, existencia)
}

// Upsert crea o actualiza una existencia de integración
func (h *ExistenciasIntegracionHandler) Upsert(c *gin.Context) {
    var params db.UpsertExistenciaIntegracionParams
    if err := c.ShouldBindJSON(&params); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
        return
    }
    existencia, err := h.queries.UpsertExistenciaIntegracion(c.Request.Context(), params)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al upsert existencia: " + err.Error()})
        return
    }
    c.JSON(http.StatusOK, existencia)
}
