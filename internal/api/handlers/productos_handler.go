package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

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
	cveProdIntegracion := c.PostForm("cve_prod_integracion")
	descripcion := c.PostForm("descripcion")
	createdByStr := c.PostForm("created_by")

	if descripcion == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "La descripción es requerida"})
		return
	}

	var createdBy int32
	if createdByStr != "" {
		val, err := strconv.Atoi(createdByStr)
		if err == nil {
			createdBy = int32(val)
		}
	}

	var fotoUrl string
	fotoFile, err := c.FormFile("foto")
	if err == nil {
		uploadDir := "uploads/productos/fotos"
		if err := os.MkdirAll(uploadDir, os.ModePerm); err == nil {
			filename := fmt.Sprintf("%d_%s", time.Now().Unix(), filepath.Base(fotoFile.Filename))
			filePath := filepath.Join(uploadDir, filename)
			if err := c.SaveUploadedFile(fotoFile, filePath); err == nil {
				fotoUrl = "/uploads/productos/fotos/" + filename
			}
		}
	}
	if fotoUrl == "" {
		fotoUrl = c.PostForm("foto_url")
	}

	var fichaTecnicaUrl string
	fichaFile, err := c.FormFile("ficha_tecnica")
	if err == nil {
		uploadDir := "uploads/productos/fichas"
		if err := os.MkdirAll(uploadDir, os.ModePerm); err == nil {
			filename := fmt.Sprintf("%d_%s", time.Now().Unix(), filepath.Base(fichaFile.Filename))
			filePath := filepath.Join(uploadDir, filename)
			if err := c.SaveUploadedFile(fichaFile, filePath); err == nil {
				fichaTecnicaUrl = "/uploads/productos/fichas/" + filename
			}
		}
	}
	if fichaTecnicaUrl == "" {
		fichaTecnicaUrl = c.PostForm("ficha_tecnica")
	}

	producto, err := h.queries.CrearProductoPadre(c.Request.Context(), db.CrearProductoPadreParams{
		CveProdIntegracion: pgtype.Text{String: cveProdIntegracion, Valid: cveProdIntegracion != ""},
		Descripcion:        pgtype.Text{String: descripcion, Valid: descripcion != ""},
		FotoUrl:            pgtype.Text{String: fotoUrl, Valid: fotoUrl != ""},
		FichaTecnica:       pgtype.Text{String: fichaTecnicaUrl, Valid: fichaTecnicaUrl != ""},
		CreatedBy:          pgtype.Int4{Int32: createdBy, Valid: createdBy != 0},
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

	cveProdIntegracion := c.PostForm("cve_prod_integracion")
	descripcion := c.PostForm("descripcion")
	updatedByStr := c.PostForm("updated_by")

	if descripcion == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "La descripción es requerida"})
		return
	}

	var updatedBy int32
	if updatedByStr != "" {
		val, err := strconv.Atoi(updatedByStr)
		if err == nil {
			updatedBy = int32(val)
		}
	}

	var fotoUrl string
	fotoFile, err := c.FormFile("foto")
	if err == nil {
		uploadDir := "uploads/productos/fotos"
		if err := os.MkdirAll(uploadDir, os.ModePerm); err == nil {
			filename := fmt.Sprintf("%d_%s", time.Now().Unix(), filepath.Base(fotoFile.Filename))
			filePath := filepath.Join(uploadDir, filename)
			if err := c.SaveUploadedFile(fotoFile, filePath); err == nil {
				fotoUrl = "/uploads/productos/fotos/" + filename
			}
		}
	}
	if fotoUrl == "" {
		fotoUrl = c.PostForm("foto_url")
	}

	var fichaTecnicaUrl string
	fichaFile, err := c.FormFile("ficha_tecnica")
	if err == nil {
		uploadDir := "uploads/productos/fichas"
		if err := os.MkdirAll(uploadDir, os.ModePerm); err == nil {
			filename := fmt.Sprintf("%d_%s", time.Now().Unix(), filepath.Base(fichaFile.Filename))
			filePath := filepath.Join(uploadDir, filename)
			if err := c.SaveUploadedFile(fichaFile, filePath); err == nil {
				fichaTecnicaUrl = "/uploads/productos/fichas/" + filename
			}
		}
	}
	if fichaTecnicaUrl == "" {
		fichaTecnicaUrl = c.PostForm("ficha_tecnica")
	}

	producto, err := h.queries.ActualizarProductoPadre(c.Request.Context(), db.ActualizarProductoPadreParams{
		ID:                 int32(id),
		CveProdIntegracion: pgtype.Text{String: cveProdIntegracion, Valid: cveProdIntegracion != ""},
		Descripcion:        pgtype.Text{String: descripcion, Valid: descripcion != ""},
		FotoUrl:            pgtype.Text{String: fotoUrl, Valid: fotoUrl != ""},
		FichaTecnica:       pgtype.Text{String: fichaTecnicaUrl, Valid: fichaTecnicaUrl != ""},
		UpdatedBy:          pgtype.Int4{Int32: updatedBy, Valid: updatedBy != 0},
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
		Sku                string  `json:"sku" binding:"required"`
		Medida             string  `json:"medida"`
		PrecioDistribuidor float64 `json:"precio_distribuidor" binding:"required"`
		PrecioLista        float64 `json:"precio_lista" binding:"required"`
		PrecioPublico      float64 `json:"precio_publico" binding:"required"`
		StockActual        int32   `json:"stock_actual"`
		UnidadMedida       string  `json:"unidad_medida" binding:"required"`
		LeadTimeDias       int32   `json:"lead_time_dias"`
		Especificaciones   string  `json:"especificaciones"`
		Categoria          string  `json:"categoria"`
		Subgrupo           string  `json:"subgrupo"`
		Modelo             string  `json:"modelo"`
		Tipo               string  `json:"tipo"`
		Marca              string  `json:"marca"`
		CreatedBy          int32   `json:"created_by"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos de variante inválidos: " + err.Error()})
		return
	}

	variante, err := h.queries.CrearVariante(c.Request.Context(), db.CrearVarianteParams{
		PadreID:            pgtype.Int4{Int32: int32(padreID), Valid: true},
		Sku:                input.Sku,
		Medida:             pgtype.Text{String: input.Medida, Valid: input.Medida != ""},
		PrecioDistribuidor: utils.ToNumeric(input.PrecioDistribuidor),
		PrecioLista:        utils.ToNumeric(input.PrecioLista),
		PrecioPublico:      utils.ToNumeric(input.PrecioPublico),
		StockActual:        input.StockActual,
		UnidadMedida:       input.UnidadMedida,
		LeadTimeDias:       pgtype.Int4{Int32: input.LeadTimeDias, Valid: input.LeadTimeDias != 0},
		Especificaciones:   pgtype.Text{String: input.Especificaciones, Valid: input.Especificaciones != ""},
		Categoria:          pgtype.Text{String: input.Categoria, Valid: input.Categoria != ""},
		Subgrupo:           pgtype.Text{String: input.Subgrupo, Valid: input.Subgrupo != ""},
		Modelo:             pgtype.Text{String: input.Modelo, Valid: input.Modelo != ""},
		Tipo:               pgtype.Text{String: input.Tipo, Valid: input.Tipo != ""},
		Marca:              pgtype.Text{String: input.Marca, Valid: input.Marca != ""},
		CreatedBy:          pgtype.Int4{Int32: input.CreatedBy, Valid: input.CreatedBy != 0},
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
		PadreID            int32   `json:"padre_id"`
		Sku                string  `json:"sku" binding:"required"`
		Medida             string  `json:"medida"`
		PrecioDistribuidor float64 `json:"precio_distribuidor" binding:"required"`
		PrecioLista        float64 `json:"precio_lista" binding:"required"`
		PrecioPublico      float64 `json:"precio_publico" binding:"required"`
		StockActual        int32   `json:"stock_actual"`
		UnidadMedida       string  `json:"unidad_medida" binding:"required"`
		LeadTimeDias       int32   `json:"lead_time_dias"`
		Especificaciones   string  `json:"especificaciones"`
		Categoria          string  `json:"categoria"`
		Subgrupo           string  `json:"subgrupo"`
		Modelo             string  `json:"modelo"`
		Tipo               string  `json:"tipo"`
		Marca              string  `json:"marca"`
		UpdatedBy          int32   `json:"updated_by"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos de actualización inválidos: " + err.Error()})
		return
	}

	variante, err := h.queries.ActualizarVariante(c.Request.Context(), db.ActualizarVarianteParams{
		ID:                 int32(id),
		PadreID:            pgtype.Int4{Int32: input.PadreID, Valid: input.PadreID != 0},
		Sku:                input.Sku,
		Medida:             pgtype.Text{String: input.Medida, Valid: input.Medida != ""},
		PrecioDistribuidor: utils.ToNumeric(input.PrecioDistribuidor),
		PrecioLista:        utils.ToNumeric(input.PrecioLista),
		PrecioPublico:      utils.ToNumeric(input.PrecioPublico),
		StockActual:        input.StockActual,
		UnidadMedida:       input.UnidadMedida,
		LeadTimeDias:       pgtype.Int4{Int32: input.LeadTimeDias, Valid: input.LeadTimeDias != 0},
		Especificaciones:   pgtype.Text{String: input.Especificaciones, Valid: input.Especificaciones != ""},
		Categoria:          pgtype.Text{String: input.Categoria, Valid: input.Categoria != ""},
		Subgrupo:           pgtype.Text{String: input.Subgrupo, Valid: input.Subgrupo != ""},
		Modelo:             pgtype.Text{String: input.Modelo, Valid: input.Modelo != ""},
		Tipo:               pgtype.Text{String: input.Tipo, Valid: input.Tipo != ""},
		Marca:              pgtype.Text{String: input.Marca, Valid: input.Marca != ""},
		UpdatedBy:          pgtype.Int4{Int32: input.UpdatedBy, Valid: input.UpdatedBy != 0},
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
