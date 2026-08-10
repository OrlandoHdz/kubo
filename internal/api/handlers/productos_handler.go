package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/OrlandoHdz/kubo/internal/db"
	"github.com/OrlandoHdz/kubo/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type ProductosHandler struct {
	queries   *db.Queries
	ossConfig *utils.ConfigOSS
}

func NewProductosHandler(q *db.Queries, ossCfg *utils.ConfigOSS) *ProductosHandler {
	return &ProductosHandler{queries: q, ossConfig: ossCfg}
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
	descripcionExtendida := c.PostForm("descripcion_extendida")
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
		fotoUrl, err = h.ossConfig.ProcesarYSubirImagen(fotoFile)
		if err != nil {
			log.Printf("Error subiendo foto a OSS: %v", err)
			fotoUrl = ""
		}
	}
	if fotoUrl == "" {
		fotoUrl = c.PostForm("foto_url")
	}

	fotoUrl2 := c.PostForm("foto_url2")
	foto2File, err2 := c.FormFile("foto2")
	if err2 == nil {
		fotoUrl2, err2 = h.ossConfig.ProcesarYSubirImagen(foto2File)
		if err2 != nil {
			log.Printf("Error subiendo foto2 a OSS: %v", err2)
			fotoUrl2 = c.PostForm("foto_url2")
		}
	}

	fotoUrl3 := c.PostForm("foto_url_3")
	foto3File, err3 := c.FormFile("foto3")
	if err3 == nil {
		fotoUrl3, err3 = h.ossConfig.ProcesarYSubirImagen(foto3File)
		if err3 != nil {
			log.Printf("Error subiendo foto3 a OSS: %v", err3)
			fotoUrl3 = c.PostForm("foto_url_3")
		}
	}

	fotoUrl4 := c.PostForm("foto_url_4")
	foto4File, err4 := c.FormFile("foto4")
	if err4 == nil {
		fotoUrl4, err4 = h.ossConfig.ProcesarYSubirImagen(foto4File)
		if err4 != nil {
			log.Printf("Error subiendo foto4 a OSS: %v", err4)
			fotoUrl4 = c.PostForm("foto_url_4")
		}
	}

	fotoUrl5 := c.PostForm("foto_url_5")
	foto5File, err5 := c.FormFile("foto5")
	if err5 == nil {
		fotoUrl5, err5 = h.ossConfig.ProcesarYSubirImagen(foto5File)
		if err5 != nil {
			log.Printf("Error subiendo foto5 a OSS: %v", err5)
			fotoUrl5 = c.PostForm("foto_url_5")
		}
	}

	fotoUrl6 := c.PostForm("foto_url_6")
	foto6File, err6 := c.FormFile("foto6")
	if err6 == nil {
		fotoUrl6, err6 = h.ossConfig.ProcesarYSubirImagen(foto6File)
		if err6 != nil {
			log.Printf("Error subiendo foto6 a OSS: %v", err6)
			fotoUrl6 = c.PostForm("foto_url_6")
		}
	}

	fotoUrl7 := c.PostForm("foto_url_7")
	foto7File, err7 := c.FormFile("foto7")
	if err7 == nil {
		fotoUrl7, err7 = h.ossConfig.ProcesarYSubirImagen(foto7File)
		if err7 != nil {
			log.Printf("Error subiendo foto7 a OSS: %v", err7)
			fotoUrl7 = c.PostForm("foto_url_7")
		}
	}

	fotoUrl8 := c.PostForm("foto_url_8")
	foto8File, err8 := c.FormFile("foto8")
	if err8 == nil {
		fotoUrl8, err8 = h.ossConfig.ProcesarYSubirImagen(foto8File)
		if err8 != nil {
			log.Printf("Error subiendo foto8 a OSS: %v", err8)
			fotoUrl8 = c.PostForm("foto_url_8")
		}
	}

	var fichaTecnicaUrl string
	fichaFile, err := c.FormFile("ficha_tecnica")
	if err == nil {
		fichaTecnicaUrl, err = h.ossConfig.SubirFichaTecnica(fichaFile)
		if err != nil {
			log.Printf("Error subiendo ficha técnica a OSS: %v", err)
			fichaTecnicaUrl = ""
		}
	}
	if fichaTecnicaUrl == "" {
		fichaTecnicaUrl = c.PostForm("ficha_tecnica")
	}

	titulo := c.PostForm("titulo")

	if cveProdIntegracion != "" {
		existeEliminado, err := h.queries.ExisteProductoPadreEliminadoPorCve(c.Request.Context(), pgtype.Text{String: cveProdIntegracion, Valid: true})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al validar producto: " + err.Error()})
			return
		}
		if existeEliminado {
			c.JSON(http.StatusConflict, gin.H{"error": "Ya existe un producto eliminado con ese cve_prod_integracion"})
			return
		}
	}

	producto, err := h.queries.CrearProductoPadre(c.Request.Context(), db.CrearProductoPadreParams{
		CveProdIntegracion:   pgtype.Text{String: cveProdIntegracion, Valid: cveProdIntegracion != ""},
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
		FichaTecnica:         pgtype.Text{String: fichaTecnicaUrl, Valid: fichaTecnicaUrl != ""},
		CreatedBy:            pgtype.Int4{Int32: createdBy, Valid: createdBy != 0},
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
	descripcionExtendida := c.PostForm("descripcion_extendida")
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
		fotoUrl, err = h.ossConfig.ProcesarYSubirImagen(fotoFile)
		if err != nil {
			log.Printf("Error subiendo foto a OSS: %v", err)
			fotoUrl = ""
		}
	}
	if fotoUrl == "" {
		fotoUrl = c.PostForm("foto_url")
	}

	fotoUrl2 := c.PostForm("foto_url2")
	foto2File, err2 := c.FormFile("foto2")
	if err2 == nil {
		fotoUrl2, err2 = h.ossConfig.ProcesarYSubirImagen(foto2File)
		if err2 != nil {
			log.Printf("Error subiendo foto2 a OSS: %v", err2)
			fotoUrl2 = c.PostForm("foto_url2")
		}
	}

	fotoUrl3 := c.PostForm("foto_url_3")
	foto3File, err3 := c.FormFile("foto3")
	if err3 == nil {
		fotoUrl3, err3 = h.ossConfig.ProcesarYSubirImagen(foto3File)
		if err3 != nil {
			log.Printf("Error subiendo foto3 a OSS: %v", err3)
			fotoUrl3 = c.PostForm("foto_url_3")
		}
	}

	fotoUrl4 := c.PostForm("foto_url_4")
	foto4File, err4 := c.FormFile("foto4")
	if err4 == nil {
		fotoUrl4, err4 = h.ossConfig.ProcesarYSubirImagen(foto4File)
		if err4 != nil {
			log.Printf("Error subiendo foto4 a OSS: %v", err4)
			fotoUrl4 = c.PostForm("foto_url_4")
		}
	}

	fotoUrl5 := c.PostForm("foto_url_5")
	foto5File, err5 := c.FormFile("foto5")
	if err5 == nil {
		fotoUrl5, err5 = h.ossConfig.ProcesarYSubirImagen(foto5File)
		if err5 != nil {
			log.Printf("Error subiendo foto5 a OSS: %v", err5)
			fotoUrl5 = c.PostForm("foto_url_5")
		}
	}

	fotoUrl6 := c.PostForm("foto_url_6")
	foto6File, err6 := c.FormFile("foto6")
	if err6 == nil {
		fotoUrl6, err6 = h.ossConfig.ProcesarYSubirImagen(foto6File)
		if err6 != nil {
			log.Printf("Error subiendo foto6 a OSS: %v", err6)
			fotoUrl6 = c.PostForm("foto_url_6")
		}
	}

	fotoUrl7 := c.PostForm("foto_url_7")
	foto7File, err7 := c.FormFile("foto7")
	if err7 == nil {
		fotoUrl7, err7 = h.ossConfig.ProcesarYSubirImagen(foto7File)
		if err7 != nil {
			log.Printf("Error subiendo foto7 a OSS: %v", err7)
			fotoUrl7 = c.PostForm("foto_url_7")
		}
	}

	fotoUrl8 := c.PostForm("foto_url_8")
	foto8File, err8 := c.FormFile("foto8")
	if err8 == nil {
		fotoUrl8, err8 = h.ossConfig.ProcesarYSubirImagen(foto8File)
		if err8 != nil {
			log.Printf("Error subiendo foto8 a OSS: %v", err8)
			fotoUrl8 = c.PostForm("foto_url_8")
		}
	}

	var fichaTecnicaUrl string
	fichaFile, err := c.FormFile("ficha_tecnica")
	if err == nil {
		fichaTecnicaUrl, err = h.ossConfig.SubirFichaTecnica(fichaFile)
		if err != nil {
			log.Printf("Error subiendo ficha técnica a OSS: %v", err)
			fichaTecnicaUrl = ""
		}
	}
	if fichaTecnicaUrl == "" {
		fichaTecnicaUrl = c.PostForm("ficha_tecnica")
	}

	titulo := c.PostForm("titulo")

	producto, err := h.queries.ActualizarProductoPadre(c.Request.Context(), db.ActualizarProductoPadreParams{
		ID:                   int32(id),
		CveProdIntegracion:   pgtype.Text{String: cveProdIntegracion, Valid: cveProdIntegracion != ""},
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
		FichaTecnica:         pgtype.Text{String: fichaTecnicaUrl, Valid: fichaTecnicaUrl != ""},
		UpdatedBy:            pgtype.Int4{Int32: updatedBy, Valid: updatedBy != 0},
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
		PrecioDistribuidor float64 `json:"precio_distribuidor" `
		PrecioLista        float64 `json:"precio_lista" `
		PrecioPublico      float64 `json:"precio_publico" `
		StockActual        int32   `json:"stock_actual"`
		UnidadMedida       string  `json:"unidad_medida" `
		LeadTimeDias       int32   `json:"lead_time_dias"`
		Especificaciones   string  `json:"especificaciones"`
		Categoria          string  `json:"categoria"`
		Subgrupo           string  `json:"subgrupo"`
		Modelo             string  `json:"modelo"`
		Tipo               string  `json:"tipo"`
		Marca              string  `json:"marca"`
		Multipos           int32   `json:"multiplos"`
		PermitirBackorder  bool    `json:"permitir_backorder"`
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
		Multiplos:          pgtype.Int4{Int32: input.Multipos, Valid: true},
		PermitirBackorder:  pgtype.Bool{Bool: input.PermitirBackorder, Valid: true},
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
		Multipos           int32   `json:"multiplos"`
		PermitirBackorder  bool    `json:"permitir_backorder"`
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
		Multiplos:          pgtype.Int4{Int32: input.Multipos, Valid: true},
		PermitirBackorder:  pgtype.Bool{Bool: input.PermitirBackorder, Valid: true},
		UpdatedBy:          pgtype.Int4{Int32: input.UpdatedBy, Valid: input.UpdatedBy != 0},
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar variante: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, variante)
}

// ObtenerVariante devuelve una variante con su existencia total
func (h *ProductosHandler) ObtenerVariante(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de variante inválido"})
		return
	}

	variante, err := h.queries.GetVarianteConExistencia(c.Request.Context(), int32(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Variante no encontrada"})
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
