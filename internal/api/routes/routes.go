package routes

import (
	"github.com/OrlandoHdz/kubo/internal/api/handlers"
	"github.com/OrlandoHdz/kubo/internal/auth"
	"github.com/OrlandoHdz/kubo/internal/db"
	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, queries *db.Queries) {
	userHandler := handlers.NewUsuarioHandler(queries)
	authHandler := handlers.NewAuthHandler(queries)
	solicitudHandler := handlers.NewSolicitudRegistroHandler(queries)
	clientesIntegracionHandler := handlers.NewClientesIntegracionHandler(queries)

	// Servir archivos subidos estáticamente (ej. para que la URL /uploads/... devuelva el PDF)
	r.Static("/uploads", "./uploads")

	clientesHandler := handlers.NewClientesHandler(queries)

	v1 := r.Group("/api/v1")
	{
		// 1. Rutas totalmente públicas
		v1.POST("/login", authHandler.Login)
		v1.POST("/solicitud-registro", solicitudHandler.Crear)
		v1.POST("/parse-csf", solicitudHandler.ParseCSF)

		// 2. Rutas protegidas
		v1.Use(auth.AuthMiddleware())

		// Solicitudes (Admin)
		solicitudes := v1.Group("/solicitud-registro")
		{
			solicitudes.GET("", solicitudHandler.Listar)
			solicitudes.PATCH("/:id/estado", solicitudHandler.ActualizarEstado)
		}

		// Usuarios
		usuarios := v1.Group("/usuarios")
		{
			usuarios.POST("/", userHandler.Crear)
			usuarios.GET("/", userHandler.Listar)
			usuarios.PATCH("/:id/status", userHandler.ActualizarEstado)
			usuarios.PATCH("/:id/password", userHandler.CambiarPassword)
		}

		// Clientes Integración
		clientesIntegracion := v1.Group("/clientes-integracion")
		{
			clientesIntegracion.GET("", clientesIntegracionHandler.Listar)
			clientesIntegracion.GET("/:id", clientesIntegracionHandler.Obtener)
			clientesIntegracion.GET("/cve/:cve", clientesIntegracionHandler.ObtenerPorCveCte)
			clientesIntegracion.POST("/", clientesIntegracionHandler.Crear)
		}

		// Clientes (Sistema Local)
		clientes := v1.Group("/clientes")
		{
			clientes.GET("", clientesHandler.Listar)
			clientes.GET("/:id", clientesHandler.Obtener)
			clientes.POST("/", clientesHandler.Crear)
			clientes.PUT("/:id", clientesHandler.Actualizar)
			clientes.DELETE("/:id", clientesHandler.Eliminar)
			clientes.PATCH("/:id/saldo", clientesHandler.ActualizarSaldo)
		}
	}
}
