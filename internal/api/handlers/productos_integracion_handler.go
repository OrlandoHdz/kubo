package handlers

import (
	"net/http"
	"strconv"

	"github.com/OrlandoHdz/kubo/internal/db"
	"github.com/gin-gonic/gin"
)

type ProductosIntegracionHandler struct {
	queries *db.Queries
}

func NewProductosIntegracionHandler(q *db.Queries) *ProductosIntegracionHandler {
	return &ProductosIntegracionHandler{queries: q}
}

// Listar devuelve todos los productos de integración
func (h *ProductosIntegracionHandler) Listar(c *gin.Context) {
	productos, err := h.queries.ListProductos(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener productos: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, productos)
}

// Obtener devuelve un producto de integración por su ID
func (h *ProductosIntegracionHandler) Obtener(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	producto, err := h.queries.GetProductoByID(c.Request.Context(), int32(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Producto no encontrado"})
		return
	}
	c.JSON(http.StatusOK, producto)
}

// Crear crea un nuevo producto de integración
func (h *ProductosIntegracionHandler) Crear(c *gin.Context) {
	var params db.CreateProductoParams
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
		return
	}
	producto, err := h.queries.CreateProducto(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear producto: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, producto)
}
