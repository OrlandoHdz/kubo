package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"
)

type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

type ToolOutput struct {
	Content    any
	Products   []ProductCard
	Escalation *EscalationEvent
}

type ToolError struct {
	UserMessage string
	Err         error
}

func (e *ToolError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.UserMessage, e.Err)
	}
	return e.UserMessage
}

func (e *ToolError) Unwrap() error { return e.Err }

type Tool struct {
	Definition ToolDefinition
	Handler    func(ctx context.Context, args json.RawMessage, tc ToolContext) (ToolOutput, error)
}

type ToolRegistry struct {
	tools map[string]Tool
	order []string
}

type ToolDeps struct {
	Catalogo     CatalogoStore
	Pedidos      PedidosStore
	Backorders   BackordersStore
	Devoluciones DevolucionesStore
	Recompensas  RecompensasStore
	Auditoria    AuditoriaStore
	Notifier     EscalationNotifier
}

func NewToolRegistry(deps ToolDeps) *ToolRegistry {
	r := &ToolRegistry{tools: make(map[string]Tool)}

	if deps.Catalogo != nil {
		r.add(Tool{
			Definition: ToolDefinition{
				Name:        "buscar_productos",
				Description: "Busca productos reales del catálogo de soldadura. Úsala antes de recomendar o cotizar. Devuelve hasta 8 resultados con SKU, precio, disponibilidad, proceso y atributos técnicos. No inventes datos que no devuelva esta herramienta.",
				InputSchema: json.RawMessage(buscarProductosSchema),
			},
			Handler: buscarProductosHandler(deps.Catalogo),
		})
		r.add(Tool{
			Definition: ToolDefinition{
				Name:        "obtener_producto",
				Description: "Obtiene el detalle completo de un producto por SKU: especificaciones, atributos técnicos, stock, precio y presentación. Úsala cuando el usuario pregunte por un SKU concreto o necesites más detalle.",
				InputSchema: json.RawMessage(obtenerProductoSchema),
			},
			Handler: obtenerProductoHandler(deps.Catalogo),
		})
	}

	if deps.Pedidos != nil {
		r.add(Tool{
			Definition: ToolDefinition{
				Name:        "consultar_pedido",
				Description: "Consulta un pedido del cliente autenticado usando su folio. El servidor toma el cliente de la sesión; nunca aceptes ni envíes un cliente_id. Si el usuario no está autenticado, la herramienta responde que debe iniciar sesión.",
				InputSchema: json.RawMessage(consultarPedidoSchema),
			},
			Handler: consultarPedidoHandler(deps.Pedidos),
		})
		r.add(Tool{
			Definition: ToolDefinition{
				Name:        "listar_pedidos_cliente",
				Description: "Lista los últimos pedidos del cliente autenticado. El límite máximo es 10 y lo fija el servidor. Úsala para preguntas como 'mis pedidos' o 'mis últimas compras'.",
				InputSchema: json.RawMessage(listarPedidosSchema),
			},
			Handler: listarPedidosHandler(deps.Pedidos),
		})
	}

	if deps.Backorders != nil {
		r.add(Tool{
			Definition: ToolDefinition{
				Name:        "consultar_backorder",
				Description: "Consulta un backorder del cliente autenticado por folio. Devuelve estado, fecha, total, guía, pedido de origen e items con su disponibilidad. El servidor toma el cliente de la sesión; nunca aceptes un cliente_id.",
				InputSchema: json.RawMessage(consultarBackorderSchema),
			},
			Handler: consultarBackorderHandler(deps.Backorders),
		})
		r.add(Tool{
			Definition: ToolDefinition{
				Name:        "listar_backorders_cliente",
				Description: "Lista los últimos backorders del cliente autenticado. El límite máximo es 10 y lo fija el servidor. Úsala para 'mis backorders', 'mis reservas' o 'productos pendientes'.",
				InputSchema: json.RawMessage(listarBackordersSchema),
			},
			Handler: listarBackordersHandler(deps.Backorders),
		})
	}

	if deps.Devoluciones != nil {
		r.add(Tool{
			Definition: ToolDefinition{
				Name:        "consultar_devolucion",
				Description: "Consulta una devolución o garantía del cliente autenticado por folio. Devuelve pedido, tipo, estatus, números de parte, cantidades y fechas. No expone notas internas del administrador.",
				InputSchema: json.RawMessage(consultarDevolucionSchema),
			},
			Handler: consultarDevolucionHandler(deps.Devoluciones),
		})
		r.add(Tool{
			Definition: ToolDefinition{
				Name:        "listar_devoluciones_cliente",
				Description: "Lista las últimas devoluciones y garantías del cliente autenticado. El límite máximo es 10 y lo fija el servidor.",
				InputSchema: json.RawMessage(listarDevolucionesSchema),
			},
			Handler: listarDevolucionesHandler(deps.Devoluciones),
		})
	}

	if deps.Recompensas != nil {
		r.add(Tool{
			Definition: ToolDefinition{
				Name:        "consultar_recompensas_cliente",
				Description: "Consulta los puntos, valor de puntos, nivel actual, siguiente nivel, promedio de compra de 90 días y últimas transacciones del cliente autenticado. El cliente sale de la sesión, nunca de los argumentos.",
				InputSchema: json.RawMessage(consultarRecompensasSchema),
			},
			Handler: consultarRecompensasHandler(deps.Recompensas),
		})
		r.add(Tool{
			Definition: ToolDefinition{
				Name:        "listar_niveles_recompensa",
				Description: "Lista el catálogo público de niveles de recompensa con su número, promedio de compra trimestral requerido, porcentaje de descuento y reglas de puntos. No requiere sesión.",
				InputSchema: json.RawMessage(listarNivelesSchema),
			},
			Handler: listarNivelesRecompensaHandler(deps.Recompensas),
		})
	}

	if deps.Auditoria != nil {
		r.add(Tool{
			Definition: ToolDefinition{
				Name:        "escalar_a_humano",
				Description: "Registra una solicitud de atención humana con los datos de contacto que el usuario capturó en el formulario del chat y notifica al equipo. Úsala cuando el usuario pida un asesor, cuando no puedas resolver la consulta o cuando el caso requiera validación técnica. Si la herramienta responde requiere_contacto, pide al usuario que use el botón 'Hablar con un asesor' para dejar sus datos.",
				InputSchema: json.RawMessage(escalarSchema),
			},
			Handler: escalarHandler(deps.Auditoria, deps.Notifier),
		})
	}

	return r
}

