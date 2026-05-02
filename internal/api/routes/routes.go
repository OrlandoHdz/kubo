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

	v1 := r.Group("/api/v1")
	{
		// 1. Rutas totalmente públicas
		v1.POST("/login", authHandler.Login)
		v1.POST("/solicitud-registro", solicitudHandler.Crear)
		v1.POST("/parse-csf", solicitudHandler.ParseCSF)

		// Rutas protegidas de solicitudes (Admin)
		solicitudes := v1.Group("/solicitud-registro")
		solicitudes.Use(auth.AuthMiddleware())
		{
			solicitudes.GET("", solicitudHandler.Listar)
			solicitudes.PATCH("/:id/estado", solicitudHandler.ActualizarEstado)
		}

		// 2. Definimos el grupo de usuarios
		usuarios := v1.Group("/usuarios")

		// 3. APLICAMOS el middleware directamente a este grupo
		// Todo lo que esté debajo de 'usuarios' pasará por aquí
		usuarios.Use(auth.AuthMiddleware())
		{
			usuarios.POST("/", userHandler.Crear)
			usuarios.GET("/", userHandler.Listar)
			// Aquí irán los futuros usuarios.GET("/:id", ...)
		}

		// 4. Rutas de Clientes Integración
		clientesIntegracion := v1.Group("/clientes-integracion")
		clientesIntegracion.Use(auth.AuthMiddleware())
		{
			clientesIntegracion.GET("", clientesIntegracionHandler.Listar)
			clientesIntegracion.GET("/:id", clientesIntegracionHandler.Obtener)
			clientesIntegracion.GET("/cve/:cve", clientesIntegracionHandler.ObtenerPorCveCte)
			clientesIntegracion.POST("/", clientesIntegracionHandler.Crear)
		}
	}
}
