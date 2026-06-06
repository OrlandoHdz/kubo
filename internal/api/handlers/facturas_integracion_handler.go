package handlers

import (
	"net/http"
	"strconv"

	"github.com/OrlandoHdz/kubo/internal/db"
	"github.com/gin-gonic/gin"
)

type FacturasIntegracionHandler struct {
	queries *db.Queries
}

func NewFacturasIntegracionHandler(q *db.Queries) *FacturasIntegracionHandler {
	return &FacturasIntegracionHandler{queries: q}
}

// Listar devuelve todas las facturas de integración
func (h *FacturasIntegracionHandler) Listar(c *gin.Context) {
	facturas, err := h.queries.ObtenerFacturasIntegracion(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener facturas: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, facturas)
}

// Obtener devuelve una factura de integración por su ID
func (h *FacturasIntegracionHandler) Obtener(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	factura, err := h.queries.ObtenerFacturaIntegracion(c.Request.Context(), int32(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Factura no encontrada"})
		return
	}
	c.JSON(http.StatusOK, factura)
}

// Crear crea una nueva factura de integración
func (h *FacturasIntegracionHandler) Crear(c *gin.Context) {
	var params db.CrearFacturaIntegracionParams
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
		return
	}
	// Usar la función de creación del repositorio
	factura, err := h.queries.CrearFacturaIntegracion(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear factura: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, factura)
}
