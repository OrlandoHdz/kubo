package handlers

import (
	"net/http"

	"strconv"

	"github.com/OrlandoHdz/kubo/internal/auth"
	"github.com/OrlandoHdz/kubo/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type UsuarioHandler struct {
	queries *db.Queries
}

func NewUsuarioHandler(q *db.Queries) *UsuarioHandler {
	return &UsuarioHandler{queries: q}
}

func (h *UsuarioHandler) Crear(c *gin.Context) {
	var input struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=8"`
		Rol      string `json:"rol" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	// 1. Cifrar contraseña con bcrypt
	hash, err := auth.HashPassword(input.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	// 2. Guardar en DB
	user, err := h.queries.CrearUsuario(c.Request.Context(), db.CrearUsuarioParams{
		Email:        input.Email,
		PasswordHash: hash,
		Rol:          input.Rol,
		IsActive:     pgtype.Bool{Bool: true, Valid: true},
	})

	if err != nil {
		c.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, user)
}

func (h *UsuarioHandler) Listar(c *gin.Context) {
	usuarios, err := h.queries.ListarUsuariosConCliente(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, usuarios)
}

func (h *UsuarioHandler) ActualizarEstado(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)

	var input struct {
		IsActive bool `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Obtener usuario actual para no perder otros campos
	current, err := h.queries.GetUsuarioByID(c.Request.Context(), int32(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Usuario no encontrado"})
		return
	}

	user, err := h.queries.ActualizarUsuario(c.Request.Context(), db.ActualizarUsuarioParams{
		ID:       int32(id),
		Email:    current.Email,
		Rol:      current.Rol,
		IsActive: pgtype.Bool{Bool: input.IsActive, Valid: true},
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *UsuarioHandler) CambiarPassword(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)

	var input struct {
		Password string `json:"password" binding:"required,min=8"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hash, err := auth.HashPassword(input.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al cifrar contraseña"})
		return
	}

	err = h.queries.ActualizarPassword(c.Request.Context(), db.ActualizarPasswordParams{
		ID:           int32(id),
		PasswordHash: hash,
		UpdatedBy:    pgtype.Int4{Valid: false}, // Por ahora no registramos quién hizo el cambio
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Contraseña actualizada correctamente"})
}
