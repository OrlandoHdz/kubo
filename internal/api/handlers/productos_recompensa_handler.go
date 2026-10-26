package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/OrlandoHdz/kubo/internal/db"
	"github.com/OrlandoHdz/kubo/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type ProductosRecompensaHandler struct {
	queries *db.Queries
	ossCfg  *utils.ConfigOSS
}

func NewProductosRecompensaHandler(q *db.Queries, ossCfg *utils.ConfigOSS) *ProductosRecompensaHandler {
	return &ProductosRecompensaHandler{queries: q, ossCfg: ossCfg}
}

// ===== PRODUCTOS PADRE =====

func (h *ProductosRecompensaHandler) ListarPadres(c *gin.Context) {
	padres, err := h.queries.ListarProductosRecompensaPadre(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al listar productos: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, padres)
}

func (h *ProductosRecompensaHandler) ObtenerPadre(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de producto inválido"})
		return
	}

	padre, err := h.queries.GetProductoRecompensaPadre(c.Request.Context(), int32(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Producto no encontrado"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"padre":     padre,
		"variantes": []db.ProductosRecompensaVariante{},
	})
}

func (h *ProductosRecompensaHandler) CrearPadre(c *gin.Context) {
	cveProd := c.PostForm("cve_prod_integracion")
	titulo := c.PostForm("titulo")
	descripcion := c.PostForm("descripcion")
	if descripcion == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "La descripción es requerida"})
		return
	}
	descripcionExtendida := c.PostForm("descripcion_extendida")
	fichaTecnicaURL := c.PostForm("ficha_tecnica_url")

	var createdBy int32
	if createdByStr := c.PostForm("created_by"); createdByStr != "" {
		if val, err := strconv.Atoi(createdByStr); err == nil {
			createdBy = int32(val)
		}
	}

	var fotoUrl, fotoUrl2, fotoUrl3, fotoUrl4, fotoUrl5, fotoUrl6, fotoUrl7, fotoUrl8 string
	var fichaUrl string

	// Upload 8 fotos
	for i, fieldName := range []string{"foto", "foto2", "foto3", "foto4", "foto5", "foto6", "foto7", "foto8"} {
		file, err := c.FormFile(fieldName)
		urlVar := &fotoUrl
		if i == 1 {
			urlVar = &fotoUrl2
		}
		if i == 2 {
			urlVar = &fotoUrl3
		}
		if i == 3 {
			urlVar = &fotoUrl4
		}
		if i == 4 {
			urlVar = &fotoUrl5
		}
		if i == 5 {
			urlVar = &fotoUrl6
		}
		if i == 6 {
			urlVar = &fotoUrl7
		}
		if i == 7 {
			urlVar = &fotoUrl8
		}

		if err == nil {
			if h.ossCfg != nil {
				*urlVar, _ = h.ossCfg.ProcesarYSubirImagen(file)
			} else {
				ext := strings.ToLower(filepath.Ext(file.Filename))
				filename := fmt.Sprintf("recompensas/%d_%d%s", time.Now().UnixMilli(), time.Now().UnixNano(), ext)
				filePath := filepath.Join("uploads", filename)
				if err := c.SaveUploadedFile(file, filePath); err == nil {
					*urlVar = "/" + filePath
				}
			}
		}
	}

	// Upload ficha técnica PDF
	fichaFile, err := c.FormFile("ficha_tecnica")
	if err == nil {
		dir := "uploads/fichas"
		os.MkdirAll(dir, 0755)
		filename := fmt.Sprintf("recompensas/%d_%d.pdf", time.Now().UnixMilli(), time.Now().UnixNano())
		filePath := filepath.Join(dir, filename)
		if err := c.SaveUploadedFile(fichaFile, filePath); err == nil {
			fichaUrl = "/" + filePath
		}
	}
	if fichaUrl == "" {
		fichaUrl = fichaTecnicaURL
	}

	padre, err := h.queries.CrearProductoRecompensaPadre(c.Request.Context(), db.CrearProductoRecompensaPadreParams{
		CveProdIntegracion:   pgtype.Text{String: cveProd, Valid: cveProd != ""},
		Descripcion:          pgtype.Text{String: descripcion, Valid: descripcion != ""},
		Titulo:               pgtype.Text{String: titulo, Valid: titulo != ""},
		DescripcionExtendida: pgtype.Text{String: descripcionExtendida, Valid: descripcionExtendida != ""},
		FotoUrl:              pgtype.Text{String: fotoUrl, Valid: fotoUrl != ""},
		FotoUrl2:             pgtype.Text{String: fotoUrl2, Valid: fotoUrl2 != ""},
		FotoUrl3:             pgtype.Text{String: fotoUrl3, Valid: fotoUrl3 != ""},
		FotoUrl4:             pgtype.Text{String: fotoUrl4, Valid: fotoUrl4 != ""},
		FotoUrl5:             pgtype.Text{String: fotoUrl5, Valid: fotoUrl5 != ""},
		FotoUrl6:             pgtype.Text{String: fotoUrl6, Valid: fotoUrl6 != ""},
		FotoUrl7:             pgtype.Text{String: fotoUrl7, Valid: fotoUrl7 != ""},
		FotoUrl8:             pgtype.Text{String: fotoUrl8, Valid: fotoUrl8 != ""},
		FichaTecnica:         pgtype.Text{String: fichaUrl, Valid: fichaUrl != ""},
		CreatedBy:            pgtype.Int4{Int32: createdBy, Valid: createdBy != 0},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear producto padre: " + err.Error()})
		return
	}

	// Crear variante default vacía
	_, _ = h.queries.CrearVarianteRecompensa(c.Request.Context(), db.CrearVarianteRecompensaParams{
		PadreID:      padre.ID,
		Sku:          fmt.Sprintf("REC-%d-01", padre.ID),
		PrecioPuntos: 0,
		StockActual:  0,
		UnidadMedida: "Pza",
		CreatedBy:    pgtype.Int4{Int32: createdBy, Valid: createdBy != 0},
	})

	c.JSON(http.StatusCreated, padre)
}

