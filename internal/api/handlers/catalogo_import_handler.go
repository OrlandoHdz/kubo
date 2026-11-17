package handlers

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/OrlandoHdz/kubo/internal/services"
	"github.com/gin-gonic/gin"
)

type CatalogoImportHandler struct {
	service *services.CatalogoExcelService
}

func NewCatalogoImportHandler(service *services.CatalogoExcelService) *CatalogoImportHandler {
	return &CatalogoImportHandler{service: service}
}

// ImportarExcel recibe un archivo .xlsx y lo importa al catálogo.
// El campo multipart esperado es "archivo" (también se acepta "file").
func (h *CatalogoImportHandler) ImportarExcel(c *gin.Context) {
	archivo, err := c.FormFile("archivo")
	if err != nil {
		archivo, err = c.FormFile("file")
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "se requiere un archivo Excel en el campo 'archivo'"})
		return
	}

	if !strings.EqualFold(filepath.Ext(archivo.Filename), ".xlsx") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "el archivo debe tener extensión .xlsx"})
		return
	}

	origen, err := archivo.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no se pudo abrir el archivo: " + err.Error()})
		return
	}
	defer origen.Close()

	temporal, err := os.CreateTemp("", "catalogo-*.xlsx")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo crear el archivo temporal: " + err.Error()})
		return
	}
	rutaTemporal := temporal.Name()
	defer os.Remove(rutaTemporal)
	defer os.Remove(rutaTemporal + ".ignorados.log")

	if _, err := io.Copy(temporal, origen); err != nil {
		temporal.Close()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo guardar el archivo temporal: " + err.Error()})
		return
	}
	if err := temporal.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo finalizar el archivo temporal: " + err.Error()})
		return
	}

	resumen, err := h.service.ImportCatalogFromExcel(c.Request.Context(), rutaTemporal)
	if resumen != nil {
		// El archivo temporal (y su reporte) se elimina al terminar la petición.
		// En HTTP el reporte útil es la lista "ignoradas" del propio resumen.
		resumen.ReporteIgnorados = ""
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "error al importar el catálogo: " + err.Error(),
			"resumen": resumen,
		})
		return
	}

	c.JSON(http.StatusOK, resumen)
}
