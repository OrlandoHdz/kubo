package chat

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type mockCatalogo struct {
	buscarInput  BuscarProductosInput
	buscarRows   []ProductoResumen
	buscarErr    error
	buscarCalls  int
	detalle      *ProductoDetalle
	detalleErr   error
	detalleSKU   string
	detalleDist  bool
	detalleCalls int
}

func (m *mockCatalogo) BuscarProductos(ctx context.Context, in BuscarProductosInput) ([]ProductoResumen, error) {
	m.buscarCalls++
	m.buscarInput = in
	return m.buscarRows, m.buscarErr
}

func (m *mockCatalogo) ObtenerProductoPorSKU(ctx context.Context, sku string, esDistribuidor bool) (*ProductoDetalle, error) {
	m.detalleCalls++
	m.detalleSKU = sku
	m.detalleDist = esDistribuidor
	return m.detalle, m.detalleErr
}

type mockPedidosStore struct {
	called          bool
	clienteID       int32
	pedido          *PedidoResumen
	items           []PedidoItem
	pedidos         []PedidoResumen
	err             error
	lastFolio       string
	lastLimite      int
	lastItemOrderID int32
	lastItemCliente int32
	lastListCliente int32
}

func (m *mockPedidosStore) ConCliente(ctx context.Context, clienteID int32, fn func(PedidosQuerier) error) error {
	m.called = true
	m.clienteID = clienteID
	if m.err != nil {
		return m.err
	}
	return fn(&mockPedidosQuerier{store: m})
}

type mockPedidosQuerier struct {
	store *mockPedidosStore
}

func (q *mockPedidosQuerier) ObtenerPedido(ctx context.Context, folio string, clienteID int32) (*PedidoResumen, error) {
	q.store.lastFolio = folio
	return q.store.pedido, nil
}

func (q *mockPedidosQuerier) ListarItems(ctx context.Context, pedidoID int32, clienteID int32) ([]PedidoItem, error) {
	q.store.lastItemOrderID = pedidoID
	q.store.lastItemCliente = clienteID
	return q.store.items, nil
}

func (q *mockPedidosQuerier) ListarPedidos(ctx context.Context, clienteID int32, limite int) ([]PedidoResumen, error) {
	q.store.lastListCliente = clienteID
	q.store.lastLimite = limite
	return q.store.pedidos, nil
}

type mockAuditoria struct {
	called         bool
	conversationID string
	usuarioID      *int32
	motivo         string
	nombre         string
	email          string
	telefono       string
	err            error
}

func (m *mockAuditoria) CrearConversacion(ctx context.Context, id string, usuarioID, clienteID *int32) error {
	return nil
}

func (m *mockAuditoria) CrearMensaje(ctx context.Context, conversationID string, usuarioID *int32, role, content string, toolCalls json.RawMessage, inputTokens, outputTokens int32) error {
	return nil
}

func (m *mockAuditoria) CrearEscalamiento(ctx context.Context, conversationID string, usuarioID *int32, motivo, nombre, email, telefono string) error {
	m.called = true
	m.conversationID = conversationID
	m.usuarioID = usuarioID
	m.motivo = motivo
	m.nombre = nombre
	m.email = email
	m.telefono = telefono
	return m.err
}

type mockNotifier struct {
	called bool
	data   EscalationNotification
}

func (m *mockNotifier) Notificar(data EscalationNotification) {
	m.called = true
	m.data = data
}