func (r *ToolRegistry) add(tool Tool) {
	if _, exists := r.tools[tool.Definition.Name]; exists {
		return
	}
	r.tools[tool.Definition.Name] = tool
	r.order = append(r.order, tool.Definition.Name)
}

func (r *ToolRegistry) Definitions() []ToolDefinition {
	if r == nil {
		return nil
	}
	out := make([]ToolDefinition, 0, len(r.order))
	for _, name := range r.order {
		out = append(out, r.tools[name].Definition)
	}
	return out
}

func (r *ToolRegistry) Execute(ctx context.Context, name string, raw json.RawMessage, tc ToolContext) (out ToolOutput, err error) {
	if r == nil {
		return ToolOutput{}, &ToolError{UserMessage: "Las herramientas no están disponibles en este momento."}
	}
	tool, ok := r.tools[name]
	if !ok {
		return ToolOutput{}, &ToolError{
			UserMessage: "La herramienta solicitada no está disponible.",
			Err:         fmt.Errorf("herramienta desconocida: %s", name),
		}
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			out = ToolOutput{}
			err = &ToolError{
				UserMessage: "La herramienta no pudo completar la consulta.",
				Err:         fmt.Errorf("panic en herramienta %s: %v", name, recovered),
			}
		}
	}()

	return tool.Handler(ctx, raw, tc)
}

const buscarProductosSchema = `{
  "type": "object",
  "properties": {
    "consulta": {
      "type": "string",
      "description": "Texto libre: producto, uso, marca o SKU parcial. No incluyas datos personales.",
      "minLength": 1,
      "maxLength": 200
    },
    "proceso": {
      "type": "string",
      "description": "Proceso de soldadura, si el usuario lo indicó.",
      "enum": ["MIG", "TIG", "MMA", "FCAW", "SAW"]
    },
    "filtros": {
      "type": "object",
      "description": "Filtros técnicos opcionales. Solo usa los que el usuario haya mencionado.",
      "properties": {
        "material_base": {"type": "string", "maxLength": 120},
        "diametro_mm": {"type": "number"},
        "clasificacion": {"type": "string", "maxLength": 80},
        "polaridad": {"type": "string", "maxLength": 40},
        "gas": {"type": "string", "maxLength": 80},
        "amperaje_min": {"type": "number"},
        "amperaje_max": {"type": "number"},
        "en_stock": {"type": "boolean"}
      },
      "additionalProperties": false
    }
  },
  "required": ["consulta"],
  "additionalProperties": false
}`

const obtenerProductoSchema = `{
  "type": "object",
  "properties": {
    "sku": {
      "type": "string",
      "description": "SKU exacto del producto.",
      "minLength": 1,
      "maxLength": 50
    }
  },
  "required": ["sku"],
  "additionalProperties": false
}`

