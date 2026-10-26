package handlers

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/OrlandoHdz/kubo/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type ChatAdminHandler struct {
	queries *db.Queries
}

func NewChatAdminHandler(q *db.Queries) *ChatAdminHandler {
	return &ChatAdminHandler{queries: q}
}

func requireAdmin(c *gin.Context) bool {
	rolValue, ok := c.Get("userRol")
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "Se requiere rol de administrador."})
		return false
	}
	rol, _ := rolValue.(string)
	if !strings.EqualFold(rol, "admin") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Se requiere rol de administrador."})
		return false
	}
	return true
}

// Listar devuelve los escalamientos del chat para la bandeja del panel.
func (h *ChatAdminHandler) Listar(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}

	estado := strings.ToLower(strings.TrimSpace(c.Query("estado")))
	switch estado {
	case "", "pendiente", "atendida", "cerrada":
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Estado inválido."})
		return
	}

	limite := 50
	if raw := strings.TrimSpace(c.Query("limite")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Límite inválido."})
			return
		}
		limite = parsed
	}
	if limite < 1 {
		limite = 1
	}
	if limite > 100 {
		limite = 100
	}

	rows, err := h.queries.ListarEscalamientosChat(c.Request.Context(), db.ListarEscalamientosChatParams{
		Estado: pgtype.Text{String: estado, Valid: estado != ""},
		Limite: int32(limite),
	})
	if err != nil {
		log.Printf("chat admin: error listando escalamientos: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudieron cargar los escalamientos."})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"escalamientos": rows,
		"total":         len(rows),
	})
}

// Obtener devuelve un escalamiento y la conversación asociada.
func (h *ChatAdminHandler) Obtener(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido."})
		return
	}

	escalamiento, err := h.queries.ObtenerEscalamientoChat(c.Request.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Escalamiento no encontrado."})
		return
	}
	if err != nil {
		log.Printf("chat admin: error obteniendo escalamiento %d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo cargar el escalamiento."})
		return
	}

	mensajes, err := h.queries.ListarMensajesConversacionChat(c.Request.Context(), escalamiento.ConversationID)
	if err != nil {
		log.Printf("chat admin: error cargando conversación %s: %v", escalamiento.ConversationID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo cargar la conversación."})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"escalamiento": escalamiento,
		"mensajes":     mensajes,
	})
}

// ActualizarEstado marca un escalamiento como atendido/cerrado y guarda una nota.
func (h *ChatAdminHandler) ActualizarEstado(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido."})
		return
	}

	var input struct {
		Estado    string `json:"estado" binding:"required"`
		NotaAdmin string `json:"nota_admin"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos."})
		return
	}

	estado := strings.ToLower(strings.TrimSpace(input.Estado))
	switch estado {
	case "pendiente", "atendida", "cerrada":
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Estado inválido."})
		return
	}

	nota := strings.TrimSpace(input.NotaAdmin)
	if utf8.RuneCountInString(nota) > 2000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "La nota no puede superar 2000 caracteres."})
		return
	}

	atendidaPor := pgtype.Int4{Valid: false}
	if estado != "pendiente" {
		if raw, ok := c.Get("userID"); ok {
			if userID, ok := raw.(int32); ok {
				atendidaPor = pgtype.Int4{Int32: userID, Valid: true}
			}
		}
	}

	row, err := h.queries.ActualizarEstadoEscalamientoChat(c.Request.Context(), db.ActualizarEstadoEscalamientoChatParams{
		Estado:      estado,
		NotaAdmin:   nota,
		AtendidaPor: atendidaPor,
		ID:          id,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Escalamiento no encontrado."})
		return
	}
	if err != nil {
		log.Printf("chat admin: error actualizando escalamiento %d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo actualizar el escalamiento."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"escalamiento": row})
}