func TestBuscarProductosTool(t *testing.T) {
	cat := &mockCatalogo{buscarRows: []ProductoResumen{
		{
			ID: 1, PadreID: 10, SKU: "E6013-18", Nombre: "Electrodo 6013 1/8",
			Precio: 120.5, Moneda: "MXN", Proceso: "MMA", EnStock: true, Stock: 30,
			Atributos: json.RawMessage(`{"material_base":"acero al carbono"}`),
		},
		{
			ID: 2, PadreID: 11, SKU: "E7018-18", Nombre: "Electrodo 7018 1/8",
			Precio: 145, Moneda: "MXN", Proceso: "MMA", EnStock: true, Stock: 12,
		},
	}}
	registry := NewToolRegistry(ToolDeps{Catalogo: cat})

	raw := json.RawMessage(`{
		"consulta":"  6013 1/8 ",
		"proceso":"mma",
		"filtros":{
			"material_base":"acero al carbono",
			"amperaje_min":80,
			"amperaje_max":120,
			"en_stock":true
		}
	}`)
	out, err := registry.Execute(context.Background(), "buscar_productos", raw, ToolContext{EsDistribuidor: true})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if cat.buscarCalls != 1 {
		t.Fatalf("llamadas = %d", cat.buscarCalls)
	}
	if cat.buscarInput.Consulta != "6013 1/8" {
		t.Fatalf("consulta = %q", cat.buscarInput.Consulta)
	}
	if cat.buscarInput.Proceso != "MMA" {
		t.Fatalf("proceso = %q", cat.buscarInput.Proceso)
	}
	if !cat.buscarInput.EsDistribuidor {
		t.Fatal("se esperaba es_distribuidor=true")
	}
	if cat.buscarInput.AmperajeMin == nil || *cat.buscarInput.AmperajeMin != 80 {
		t.Fatalf("amperaje_min = %v", cat.buscarInput.AmperajeMin)
	}
	if cat.buscarInput.EnStock == nil || !*cat.buscarInput.EnStock {
		t.Fatalf("en_stock = %v", cat.buscarInput.EnStock)
	}

	var filtros map[string]any
	if err := json.Unmarshal(cat.buscarInput.Filtros, &filtros); err != nil {
		t.Fatalf("filtros JSON: %v", err)
	}
	if filtros["material_base"] != "acero al carbono" {
		t.Fatalf("filtros = %+v", filtros)
	}
	if _, exists := filtros["amperaje_min"]; exists {
		t.Fatalf("amperaje_min no debe ir en el JSONB de contención: %+v", filtros)
	}
	if _, exists := filtros["en_stock"]; exists {
		t.Fatalf("en_stock no debe ir en el JSONB de contención: %+v", filtros)
	}
	if len(out.Products) != 2 {
		t.Fatalf("tarjetas = %d", len(out.Products))
	}
	content, ok := out.Content.(map[string]any)
	if !ok || content["total"] != 2 {
		t.Fatalf("contenido = %+v", out.Content)
	}
}