const consultarPedidoSchema = `{
  "type": "object",
  "properties": {
    "numero_pedido": {
      "type": "string",
      "description": "Folio del pedido del cliente autenticado.",
      "minLength": 1,
      "maxLength": 20
    }
  },
  "required": ["numero_pedido"],
  "additionalProperties": false
}`

const listarPedidosSchema = `{
  "type": "object",
  "properties": {
    "limite": {
      "type": "integer",
      "description": "Número de pedidos a devolver. Máximo 10.",
      "minimum": 1,
      "maximum": 10
    }
  },
  "additionalProperties": false
}`

const escalarSchema = `{
  "type": "object",
  "properties": {
    "motivo": {
      "type": "string",
      "description": "Motivo breve de la solicitud de atención humana.",
      "minLength": 1,
      "maxLength": 500
    }
  },
  "required": ["motivo"],
  "additionalProperties": false
}`

const consultarBackorderSchema = `{
  "type": "object",
  "properties": {
    "numero_backorder": {
      "type": "string",
      "description": "Folio del backorder del cliente autenticado.",
      "minLength": 1,
      "maxLength": 20
    }
  },
  "required": ["numero_backorder"],
  "additionalProperties": false
}`

const listarBackordersSchema = `{
  "type": "object",
  "properties": {
    "limite": {
      "type": "integer",
      "description": "Número de backorders a devolver. Máximo 10.",
      "minimum": 1,
      "maximum": 10
    }
  },
  "additionalProperties": false
}`

const consultarDevolucionSchema = `{
  "type": "object",
  "properties": {
    "numero_devolucion": {
      "type": "string",
      "description": "Folio de la devolución o garantía del cliente autenticado.",
      "minLength": 1,
      "maxLength": 20
    }
  },
  "required": ["numero_devolucion"],
  "additionalProperties": false
}`

const listarDevolucionesSchema = `{
  "type": "object",
  "properties": {
    "limite": {
      "type": "integer",
      "description": "Número de devoluciones a devolver. Máximo 10.",
      "minimum": 1,
      "maximum": 10
    }
  },
  "additionalProperties": false
}`

const consultarRecompensasSchema = `{
  "type": "object",
  "properties": {},
  "additionalProperties": false
}`

const listarNivelesSchema = `{
  "type": "object",
  "properties": {},
  "additionalProperties": false
}`

type buscarProductosArgs struct {
	Consulta string           `json:"consulta"`
	Proceso  string           `json:"proceso"`
	Filtros  *filtrosProducto `json:"filtros"`

	// Algunos modelos envían los filtros en el nivel superior en lugar de
	// anidarlos en "filtros". Se aceptan y se mezclan para no perder la
	// intención de la búsqueda.
	MaterialBase  string   `json:"material_base"`
	DiametroMM    *float64 `json:"diametro_mm"`
	Clasificacion string   `json:"clasificacion"`
	Polaridad     string   `json:"polaridad"`
	Gas           string   `json:"gas"`
	AmperajeMin   *float64 `json:"amperaje_min"`
	AmperajeMax   *float64 `json:"amperaje_max"`
	EnStock       *bool    `json:"en_stock"`
}

type filtrosProducto struct {
	MaterialBase  string   `json:"material_base"`
	DiametroMM    *float64 `json:"diametro_mm"`
	Clasificacion string   `json:"clasificacion"`
	Polaridad     string   `json:"polaridad"`
	Gas           string   `json:"gas"`
	AmperajeMin   *float64 `json:"amperaje_min"`
	AmperajeMax   *float64 `json:"amperaje_max"`
	EnStock       *bool    `json:"en_stock"`
}

type productoBusquedaItem struct {
	ID        int32           `json:"id"`
	SKU       string          `json:"sku"`
	Nombre    string          `json:"nombre"`
	Precio    float64         `json:"precio"`
	Moneda    string          `json:"moneda"`
	Proceso   string          `json:"proceso"`
	Atributos json.RawMessage `json:"atributos"`
	EnStock   bool            `json:"en_stock"`
}

