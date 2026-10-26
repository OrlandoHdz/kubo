package chat

import (
	"context"
	"log"

	"github.com/OrlandoHdz/kubo/pkg/email"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewFromEnv construye el handler completo usando variables de entorno.
// Si falta ANTHROPIC_API_KEY devuelve ErrDisabled. Si CHAT_DB_URL no está
// configurado, el chat funciona en modo texto sin herramientas de catálogo.
func NewFromEnv(ctx context.Context, emailCfg *email.Config) (*Handler, error) {
	cfg := LoadConfig()
	if cfg.APIKey == "" {
		return nil, ErrDisabled
	}

	client := NewClient(cfg)

	var (
		readPool     *pgxpool.Pool
		catalogo     CatalogoStore
		pedidos      PedidosStore
		backorders   BackordersStore
		devoluciones DevolucionesStore
		recompensas  RecompensasStore
		contexto     ContextoStore
		auditPool    *pgxpool.Pool
		auditoria    AuditoriaStore
	)

	if cfg.ReadDBURL != "" {
		pool, err := newPool(ctx, cfg.ReadDBURL, 5, "kubo-chat-ro")
		if err != nil {
			return nil, err
		}
		readPool = pool
		store := newPGReadStore(pool)
		catalogo = store
		pedidos = store
		backorders = store
		devoluciones = store
		recompensas = store
		contexto = store
	} else {
		log.Printf("chat: CHAT_DB_URL no configurado; herramientas de catálogo, pedidos, backorders, devoluciones y recompensas deshabilitadas")
	}

	if cfg.AuditDBURL != "" {
		pool, err := newPool(ctx, cfg.AuditDBURL, 2, "kubo-chat-audit")
		if err != nil {
			log.Printf("chat: auditoría deshabilitada: %v", err)
		} else {
			auditPool = pool
			auditoria = newPGAuditStore(pool)
		}
	} else {
		log.Printf("chat: CHAT_AUDIT_DB_URL no configurado; auditoría deshabilitada")
	}

	registry := NewToolRegistry(ToolDeps{
		Catalogo:     catalogo,
		Pedidos:      pedidos,
		Backorders:   backorders,
		Devoluciones: devoluciones,
		Recompensas:  recompensas,
		Auditoria:    auditoria,
		Notifier:     newEmailEscalationNotifier(emailCfg),
	})
	agent := NewAgent(client, registry, SystemPrompt, cfg)
	handler := NewHandler(cfg, agent, contexto, auditoria)
	handler.readPool = readPool
	handler.auditPool = auditPool
	return handler, nil
}
