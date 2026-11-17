package routes

import (
	"github.com/OrlandoHdz/kubo/internal/api/handlers"
	"github.com/OrlandoHdz/kubo/internal/auth"
	"github.com/OrlandoHdz/kubo/internal/chat"
	"github.com/OrlandoHdz/kubo/internal/db"
	"github.com/OrlandoHdz/kubo/internal/services"
	"github.com/OrlandoHdz/kubo/pkg/email"
	"github.com/OrlandoHdz/kubo/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func SetupRoutes(r *gin.Engine, queries *db.Queries, pool *pgxpool.Pool, ossCfg *utils.ConfigOSS, emailCfg *email.Config, chatHandler *chat.Handler) {
	userHandler := handlers.NewUsuarioHandler(queries)
	authHandler := handlers.NewAuthHandler(queries)
	permisoHandler := handlers.NewPermisoHandler(queries)
	solicitudHandler := handlers.NewSolicitudRegistroHandler(queries, emailCfg)
	clientesIntegracionHandler := handlers.NewClientesIntegracionHandler(queries)
	facturasIntegracionHandler := handlers.NewFacturasIntegracionHandler(queries)
	existenciasIntegracionHandler := handlers.NewExistenciasIntegracionHandler(queries)
	creditosIntegracionHandler := handlers.NewCreditosIntegracionHandler(queries)
	productosIntegracionHandler := handlers.NewProductosIntegracionHandler(queries)
	catalogoImportService := services.NewCatalogoExcelService(queries, pool)
	catalogoImportHandler := handlers.NewCatalogoImportHandler(catalogoImportService)

	// Servir archivos subidos estáticamente (ej. para que la URL /uploads/... devuelva el PDF)
	r.Static("/uploads", "./uploads")

	// Chat inteligente. La autenticación es opcional a nivel de ruta: el handler
	// permite invitados para catálogo y exige sesión para las herramientas de pedidos.
	// Se registra fuera del grupo protegido para no bloquear a los invitados.
	if chatHandler != nil {
		r.POST("/api/chat", chatHandler.Chat)
		r.POST("/api/v1/chat", chatHandler.Chat)
	}

	clientesHandler := handlers.NewClientesHandler(queries)
	productosHandler := handlers.NewProductosHandler(queries, ossCfg)
	productosRecompensaHandler := handlers.NewProductosRecompensaHandler(queries, ossCfg)
	pedidosHandler := handlers.NewPedidosHandler(queries, pool, emailCfg)
	backordersHandler := handlers.NewBackordersHandler(queries, pool, emailCfg)
	pagosTarjetaHandler := handlers.NewPagosTarjetaHandler(queries)
	spyWebhookHandler := handlers.NewSpyWebhookHandler(queries)
	devolucionesHandler := handlers.NewDevolucionesHandler(queries, ossCfg, emailCfg)
	pagoFacturasHandler := handlers.NewPagoFacturasHandler(queries, emailCfg)
	estadoCuentaHandler := handlers.NewEstadoCuentaHandler(queries)
	dashboardClienteHandler := handlers.NewDashboardClienteHandler(queries)
	adminDashboardHandler := handlers.NewAdminDashboardHandler(queries)
	chatAdminHandler := handlers.NewChatAdminHandler(queries)
	recompensasHandler := handlers.NewRecompensasHandlerWithPoolAndEmail(queries, pool, emailCfg)
	ajustesInventarioHandler := handlers.NewAjustesInventarioHandler(queries, pool)

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

		// Perfil del usuario autenticado (incluye permisos del menú)
		v1.GET("/perfil", authHandler.Perfil)

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

		// Permisos del menú (Panel de Control)
		permisos := v1.Group("/permisos")
		{
			permisos.GET("", permisoHandler.ListarPermisos)
			permisos.GET("/usuario/:id", permisoHandler.ListarPermisosUsuario)
			permisos.PUT("/usuario/:id", permisoHandler.ActualizarPermisosUsuario)
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
		// Importación de catálogo desde Excel (Privado - Gestión)
		catalogo := v1.Group("/catalogo")
		{
			catalogo.POST("/importar-excel", catalogoImportHandler.ImportarExcel)
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

		// Productos Recompensa (Privados - Gestión) — espejo de productos
		productosRecompensa := v1.Group("/recompensas/productos")
		{
			productosRecompensa.GET("", productosRecompensaHandler.ListarPadres)
			productosRecompensa.GET("/:id", productosRecompensaHandler.ObtenerPadre)
			productosRecompensa.POST("", productosRecompensaHandler.CrearPadre)
			productosRecompensa.PATCH("/:id", productosRecompensaHandler.ActualizarPadre)
			productosRecompensa.DELETE("/:id", productosRecompensaHandler.EliminarPadre)

			// Variantes ligadas al padre
			productosRecompensa.GET("/:id/variantes", productosRecompensaHandler.ListarVariantes)
			productosRecompensa.POST("/:id/variantes", productosRecompensaHandler.CrearVariante)
		}

		// Variantes (operaciones directas)
		variantes := v1.Group("/variantes")
		{
			variantes.GET("/:id", productosHandler.ObtenerVariante)
			variantes.PATCH("/:id", productosHandler.ActualizarVariante)
			variantes.PATCH("/:id/stock", productosHandler.ActualizarStock)
			variantes.DELETE("/:id", productosHandler.EliminarVariante)
		}

		// Variantes Recompensa (operaciones directas) — espejo de variantes
		variantesRecompensa := v1.Group("/recompensas/variantes")
		{
			variantesRecompensa.GET("/:id", productosRecompensaHandler.ObtenerVariante)
			variantesRecompensa.PATCH("/:id", productosRecompensaHandler.ActualizarVariante)
			variantesRecompensa.PATCH("/:id/stock", productosRecompensaHandler.ActualizarStock)
			variantesRecompensa.DELETE("/:id", productosRecompensaHandler.EliminarVariante)
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

		// Auditoría de Pedidos (modificaciones con backorder)
		v1.GET("/pedidos/modificaciones", pedidosHandler.ListarModificaciones)

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

		// Bandeja de escalamientos del chat (solo rol Admin)
		adminChat := v1.Group("/admin/chat")
		{
			adminChat.GET("/escalations", chatAdminHandler.Listar)
			adminChat.GET("/escalations/:id", chatAdminHandler.Obtener)
			adminChat.PATCH("/escalations/:id/estado", chatAdminHandler.ActualizarEstado)
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

		// ===== Módulo de Parámetros y Niveles =====
		v1.GET("/recompensas/parametros", recompensasHandler.GetParametros)
		v1.PUT("/recompensas/parametros", recompensasHandler.UpdateParametros)

		niveles := v1.Group("/recompensas/niveles")
		{
			niveles.GET("", recompensasHandler.ListarNiveles)
			niveles.POST("", recompensasHandler.CrearNivel)
			niveles.PATCH("/:id", recompensasHandler.ActualizarNivel)
			niveles.DELETE("/:id", recompensasHandler.EliminarNivel)
		}

		v1.GET("/recompensas/transacciones/:cliente_id", recompensasHandler.ListarTransaccionesCliente)
		v1.POST("/recompensas/evaluar-niveles", recompensasHandler.ForzarEvaluacionNiveles)
		v1.POST("/recompensas/canje", recompensasHandler.Canjear)
		v1.GET("/recompensas/cliente/:id", recompensasHandler.GetInfoRecompensasCliente)
		v1.PATCH("/recompensas/cliente/:id/nivel", recompensasHandler.AsignarNivelManual)

		// Ajustes de Inventario Recompensa
		ajustes := v1.Group("/recompensas/ajustes")
		{
			ajustes.POST("", ajustesInventarioHandler.CrearAjuste)
			ajustes.GET("", ajustesInventarioHandler.ListarAjustes)
			ajustes.GET("/variantes", ajustesInventarioHandler.ListarVariantes)
		}
	}
}
