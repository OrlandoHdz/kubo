package chat

import (
	"encoding/json"
	"time"
)

// Usage acumula tokens de una o varias llamadas a Claude.
type Usage struct {
	InputTokens  int32 `json:"input_tokens"`
	OutputTokens int32 `json:"output_tokens"`
}

// ContentBlock es la representación interna de un bloque de contenido de
// Claude: texto, tool_use, tool_result o thinking.
type ContentBlock struct {
	Type        string          `json:"type"`
	Text        string          `json:"text,omitempty"`
	ID          string          `json:"id,omitempty"`
	Name        string          `json:"name,omitempty"`
	Input       json.RawMessage `json:"input,omitempty"`
	ToolUseID   string          `json:"tool_use_id,omitempty"`
	ToolContent string          `json:"content,omitempty"`
	IsError     bool            `json:"is_error,omitempty"`
	Thinking    string          `json:"thinking,omitempty"`
	Signature   string          `json:"signature,omitempty"`
	Data        string          `json:"data,omitempty"`
}

type Message struct {
	Role    string
	Content []ContentBlock
}

// ProductoResumen es lo que devuelve la búsqueda del catálogo.
type ProductoResumen struct {
	ID               int32           `json:"id"`
	PadreID          int32           `json:"padre_id"`
	SKU              string          `json:"sku"`
	Nombre           string          `json:"nombre"`
	Imagen           string          `json:"imagen"`
	Precio           float64         `json:"precio"`
	Moneda           string          `json:"moneda"`
	Proceso          string          `json:"proceso"`
	Atributos        json.RawMessage `json:"atributos"`
	Stock            float64         `json:"stock"`
	EnStock          bool            `json:"en_stock"`
	Marca            string          `json:"marca,omitempty"`
	Categoria        string          `json:"categoria,omitempty"`
	Descripcion      string          `json:"descripcion,omitempty"`
	Especificaciones string          `json:"especificaciones,omitempty"`
}

// ProductoDetalle incluye especificaciones completas para obtener_producto.
type ProductoDetalle struct {
	ProductoResumen
	DescripcionCorta  string `json:"descripcion_corta"`
	FichaTecnica      string `json:"ficha_tecnica"`
	Medida            string `json:"medida"`
	UnidadMedida      string `json:"unidad_medida"`
	Subgrupo          string `json:"subgrupo"`
	Modelo            string `json:"modelo"`
	Tipo              string `json:"tipo"`
	Multiplos         int32  `json:"multiplos"`
	LeadTimeDias      int32  `json:"lead_time_dias"`
	PermitirBackorder bool   `json:"permitir_backorder"`
}

// ProductCard es el subconjunto que se envía por SSE para pintar una tarjeta.
type ProductCard struct {
	ID      int32   `json:"id"`
	PadreID int32   `json:"padre_id"`
	SKU     string  `json:"sku"`
	Nombre  string  `json:"nombre"`
	Imagen  string  `json:"imagen"`
	Precio  float64 `json:"precio"`
	Moneda  string  `json:"moneda"`
	Stock   float64 `json:"stock"`
	EnStock bool    `json:"en_stock"`
	Proceso string  `json:"proceso,omitempty"`
}

// PedidoResumen es la cabecera visible por el cliente autenticado.
type PedidoResumen struct {
	ID              int32   `json:"-"`
	Folio           string  `json:"folio"`
	Estado          string  `json:"estado"`
	FechaPedido     string  `json:"fecha_pedido"`
	Total           float64 `json:"total"`
	Moneda          string  `json:"moneda"`
	Guia            string  `json:"guia"`
	Paqueteria      string  `json:"paqueteria"`
	EntregaEstimada string  `json:"entrega_estimada"`
	HasBackorder    bool    `json:"has_backorder"`
}

type PedidoItem struct {
	SKU            string  `json:"sku"`
	Nombre         string  `json:"nombre"`
	Cantidad       int32   `json:"cantidad"`
	PrecioUnitario float64 `json:"precio_unitario"`
	Enviado        int32   `json:"enviado"`
	Backorder      int32   `json:"backorder"`
}

// ContextoCliente se resuelve desde el JWT + usuarios.cliente_id. Nunca
// proviene de argumentos del modelo.
type ContextoCliente struct {
	ClienteID      *int32
	EsDistribuidor bool
}

// ToolContext es el contexto de servidor que reciben las herramientas.
type ToolContext struct {
	ConversationID string
	UsuarioID      *int32
	UsuarioEmail   string
	ClienteID      *int32
	EsDistribuidor bool
	Contacto       *ContactoEscalamiento
}

// ContactoEscalamiento se captura en el formulario del chat o se completa con
// el correo de la sesión. Nunca se toma de argumentos libres del modelo.
type ContactoEscalamiento struct {
	Nombre   string
	Email    string
	Telefono string
	Motivo   string
}

type ToolCallRecord struct {
	Name       string          `json:"name"`
	Input      json.RawMessage `json:"input,omitempty"`
	IsError    bool            `json:"is_error"`
	DurationMS int64           `json:"duration_ms"`
}

type EscalationEvent struct {
	Motivo string    `json:"motivo"`
	Estado string    `json:"estado"`
	Fecha  time.Time `json:"fecha"`
}