func buscarProductosHandler(store CatalogoStore) func(context.Context, json.RawMessage, ToolContext) (ToolOutput, error) {
	return func(ctx context.Context, raw json.RawMessage, tc ToolContext) (ToolOutput, error) {
		var args buscarProductosArgs
		if err := decodeArgs(raw, &args); err != nil {
			return ToolOutput{}, &ToolError{UserMessage: "Los parámetros de búsqueda no son válidos.", Err: err}
		}

		consulta := strings.TrimSpace(args.Consulta)
		if consulta == "" || utf8.RuneCountInString(consulta) > 200 {
			return ToolOutput{}, &ToolError{UserMessage: "La consulta debe tener entre 1 y 200 caracteres."}
		}

		proceso := strings.ToUpper(strings.TrimSpace(args.Proceso))
		if proceso != "" && !procesoValido(proceso) {
			return ToolOutput{}, &ToolError{UserMessage: "El proceso indicado no es válido. Usa MIG, TIG, MMA, FCAW o SAW."}
		}

		var (
			filtrosMap map[string]any
			amin       *float64
			amax       *float64
			enStock    *bool
		)
		filtros := args.Filtros
		if filtros == nil {
			filtros = &filtrosProducto{}
		}
		if filtros.MaterialBase == "" {
			filtros.MaterialBase = args.MaterialBase
		}
		if filtros.DiametroMM == nil {
			filtros.DiametroMM = args.DiametroMM
		}
		if filtros.Clasificacion == "" {
			filtros.Clasificacion = args.Clasificacion
		}
		if filtros.Polaridad == "" {
			filtros.Polaridad = args.Polaridad
		}
		if filtros.Gas == "" {
			filtros.Gas = args.Gas
		}
		if filtros.AmperajeMin == nil {
			filtros.AmperajeMin = args.AmperajeMin
		}
		if filtros.AmperajeMax == nil {
			filtros.AmperajeMax = args.AmperajeMax
		}
		if filtros.EnStock == nil {
			filtros.EnStock = args.EnStock
		}

		filtrosMap = make(map[string]any)
		if v := strings.TrimSpace(filtros.MaterialBase); v != "" {
			filtrosMap["material_base"] = v
		}
		if filtros.DiametroMM != nil {
			filtrosMap["diametro_mm"] = *filtros.DiametroMM
		}
		if v := strings.TrimSpace(filtros.Clasificacion); v != "" {
			filtrosMap["clasificacion"] = v
		}
		if v := strings.TrimSpace(filtros.Polaridad); v != "" {
			filtrosMap["polaridad"] = v
		}
		if v := strings.TrimSpace(filtros.Gas); v != "" {
			filtrosMap["gas"] = v
		}
		amin = filtros.AmperajeMin
		amax = filtros.AmperajeMax
		enStock = filtros.EnStock
		if amin != nil && amax != nil && *amin > *amax {
			return ToolOutput{}, &ToolError{UserMessage: "El amperaje mínimo no puede ser mayor que el máximo."}
		}

		var filtrosJSON json.RawMessage
		if len(filtrosMap) > 0 {
			encoded, err := json.Marshal(filtrosMap)
			if err != nil {
				return ToolOutput{}, &ToolError{UserMessage: "No se pudieron interpretar los filtros.", Err: err}
			}
			filtrosJSON = encoded
		}

		productos, err := store.BuscarProductos(ctx, BuscarProductosInput{
			Consulta:       consulta,
			Proceso:        proceso,
			Filtros:        filtrosJSON,
			AmperajeMin:    amin,
			AmperajeMax:    amax,
			EnStock:        enStock,
			EsDistribuidor: tc.EsDistribuidor,
		})
		if err != nil {
			return ToolOutput{}, &ToolError{UserMessage: "No pude consultar el catálogo en este momento.", Err: err}
		}

		items := make([]productoBusquedaItem, 0, len(productos))
		cards := make([]ProductCard, 0, len(productos))
		for _, p := range productos {
			items = append(items, productoBusquedaItem{
				ID:        p.ID,
				SKU:       p.SKU,
				Nombre:    p.Nombre,
				Precio:    p.Precio,
				Moneda:    p.Moneda,
				Proceso:   p.Proceso,
				Atributos: p.Atributos,
				EnStock:   p.EnStock,
			})
			cards = append(cards, cardFromResumen(p))
		}

		content := map[string]any{
			"productos": items,
			"total":     len(items),
		}
		if len(items) == 0 {
			content["mensaje"] = "No encontré productos que coincidan con la búsqueda. Prueba con otro nombre, marca o característica."
		}
		return ToolOutput{Content: content, Products: cards}, nil
	}
}

