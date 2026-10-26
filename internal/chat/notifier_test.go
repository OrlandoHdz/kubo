package chat

import (
	"strings"
	"testing"
	"time"
)

func TestBuildEscalationEmailEscapesContact(t *testing.T) {
	body := buildEscalationEmail(EscalationNotification{
		ConversationID: "abc123",
		Nombre:         `<script>alert(1)</script>`,
		Email:          "ana@example.com",
		Telefono:       "5551234567",
		Motivo:         `<b>Necesito ayuda</b>`,
		CreatedAt:      time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC),
	}, "http://localhost:9000/app/dashboard")

	if strings.Contains(body, "<script>") {
		t.Fatalf("el correo contiene HTML sin escapar: %s", body)
	}
	if !strings.Contains(body, "&lt;script&gt;") {
		t.Fatalf("no se escapó el nombre: %s", body)
	}
	if !strings.Contains(body, "ana@example.com") || !strings.Contains(body, "abc123") {
		t.Fatalf("faltan datos del escalamiento: %s", body)
	}
}

func TestNewEmailEscalationNotifierDisabled(t *testing.T) {
	if notifier := newEmailEscalationNotifier(nil); notifier != nil {
		t.Fatal("sin configuración no debe crear notifier")
	}
}
