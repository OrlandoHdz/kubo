package handlers

import (
	"log"
	"net/http"

	"github.com/OrlandoHdz/kubo/internal/auth"
	"github.com/OrlandoHdz/kubo/internal/db"
	"github.com/OrlandoHdz/kubo/pkg/utils"
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

	userMap := map[string]any{
		"id":    user.ID,
		"email": user.Email,
		"rol":   user.Rol,
	}
	if user.ClienteID.Valid {
		userMap["cliente_id"] = user.ClienteID.Int32
	}

	response := map[string]any{
		"token": token,
		"user":  userMap,
	}

	if user.Rol == "cliente" && user.ClienteID.Valid {
		cliente, err := h.queries.GetCliente(c.Request.Context(), user.ClienteID.Int32)
		if err != nil {
			log.Printf("Error al obtener cliente para usuario ID %d: %v", user.ID, err)
			response["credito_disponible"] = 0.0
		} else {
			total, err1 := utils.NumericToFloat64(cliente.LineaCreditoTotal)
			utilizada, err2 := utils.NumericToFloat64(cliente.LineaCreditoUtilizada)
			if err1 != nil || err2 != nil {
				log.Printf("Error al convertir limites de credito a float64 para cliente ID %d: err1=%v, err2=%v", cliente.ID, err1, err2)
				response["credito_disponible"] = 0.0
			} else {
				response["credito_disponible"] = total - utilizada
			}
		}
	}

	c.JSON(http.StatusOK, response)
}
