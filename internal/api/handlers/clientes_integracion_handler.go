package handlers

import (
	"net/http"
	"strconv"

	"github.com/OrlandoHdz/kubo/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type ClientesIntegracionHandler struct {
	queries *db.Queries
}

func NewClientesIntegracionHandler(q *db.Queries) *ClientesIntegracionHandler {
	return &ClientesIntegracionHandler{queries: q}
}

// Listar devuelve todos los clientes de integración
func (h *ClientesIntegracionHandler) Listar(c *gin.Context) {
	clientes, err := h.queries.ObtenerClientesIntegracion(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener clientes: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, clientes)
}

// Obtener devuelve un cliente de integración por su ID
func (h *ClientesIntegracionHandler) Obtener(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	cliente, err := h.queries.ObtenerClienteIntegracion(c.Request.Context(), int32(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cliente no encontrado"})
		return
	}
	c.JSON(http.StatusOK, cliente)
}

// ObtenerPorCveCte devuelve un cliente de integración por su CVE_CTE
func (h *ClientesIntegracionHandler) ObtenerPorCveCte(c *gin.Context) {
	cveStr := c.Param("cve")
	cve, err := strconv.Atoi(cveStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "CVE_CTE inválido"})
		return
	}

	cliente, err := h.queries.ObtenerClientesIntegracionPorCveCte(c.Request.Context(), pgtype.Int4{Int32: int32(cve), Valid: true})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cliente no encontrado"})
		return
	}
	c.JSON(http.StatusOK, cliente)
}

// Crear crea un nuevo cliente de integración
func (h *ClientesIntegracionHandler) Crear(c *gin.Context) {
	var params db.CrearClienteIntegracionParams
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
		return
	}

	cliente, err := h.queries.CrearClienteIntegracion(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear cliente: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, cliente)
}
