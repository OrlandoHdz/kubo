package handlers

import (
	"net/http"
	"strconv"

	"github.com/OrlandoHdz/kubo/internal/auth"
	"github.com/OrlandoHdz/kubo/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type ClientesHandler struct {
	queries *db.Queries
}

func NewClientesHandler(q *db.Queries) *ClientesHandler {
	return &ClientesHandler{queries: q}
}

// Listar devuelve todos los clientes activos
func (h *ClientesHandler) Listar(c *gin.Context) {
	clientes, err := h.queries.ListarClientesActivos(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al listar clientes: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, clientes)
}

// Obtener devuelve un cliente por su ID
func (h *ClientesHandler) Obtener(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de cliente inválido"})
		return
	}

	cliente, err := h.queries.GetCliente(c.Request.Context(), int32(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cliente no encontrado"})
		return
	}
	c.JSON(http.StatusOK, cliente)
}

// Crear registra un nuevo cliente y un usuario asociado por defecto
func (h *ClientesHandler) Crear(c *gin.Context) {
	var input struct {
		db.CrearClienteParams
		Email string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos de cliente o usuario inválidos: " + err.Error()})
		return
	}

	// 1. Crear el Cliente
	cliente, err := h.queries.CrearCliente(c.Request.Context(), input.CrearClienteParams)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear cliente: " + err.Error()})
		return
	}

	// 2. Crear el Usuario por defecto para ese cliente
	// Generamos una contraseña temporal (por ejemplo: Kubo + RFC o una fija)
	// Aquí usaremos una fija por ahora "Kubo12345!"
	tempPassword := "Kubo123!"
	hashedPassword, err := auth.HashPassword(tempPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al procesar credenciales de usuario: " + err.Error()})
		return
	}

	_, err = h.queries.CrearUsuario(c.Request.Context(), db.CrearUsuarioParams{
		ClienteID:    pgtype.Int4{Int32: cliente.ID, Valid: true},
		Email:        input.Email,
		PasswordHash: hashedPassword,
		Rol:          "cliente",
		IsActive:     pgtype.Bool{Bool: true, Valid: true},
		CreatedBy:    input.CreatedBy,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Cliente creado pero hubo un error al crear su usuario: " + err.Error(),
			"cliente": cliente,
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Cliente y usuario creados exitosamente",
		"cliente": cliente,
		"usuario": gin.H{
			"email":    input.Email,
			"password": tempPassword, // Se devuelve la temporal solo esta vez
			"rol":      "cliente",
		},
	})
}

// Actualizar modifica los datos de un cliente existente
func (h *ClientesHandler) Actualizar(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de cliente inválido"})
		return
	}

	var params db.ActualizarClienteParams
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos de actualización inválidos: " + err.Error()})
		return
	}

	// Asegurar que el ID del body coincida con el de la URL
	params.ID = int32(id)

	cliente, err := h.queries.ActualizarCliente(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar cliente: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, cliente)
}

// Eliminar realiza un borrado lógico del cliente
func (h *ClientesHandler) Eliminar(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de cliente inválido"})
		return
	}

	// Por ahora el deleted_by es opcional o lo tomamos del contexto si existiera
	// Aquí podrías obtener el ID del usuario autenticado si es necesario
	params := db.SoftDeleteClienteParams{
		ID:        int32(id),
		DeletedBy: pgtype.Int4{Valid: false}, // Opcional
	}

	err = h.queries.SoftDeleteCliente(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al eliminar cliente: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cliente eliminado correctamente"})
}

// ActualizarSaldo modifica solo el saldo utilizado del cliente
func (h *ClientesHandler) ActualizarSaldo(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de cliente inválido"})
		return
	}

	var params db.ActualizarSaldoCreditoParams
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos de saldo inválidos: " + err.Error()})
		return
	}

	params.ID = int32(id)

	err = h.queries.ActualizarSaldoCredito(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar saldo del cliente: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Saldo actualizado correctamente"})
}