func obtenerProductoHandler(store CatalogoStore) func(context.Context, json.RawMessage, ToolContext) (ToolOutput, error) {
	return func(ctx context.Context, raw json.RawMessage, tc ToolContext) (ToolOutput, error) {
		var args struct {
			SKU string `json:"sku"`
		}
		if err := decodeArgs(raw, &args); err != nil {
			return ToolOutput{}, &ToolError{UserMessage: "El SKU no es válido.", Err: err}
		}
		sku := strings.TrimSpace(args.SKU)
		if sku == "" || utf8.RuneCountInString(sku) > 50 {
			return ToolOutput{}, &ToolError{UserMessage: "El SKU debe tener entre 1 y 50 caracteres."}
		}

		detalle, err := store.ObtenerProductoPorSKU(ctx, sku, tc.EsDistribuidor)
		if err != nil {
			return ToolOutput{}, &ToolError{UserMessage: "No pude consultar el producto en este momento.", Err: err}
		}
		if detalle == nil {
			return ToolOutput{
				Content: map[string]any{
					"encontrado": false,
					"mensaje":    "No encontré un producto con ese SKU en el catálogo.",
				},
			}, nil
		}

		return ToolOutput{
			Content: map[string]any{
				"encontrado": true,
				"producto":   detalle,
			},
			Products: []ProductCard{cardFromResumen(detalle.ProductoResumen)},
		}, nil
	}
}

func consultarPedidoHandler(store PedidosStore) func(context.Context, json.RawMessage, ToolContext) (ToolOutput, error) {
	return func(ctx context.Context, raw json.RawMessage, tc ToolContext) (ToolOutput, error) {
		var args struct {
			NumeroPedido string `json:"numero_pedido"`
		}
		if err := decodeArgs(raw, &args); err != nil {
			return ToolOutput{}, &ToolError{UserMessage: "El número de pedido no es válido.", Err: err}
		}
		folio := strings.ToUpper(strings.TrimSpace(args.NumeroPedido))
		if folio == "" || utf8.RuneCountInString(folio) > 20 {
			return ToolOutput{}, &ToolError{UserMessage: "El número de pedido debe tener entre 1 y 20 caracteres."}
		}
		if tc.ClienteID == nil {
			return noCustomerOutput(tc), nil
		}

		var (
			pedido *PedidoResumen
			items  []PedidoItem
		)
		err := store.ConCliente(ctx, *tc.ClienteID, func(q PedidosQuerier) error {
			p, err := q.ObtenerPedido(ctx, folio, *tc.ClienteID)
			if err != nil {
				return err
			}
			if p == nil {
				return nil
			}
			pedido = p
			it, err := q.ListarItems(ctx, p.ID, *tc.ClienteID)
			if err != nil {
				return err
			}
			items = it
			return nil
		})
		if err != nil {
			return ToolOutput{}, &ToolError{UserMessage: "No pude consultar el pedido en este momento.", Err: err}
		}
		if pedido == nil {
			return ToolOutput{
				Content: map[string]any{
					"encontrado": false,
					"mensaje":    "No encontré un pedido con ese folio en tu cuenta.",
				},
			}, nil
		}

		return ToolOutput{
			Content: map[string]any{
				"encontrado": true,
				"pedido":     pedido,
				"items":      items,
			},
		}, nil
	}
}

func listarPedidosHandler(store PedidosStore) func(context.Context, json.RawMessage, ToolContext) (ToolOutput, error) {
	return func(ctx context.Context, raw json.RawMessage, tc ToolContext) (ToolOutput, error) {
		var args struct {
			Limite *int `json:"limite"`
		}
		if err := decodeArgs(raw, &args); err != nil {
			return ToolOutput{}, &ToolError{UserMessage: "El límite no es válido.", Err: err}
		}
		limite := 5
		if args.Limite != nil {
			limite = *args.Limite
		}
		if limite < 1 {
			limite = 1
		}
		if limite > 10 {
			limite = 10
		}
		if tc.ClienteID == nil {
			return noCustomerOutput(tc), nil
		}

		var pedidos []PedidoResumen
		err := store.ConCliente(ctx, *tc.ClienteID, func(q PedidosQuerier) error {
			p, err := q.ListarPedidos(ctx, *tc.ClienteID, limite)
			if err != nil {
				return err
			}
			pedidos = p
			return nil
		})
		if err != nil {
			return ToolOutput{}, &ToolError{UserMessage: "No pude consultar tus pedidos en este momento.", Err: err}
		}

		content := map[string]any{
			"pedidos": pedidos,
			"total":   len(pedidos),
			"limite":  limite,
		}
		if len(pedidos) == 0 {
			content["mensaje"] = "No encontré pedidos en tu cuenta."
		}
		return ToolOutput{Content: content}, nil
	}
}

