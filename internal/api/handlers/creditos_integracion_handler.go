package handlers

import (
	"net/http"
	"strconv"

	"github.com/OrlandoHdz/kubo/internal/db"
	"github.com/gin-gonic/gin"
)

type CreditosIntegracionHandler struct {
	queries *db.Queries
}

func NewCreditosIntegracionHandler(q *db.Queries) *CreditosIntegracionHandler {
	return &CreditosIntegracionHandler{queries: q}
}

// Listar devuelve todos los créditos de integración
func (h *CreditosIntegracionHandler) Listar(c *gin.Context) {
	creditos, err := h.queries.ListAllCreditos(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener créditos: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, creditos)
}

// Obtener devuelve un crédito de integración por su ID
func (h *CreditosIntegracionHandler) Obtener(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	credito, err := h.queries.GetCreditoByID(c.Request.Context(), int32(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Crédito no encontrado"})
		return
	}
	c.JSON(http.StatusOK, credito)
}

// Crear crea un nuevo crédito de integración
func (h *CreditosIntegracionHandler) Crear(c *gin.Context) {
	var params db.CreateCreditoParams
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
		return
	}
	credito, err := h.queries.CreateCredito(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear crédito: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, credito)
}
