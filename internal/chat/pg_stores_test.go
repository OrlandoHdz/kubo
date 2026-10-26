package chat

import (
	"context"
	"testing"
)

func TestPGPedidosQuerierRejectsCustomerMismatch(t *testing.T) {
	q := &pgPedidosQuerier{clienteID: 7}
	ctx := context.Background()

	if _, err := q.ObtenerPedido(ctx, "KUBO-1", 8); err == nil {
		t.Fatal("ObtenerPedido aceptó un cliente distinto al de la sesión")
	}
	if _, err := q.ListarItems(ctx, 1, 8); err == nil {
		t.Fatal("ListarItems aceptó un cliente distinto al de la sesión")
	}
	if _, err := q.ListarPedidos(ctx, 8, 5); err == nil {
		t.Fatal("ListarPedidos aceptó un cliente distinto al de la sesión")
	}
}