func consultarBackorderHandler(store BackordersStore) func(context.Context, json.RawMessage, ToolContext) (ToolOutput, error) {
	return func(ctx context.Context, raw json.RawMessage, tc ToolContext) (ToolOutput, error) {
		var args struct {
			NumeroBackorder string `json:"numero_backorder"`
		}
		if err := decodeArgs(raw, &args); err != nil {
			return ToolOutput{}, &ToolError{UserMessage: "El número de backorder no es válido.", Err: err}
		}
		folio := strings.ToUpper(strings.TrimSpace(args.NumeroBackorder))
		if folio == "" || utf8.RuneCountInString(folio) > 20 {
			return ToolOutput{}, &ToolError{UserMessage: "El número de backorder debe tener entre 1 y 20 caracteres."}
		}
		if tc.ClienteID == nil {
			return noCustomerOutput(tc), nil
		}

		var (
			backorder *BackorderResumen
			items     []BackorderItem
		)
		err := store.ConClienteBackorders(ctx, *tc.ClienteID, func(q BackordersQuerier) error {
			b, err := q.ObtenerBackorder(ctx, folio, *tc.ClienteID)
			if err != nil {
				return err
			}
			if b == nil {
				return nil
			}
			backorder = b
			it, err := q.ListarBackorderItems(ctx, b.ID, *tc.ClienteID)
			if err != nil {
				return err
			}
			items = it
			return nil
		})
		if err != nil {
			return ToolOutput{}, &ToolError{UserMessage: "No pude consultar el backorder en este momento.", Err: err}
		}
		if backorder == nil {
			return ToolOutput{
				Content: map[string]any{
					"encontrado": false,
					"mensaje":    "No encontré un backorder con ese folio en tu cuenta.",
				},
			}, nil
		}

		return ToolOutput{
			Content: map[string]any{
				"encontrado": true,
				"backorder":  backorder,
				"items":      items,
			},
		}, nil
	}
}

func listarBackordersHandler(store BackordersStore) func(context.Context, json.RawMessage, ToolContext) (ToolOutput, error) {
	return func(ctx context.Context, raw json.RawMessage, tc ToolContext) (ToolOutput, error) {
		var args struct {
			Limite *int `json:"limite"`
		}
		if err := decodeArgs(raw, &args); err != nil {
			return ToolOutput{}, &ToolError{UserMessage: "El límite no es válido.", Err: err}
		}
		limite := 5
		if args.Limite != nil {
			limite = *args.Limite
		}
		if limite < 1 {
			limite = 1
		}
		if limite > 10 {
			limite = 10
		}
		if tc.ClienteID == nil {
			return noCustomerOutput(tc), nil
		}

		var backorders []BackorderResumen
		err := store.ConClienteBackorders(ctx, *tc.ClienteID, func(q BackordersQuerier) error {
			result, err := q.ListarBackorders(ctx, *tc.ClienteID, limite)
			if err != nil {
				return err
			}
			backorders = result
			return nil
		})
		if err != nil {
			return ToolOutput{}, &ToolError{UserMessage: "No pude consultar tus backorders en este momento.", Err: err}
		}

		content := map[string]any{
			"backorders": backorders,
			"total":      len(backorders),
			"limite":     limite,
		}
		if len(backorders) == 0 {
			content["mensaje"] = "No encontré backorders en tu cuenta."
		}
		return ToolOutput{Content: content}, nil
	}
}

func consultarDevolucionHandler(store DevolucionesStore) func(context.Context, json.RawMessage, ToolContext) (ToolOutput, error) {
	return func(ctx context.Context, raw json.RawMessage, tc ToolContext) (ToolOutput, error) {
		var args struct {
			NumeroDevolucion string `json:"numero_devolucion"`
		}
		if err := decodeArgs(raw, &args); err != nil {
			return ToolOutput{}, &ToolError{UserMessage: "El número de devolución no es válido.", Err: err}
		}
		folio := strings.ToUpper(strings.TrimSpace(args.NumeroDevolucion))
		if folio == "" || utf8.RuneCountInString(folio) > 20 {
			return ToolOutput{}, &ToolError{UserMessage: "El número de devolución debe tener entre 1 y 20 caracteres."}
		}
		if tc.ClienteID == nil {
			return noCustomerOutput(tc), nil
		}

		var devolucion *DevolucionResumen
		err := store.ConClienteDevoluciones(ctx, *tc.ClienteID, func(q DevolucionesQuerier) error {
			d, err := q.ObtenerDevolucion(ctx, folio, *tc.ClienteID)
			if err != nil {
				return err
			}
			devolucion = d
			return nil
		})
		if err != nil {
			return ToolOutput{}, &ToolError{UserMessage: "No pude consultar la devolución en este momento.", Err: err}
		}
		if devolucion == nil {
			return ToolOutput{
				Content: map[string]any{
					"encontrado": false,
					"mensaje":    "No encontré una devolución o garantía con ese folio en tu cuenta.",
				},
			}, nil
		}

		return ToolOutput{
			Content: map[string]any{
				"encontrado": true,
				"devolucion": devolucion,
			},
		}, nil
	}
}

