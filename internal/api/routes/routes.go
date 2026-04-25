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

	v1 := r.Group("/api/v1")
	{
		// 1. Rutas totalmente públicas
		v1.POST("/login", authHandler.Login)

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
	}
}
