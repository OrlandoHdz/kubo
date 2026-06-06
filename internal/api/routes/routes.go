package routes

import (
	"github.com/OrlandoHdz/kubo/internal/api/handlers"
	"github.com/OrlandoHdz/kubo/internal/auth"
	"github.com/OrlandoHdz/kubo/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func SetupRoutes(r *gin.Engine, queries *db.Queries, pool *pgxpool.Pool) {
	userHandler := handlers.NewUsuarioHandler(queries)
	authHandler := handlers.NewAuthHandler(queries)
	solicitudHandler := handlers.NewSolicitudRegistroHandler(queries)
	clientesIntegracionHandler := handlers.NewClientesIntegracionHandler(queries)
	facturasIntegracionHandler := handlers.NewFacturasIntegracionHandler(queries)
	existenciasIntegracionHandler := handlers.NewExistenciasIntegracionHandler(queries)

	// Servir archivos subidos estáticamente (ej. para que la URL /uploads/... devuelva el PDF)
	r.Static("/uploads", "./uploads")

	clientesHandler := handlers.NewClientesHandler(queries)
	productosHandler := handlers.NewProductosHandler(queries)
	pedidosHandler := handlers.NewPedidosHandler(queries, pool)

	v1 := r.Group("/api/v1")
	{
		// 1. Rutas totalmente públicas
		v1.POST("/login", authHandler.Login)
		v1.POST("/solicitud-registro", solicitudHandler.Crear)
		v1.POST("/parse-csf", solicitudHandler.ParseCSF)

		// Productos (Públicos)
		publicProductos := v1.Group("/productos")
		{
			publicProductos.GET("", productosHandler.ListarPadres)
			publicProductos.GET("/:id", productosHandler.ObtenerPadre)
			publicProductos.GET("/:id/variantes", productosHandler.ListarVariantes)
		}

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
		// Facturas Integración
		facturasIntegracion := v1.Group("/facturas-integracion")
		{
			facturasIntegracion.GET("", facturasIntegracionHandler.Listar)
			facturasIntegracion.GET("/:id", facturasIntegracionHandler.Obtener)
			facturasIntegracion.POST("/", facturasIntegracionHandler.Crear)
		}
		// Facturas SAI
		facturasSai := v1.Group("/facturas-sai")
		{
			facturasSai.GET("", facturasIntegracionHandler.Listar)
			facturasSai.GET("/:id", facturasIntegracionHandler.Obtener)
			facturasSai.POST("/", facturasIntegracionHandler.Crear)
		}

		// Existencias Integración
		existenciasIntegracion := v1.Group("/existencias-integracion")
		{
			existenciasIntegracion.GET("", existenciasIntegracionHandler.Listar)
			existenciasIntegracion.GET("/:id", existenciasIntegracionHandler.Obtener)
			existenciasIntegracion.POST("", existenciasIntegracionHandler.Crear)
			existenciasIntegracion.PUT("", existenciasIntegracionHandler.Upsert)
		}
		// Productos (Privados - Gestión)
		productos := v1.Group("/productos")
		{
			productos.POST("", productosHandler.CrearPadre)
			productos.PATCH("/:id", productosHandler.ActualizarPadre)
			productos.DELETE("/:id", productosHandler.EliminarPadre)

			// Variantes ligadas al padre
			productos.POST("/:id/variantes", productosHandler.CrearVariante)
		}

		// Variantes (operaciones directas)
		variantes := v1.Group("/variantes")
		{
			variantes.PATCH("/:id", productosHandler.ActualizarVariante)
			variantes.PATCH("/:id/stock", productosHandler.ActualizarStock)
			variantes.DELETE("/:id", productosHandler.EliminarVariante)
		}

		// Pedidos
		pedidos := v1.Group("/pedidos")
		{
			pedidos.GET("", pedidosHandler.Listar)
			pedidos.GET("/:id", pedidosHandler.Obtener)
			pedidos.GET("/cliente/:cliente_id", pedidosHandler.ListarPorCliente)
			pedidos.POST("", pedidosHandler.Crear)
			pedidos.PATCH("/:id/estado", pedidosHandler.ActualizarEstado)
			pedidos.PATCH("/:id/detalles/:detalle_id/cancelar", pedidosHandler.CancelarDetalle)
		pedidos.POST("/:id/agregar-productos", pedidosHandler.AgregarProductos)
		}

		// Clientes (Sistema Local)
		clientes := v1.Group("/clientes")
		{
			clientes.GET("", clientesHandler.Listar)
			clientes.GET("/:id", clientesHandler.Obtener)
			clientes.POST("/", clientesHandler.Crear)
			clientes.PATCH("/:id", clientesHandler.Actualizar)
			clientes.DELETE("/:id", clientesHandler.Eliminar)
			clientes.PATCH("/:id/saldo", clientesHandler.ActualizarSaldo)
		}
	}
}