func listarDevolucionesHandler(store DevolucionesStore) func(context.Context, json.RawMessage, ToolContext) (ToolOutput, error) {
	return func(ctx context.Context, raw json.RawMessage, tc ToolContext) (ToolOutput, error) {
		var args struct {
			Limite *int `json:"limite"`
		}
		if err := decodeArgs(raw, &args); err != nil {
			return ToolOutput{}, &ToolError{UserMessage: "El límite no es válido.", Err: err}
		}
		limite := 5
		if args.Limite != nil {
			limite = *args.Limite
		}
		if limite < 1 {
			limite = 1
		}
		if limite > 10 {
			limite = 10
		}
		if tc.ClienteID == nil {
			return noCustomerOutput(tc), nil
		}

		var devoluciones []DevolucionResumen
		err := store.ConClienteDevoluciones(ctx, *tc.ClienteID, func(q DevolucionesQuerier) error {
			result, err := q.ListarDevoluciones(ctx, *tc.ClienteID, limite)
			if err != nil {
				return err
			}
			devoluciones = result
			return nil
		})
		if err != nil {
			return ToolOutput{}, &ToolError{UserMessage: "No pude consultar tus devoluciones en este momento.", Err: err}
		}

		content := map[string]any{
			"devoluciones": devoluciones,
			"total":        len(devoluciones),
			"limite":       limite,
		}
		if len(devoluciones) == 0 {
			content["mensaje"] = "No encontré devoluciones o garantías en tu cuenta."
		}
		return ToolOutput{Content: content}, nil
	}
}

func consultarRecompensasHandler(store RecompensasStore) func(context.Context, json.RawMessage, ToolContext) (ToolOutput, error) {
	return func(ctx context.Context, raw json.RawMessage, tc ToolContext) (ToolOutput, error) {
		if err := decodeArgs(raw, &struct{}{}); err != nil {
			return ToolOutput{}, &ToolError{UserMessage: "Esta herramienta no acepta argumentos.", Err: err}
		}
		if tc.ClienteID == nil {
			return noCustomerOutput(tc), nil
		}

		var (
			info          *RecompensasCliente
			niveles       []NivelRecompensa
			transacciones []TransaccionPuntos
		)
		err := store.ConClienteRecompensas(ctx, *tc.ClienteID, func(q RecompensasQuerier) error {
			var err error
			info, err = q.ObtenerRecompensasCliente(ctx, *tc.ClienteID)
			if err != nil {
				return err
			}
			niveles, err = q.ListarNivelesRecompensa(ctx)
			if err != nil {
				return err
			}
			transacciones, err = q.ListarTransaccionesPuntos(ctx, *tc.ClienteID, 10)
			return err
		})
		if err != nil {
			return ToolOutput{}, &ToolError{UserMessage: "No pude consultar tus recompensas en este momento.", Err: err}
		}
		if info == nil {
			return ToolOutput{
				Content: map[string]any{
					"encontrado": false,
					"mensaje":    "No encontré tu perfil de recompensas.",
				},
			}, nil
		}

		info.Niveles = niveles
		info.UltimasTransacciones = transacciones
		info.SiguienteNivel = siguienteNivel(info.NivelActual, niveles)
		if info.SiguienteNivel != nil {
			faltante := info.SiguienteNivel.PromedioCompraTrimestral - info.PromedioTrimestral
			if faltante < 0 {
				faltante = 0
			}
			info.FaltanteParaSiguiente = faltante
		}

		return ToolOutput{Content: map[string]any{"recompensas": info}}, nil
	}
}

func listarNivelesRecompensaHandler(store RecompensasStore) func(context.Context, json.RawMessage, ToolContext) (ToolOutput, error) {
	return func(ctx context.Context, raw json.RawMessage, tc ToolContext) (ToolOutput, error) {
		if err := decodeArgs(raw, &struct{}{}); err != nil {
			return ToolOutput{}, &ToolError{UserMessage: "Esta herramienta no acepta argumentos.", Err: err}
		}
		result, err := store.ListarNivelesRecompensa(ctx)
		if err != nil {
			return ToolOutput{}, &ToolError{UserMessage: "No pude consultar los niveles de recompensa.", Err: err}
		}
		return ToolOutput{
			Content: map[string]any{
				"niveles":                result.Niveles,
				"total":                  len(result.Niveles),
				"monto_compra_por_punto": result.MontoCompraPorPunto,
			},
		}, nil
	}
}

func siguienteNivel(actual *NivelRecompensa, niveles []NivelRecompensa) *NivelRecompensa {
	if len(niveles) == 0 {
		return nil
	}
	if actual == nil {
		return &niveles[0]
	}
	for i := range niveles {
		if niveles[i].NumeroNivel > actual.NumeroNivel {
			return &niveles[i]
		}
	}
	return nil
}

