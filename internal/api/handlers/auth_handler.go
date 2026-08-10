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

	// Permisos del menú del Panel de Control (solo aplican a usuarios Admin/Staff)
	// - Si el usuario NO tiene permisos configurados aún: acceso total (permisos = nil)
	// - Si ya se configuraron: solo las claves activas
	if user.Rol != "cliente" {
		configurados, errCount := h.queries.ContarPermisosConfiguradosDeUsuario(c.Request.Context(), user.ID)
		if errCount != nil {
			log.Printf("Error al contar permisos para usuario ID %d: %v", user.ID, errCount)
			response["permisos"] = nil
		} else if configurados == 0 {
			response["permisos"] = nil
		} else {
			permisos, errPermisos := h.queries.ListarPermisosActivosDeUsuario(c.Request.Context(), user.ID)
			if errPermisos != nil {
				log.Printf("Error al obtener permisos para usuario ID %d: %v", user.ID, errPermisos)
				response["permisos"] = []string{}
			} else {
				response["permisos"] = permisos
			}
		}
	} else {
		response["permisos"] = []string{}
	}

	if user.Rol == "cliente" && user.ClienteID.Valid {
		cliente, err := h.queries.GetCliente(c.Request.Context(), user.ClienteID.Int32)
		if err != nil {
			log.Printf("Error al obtener cliente para usuario ID %d: %v", user.ID, err)
			response["credito_disponible"] = 0.0
		} else {
			userMap["tiene_precio_distribuidor"] = cliente.TienePrecioDistribuidor
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

// Perfil devuelve los datos del usuario autenticado junto con sus permisos del menú.
// GET /api/v1/perfil
func (h *AuthHandler) Perfil(c *gin.Context) {
	userID, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, map[string]any{"error": "No autenticado"})
		return
	}

	user, err := h.queries.GetUsuarioByID(c.Request.Context(), userID.(int32))
	if err != nil {
		c.JSON(http.StatusNotFound, map[string]any{"error": "Usuario no encontrado"})
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
		"user": userMap,
	}

	if user.Rol != "cliente" {
		configurados, errCount := h.queries.ContarPermisosConfiguradosDeUsuario(c.Request.Context(), user.ID)
		if errCount != nil {
			log.Printf("Error al contar permisos para usuario ID %d: %v", user.ID, errCount)
			response["permisos"] = nil
		} else if configurados == 0 {
			response["permisos"] = nil
		} else {
			permisos, errPermisos := h.queries.ListarPermisosActivosDeUsuario(c.Request.Context(), user.ID)
			if errPermisos != nil {
				log.Printf("Error al obtener permisos para usuario ID %d: %v", user.ID, errPermisos)
				response["permisos"] = []string{}
			} else {
				response["permisos"] = permisos
			}
		}
	} else {
		response["permisos"] = []string{}
	}

	if user.Rol == "cliente" && user.ClienteID.Valid {
		cliente, err := h.queries.GetCliente(c.Request.Context(), user.ClienteID.Int32)
		if err != nil {
			log.Printf("Error al obtener cliente para usuario ID %d: %v", user.ID, err)
		} else {
			userMap["tiene_precio_distribuidor"] = cliente.TienePrecioDistribuidor
		}
	}

	c.JSON(http.StatusOK, response)
}
