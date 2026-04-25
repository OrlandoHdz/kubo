package handlers

import (
	"net/http"

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
	usuarios, err := h.queries.ListarUsuarios(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, usuarios)
}
