package chat

import (
	"fmt"
	"html"
	"log"
	"strings"
	"time"

	"github.com/OrlandoHdz/kubo/pkg/email"
)

// EscalationNotification es el contenido que se envía al equipo cuando un
// cliente solicita atención humana.
type EscalationNotification struct {
	ConversationID string
	UsuarioID      *int32
	UsuarioEmail   string
	ClienteID      *int32
	Nombre         string
	Email          string
	Telefono       string
	Motivo         string
	CreatedAt      time.Time
}

type EscalationNotifier interface {
	Notificar(EscalationNotification)
}

type emailEscalationNotifier struct {
	cfg        *email.Config
	recipients []string
	panelURL   string
}

func newEmailEscalationNotifier(cfg *email.Config) EscalationNotifier {
	if cfg == nil || cfg.Password == "" {
		return nil
	}
	recipients := splitEmails(env("CHAT_ESCALATION_EMAILS"))
	if len(recipients) == 0 {
		recipients = cfg.DefaultRecipients
	}
	if len(recipients) == 0 {
		return nil
	}
	return &emailEscalationNotifier{
		cfg:        cfg,
		recipients: recipients,
		panelURL:   envDefault("CHAT_ADMIN_PANEL_URL", "http://localhost:9000/#/app/dashboard"),
	}
}

func splitEmails(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			out = append(out, value)
		}
	}
	return out
}

func (n *emailEscalationNotifier) Notificar(data EscalationNotification) {
	if n == nil || n.cfg == nil {
		return
	}
	// No bloquea la respuesta del chat si el SMTP tarda.
	go func() {
		subject := "Nueva solicitud de asesor humano - Arsenal Welds"
		body := buildEscalationEmail(data, n.panelURL)
		if err := n.cfg.SendToMultiple(n.recipients, subject, body); err != nil {
			log.Printf(
				"chat: no se pudo enviar correo de escalamiento conversation=%s: %v",
				data.ConversationID, err,
			)
		}
	}()
}

func buildEscalationEmail(data EscalationNotification, panelURL string) string {
	value := func(s string) string {
		s = strings.TrimSpace(s)
		if s == "" {
			return "No proporcionado"
		}
		return html.EscapeString(s)
	}
	id := func(v *int32) string {
		if v == nil {
			return "No aplica"
		}
		return fmt.Sprintf("%d", *v)
	}
	if panelURL == "" {
		panelURL = "#"
	}

	return fmt.Sprintf(`
		<div style="font-family:Arial,sans-serif;color:#222;line-height:1.45;">
			<h2 style="color:#491212;margin:0 0 12px;">Nueva solicitud de asesor humano</h2>
			<p>Un cliente solicitó atención humana desde el chat de Kubo, el asesor virtual de Arsenal Welds.</p>
			<table style="border-collapse:collapse;width:100%%;max-width:680px;">
				<tr><td style="padding:6px;border:1px solid #ddd;"><strong>Conversación</strong></td><td style="padding:6px;border:1px solid #ddd;">%s</td></tr>
				<tr><td style="padding:6px;border:1px solid #ddd;"><strong>Cliente ID</strong></td><td style="padding:6px;border:1px solid #ddd;">%s</td></tr>
				<tr><td style="padding:6px;border:1px solid #ddd;"><strong>Usuario ID</strong></td><td style="padding:6px;border:1px solid #ddd;">%s</td></tr>
				<tr><td style="padding:6px;border:1px solid #ddd;"><strong>Correo de sesión</strong></td><td style="padding:6px;border:1px solid #ddd;">%s</td></tr>
				<tr><td style="padding:6px;border:1px solid #ddd;"><strong>Nombre de contacto</strong></td><td style="padding:6px;border:1px solid #ddd;">%s</td></tr>
				<tr><td style="padding:6px;border:1px solid #ddd;"><strong>Correo de contacto</strong></td><td style="padding:6px;border:1px solid #ddd;">%s</td></tr>
				<tr><td style="padding:6px;border:1px solid #ddd;"><strong>Teléfono</strong></td><td style="padding:6px;border:1px solid #ddd;">%s</td></tr>
				<tr><td style="padding:6px;border:1px solid #ddd;"><strong>Motivo</strong></td><td style="padding:6px;border:1px solid #ddd;">%s</td></tr>
				<tr><td style="padding:6px;border:1px solid #ddd;"><strong>Fecha</strong></td><td style="padding:6px;border:1px solid #ddd;">%s</td></tr>
			</table>
			<p style="margin-top:16px;">
				<a href="%s" style="background:#491212;color:#fff;padding:10px 16px;text-decoration:none;border-radius:6px;">
					Abrir Panel de Control
				</a>
			</p>
			<p style="color:#666;font-size:12px;">
				La conversación completa está disponible en el Panel de Control, sección
				Chat / Asesor Humano.
			</p>
		</div>`,
		value(data.ConversationID),
		id(data.ClienteID),
		id(data.UsuarioID),
		value(data.UsuarioEmail),
		value(data.Nombre),
		value(data.Email),
		value(data.Telefono),
		value(data.Motivo),
		data.CreatedAt.Format("2006-01-02 15:04:05"),
		html.EscapeString(panelURL),
	)
}