func escalarHandler(store AuditoriaStore, notifier EscalationNotifier) func(context.Context, json.RawMessage, ToolContext) (ToolOutput, error) {
	return func(ctx context.Context, raw json.RawMessage, tc ToolContext) (ToolOutput, error) {
		var args struct {
			Motivo string `json:"motivo"`
		}
		if err := decodeArgs(raw, &args); err != nil {
			return ToolOutput{}, &ToolError{UserMessage: "El motivo no es válido.", Err: err}
		}
		motivo := strings.TrimSpace(args.Motivo)
		if tc.Contacto != nil && strings.TrimSpace(tc.Contacto.Motivo) != "" {
			motivo = strings.TrimSpace(tc.Contacto.Motivo)
		}
		if motivo == "" || utf8.RuneCountInString(motivo) > 500 {
			return ToolOutput{}, &ToolError{UserMessage: "El motivo debe tener entre 1 y 500 caracteres."}
		}
		if tc.ConversationID == "" {
			return ToolOutput{}, &ToolError{UserMessage: "No pude identificar la conversación para escalar."}
		}

		nombre, email, telefono := "", "", ""
		if tc.Contacto != nil {
			nombre = strings.TrimSpace(tc.Contacto.Nombre)
			email = strings.TrimSpace(tc.Contacto.Email)
			telefono = strings.TrimSpace(tc.Contacto.Telefono)
		}
		if email == "" {
			email = strings.TrimSpace(tc.UsuarioEmail)
		}
		if utf8.RuneCountInString(nombre) > 120 || utf8.RuneCountInString(email) > 160 || utf8.RuneCountInString(telefono) > 30 {
			return ToolOutput{}, &ToolError{UserMessage: "Los datos de contacto no son válidos."}
		}
		if email == "" && telefono == "" {
			return ToolOutput{
				Content: map[string]any{
					"escalado":          false,
					"requiere_contacto": true,
					"mensaje":           "Para registrar la solicitud necesito que uses el botón 'Hablar con un asesor' y dejes tu nombre y un correo o teléfono de contacto.",
				},
			}, nil
		}

		if err := store.CrearEscalamiento(ctx, tc.ConversationID, tc.UsuarioID, motivo, nombre, email, telefono); err != nil {
			return ToolOutput{}, &ToolError{UserMessage: "No pude registrar la solicitud de asesor en este momento.", Err: err}
		}

		creado := time.Now()
		if notifier != nil {
			notifier.Notificar(EscalationNotification{
				ConversationID: tc.ConversationID,
				UsuarioID:      tc.UsuarioID,
				UsuarioEmail:   tc.UsuarioEmail,
				ClienteID:      tc.ClienteID,
				Nombre:         nombre,
				Email:          email,
				Telefono:       telefono,
				Motivo:         motivo,
				CreatedAt:      creado,
			})
		}

		return ToolOutput{
			Content: map[string]any{
				"escalado": true,
				"mensaje":  "La solicitud de atención humana quedó registrada. Un asesor revisará los datos de contacto.",
			},
			Escalation: &EscalationEvent{
				Motivo: motivo,
				Estado: "pendiente",
				Fecha:  creado,
			},
		}, nil
	}
}

func noCustomerOutput(tc ToolContext) ToolOutput {
	if tc.UsuarioID != nil {
		return ToolOutput{
			Content: map[string]any{
				"requiere_login": false,
				"mensaje":        "Tu usuario no está asociado a una cuenta de cliente. Para consultar datos de tu cuenta usa una cuenta de cliente.",
			},
		}
	}
	return ToolOutput{
		Content: map[string]any{
			"requiere_login": true,
			"mensaje":        "Para consultar datos de tu cuenta necesitas iniciar sesión con tu cuenta de cliente.",
		},
	}
}

func cardFromResumen(p ProductoResumen) ProductCard {
	return ProductCard{
		ID:      p.ID,
		PadreID: p.PadreID,
		SKU:     p.SKU,
		Nombre:  p.Nombre,
		Imagen:  p.Imagen,
		Precio:  p.Precio,
		Moneda:  p.Moneda,
		Stock:   p.Stock,
		EnStock: p.EnStock,
		Proceso: p.Proceso,
	}
}

func procesoValido(v string) bool {
	switch v {
	case "MIG", "TIG", "MMA", "FCAW", "SAW":
		return true
	default:
		return false
	}
}

func decodeArgs(raw json.RawMessage, dst any) error {
	if len(bytes.TrimSpace(raw)) == 0 {
		raw = json.RawMessage(`{}`)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("argumentos adicionales")
	}
	return nil
}
