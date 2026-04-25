package handlers

import (
	"net/http"

	"github.com/OrlandoHdz/kubo/internal/auth"
	"github.com/OrlandoHdz/kubo/internal/db"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	queries *db.Queries
}

func NewAuthHandler(q *db.Queries) *AuthHandler {
	return &AuthHandler{queries: q}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var input struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, map[string]any{"error": "Datos inválidos"})
		return
	}

	// 1. Buscar usuario por email (Necesitas crear este Query en sqlc)
	user, err := h.queries.GetUsuarioByEmail(c.Request.Context(), input.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, map[string]any{"error": "Credenciales incorrectas"})
		return
	}

	// 2. Comparar Password Hash
	if !auth.CheckPasswordHash(input.Password, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, map[string]any{"error": "Credenciales incorrectas"})
		return
	}

	// 3. Generar Token
	token, err := auth.GenerarToken(user.ID, user.Email, user.Rol)
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]any{"error": "Error al generar sesión"})
		return
	}

	c.JSON(http.StatusOK, map[string]any{
		"token": token,
		"user": map[string]any{
			"email": user.Email,
			"rol":   user.Rol,
		},
	})
}
