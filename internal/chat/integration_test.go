//go:build integration

package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// El SQL de este archivo se usa únicamente para crear y limpiar fixtures de
// prueba. El acceso a datos del chat en producción sigue siendo vía sqlc.
//
// TestPedidosAislamientoEntreClientes requiere una base de pruebas con las
// migraciones 30-34 aplicadas y dos URLs:
//
//	CHAT_TEST_DB_URL        -> usuario chat_ro (solo lectura)
//	CHAT_TEST_ADMIN_DB_URL  -> usuario con permisos de escritura para fixtures
//
// El test crea dos clientes, dos pedidos y verifica que las herramientas del
// chat nunca devuelvan el pedido de otro cliente.
func TestPedidosAislamientoEntreClientes(t *testing.T) {
	readURL := os.Getenv("CHAT_TEST_DB_URL")
	adminURL := os.Getenv("CHAT_TEST_ADMIN_DB_URL")
	if readURL == "" || adminURL == "" {
		t.Skip("definir CHAT_TEST_DB_URL y CHAT_TEST_ADMIN_DB_URL para ejecutar la integración")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	admin, err := pgxpool.New(ctx, adminURL)
	if err != nil {
		t.Fatalf("pool admin: %v", err)
	}
	t.Cleanup(func() { admin.Close() })

	read, err := pgxpool.New(ctx, readURL)
	if err != nil {
		t.Fatalf("pool chat_ro: %v", err)
	}
	t.Cleanup(func() { read.Close() })

	suffix := time.Now().UnixNano()

	var clienteA, clienteB, usuarioA, usuarioB, pedidoA, pedidoB int32
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if pedidoA != 0 || pedidoB != 0 {
			_, _ = admin.Exec(cleanupCtx, `DELETE FROM pedido_detalles WHERE pedido_id = ANY($1)`, []int32{pedidoA, pedidoB})
			_, _ = admin.Exec(cleanupCtx, `DELETE FROM pedidos WHERE id = ANY($1)`, []int32{pedidoA, pedidoB})
		}
		if usuarioA != 0 || usuarioB != 0 {
			_, _ = admin.Exec(cleanupCtx, `DELETE FROM usuarios WHERE id = ANY($1)`, []int32{usuarioA, usuarioB})
		}
		if clienteA != 0 || clienteB != 0 {
			_, _ = admin.Exec(cleanupCtx, `DELETE FROM clientes WHERE id = ANY($1)`, []int32{clienteA, clienteB})
		}
	})
	if err := admin.QueryRow(ctx, `
		INSERT INTO clientes (nombre_comercial, razon_social, rfc)
		VALUES ($1, $2, $3) RETURNING id`,
		fmt.Sprintf("Chat Test A %d", suffix),
		fmt.Sprintf("Chat Test A SA %d", suffix),
		fmt.Sprintf("A%012d", suffix%1000000000000),
	).Scan(&clienteA); err != nil {
		t.Fatalf("crear cliente A: %v", err)
	}
	if err := admin.QueryRow(ctx, `
		INSERT INTO clientes (nombre_comercial, razon_social, rfc)
		VALUES ($1, $2, $3) RETURNING id`,
		fmt.Sprintf("Chat Test B %d", suffix),
		fmt.Sprintf("Chat Test B SA %d", suffix),
		fmt.Sprintf("B%012d", (suffix+1)%1000000000000),
	).Scan(&clienteB); err != nil {
		t.Fatalf("crear cliente B: %v", err)
	}

	if err := admin.QueryRow(ctx, `
		INSERT INTO usuarios (cliente_id, email, password_hash, rol, is_active)
		VALUES ($1, $2, 'integration-test', 'cliente', true) RETURNING id`,
		clienteA, fmt.Sprintf("chat-a-%d@test.local", suffix),
	).Scan(&usuarioA); err != nil {
		t.Fatalf("crear usuario A: %v", err)
	}
	if err := admin.QueryRow(ctx, `
		INSERT INTO usuarios (cliente_id, email, password_hash, rol, is_active)
		VALUES ($1, $2, 'integration-test', 'cliente', true) RETURNING id`,
		clienteB, fmt.Sprintf("chat-b-%d@test.local", suffix),
	).Scan(&usuarioB); err != nil {
		t.Fatalf("crear usuario B: %v", err)
	}

	folioA := fmt.Sprintf("CHA%d", suffix%1_000_000_000)
	folioB := fmt.Sprintf("CHB%d", suffix%1_000_000_000)

	if err := admin.QueryRow(ctx, `
		INSERT INTO pedidos (folio, cliente_id, usuario_id, estado, metodo_pago, subtotal, iva, total_orden, fecha_pedido)
		VALUES ($1, $2, $3, 'Enviado', 'Tarjeta', 100, 16, 116, CURRENT_TIMESTAMP) RETURNING id`,
		folioA, clienteA, usuarioA,
	).Scan(&pedidoA); err != nil {
		t.Fatalf("crear pedido A: %v", err)
	}
	if err := admin.QueryRow(ctx, `
		INSERT INTO pedidos (folio, cliente_id, usuario_id, estado, metodo_pago, subtotal, iva, total_orden, fecha_pedido)
		VALUES ($1, $2, $3, 'Pendiente', 'Tarjeta', 200, 32, 232, CURRENT_TIMESTAMP) RETURNING id`,
		folioB, clienteB, usuarioB,
	).Scan(&pedidoB); err != nil {
		t.Fatalf("crear pedido B: %v", err)
	}

	if _, err := admin.Exec(ctx, `
		INSERT INTO pedido_detalles (pedido_id, variante_id, cantidad, precio_unitario_aplicado)
		VALUES ($1, NULL, 2, 50)`, pedidoA); err != nil {
		t.Fatalf("detalle pedido A: %v", err)
	}
	if _, err := admin.Exec(ctx, `
		INSERT INTO pedido_detalles (pedido_id, variante_id, cantidad, precio_unitario_aplicado)
		VALUES ($1, NULL, 4, 50)`, pedidoB); err != nil {
		t.Fatalf("detalle pedido B: %v", err)
	}

	store := newPGReadStore(read)

	contextoA, err := store.ObtenerContexto(ctx, usuarioA)
	if err != nil {
		t.Fatalf("contexto A: %v", err)
	}
	if contextoA.ClienteID == nil || *contextoA.ClienteID != clienteA {
		t.Fatalf("cliente A resuelto = %+v", contextoA)
	}

	registry := NewToolRegistry(ToolDeps{Catalogo: store, Pedidos: store, Backorders: store, Devoluciones: store, Recompensas: store})

	out, err := registry.Execute(
		ctx,
		"consultar_pedido",
		json.RawMessage(fmt.Sprintf(`{"numero_pedido":%q}`, folioA)),
		ToolContext{ClienteID: &clienteA},
	)
	if err != nil {
		t.Fatalf("consultar pedido A: %v", err)
	}
	content := out.Content.(map[string]any)
	if content["encontrado"] != true {
		t.Fatalf("pedido A no encontrado: %+v", content)
	}
	encoded, _ := json.Marshal(content)
	if !strings.Contains(string(encoded), folioA) {
		t.Fatalf("respuesta sin folio A: %s", encoded)
	}

	// El mismo cliente A intenta consultar el folio de B.
	out, err = registry.Execute(
		ctx,
		"consultar_pedido",
		json.RawMessage(fmt.Sprintf(`{"numero_pedido":%q}`, folioB)),
		ToolContext{ClienteID: &clienteA},
	)
	if err != nil {
		t.Fatalf("consultar pedido B: %v", err)
	}
	content = out.Content.(map[string]any)
	if content["encontrado"] != false {
		t.Fatalf("el cliente A pudo ver el pedido B: %+v", content)
	}
	encoded, _ = json.Marshal(content)
	if strings.Contains(string(encoded), folioB) {
		t.Fatalf("la respuesta filtró el folio B: %s", encoded)
	}

	// El listado de A solo debe contener su pedido.
	out, err = registry.Execute(
		ctx,
		"listar_pedidos_cliente",
		json.RawMessage(`{"limite":10}`),
		ToolContext{ClienteID: &clienteA},
	)
	if err != nil {
		t.Fatalf("listar pedidos A: %v", err)
	}
	listContent := out.Content.(map[string]any)
	pedidos, ok := listContent["pedidos"].([]PedidoResumen)
	if !ok {
		t.Fatalf("lista inesperada: %+v", listContent["pedidos"])
	}
	for _, pedido := range pedidos {
		if pedido.Folio == folioB {
			t.Fatalf("el listado de A incluyó el pedido B: %+v", pedidos)
		}
	}
	listEncoded, _ := json.Marshal(listContent)
	if strings.Contains(string(listEncoded), folioB) {
		t.Fatalf("el listado de A filtró el folio B: %s", listEncoded)
	}
}
