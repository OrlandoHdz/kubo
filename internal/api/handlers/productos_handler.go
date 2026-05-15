package handlers

import (
	"net/http"
	"strconv"

	"github.com/OrlandoHdz/kubo/internal/db"
	"github.com/OrlandoHdz/kubo/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type ProductosHandler struct {
	queries *db.Queries
}

func NewProductosHandler(q *db.Queries) *ProductosHandler {
	return &ProductosHandler{queries: q}
}

// ==========================================
// CRUD PRODUCTOS PADRE
// ==========================================

// ListarPadres devuelve todos los productos padre activos
func (h *ProductosHandler) ListarPadres(c *gin.Context) {
	productos, err := h.queries.ListarProductosPadre(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al listar productos: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, productos)
}

// ObtenerPadre devuelve un producto padre por su ID
func (h *ProductosHandler) ObtenerPadre(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de producto inválido"})
		return
	}

	producto, err := h.queries.GetProductoPadre(c.Request.Context(), int32(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Producto no encontrado"})
		return
	}
	c.JSON(http.StatusOK, producto)
}

// CrearPadre registra un nuevo producto padre
func (h *ProductosHandler) CrearPadre(c *gin.Context) {
	var input struct {
		NombreTecnico    string `json:"nombre_tecnico" binding:"required"`
		Descripcion      string `json:"descripcion"`
		Categoria        string `json:"categoria" binding:"required"`
		Marca            string `json:"marca"`
		DocumentacionUrl string `json:"documentacion_url"`
		CreatedBy        int32  `json:"created_by"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos de producto inválidos: " + err.Error()})
		return
	}

	producto, err := h.queries.CrearProductoPadre(c.Request.Context(), db.CrearProductoPadreParams{
		NombreTecnico:    input.NombreTecnico,
		Descripcion:      pgtype.Text{String: input.Descripcion, Valid: input.Descripcion != ""},
		Categoria:        input.Categoria,
		Marca:            pgtype.Text{String: input.Marca, Valid: input.Marca != ""},
		DocumentacionUrl: pgtype.Text{String: input.DocumentacionUrl, Valid: input.DocumentacionUrl != ""},
		CreatedBy:        pgtype.Int4{Int32: input.CreatedBy, Valid: input.CreatedBy != 0},
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear producto: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, producto)
}

// ActualizarPadre modifica los datos de un producto padre existente
func (h *ProductosHandler) ActualizarPadre(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de producto inválido"})
		return
	}

	var input struct {
		NombreTecnico    string `json:"nombre_tecnico" binding:"required"`
		Descripcion      string `json:"descripcion"`
		Categoria        string `json:"categoria" binding:"required"`
		Marca            string `json:"marca"`
		DocumentacionUrl string `json:"documentacion_url"`
		UpdatedBy        int32  `json:"updated_by"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos de actualización inválidos: " + err.Error()})
		return
	}

	producto, err := h.queries.ActualizarProductoPadre(c.Request.Context(), db.ActualizarProductoPadreParams{
		ID:               int32(id),
		NombreTecnico:    input.NombreTecnico,
		Descripcion:      pgtype.Text{String: input.Descripcion, Valid: input.Descripcion != ""},
		Categoria:        input.Categoria,
		Marca:            pgtype.Text{String: input.Marca, Valid: input.Marca != ""},
		DocumentacionUrl: pgtype.Text{String: input.DocumentacionUrl, Valid: input.DocumentacionUrl != ""},
		UpdatedBy:        pgtype.Int4{Int32: input.UpdatedBy, Valid: input.UpdatedBy != 0},
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar producto: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, producto)
}

// EliminarPadre realiza un borrado lógico del producto padre
func (h *ProductosHandler) EliminarPadre(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de producto inválido"})
		return
	}

	err = h.queries.SoftDeleteProductoPadre(c.Request.Context(), db.SoftDeleteProductoPadreParams{
		ID:        int32(id),
		DeletedBy: pgtype.Int4{Valid: false},
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al eliminar producto: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Producto eliminado correctamente"})
}

// ==========================================
// CRUD VARIANTES
// ==========================================

// ListarVariantes devuelve todas las variantes de un producto padre
func (h *ProductosHandler) ListarVariantes(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de producto padre inválido"})
		return
	}

	variantes, err := h.queries.ListarVariantesPorPadre(c.Request.Context(), pgtype.Int4{Int32: int32(id), Valid: true})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al listar variantes: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, variantes)
}