func (h *ProductosRecompensaHandler) ActualizarPadre(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de producto inválido"})
		return
	}

	existing, err := h.queries.GetProductoRecompensaPadre(c.Request.Context(), int32(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Producto no encontrado"})
		return
	}

	cveProd := c.PostForm("cve_prod_integracion")
	titulo := c.PostForm("titulo")
	descripcion := c.PostForm("descripcion")
	descripcionExtendida := c.PostForm("descripcion_extendida")

	var updatedBy int32
	if updatedByStr := c.PostForm("updated_by"); updatedByStr != "" {
		if val, err := strconv.Atoi(updatedByStr); err == nil {
			updatedBy = int32(val)
		}
	}

	fotoUrl := existing.FotoUrl.String
	fotoUrl2 := existing.FotoUrl2.String
	fotoUrl3 := existing.FotoUrl3.String
	fotoUrl4 := existing.FotoUrl4.String
	fotoUrl5 := existing.FotoUrl5.String
	fotoUrl6 := existing.FotoUrl6.String
	fotoUrl7 := existing.FotoUrl7.String
	fotoUrl8 := existing.FotoUrl8.String
	fichaUrl := existing.FichaTecnica.String

	for i, fieldName := range []string{"foto", "foto2", "foto3", "foto4", "foto5", "foto6", "foto7", "foto8"} {
		file, err := c.FormFile(fieldName)
		urlVar := &fotoUrl
		if i == 1 {
			urlVar = &fotoUrl2
		}
		if i == 2 {
			urlVar = &fotoUrl3
		}
		if i == 3 {
			urlVar = &fotoUrl4
		}
		if i == 4 {
			urlVar = &fotoUrl5
		}
		if i == 5 {
			urlVar = &fotoUrl6
		}
		if i == 6 {
			urlVar = &fotoUrl7
		}
		if i == 7 {
			urlVar = &fotoUrl8
		}

		if err == nil {
			if h.ossCfg != nil {
				*urlVar, _ = h.ossCfg.ProcesarYSubirImagen(file)
			} else {
				ext := strings.ToLower(filepath.Ext(file.Filename))
				filename := fmt.Sprintf("recompensas/%d_%d%s", time.Now().UnixMilli(), time.Now().UnixNano(), ext)
				filePath := filepath.Join("uploads", filename)
				if err := c.SaveUploadedFile(file, filePath); err == nil {
					*urlVar = "/" + filePath
				}
			}
		}
	}

	fichaFile, err := c.FormFile("ficha_tecnica")
	if err == nil {
		dir := "uploads/fichas"
		os.MkdirAll(dir, 0755)
		filename := fmt.Sprintf("recompensas/%d_%d.pdf", time.Now().UnixMilli(), time.Now().UnixNano())
		filePath := filepath.Join(dir, filename)
		if err := c.SaveUploadedFile(fichaFile, filePath); err == nil {
			fichaUrl = "/" + filePath
		}
	}

	padre, err := h.queries.ActualizarProductoRecompensaPadre(c.Request.Context(), db.ActualizarProductoRecompensaPadreParams{
		ID:                   int32(id),
		CveProdIntegracion:   pgtype.Text{String: cveProd, Valid: cveProd != ""},
		Descripcion:          pgtype.Text{String: descripcion, Valid: descripcion != ""},
		Titulo:               pgtype.Text{String: titulo, Valid: titulo != ""},
		DescripcionExtendida: pgtype.Text{String: descripcionExtendida, Valid: descripcionExtendida != ""},
		FotoUrl:              pgtype.Text{String: fotoUrl, Valid: fotoUrl != ""},
		FotoUrl2:             pgtype.Text{String: fotoUrl2, Valid: fotoUrl2 != ""},
		FotoUrl3:             pgtype.Text{String: fotoUrl3, Valid: fotoUrl3 != ""},
		FotoUrl4:             pgtype.Text{String: fotoUrl4, Valid: fotoUrl4 != ""},
		FotoUrl5:             pgtype.Text{String: fotoUrl5, Valid: fotoUrl5 != ""},
		FotoUrl6:             pgtype.Text{String: fotoUrl6, Valid: fotoUrl6 != ""},
		FotoUrl7:             pgtype.Text{String: fotoUrl7, Valid: fotoUrl7 != ""},
		FotoUrl8:             pgtype.Text{String: fotoUrl8, Valid: fotoUrl8 != ""},
		FichaTecnica:         pgtype.Text{String: fichaUrl, Valid: fichaUrl != ""},
		UpdatedBy:            pgtype.Int4{Int32: updatedBy, Valid: updatedBy != 0},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar producto padre: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, padre)
}

func (h *ProductosRecompensaHandler) EliminarPadre(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de producto inválido"})
		return
	}

	var input struct{ DeletedBy int32 }
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	err = h.queries.SoftDeleteProductoRecompensaPadre(c.Request.Context(), db.SoftDeleteProductoRecompensaPadreParams{
		ID: int32(id), DeletedBy: pgtype.Int4{Int32: input.DeletedBy, Valid: input.DeletedBy != 0},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al eliminar producto: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Producto eliminado"})
}

// ===== VARIANTES =====

func (h *ProductosRecompensaHandler) ListarVariantes(c *gin.Context) {
	padreIDStr := c.Param("id")
	padreID, err := strconv.Atoi(padreIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de producto inválido"})
		return
	}

	variantes, err := h.queries.ListarVariantesRecompensaPorPadre(c.Request.Context(), int32(padreID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al listar variantes: " + err.Error()})
		return
	}

	padre, _ := h.queries.GetProductoRecompensaPadre(c.Request.Context(), int32(padreID))

	c.JSON(http.StatusOK, gin.H{
		"padre":     padre,
		"variantes": variantes,
	})
}

func (h *ProductosRecompensaHandler) CrearVariante(c *gin.Context) {
	padreIDStr := c.Param("id")
	padreID, err := strconv.Atoi(padreIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de producto inválido"})
		return
	}

	var input struct {
		Sku               string `json:"sku" binding:"required"`
		Medida            string `json:"medida"`
		PrecioPuntos      int32  `json:"precio_puntos" binding:"required"`
		StockActual       int32  `json:"stock_actual"`
		UnidadMedida      string `json:"unidad_medida"`
		LeadTimeDias      int32  `json:"lead_time_dias"`
		Especificaciones  string `json:"especificaciones"`
		Categoria         string `json:"categoria"`
		Subgrupo          string `json:"subgrupo"`
		Modelo            string `json:"modelo"`
		Tipo              string `json:"tipo"`
		Marca             string `json:"marca"`
		Multiplos         int32  `json:"multiplos"`
		PermitirBackorder bool   `json:"permitir_backorder"`
		CreatedBy         int32  `json:"created_by"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
		return
	}
	if input.UnidadMedida == "" {
		input.UnidadMedida = "Pza"
	}

	variante, err := h.queries.CrearVarianteRecompensa(c.Request.Context(), db.CrearVarianteRecompensaParams{
		PadreID:           int32(padreID),
		Sku:               input.Sku,
		Medida:            pgtype.Text{String: input.Medida, Valid: input.Medida != ""},
		PrecioPuntos:      input.PrecioPuntos,
		StockActual:       0,
		UnidadMedida:      input.UnidadMedida,
		LeadTimeDias:      pgtype.Int4{Int32: input.LeadTimeDias, Valid: input.LeadTimeDias != 0},
		Especificaciones:  pgtype.Text{String: input.Especificaciones, Valid: input.Especificaciones != ""},
		Categoria:         pgtype.Text{String: input.Categoria, Valid: input.Categoria != ""},
		Subgrupo:          pgtype.Text{String: input.Subgrupo, Valid: input.Subgrupo != ""},
		Modelo:            pgtype.Text{String: input.Modelo, Valid: input.Modelo != ""},
		Tipo:              pgtype.Text{String: input.Tipo, Valid: input.Tipo != ""},
		Marca:             pgtype.Text{String: input.Marca, Valid: input.Marca != ""},
		Multiplos:         pgtype.Int4{Int32: input.Multiplos, Valid: input.Multiplos != 0},
		PermitirBackorder: pgtype.Bool{Bool: input.PermitirBackorder, Valid: true},
		CreatedBy:         pgtype.Int4{Int32: input.CreatedBy, Valid: input.CreatedBy != 0},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear variante: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, variante)
}

func (h *ProductosRecompensaHandler) ActualizarVariante(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de variante inválido"})
		return
	}

	var input struct {
		PadreID           int32  `json:"padre_id"`
		Sku               string `json:"sku" binding:"required"`
		Medida            string `json:"medida"`
		PrecioPuntos      int32  `json:"precio_puntos" binding:"required"`
		StockActual       int32  `json:"stock_actual"`
		UnidadMedida      string `json:"unidad_medida"`
		LeadTimeDias      int32  `json:"lead_time_dias"`
		Especificaciones  string `json:"especificaciones"`
		Categoria         string `json:"categoria"`
		Subgrupo          string `json:"subgrupo"`
		Modelo            string `json:"modelo"`
		Tipo              string `json:"tipo"`
		Marca             string `json:"marca"`
		Multiplos         int32  `json:"multiplos"`
		PermitirBackorder bool   `json:"permitir_backorder"`
		UpdatedBy         int32  `json:"updated_by"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
		return
	}

	// Stock gestionado solo vía Ajustes de Inventario, mantener valor actual
	existingVar, _ := h.queries.GetVarianteRecompensa(c.Request.Context(), int32(id))
	stockActual := existingVar.StockActual
	variante, err := h.queries.ActualizarVarianteRecompensa(c.Request.Context(), db.ActualizarVarianteRecompensaParams{
		ID:                int32(id),
		PadreID:           input.PadreID,
		Sku:               input.Sku,
		Medida:            pgtype.Text{String: input.Medida, Valid: input.Medida != ""},
		PrecioPuntos:      input.PrecioPuntos,
		StockActual:       stockActual,
		UnidadMedida:      input.UnidadMedida,
		LeadTimeDias:      pgtype.Int4{Int32: input.LeadTimeDias, Valid: input.LeadTimeDias != 0},
		Especificaciones:  pgtype.Text{String: input.Especificaciones, Valid: input.Especificaciones != ""},
		Categoria:         pgtype.Text{String: input.Categoria, Valid: input.Categoria != ""},
		Subgrupo:          pgtype.Text{String: input.Subgrupo, Valid: input.Subgrupo != ""},
		Modelo:            pgtype.Text{String: input.Modelo, Valid: input.Modelo != ""},
		Tipo:              pgtype.Text{String: input.Tipo, Valid: input.Tipo != ""},
		Marca:             pgtype.Text{String: input.Marca, Valid: input.Marca != ""},
		Multiplos:         pgtype.Int4{Int32: input.Multiplos, Valid: input.Multiplos != 0},
		PermitirBackorder: pgtype.Bool{Bool: input.PermitirBackorder, Valid: true},
		UpdatedBy:         pgtype.Int4{Int32: input.UpdatedBy, Valid: input.UpdatedBy != 0},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar variante: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, variante)
}

func (h *ProductosRecompensaHandler) ActualizarStock(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de variante inválido"})
		return
	}

	var input struct {
		StockActual int32 `json:"stock_actual" binding:"required"`
		UpdatedBy   int32 `json:"updated_by"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	err = h.queries.ActualizarStockVarianteRecompensa(c.Request.Context(), db.ActualizarStockVarianteRecompensaParams{
		ID:          int32(id),
		StockActual: input.StockActual,
		UpdatedBy:   pgtype.Int4{Int32: input.UpdatedBy, Valid: input.UpdatedBy != 0},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar stock: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Stock actualizado"})
}

func (h *ProductosRecompensaHandler) ObtenerVariante(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de variante inválido"})
		return
	}

	variante, err := h.queries.GetVarianteRecompensaConExistencia(c.Request.Context(), int32(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Variante no encontrada"})
		return
	}
	c.JSON(http.StatusOK, variante)
}

func (h *ProductosRecompensaHandler) EliminarVariante(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de variante inválido"})
		return
	}

	var input struct{ DeletedBy int32 }
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	err = h.queries.SoftDeleteVarianteRecompensa(c.Request.Context(), db.SoftDeleteVarianteRecompensaParams{
		ID: int32(id), DeletedBy: pgtype.Int4{Int32: input.DeletedBy, Valid: input.DeletedBy != 0},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al eliminar variante: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Variante eliminada"})
}
