package handlers

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/OrlandoHdz/kubo/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type SolicitudRegistroHandler struct {
	queries *db.Queries
}

func NewSolicitudRegistroHandler(q *db.Queries) *SolicitudRegistroHandler {
	return &SolicitudRegistroHandler{queries: q}
}

func (h *SolicitudRegistroHandler) Crear(c *gin.Context) {
	// 1. Obtener los campos del formulario (multipart/form-data)
	nombreComercial := c.PostForm("nombreComercial")
	razonSocial := c.PostForm("razonSocial")
	rfc := c.PostForm("rfc")
	tipoContribuyente := c.PostForm("tipoContribuyente")
	calle := c.PostForm("calle")
	numero := c.PostForm("numero")
	colonia := c.PostForm("colonia")
	ciudad := c.PostForm("ciudad")
	estado := c.PostForm("estado")
	cp := c.PostForm("cp")
	nombreContacto := c.PostForm("nombre_contacto")
	puestoContacto := c.PostForm("puesto_contacto")
	correoContacto := c.PostForm("correo_contacto")
	telefonoContacto := c.PostForm("telefono_contacto")
	comentarios := c.PostForm("comentarios")

	// Validaciones básicas requeridas
	if nombreComercial == "" || razonSocial == "" || rfc == "" || tipoContribuyente == "" {
		c.JSON(http.StatusBadRequest, map[string]string{"error": "Faltan campos obligatorios (nombre_comercial, razon_social, rfc, tipo_contribuyente)"})
		return
	}

	if tipoContribuyente != "Persona Moral" && tipoContribuyente != "Persona Fisica" {
		c.JSON(http.StatusBadRequest, map[string]string{"error": "tipo_contribuyente inválido. Debe ser 'Persona Moral' o 'Persona Fisica'"})
		return
	}

	// 2. Manejar la subida del archivo (Constancia SAT)
	var constanciaSatUrl string
	file, err := c.FormFile("constancia_sat")
	if err == nil {
		// Crear carpeta si no existe
		uploadDir := "uploads/constancias"
		if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
			c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error al crear directorio de subidas"})
			return
		}

		// Generar un nombre único para el archivo basado en el RFC y la fecha
		filename := fmt.Sprintf("%s_%d%s", rfc, time.Now().Unix(), filepath.Ext(file.Filename))
		filePath := filepath.Join(uploadDir, filename)

		// Guardar el archivo en el servidor
		if err := c.SaveUploadedFile(file, filePath); err != nil {
			c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error al guardar el archivo"})
			return
		}

		// Guardamos la ruta relativa para la base de datos
		constanciaSatUrl = "/" + filePath
	} else if err != http.ErrMissingFile {
		c.JSON(http.StatusBadRequest, map[string]string{"error": "Error al procesar el archivo subido"})
		return
	}

	// Preparar campos opcionales
	pgComentarios := pgtype.Text{Valid: false}
	if comentarios != "" {
		pgComentarios = pgtype.Text{String: comentarios, Valid: true}
	}

	pgConstancia := pgtype.Text{Valid: false}
	if constanciaSatUrl != "" {
		pgConstancia = pgtype.Text{String: constanciaSatUrl, Valid: true}
	}

	// 3. Guardar en la Base de Datos
	id, err := h.queries.CrearSolicitudRegistroNuevoCliente(c.Request.Context(), db.CrearSolicitudRegistroNuevoClienteParams{
		NombreComercial:   nombreComercial,
		RazonSocial:       razonSocial,
		Rfc:               rfc,
		TipoContribuyente: tipoContribuyente,
		Calle:             calle,
		Numero:            numero,
		Colonia:           colonia,
		Ciudad:            ciudad,
		Estado:            estado,
		Cp:                cp,
		NombreContacto:    nombreContacto,
		PuestoContacto:    puestoContacto,
		CorreoContacto:    correoContacto,
		TelefonoContacto:  telefonoContacto,
		Comentarios:       pgComentarios,
		ConstanciaSatUrl:  pgConstancia,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error al guardar en la base de datos: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, map[string]any{
		"message": "Solicitud creada exitosamente",
		"id":      id,
	})
}