// CrearVariante registra una nueva variante para un producto padre
func (h *ProductosHandler) CrearVariante(c *gin.Context) {
	idStr := c.Param("id")
	padreID, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de producto padre inválido"})
		return
	}

	var input struct {
		Sku              string  `json:"sku" binding:"required"`
		Medida           string  `json:"medida"`
		PrecioLista      float64 `json:"precio_lista" binding:"required"`
		StockActual      int32   `json:"stock_actual"`
		UnidadMedida     string  `json:"unidad_medida" binding:"required"`
		LeadTimeDias     int32   `json:"lead_time_dias"`
		Especificaciones string  `json:"especificaciones"`
		CreatedBy        int32   `json:"created_by"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos de variante inválidos: " + err.Error()})
		return
	}

	variante, err := h.queries.CrearVariante(c.Request.Context(), db.CrearVarianteParams{
		PadreID:      pgtype.Int4{Int32: int32(padreID), Valid: true},
		Sku:              input.Sku,
		Medida:           pgtype.Text{String: input.Medida, Valid: input.Medida != ""},
		PrecioLista:      utils.ToNumeric(input.PrecioLista),
		StockActual:      input.StockActual,
		UnidadMedida:     input.UnidadMedida,
		LeadTimeDias:     pgtype.Int4{Int32: input.LeadTimeDias, Valid: input.LeadTimeDias != 0},
		Especificaciones: pgtype.Text{String: input.Especificaciones, Valid: input.Especificaciones != ""},
		CreatedBy:        pgtype.Int4{Int32: input.CreatedBy, Valid: input.CreatedBy != 0},
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear variante: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, variante)
}

// ActualizarStock modifica solo el stock de una variante
func (h *ProductosHandler) ActualizarStock(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de variante inválido"})
		return
	}

	var input struct {
		StockActual int32 `json:"stock_actual"`
		UpdatedBy   int32 `json:"updated_by"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos de stock inválidos: " + err.Error()})
		return
	}

	err = h.queries.ActualizarStock(c.Request.Context(), db.ActualizarStockParams{
		ID:          int32(id),
		StockActual: input.StockActual,
		UpdatedBy:   pgtype.Int4{Int32: input.UpdatedBy, Valid: input.UpdatedBy != 0},
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar stock: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Stock actualizado correctamente"})
}

// ActualizarVariante modifica los datos de una variante existente
func (h *ProductosHandler) ActualizarVariante(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de variante inválido"})
		return
	}

	var input struct {
		Sku              string  `json:"sku" binding:"required"`
		Medida           string  `json:"medida"`
		PrecioLista      float64 `json:"precio_lista" binding:"required"`
		StockActual      int32   `json:"stock_actual"`
		UnidadMedida     string  `json:"unidad_medida" binding:"required"`
		LeadTimeDias     int32   `json:"lead_time_dias"`
		Especificaciones string  `json:"especificaciones"`
		UpdatedBy        int32   `json:"updated_by"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos de actualización inválidos: " + err.Error()})
		return
	}

	variante, err := h.queries.ActualizarVariante(c.Request.Context(), db.ActualizarVarianteParams{
		ID:               int32(id),
		Sku:              input.Sku,
		Medida:           pgtype.Text{String: input.Medida, Valid: input.Medida != ""},
		PrecioLista:      utils.ToNumeric(input.PrecioLista),
		StockActual:      input.StockActual,
		UnidadMedida:     input.UnidadMedida,
		LeadTimeDias:     pgtype.Int4{Int32: input.LeadTimeDias, Valid: input.LeadTimeDias != 0},
		Especificaciones: pgtype.Text{String: input.Especificaciones, Valid: input.Especificaciones != ""},
		UpdatedBy:        pgtype.Int4{Int32: input.UpdatedBy, Valid: input.UpdatedBy != 0},
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar variante: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, variante)
}

// EliminarVariante realiza un borrado lógico de la variante
func (h *ProductosHandler) EliminarVariante(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de variante inválido"})
		return
	}

	err = h.queries.SoftDeleteVariante(c.Request.Context(), db.SoftDeleteVarianteParams{
		ID:        int32(id),
		DeletedBy: pgtype.Int4{Valid: false},
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al eliminar variante: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Variante eliminada correctamente"})
}
