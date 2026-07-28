package routes

import (
	"github.com/OrlandoHdz/kubo/internal/api/handlers"
	"github.com/OrlandoHdz/kubo/internal/auth"
	"github.com/OrlandoHdz/kubo/internal/db"
	"github.com/OrlandoHdz/kubo/pkg/email"
	"github.com/OrlandoHdz/kubo/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func SetupRoutes(r *gin.Engine, queries *db.Queries, pool *pgxpool.Pool, ossCfg *utils.ConfigOSS, emailCfg *email.Config) {
	userHandler := handlers.NewUsuarioHandler(queries)
	authHandler := handlers.NewAuthHandler(queries)
	solicitudHandler := handlers.NewSolicitudRegistroHandler(queries)
	clientesIntegracionHandler := handlers.NewClientesIntegracionHandler(queries)
	facturasIntegracionHandler := handlers.NewFacturasIntegracionHandler(queries)
	existenciasIntegracionHandler := handlers.NewExistenciasIntegracionHandler(queries)
	creditosIntegracionHandler := handlers.NewCreditosIntegracionHandler(queries)
	productosIntegracionHandler := handlers.NewProductosIntegracionHandler(queries)

	// Servir archivos subidos estáticamente (ej. para que la URL /uploads/... devuelva el PDF)
	r.Static("/uploads", "./uploads")

	clientesHandler := handlers.NewClientesHandler(queries)
	productosHandler := handlers.NewProductosHandler(queries, ossCfg)
	pedidosHandler := handlers.NewPedidosHandler(queries, pool, emailCfg)
	backordersHandler := handlers.NewBackordersHandler(queries, pool, emailCfg)
	pagosTarjetaHandler := handlers.NewPagosTarjetaHandler(queries)
	spyWebhookHandler := handlers.NewSpyWebhookHandler(queries)
	devolucionesHandler := handlers.NewDevolucionesHandler(queries, ossCfg, emailCfg)
	pagoFacturasHandler := handlers.NewPagoFacturasHandler(queries, emailCfg)
	estadoCuentaHandler := handlers.NewEstadoCuentaHandler(queries)
	dashboardClienteHandler := handlers.NewDashboardClienteHandler(queries)
	adminDashboardHandler := handlers.NewAdminDashboardHandler(queries)

	v1 := r.Group("/api/v1")
	{
		// 1. Rutas totalmente públicas
		v1.POST("/login", authHandler.Login)
		v1.POST("/solicitud-registro", solicitudHandler.Crear)
		v1.POST("/parse-csf", solicitudHandler.ParseCSF)
		v1.POST("/spy-webhook", spyWebhookHandler.SpyWebhook)
		v1.GET("/spy-webhook", spyWebhookHandler.SpyWebhook)

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

		// Créditos Integración
		creditosIntegracion := v1.Group("/creditos-integracion")
		{
			creditosIntegracion.GET("", creditosIntegracionHandler.Listar)
			creditosIntegracion.GET("/:id", creditosIntegracionHandler.Obtener)
			creditosIntegracion.POST("/", creditosIntegracionHandler.Crear)
		}

		// Productos Integración
		productosIntegracion := v1.Group("/productos-integracion")
		{
			productosIntegracion.GET("", productosIntegracionHandler.Listar)
			productosIntegracion.GET("/:id", productosIntegracionHandler.Obtener)
			productosIntegracion.POST("/", productosIntegracionHandler.Crear)
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
			variantes.GET("/:id", productosHandler.ObtenerVariante)
			variantes.PATCH("/:id", productosHandler.ActualizarVariante)
			variantes.PATCH("/:id/stock", productosHandler.ActualizarStock)
			variantes.DELETE("/:id", productosHandler.EliminarVariante)
		}

		// Pagos Tarjeta (Transacciones BanRegio)
		pagosTarjeta := v1.Group("/pagos-tarjeta")
		{
			pagosTarjeta.GET("", pagosTarjetaHandler.Listar)
			pagosTarjeta.GET("/:id", pagosTarjetaHandler.Obtener)
			pagosTarjeta.GET("/cliente/:cliente_id", pagosTarjetaHandler.ListarPorCliente)
			pagosTarjeta.POST("", pagosTarjetaHandler.Crear)
			pagosTarjeta.DELETE("/:id", pagosTarjetaHandler.Eliminar)
		}

		v1.GET("/transacciones-banregio/verificar", pagosTarjetaHandler.Verificar)

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
			pedidos.POST("/:id/ship", pedidosHandler.ShipOrder)
		}

		// Backorders
		backorders := v1.Group("/backorders")
		{
			backorders.GET("", backordersHandler.Listar)
			backorders.GET("/:id", backordersHandler.Obtener)
			backorders.GET("/cliente/:cliente_id", backordersHandler.ListarPorCliente)
			backorders.POST("", backordersHandler.Crear)
			backorders.PATCH("/:id/estado", backordersHandler.ActualizarEstado)
			backorders.PATCH("/:id/detalles/:detalle_id/disponible", backordersHandler.MarcarDetalleDisponible)
			backorders.POST("/:id/convertir", backordersHandler.ConvertirAPedido)
		}

		// Devoluciones y Garantías
		devoluciones := v1.Group("/devoluciones")
		{
			devoluciones.GET("", devolucionesHandler.Listar)
			devoluciones.GET("/cliente/:cliente_id", devolucionesHandler.ListarPorCliente)
			devoluciones.POST("", devolucionesHandler.Crear)
			devoluciones.PUT("/:id/estatus", devolucionesHandler.ActualizarEstatus)
			devoluciones.PUT("/:id/cancelar", devolucionesHandler.Cancelar)
		}

		// Pago de Facturas
		pagoFacturas := v1.Group("/pago-facturas")
		{
			pagoFacturas.GET("/emitidas", pagoFacturasHandler.ListarEmitidas)
			pagoFacturas.POST("", pagoFacturasHandler.ProcesarPago)
			pagoFacturas.GET("", pagoFacturasHandler.ListarPagadas)
			pagoFacturas.GET("/:id", pagoFacturasHandler.ObtenerPagada)
		}

			// Estado de Cuenta
		v1.GET("/estado-cuenta", estadoCuentaHandler.ObtenerEstadoCuenta)

		// Dashboard Principal del Cliente
		v1.GET("/dashboard-cliente", dashboardClienteHandler.ObtenerDashboard)

		// Dashboard del Administrador
		v1.GET("/admin/dashboard", adminDashboardHandler.ObtenerDashboard)

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