// ParseCSF recibe un PDF, extrae su texto y parsea los campos solicitados
func (h *SolicitudRegistroHandler) ParseCSF(c *gin.Context) {
	file, err := c.FormFile("archivo")
	if err != nil {
		c.JSON(http.StatusBadRequest, map[string]string{"error": "El archivo PDF 'archivo' es requerido"})
		return
	}

	// Abrir el archivo subido
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error al abrir el archivo subido"})
		return
	}
	defer src.Close()

	// Crear archivo temporal para pdftotext
	tempFile, err := os.CreateTemp("", "csf_*.pdf")
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error al crear archivo temporal"})
		return
	}
	defer os.Remove(tempFile.Name()) // Limpiar archivo temporal al final
	defer tempFile.Close()

	if _, err := io.Copy(tempFile, src); err != nil {
		c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error al escribir archivo temporal"})
		return
	}

	// Extraer texto usando pdftotext (poppler-utils)
	cmd := exec.Command("pdftotext", "-layout", tempFile.Name(), "-")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error al extraer texto usando pdftotext: " + err.Error()})
		return
	}

	text := out.String()

	// Parsear con Regex
	extract := func(pattern string) string {
		re := regexp.MustCompile(pattern)
		match := re.FindStringSubmatch(text)
		if len(match) > 1 {
			return strings.TrimSpace(match[1])
		}
		return ""
	}

	rfc := extract(`(?i)RFC:\s*([A-Z0-9]+)`)
	razonSocial := extract(`(?i)Denominación/Razón\s*Social:\s*(.*?)(?:\s{2,}|$)`)
	nombreComercial := extract(`(?i)Nombre\s*Comercial:\s*(.*?)(?:\s{2,}|$)`)
	codigoPostal := extract(`(?i)Código\s*Postal:\s*(\d{5})`)
	nombreVialidad := extract(`(?i)Nombre\s*de\s*Vialidad:\s*(.*?)(?:\s{2,}|$)`)
	numeroExterior := extract(`(?i)Número\s*Exterior:\s*(.*?)(?:\s{2,}|$)`)
	nombreColonia := extract(`(?i)Nombre\s*de\s*la\s*Colonia:\s*(.*?)(?:\s{2,}|$)`)
	nombreLocalidad := extract(`(?i)Nombre\s*de\s*la\s*Localidad:\s*(.*?)(?:\s{2,}|$)`)
	nombreMunicipio := extract(`(?i)Nombre\s*del\s*Municipio\s*o\s*Demarcación\s*Territorial:\s*(.*?)(?:\s{2,}|$)`)
	nombreEntidad := extract(`(?i)Nombre\s*de\s*la\s*Entidad\s*Federativa:\s*(.*?)(?:\s{2,}|$)`)

	c.JSON(http.StatusOK, gin.H{
		"rfc":                rfc,
		"razon_social":       razonSocial,
		"nombre_comercial":   nombreComercial,
		"codigo_postal":      codigoPostal,
		"vialidad":           nombreVialidad,
		"numero_exterior":    numeroExterior,
		"colonia":            nombreColonia,
		"localidad":          nombreLocalidad,
		"municipio":          nombreMunicipio,
		"entidad_federativa": nombreEntidad,
	})
}

// Listar devuelve todas las solicitudes de registro no borradas
func (h *SolicitudRegistroHandler) Listar(c *gin.Context) {
	solicitudes, err := h.queries.ListarTodasLasSolicitudes(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al listar solicitudes: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, solicitudes)
}

// ActualizarEstado actualiza el estado de una solicitud
func (h *SolicitudRegistroHandler) ActualizarEstado(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	var req struct {
		Estado      string `json:"solicitud_estado"`
		Observacion string `json:"observacion"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
		return
	}

	if req.Estado == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "El estado es requerido"})
		return
	}

	pgObservacion := pgtype.Text{Valid: false}
	if req.Observacion != "" {
		pgObservacion = pgtype.Text{String: req.Observacion, Valid: true}
	}

	// Por ahora el updated_by lo dejamos como null o un valor fijo si no tenemos el usuario del context
	// Si el middleware de auth está activado, podríamos sacar el ID del usuario de ahí.
	// Por ahora lo dejamos como nulo (Valid: false)
	updatedBy := pgtype.Int4{Valid: false}

	err = h.queries.ActualizarSolicitudEstado(c.Request.Context(), db.ActualizarSolicitudEstadoParams{
		ID:              int32(id),
		SolicitudEstado: req.Estado,
		Observacion:     pgObservacion,
		UpdatedBy:       updatedBy,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar estado: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Estado actualizado correctamente"})
}
