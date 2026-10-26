package chat

import (
	"context"
	"encoding/json"
)

// BuscarProductosInput es el contrato interno del catálogo.
type BuscarProductosInput struct {
	Consulta       string
	Proceso        string
	Filtros        json.RawMessage
	AmperajeMin    *float64
	AmperajeMax    *float64
	EnStock        *bool
	EsDistribuidor bool
}

// CatalogoStore abstrae las consultas sqlc de productos.
type CatalogoStore interface {
	BuscarProductos(ctx context.Context, in BuscarProductosInput) ([]ProductoResumen, error)
	ObtenerProductoPorSKU(ctx context.Context, sku string, esDistribuidor bool) (*ProductoDetalle, error)
}

// ContextoStore resuelve usuario -> cliente. Es la única fuente permitida para
// el cliente_id de las consultas de pedidos.
type ContextoStore interface {
	ObtenerContexto(ctx context.Context, usuarioID int32) (ContextoCliente, error)
}

// PedidosStore ejecuta las consultas de pedidos dentro de una transacción
// read-only con set_config('app.customer_id', ..., true).
type PedidosStore interface {
	ConCliente(ctx context.Context, clienteID int32, fn func(PedidosQuerier) error) error
}

// PedidosQuerier es la interfaz pequeña que consumen las herramientas.
type PedidosQuerier interface {
	ObtenerPedido(ctx context.Context, folio string, clienteID int32) (*PedidoResumen, error)
	ListarItems(ctx context.Context, pedidoID int32, clienteID int32) ([]PedidoItem, error)
	ListarPedidos(ctx context.Context, clienteID int32, limite int) ([]PedidoResumen, error)
}

// BackordersStore ejecuta consultas de backorders del cliente autenticado.
type BackordersStore interface {
	ConClienteBackorders(ctx context.Context, clienteID int32, fn func(BackordersQuerier) error) error
}

type BackordersQuerier interface {
	ObtenerBackorder(ctx context.Context, folio string, clienteID int32) (*BackorderResumen, error)
	ListarBackorderItems(ctx context.Context, backorderID int32, clienteID int32) ([]BackorderItem, error)
	ListarBackorders(ctx context.Context, clienteID int32, limite int) ([]BackorderResumen, error)
}

// DevolucionesStore ejecuta consultas de devoluciones y garantías.
type DevolucionesStore interface {
	ConClienteDevoluciones(ctx context.Context, clienteID int32, fn func(DevolucionesQuerier) error) error
}

type DevolucionesQuerier interface {
	ObtenerDevolucion(ctx context.Context, folio string, clienteID int32) (*DevolucionResumen, error)
	ListarDevoluciones(ctx context.Context, clienteID int32, limite int) ([]DevolucionResumen, error)
}

// RecompensasStore expone el catálogo público de niveles y los datos del
// cliente autenticado.
type RecompensasStore interface {
	ConClienteRecompensas(ctx context.Context, clienteID int32, fn func(RecompensasQuerier) error) error
	ListarNivelesRecompensa(ctx context.Context) (NivelesRecompensaResult, error)
}

type RecompensasQuerier interface {
	ObtenerRecompensasCliente(ctx context.Context, clienteID int32) (*RecompensasCliente, error)
	ListarNivelesRecompensa(ctx context.Context) ([]NivelRecompensa, error)
	ListarTransaccionesPuntos(ctx context.Context, clienteID int32, limite int) ([]TransaccionPuntos, error)
}

// AuditoriaStore es la interfaz de escritura del pool chat_audit.
type AuditoriaStore interface {
	CrearConversacion(ctx context.Context, id string, usuarioID, clienteID *int32) error
	CrearMensaje(ctx context.Context, conversationID string, usuarioID *int32, role, content string, toolCalls json.RawMessage, inputTokens, outputTokens int32) error
	CrearEscalamiento(ctx context.Context, conversationID string, usuarioID *int32, motivo, nombre, email, telefono string) error
}