func TestBuscarProductosToolAcceptsTopLevelFilters(t *testing.T) {
	cat := &mockCatalogo{}
	registry := NewToolRegistry(ToolDeps{Catalogo: cat})

	_, err := registry.Execute(
		context.Background(),
		"buscar_productos",
		json.RawMessage(`{"consulta":"6013","diametro_mm":3.2,"en_stock":true}`),
		ToolContext{},
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var filtros map[string]any
	if err := json.Unmarshal(cat.buscarInput.Filtros, &filtros); err != nil {
		t.Fatalf("filtros JSON: %v", err)
	}
	if filtros["diametro_mm"] != 3.2 {
		t.Fatalf("diametro_mm = %v", filtros["diametro_mm"])
	}
	if cat.buscarInput.EnStock == nil || !*cat.buscarInput.EnStock {
		t.Fatalf("en_stock = %v", cat.buscarInput.EnStock)
	}
}

func TestBuscarProductosToolValidation(t *testing.T) {
	cat := &mockCatalogo{}
	registry := NewToolRegistry(ToolDeps{Catalogo: cat})
	ctx := context.Background()

	cases := []struct {
		name string
		raw  string
	}{
		{"vacía", `{"consulta":"   "}`},
		{"proceso inválido", `{"consulta":"6013","proceso":"MAG"}`},
		{"campo desconocido", `{"consulta":"6013","cliente_id":99}`},
		{"amperaje invertido", `{"consulta":"6013","filtros":{"amperaje_min":150,"amperaje_max":80}}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := registry.Execute(ctx, "buscar_productos", json.RawMessage(tc.raw), ToolContext{})
			if err == nil {
				t.Fatal("se esperaba error")
			}
		})
	}
	if cat.buscarCalls != 0 {
		t.Fatalf("no debió llamar al store; llamadas = %d", cat.buscarCalls)
	}
}

func TestObtenerProductoTool(t *testing.T) {
	detalle := &ProductoDetalle{
		ProductoResumen: ProductoResumen{
			ID: 3, PadreID: 30, SKU: "TIG-308", Nombre: "Varilla TIG 308",
			Precio: 210, Moneda: "MXN", Proceso: "TIG", EnStock: true, Stock: 8,
		},
		Medida: "2.4 mm", UnidadMedida: "Kg", Multiplos: 1,
	}
	cat := &mockCatalogo{detalle: detalle}
	registry := NewToolRegistry(ToolDeps{Catalogo: cat})

	out, err := registry.Execute(context.Background(), "obtener_producto", json.RawMessage(`{"sku":"TIG-308"}`), ToolContext{EsDistribuidor: true})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if cat.detalleSKU != "TIG-308" || !cat.detalleDist {
		t.Fatalf("sku=%q dist=%v", cat.detalleSKU, cat.detalleDist)
	}
	if len(out.Products) != 1 || out.Products[0].SKU != "TIG-308" {
		t.Fatalf("tarjetas = %+v", out.Products)
	}
	content := out.Content.(map[string]any)
	if content["encontrado"] != true {
		t.Fatalf("contenido = %+v", content)
	}

	cat.detalle = nil
	out, err = registry.Execute(context.Background(), "obtener_producto", json.RawMessage(`{"sku":"NO-EXISTE"}`), ToolContext{})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.Content.(map[string]any)["encontrado"] != false {
		t.Fatalf("contenido = %+v", out.Content)
	}
}

func TestConsultarPedidoUsesSessionCustomer(t *testing.T) {
	store := &mockPedidosStore{
		pedido: &PedidoResumen{ID: 55, Folio: "KUBO-123", Estado: "Enviado", Total: 580.5, Moneda: "MXN"},
		items:  []PedidoItem{{SKU: "E6013-18", Nombre: "Electrodo", Cantidad: 2, PrecioUnitario: 120.5}},
	}
	registry := NewToolRegistry(ToolDeps{Pedidos: store})
	clienteID := int32(7)

	out, err := registry.Execute(context.Background(), "consultar_pedido", json.RawMessage(`{"numero_pedido":" kubo-123 "}`), ToolContext{ClienteID: &clienteID})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !store.called || store.clienteID != 7 {
		t.Fatalf("store called=%v cliente=%d", store.called, store.clienteID)
	}
	if store.lastFolio != "KUBO-123" {
		t.Fatalf("folio = %q", store.lastFolio)
	}
	if store.lastItemCliente != 7 {
		t.Fatalf("items cliente = %d", store.lastItemCliente)
	}
	content := out.Content.(map[string]any)
	if content["encontrado"] != true {
		t.Fatalf("contenido = %+v", content)
	}
	encoded, _ := json.Marshal(content)
	if strings.Contains(string(encoded), "cliente_id") {
		t.Fatalf("la respuesta no debe exponer cliente_id: %s", encoded)
	}
}

func TestConsultarPedidoRejectsCustomerIDArg(t *testing.T) {
	store := &mockPedidosStore{}
	registry := NewToolRegistry(ToolDeps{Pedidos: store})
	clienteID := int32(7)

	_, err := registry.Execute(context.Background(), "consultar_pedido", json.RawMessage(`{"numero_pedido":"KUBO-1","cliente_id":999}`), ToolContext{ClienteID: &clienteID})
	if err == nil {
		t.Fatal("se esperaba error por argumento desconocido")
	}
	if store.called {
		t.Fatal("no debió consultar pedidos")
	}
}

func TestConsultarPedidoRequiresLogin(t *testing.T) {
	store := &mockPedidosStore{}
	registry := NewToolRegistry(ToolDeps{Pedidos: store})

	out, err := registry.Execute(context.Background(), "consultar_pedido", json.RawMessage(`{"numero_pedido":"KUBO-1"}`), ToolContext{})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if store.called {
		t.Fatal("no debió consultar pedidos sin sesión")
	}
	content := out.Content.(map[string]any)
	if content["requiere_login"] != true {
		t.Fatalf("contenido = %+v", content)
	}
}

func TestConsultarPedidoForeignNotFound(t *testing.T) {
	store := &mockPedidosStore{pedido: nil}
	registry := NewToolRegistry(ToolDeps{Pedidos: store})
	clienteID := int32(7)

	out, err := registry.Execute(context.Background(), "consultar_pedido", json.RawMessage(`{"numero_pedido":"AJENO-1"}`), ToolContext{ClienteID: &clienteID})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	content := out.Content.(map[string]any)
	if content["encontrado"] != false {
		t.Fatalf("contenido = %+v", content)
	}
	if strings.Contains(strings.ToLower(content["mensaje"].(string)), "otro") {
		t.Fatalf("no debe revelar información sobre otros clientes: %v", content)
	}
}

func TestListarPedidosClampsLimit(t *testing.T) {
	store := &mockPedidosStore{pedidos: []PedidoResumen{{Folio: "KUBO-1"}}}
	registry := NewToolRegistry(ToolDeps{Pedidos: store})
	clienteID := int32(7)

	cases := []struct {
		raw    string
		expect int
	}{
		{`{"limite":100}`, 10},
		{`{"limite":0}`, 1},
		{`{}`, 5},
		{`{"limite":3}`, 3},
	}
	for _, tc := range cases {
		_, err := registry.Execute(context.Background(), "listar_pedidos_cliente", json.RawMessage(tc.raw), ToolContext{ClienteID: &clienteID})
		if err != nil {
			t.Fatalf("Execute(%s): %v", tc.raw, err)
		}
		if store.lastLimite != tc.expect {
			t.Fatalf("límite para %s = %d, esperado %d", tc.raw, store.lastLimite, tc.expect)
		}
		if store.lastListCliente != 7 {
			t.Fatalf("cliente = %d", store.lastListCliente)
		}
	}
}

func TestListarPedidosRequiresLogin(t *testing.T) {
	store := &mockPedidosStore{}
	registry := NewToolRegistry(ToolDeps{Pedidos: store})

	out, err := registry.Execute(context.Background(), "listar_pedidos_cliente", json.RawMessage(`{}`), ToolContext{})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if store.called {
		t.Fatal("no debió consultar sin sesión")
	}
	if out.Content.(map[string]any)["requiere_login"] != true {
		t.Fatalf("contenido = %+v", out.Content)
	}
}

func TestEscalarAHumano(t *testing.T) {
	audit := &mockAuditoria{}
	notifier := &mockNotifier{}
	registry := NewToolRegistry(ToolDeps{Auditoria: audit, Notifier: notifier})
	usuarioID := int32(9)

	out, err := registry.Execute(context.Background(), "escalar_a_humano", json.RawMessage(`{"motivo":"  Necesito validar un procedimiento  "}`), ToolContext{
		ConversationID: "abc123",
		UsuarioID:      &usuarioID,
		UsuarioEmail:   "cliente@example.com",
		Contacto: &ContactoEscalamiento{
			Nombre:   "Ana Pérez",
			Email:    "ana@example.com",
			Telefono: "5551234567",
		},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !audit.called || audit.conversationID != "abc123" || audit.motivo != "Necesito validar un procedimiento" {
		t.Fatalf("auditoría = %+v", audit)
	}
	if audit.usuarioID == nil || *audit.usuarioID != 9 {
		t.Fatalf("usuario = %v", audit.usuarioID)
	}
	if audit.nombre != "Ana Pérez" || audit.email != "ana@example.com" || audit.telefono != "5551234567" {
		t.Fatalf("contacto auditado = %+v", audit)
	}
	if !notifier.called || notifier.data.Email != "ana@example.com" {
		t.Fatalf("notifier = %+v", notifier)
	}
	if out.Escalation == nil || out.Escalation.Estado != "pendiente" {
		t.Fatalf("escalation = %+v", out.Escalation)
	}
}

func TestEscalarSinContactoPideFormulario(t *testing.T) {
	audit := &mockAuditoria{}
	registry := NewToolRegistry(ToolDeps{Auditoria: audit})

	out, err := registry.Execute(context.Background(), "escalar_a_humano", json.RawMessage(`{"motivo":"Necesito ayuda"}`), ToolContext{
		ConversationID: "abc123",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if audit.called {
		t.Fatal("no debió registrar sin contacto")
	}
	content := out.Content.(map[string]any)
	if content["requiere_contacto"] != true {
		t.Fatalf("contenido = %+v", content)
	}
}

func TestEscalarConEmailDeSesion(t *testing.T) {
	audit := &mockAuditoria{}
	registry := NewToolRegistry(ToolDeps{Auditoria: audit})
	usuarioID := int32(9)

	out, err := registry.Execute(context.Background(), "escalar_a_humano", json.RawMessage(`{"motivo":"Necesito ayuda"}`), ToolContext{
		ConversationID: "abc123",
		UsuarioID:      &usuarioID,
		UsuarioEmail:   "cliente@example.com",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !audit.called || audit.email != "cliente@example.com" {
		t.Fatalf("auditoría = %+v", audit)
	}
	if out.Escalation == nil {
		t.Fatal("se esperaba escalamiento")
	}
}

func TestToolSchemasAreValidJSON(t *testing.T) {
	registry := NewToolRegistry(ToolDeps{
		Catalogo:     &mockCatalogo{},
		Pedidos:      &mockPedidosStore{},
		Backorders:   &mockBackordersStore{},
		Devoluciones: &mockDevolucionesStore{},
		Recompensas:  &mockRecompensasStore{},
		Auditoria:    &mockAuditoria{},
	})
	definitions := registry.Definitions()
	if len(definitions) != 11 {
		t.Fatalf("herramientas registradas = %d, esperado 11", len(definitions))
	}
	for _, definition := range definitions {
		var schema any
		if err := json.Unmarshal(definition.InputSchema, &schema); err != nil {
			t.Fatalf("schema inválido para %s: %v", definition.Name, err)
		}
	}
}

func TestUnknownTool(t *testing.T) {
	registry := NewToolRegistry(ToolDeps{})
	_, err := registry.Execute(context.Background(), "borrar_todo", json.RawMessage(`{}`), ToolContext{})
	if err == nil {
		t.Fatal("se esperaba error")
	}
	if !strings.Contains(err.Error(), "no está disponible") {
		t.Fatalf("error = %v", err)
	}
}

func TestToolPanicIsRecovered(t *testing.T) {
	registry := &ToolRegistry{
		tools: map[string]Tool{
			"panic_tool": {
				Definition: ToolDefinition{Name: "panic_tool"},
				Handler: func(context.Context, json.RawMessage, ToolContext) (ToolOutput, error) {
					panic("boom")
				},
			},
		},
		order: []string{"panic_tool"},
	}
	_, err := registry.Execute(context.Background(), "panic_tool", json.RawMessage(`{}`), ToolContext{})
	if err == nil {
		t.Fatal("se esperaba error recuperado")
	}
	var te *ToolError
	if !errors.As(err, &te) {
		t.Fatalf("error = %T %v", err, err)
	}
}

type mockBackordersStore struct {
	called          bool
	clienteID       int32
	backorder       *BackorderResumen
	items           []BackorderItem
	backorders      []BackorderResumen
	err             error
	lastFolio       string
	lastLimite      int
	lastItemID      int32
	lastItemCliente int32
}

func (m *mockBackordersStore) ConClienteBackorders(ctx context.Context, clienteID int32, fn func(BackordersQuerier) error) error {
	m.called = true
	m.clienteID = clienteID
	if m.err != nil {
		return m.err
	}
	return fn(&mockBackordersQuerier{store: m})
}

type mockBackordersQuerier struct {
	store *mockBackordersStore
}

func (q *mockBackordersQuerier) ObtenerBackorder(ctx context.Context, folio string, clienteID int32) (*BackorderResumen, error) {
	q.store.lastFolio = folio
	return q.store.backorder, nil
}

func (q *mockBackordersQuerier) ListarBackorderItems(ctx context.Context, backorderID int32, clienteID int32) ([]BackorderItem, error) {
	q.store.lastItemID = backorderID
	q.store.lastItemCliente = clienteID
	return q.store.items, nil
}

func (q *mockBackordersQuerier) ListarBackorders(ctx context.Context, clienteID int32, limite int) ([]BackorderResumen, error) {
	q.store.lastLimite = limite
	return q.store.backorders, nil
}

type mockDevolucionesStore struct {
	called       bool
	clienteID    int32
	devolucion   *DevolucionResumen
	devoluciones []DevolucionResumen
	err          error
	lastFolio    string
	lastLimite   int
}

func (m *mockDevolucionesStore) ConClienteDevoluciones(ctx context.Context, clienteID int32, fn func(DevolucionesQuerier) error) error {
	m.called = true
	m.clienteID = clienteID
	if m.err != nil {
		return m.err
	}
	return fn(&mockDevolucionesQuerier{store: m})
}

type mockDevolucionesQuerier struct {
	store *mockDevolucionesStore
}

func (q *mockDevolucionesQuerier) ObtenerDevolucion(ctx context.Context, folio string, clienteID int32) (*DevolucionResumen, error) {
	q.store.lastFolio = folio
	return q.store.devolucion, nil
}

func (q *mockDevolucionesQuerier) ListarDevoluciones(ctx context.Context, clienteID int32, limite int) ([]DevolucionResumen, error) {
	q.store.lastLimite = limite
	return q.store.devoluciones, nil
}

type mockRecompensasStore struct {
	called        bool
	clienteID     int32
	info          *RecompensasCliente
	niveles       []NivelRecompensa
	transacciones []TransaccionPuntos
	publicResult  NivelesRecompensaResult
	err           error
	publicCalled  bool
	lastTxCliente int32
	lastTxLimite  int
}

func (m *mockRecompensasStore) ConClienteRecompensas(ctx context.Context, clienteID int32, fn func(RecompensasQuerier) error) error {
	m.called = true
	m.clienteID = clienteID
	if m.err != nil {
		return m.err
	}
	return fn(&mockRecompensasQuerier{store: m})
}

func (m *mockRecompensasStore) ListarNivelesRecompensa(ctx context.Context) (NivelesRecompensaResult, error) {
	m.publicCalled = true
	return m.publicResult, m.err
}

type mockRecompensasQuerier struct {
	store *mockRecompensasStore
}

func (q *mockRecompensasQuerier) ObtenerRecompensasCliente(ctx context.Context, clienteID int32) (*RecompensasCliente, error) {
	return q.store.info, nil
}

func (q *mockRecompensasQuerier) ListarNivelesRecompensa(ctx context.Context) ([]NivelRecompensa, error) {
	return q.store.niveles, nil
}

func (q *mockRecompensasQuerier) ListarTransaccionesPuntos(ctx context.Context, clienteID int32, limite int) ([]TransaccionPuntos, error) {
	q.store.lastTxCliente = clienteID
	q.store.lastTxLimite = limite
	return q.store.transacciones, nil
}

func TestConsultarBackorderUsesSessionCustomer(t *testing.T) {
	store := &mockBackordersStore{
		backorder: &BackorderResumen{ID: 77, Folio: "BO-1", Estado: "Pendiente", Total: 580.5, Moneda: "MXN", PedidoOrigenFolio: "KUBO-1"},
		items:     []BackorderItem{{SKU: "E6013-18", Nombre: "Electrodo", Cantidad: 2, PrecioUnitario: 120.5, Disponible: false}},
	}
	registry := NewToolRegistry(ToolDeps{Backorders: store})
	clienteID := int32(7)

	out, err := registry.Execute(context.Background(), "consultar_backorder", json.RawMessage(`{"numero_backorder":" bo-1 "}`), ToolContext{ClienteID: &clienteID})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !store.called || store.clienteID != 7 || store.lastFolio != "BO-1" {
		t.Fatalf("store called=%v cliente=%d folio=%q", store.called, store.clienteID, store.lastFolio)
	}
	if store.lastItemID != 77 || store.lastItemCliente != 7 {
		t.Fatalf("items id=%d cliente=%d", store.lastItemID, store.lastItemCliente)
	}
	content := out.Content.(map[string]any)
	if content["encontrado"] != true {
		t.Fatalf("contenido = %+v", content)
	}
	encoded, _ := json.Marshal(content)
	if strings.Contains(string(encoded), "cliente_id") || strings.Contains(string(encoded), "notas_admin") {
		t.Fatalf("respuesta con datos internos: %s", encoded)
	}
}

func TestConsultarBackorderRequiresLogin(t *testing.T) {
	store := &mockBackordersStore{}
	registry := NewToolRegistry(ToolDeps{Backorders: store})
	out, err := registry.Execute(context.Background(), "consultar_backorder", json.RawMessage(`{"numero_backorder":"BO-1"}`), ToolContext{})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if store.called {
		t.Fatal("no debió consultar sin sesión")
	}
	if out.Content.(map[string]any)["requiere_login"] != true {
		t.Fatalf("contenido = %+v", out.Content)
	}
}

func TestListarBackordersClampsLimit(t *testing.T) {
	store := &mockBackordersStore{backorders: []BackorderResumen{{Folio: "BO-1"}}}
	registry := NewToolRegistry(ToolDeps{Backorders: store})
	clienteID := int32(7)

	cases := []struct {
		raw    string
		expect int
	}{
		{`{"limite":100}`, 10},
		{`{"limite":0}`, 1},
		{`{}`, 5},
	}
	for _, tc := range cases {
		if _, err := registry.Execute(context.Background(), "listar_backorders_cliente", json.RawMessage(tc.raw), ToolContext{ClienteID: &clienteID}); err != nil {
			t.Fatalf("Execute(%s): %v", tc.raw, err)
		}
		if store.lastLimite != tc.expect {
			t.Fatalf("límite para %s = %d, esperado %d", tc.raw, store.lastLimite, tc.expect)
		}
	}
}

func TestConsultarDevolucionUsesSessionCustomer(t *testing.T) {
	store := &mockDevolucionesStore{
		devolucion: &DevolucionResumen{
			Folio: "DEV-1", PedidoFolio: "KUBO-1", Tipo: "Garantia",
			Estatus: "Pendiente", NumerosParte: "E6013-18", Cantidades: "2",
		},
	}
	registry := NewToolRegistry(ToolDeps{Devoluciones: store})
	clienteID := int32(7)

	out, err := registry.Execute(context.Background(), "consultar_devolucion", json.RawMessage(`{"numero_devolucion":" dev-1 "}`), ToolContext{ClienteID: &clienteID})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !store.called || store.clienteID != 7 || store.lastFolio != "DEV-1" {
		t.Fatalf("store called=%v cliente=%d folio=%q", store.called, store.clienteID, store.lastFolio)
	}
	content := out.Content.(map[string]any)
	if content["encontrado"] != true {
		t.Fatalf("contenido = %+v", content)
	}
	encoded, _ := json.Marshal(content)
	if strings.Contains(string(encoded), "nota_administrador") || strings.Contains(string(encoded), "evidencias") {
		t.Fatalf("respuesta con datos internos: %s", encoded)
	}
}

func TestConsultarDevolucionRequiresLogin(t *testing.T) {
	store := &mockDevolucionesStore{}
	registry := NewToolRegistry(ToolDeps{Devoluciones: store})
	out, err := registry.Execute(context.Background(), "consultar_devolucion", json.RawMessage(`{"numero_devolucion":"DEV-1"}`), ToolContext{})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if store.called {
		t.Fatal("no debió consultar sin sesión")
	}
	if out.Content.(map[string]any)["requiere_login"] != true {
		t.Fatalf("contenido = %+v", out.Content)
	}
}

func TestListarDevolucionesClampsLimit(t *testing.T) {
	store := &mockDevolucionesStore{devoluciones: []DevolucionResumen{{Folio: "DEV-1"}}}
	registry := NewToolRegistry(ToolDeps{Devoluciones: store})
	clienteID := int32(7)

	for _, tc := range []struct {
		raw    string
		expect int
	}{
		{`{"limite":100}`, 10},
		{`{"limite":0}`, 1},
		{`{}`, 5},
	} {
		if _, err := registry.Execute(context.Background(), "listar_devoluciones_cliente", json.RawMessage(tc.raw), ToolContext{ClienteID: &clienteID}); err != nil {
			t.Fatalf("Execute(%s): %v", tc.raw, err)
		}
		if store.lastLimite != tc.expect {
			t.Fatalf("límite para %s = %d, esperado %d", tc.raw, store.lastLimite, tc.expect)
		}
	}
}

func TestConsultarRecompensas(t *testing.T) {
	niveles := []NivelRecompensa{
		{Nombre: "Nivel 0", NumeroNivel: 0, PromedioCompraTrimestral: 0, PorcentajeDescuento: 0},
		{Nombre: "Plata", NumeroNivel: 2, PromedioCompraTrimestral: 4000, PorcentajeDescuento: 5},
		{Nombre: "Oro", NumeroNivel: 3, PromedioCompraTrimestral: 6000, PorcentajeDescuento: 8},
	}
	store := &mockRecompensasStore{
		info: &RecompensasCliente{
			SaldoPuntos: 1200, ValorPuntoPesos: 0.4, MontoCompraPorPunto: 25,
			Compra90Dias: 15000, PromedioTrimestral: 5000,
			NivelActual: &NivelRecompensa{Nombre: "Plata", NumeroNivel: 2, PromedioCompraTrimestral: 4000, PorcentajeDescuento: 5},
		},
		niveles: niveles,
		transacciones: []TransaccionPuntos{
			{Tipo: "ACUMULADO", Puntos: 50, Fecha: "2026-10-01T10:00:00", PedidoFolio: "KUBO-1"},
		},
	}
	registry := NewToolRegistry(ToolDeps{Recompensas: store})
	clienteID := int32(7)

	out, err := registry.Execute(context.Background(), "consultar_recompensas_cliente", json.RawMessage(`{}`), ToolContext{ClienteID: &clienteID})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !store.called || store.clienteID != 7 || store.lastTxCliente != 7 || store.lastTxLimite != 10 {
		t.Fatalf("store called=%v cliente=%d txCliente=%d txLimite=%d", store.called, store.clienteID, store.lastTxCliente, store.lastTxLimite)
	}
	recompensas := out.Content.(map[string]any)["recompensas"].(*RecompensasCliente)
	if recompensas.SaldoPuntos != 1200 || recompensas.EquivalenciaPesos != 480 {
		t.Fatalf("recompensas = %+v", recompensas)
	}
	if recompensas.SiguienteNivel == nil || recompensas.SiguienteNivel.Nombre != "Oro" {
		t.Fatalf("siguiente nivel = %+v", recompensas.SiguienteNivel)
	}
	if recompensas.FaltanteParaSiguiente != 1000 {
		t.Fatalf("faltante = %v", recompensas.FaltanteParaSiguiente)
	}
	if len(recompensas.Niveles) != 3 || len(recompensas.UltimasTransacciones) != 1 {
		t.Fatalf("niveles=%d tx=%d", len(recompensas.Niveles), len(recompensas.UltimasTransacciones))
	}
}

func TestConsultarRecompensasRequiresLogin(t *testing.T) {
	store := &mockRecompensasStore{}
	registry := NewToolRegistry(ToolDeps{Recompensas: store})
	out, err := registry.Execute(context.Background(), "consultar_recompensas_cliente", json.RawMessage(`{}`), ToolContext{})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if store.called {
		t.Fatal("no debió consultar sin sesión")
	}
	if out.Content.(map[string]any)["requiere_login"] != true {
		t.Fatalf("contenido = %+v", out.Content)
	}
}

func TestListarNivelesRecompensaPublic(t *testing.T) {
	store := &mockRecompensasStore{
		publicResult: NivelesRecompensaResult{
			Niveles:             []NivelRecompensa{{Nombre: "Nivel 0", NumeroNivel: 0}, {Nombre: "Oro", NumeroNivel: 3, PromedioCompraTrimestral: 6000, PorcentajeDescuento: 8}},
			MontoCompraPorPunto: 25,
			ValorPuntoPesos:     0.4,
		},
	}
	registry := NewToolRegistry(ToolDeps{Recompensas: store})

	out, err := registry.Execute(context.Background(), "listar_niveles_recompensa", json.RawMessage(`{}`), ToolContext{})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !store.publicCalled {
		t.Fatal("no se consultó el catálogo")
	}
	content := out.Content.(map[string]any)
	if content["total"] != 2 || content["monto_compra_por_punto"] != 25.0 {
		t.Fatalf("contenido = %+v", content)
	}
}

func TestSiguienteNivel(t *testing.T) {
	niveles := []NivelRecompensa{
		{Nombre: "Nivel 0", NumeroNivel: 0},
		{Nombre: "Plata", NumeroNivel: 2},
		{Nombre: "Oro", NumeroNivel: 3},
	}
	if got := siguienteNivel(nil, niveles); got == nil || got.Nombre != "Nivel 0" {
		t.Fatalf("sin nivel = %+v", got)
	}
	if got := siguienteNivel(&niveles[1], niveles); got == nil || got.Nombre != "Oro" {
		t.Fatalf("siguiente desde Plata = %+v", got)
	}
	if got := siguienteNivel(&niveles[2], niveles); got != nil {
		t.Fatalf("siguiente desde Oro = %+v", got)
	}
}
