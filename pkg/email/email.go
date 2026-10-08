package email

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/smtp"
	"strings"
)

type OrderData struct {
	Folio               string
	Fecha               string
	MetodoPago          string
	Subtotal            string
	DescuentoAplicado   string
	DescuentoPorcentaje string
	Iva                 string
	Total               string
	ClienteName         string
	ClienteID           int32
	Status              string
	NotasAdmin          string
	Guia                string
	Items               []OrderItemData
}

type OrderItemData struct {
	SKU             string
	Descripcion     string
	Cantidad        int32
	Precio          string
	Importe         string
	// Campos opcionales para mostrar ahorro por nivel de distribuidor
	PrecioLista     string // precio de lista para comparación
	AhorroNivel     string // monto ahorrado por nivel en este item
	NivelNombre     string // nombre del nivel aplicado (ej. "Platino")
}

type Config struct {
	Host              string   `yaml:"smtp_host"`
	Port              string   `yaml:"smtp_port"`
	Username          string   `yaml:"username"`
	Password          string   `yaml:"password"`
	FromName          string   `yaml:"from_name"`
	DefaultRecipients []string `yaml:"default_recipients"`
	TemplatesDir      string   `yaml:"templates_dir"`
}

type loginAuth struct {
	username, password string
	step               int
}

func LoginAuth(username, password string) smtp.Auth {
	return &loginAuth{username: username, password: password}
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", nil, nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}
	a.step++
	if a.step == 1 {
		return []byte(a.username), nil
	}
	return []byte(a.password), nil
}

func (c *Config) buildMsg(to, cc, subject, bodyHTML string) []byte {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("From: %s <%s>\r\n", c.FromName, c.Username))
	b.WriteString(fmt.Sprintf("To: %s\r\n", to))
	if cc != "" {
		b.WriteString(fmt.Sprintf("Cc: %s\r\n", cc))
	}
	b.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	b.WriteString("\r\n")
	b.WriteString(bodyHTML)
	return []byte(b.String())
}

func (c *Config) Send(to, subject, bodyHTML string) error {
	msg := c.buildMsg(to, "", subject, bodyHTML)
	auth := LoginAuth(c.Username, c.Password)
	addr := c.Host + ":" + c.Port

	if c.Port == "465" {
		return c.sendSSL(addr, auth, to, msg)
	}
	return smtp.SendMail(addr, auth, c.Username, []string{to}, msg)
}

func (c *Config) SendToMultiple(to []string, subject, bodyHTML string) error {
	if len(to) == 0 {
		return fmt.Errorf("no recipients")
	}
	auth := LoginAuth(c.Username, c.Password)
	addr := c.Host + ":" + c.Port
	msg := c.buildMsg(to[0], strings.Join(to[1:], ", "), subject, bodyHTML)

	allRecipients := to
	if c.Port == "465" {
		return c.sendSSLToMultiple(addr, auth, allRecipients, msg)
	}
	return smtp.SendMail(addr, auth, c.Username, allRecipients, msg)
}

func (c *Config) sendSSL(addr string, auth smtp.Auth, to string, msg []byte) error {
	tlsCfg := &tls.Config{ServerName: c.Host}
	conn, err := tls.Dial("tcp", addr, tlsCfg)
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, c.Host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("auth: %w", err)
	}
	if err := client.Mail(c.Username); err != nil {
		return fmt.Errorf("mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("rcpt to: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close: %w", err)
	}
	return client.Quit()
}

func (c *Config) sendSSLToMultiple(addr string, auth smtp.Auth, to []string, msg []byte) error {
	tlsCfg := &tls.Config{ServerName: c.Host}
	conn, err := tls.Dial("tcp", addr, tlsCfg)
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, c.Host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("auth: %w", err)
	}
	if err := client.Mail(c.Username); err != nil {
		return fmt.Errorf("mail from: %w", err)
	}
	for _, rcpt := range to {
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("rcpt to %s: %w", rcpt, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close: %w", err)
	}
	return client.Quit()
}

func (c *Config) SendOrderNotification(order OrderData, clientEmail string) {
	if c == nil {
		log.Println("Email: configuración de correo es nil, omitiendo notificación")
		return
	}
	if c.Password == "" {
		log.Println("Email: password de correo vacío, omitiendo notificación")
		return
	}

	subject := fmt.Sprintf("Nuevo Pedido — %s", order.Folio)
	body := buildOrderHTML(order)

	recipients := []string{clientEmail}
	recipients = append(recipients, c.DefaultRecipients...)

	log.Printf("Email: enviando notificación a %d destinatarios (primero: %s)", len(recipients), clientEmail)

	if err := c.SendToMultiple(recipients, subject, body); err != nil {
		log.Printf("Email: error al enviar notificación de pedido %s: %v", order.Folio, err)
		return
	}
	log.Printf("Email: notificación de pedido %s enviada correctamente a %s y %d copias", order.Folio, clientEmail, len(c.DefaultRecipients))
}

type BackorderData struct {
	Folio       string
	Fecha       string
	MetodoPago  string
	Subtotal    string
	Iva         string
	Total       string
	ClienteName string
	ClienteID   int32
	Items       []OrderItemData
}

func (c *Config) SendBackorderNotification(bo BackorderData, clientEmail string) {
	if c == nil {
		log.Println("Email: configuración de correo es nil, omitiendo notificación")
		return
	}
	if c.Password == "" {
		log.Println("Email: password de correo vacío, omitiendo notificación")
		return
	}

	subject := fmt.Sprintf("Nuevo Backorder — %s", bo.Folio)
	body := buildBackorderHTML(bo)

	recipients := []string{clientEmail}
	recipients = append(recipients, c.DefaultRecipients...)

	log.Printf("Email: enviando notificación de backorder a %d destinatarios (primero: %s)", len(recipients), clientEmail)

	if err := c.SendToMultiple(recipients, subject, body); err != nil {
		log.Printf("Email: error al enviar notificación de backorder %s: %v", bo.Folio, err)
		return
	}
	log.Printf("Email: notificación de backorder %s enviada correctamente a %s y %d copias", bo.Folio, clientEmail, len(c.DefaultRecipients))
}

func buildBackorderHTML(bo BackorderData) string {
	itemsRows := ""
	for _, item := range bo.Items {
		itemsRows += fmt.Sprintf(`<tr>
              <td style="padding: 10px; border-bottom: 1px solid #e0e0e0; color: #333;">%s</td>
              <td style="padding: 10px; border-bottom: 1px solid #e0e0e0; color: #333;">%s</td>
              <td style="padding: 10px; border-bottom: 1px solid #e0e0e0; color: #333; text-align: center;">%d</td>
              <td style="padding: 10px; border-bottom: 1px solid #e0e0e0; color: #333; text-align: right;">$ %s</td>
              <td style="padding: 10px; border-bottom: 1px solid #e0e0e0; color: #333; text-align: right;">$ %s</td>
            </tr>`, item.SKU, item.Descripcion, item.Cantidad, item.Precio, item.Importe)
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: Arial, sans-serif; margin: 0; padding: 0; background: #f5f5f5;">
  <table width="100%%" cellpadding="0" cellspacing="0" style="background: #f5f5f5; padding: 20px;">
    <tr>
      <td align="left">
        <table width="70%%" cellpadding="0" cellspacing="0" style="background: #ffffff; border-radius: 8px; overflow: hidden;">
          <tr>
            <td style="background: linear-gradient(135deg, #c9540e, #f57c1a); padding: 20px; text-align: center;">
              <img src="https://kubo-producto.oss-us-east-1.aliyuncs.com/logos/arsenal_logo1t.png" alt="Arsenal Welds" style="max-height: 60px; margin-bottom: 10px;">
              <h1 style="color: #ffffff; margin: 10px 0 0; font-size: 22px;">¡Nuevo Backorder Registrado!</h1>
            </td>
          </tr>
          <tr>
            <td style="padding: 30px;">
              <p style="font-size: 16px; color: #333;">Estimado cliente,</p>
              <p style="font-size: 15px; color: #555; line-height: 1.6;">
                Hemos recibido su backorder <strong>%s</strong> y se encuentra en proceso de revisión.<br>
                A continuación se detallan los artículos solicitados en espera de stock:
              </p>

              <table width="100%%" cellpadding="0" cellspacing="0" style="margin: 20px 0;">
                <tr>
                  <td style="padding: 8px 10px; background: #f8f9fa; font-weight: bold; color: #333; border-bottom: 2px solid #c9540e;">Folio</td>
                  <td style="padding: 8px 10px; background: #f8f9fa; color: #c9540e; border-bottom: 2px solid #c9540e;">%s</td>
                </tr>
                <tr>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0; font-weight: bold; color: #333;">Fecha</td>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0; color: #555;">%s</td>
                </tr>
                <tr>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0; font-weight: bold; color: #333;">Método de Pago</td>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0; color: #555;">%s</td>
                </tr>
              </table>

              <h3 style="color: #c9540e; margin: 20px 0 10px;">Detalle del Backorder</h3>
              <table width="100%%" cellpadding="0" cellspacing="0" style="border-collapse: collapse;">
                <thead>
                  <tr style="background: #c9540e; color: #ffffff;">
                    <th style="padding: 10px; text-align: left;">CÓDIGO</th>
                    <th style="padding: 10px; text-align: left;">Producto</th>
                    <th style="padding: 10px; text-align: center;">Cant.</th>
                    <th style="padding: 10px; text-align: right;">Precio Unit.</th>
                    <th style="padding: 10px; text-align: right;">Importe</th>
                  </tr>
                </thead>
                <tbody>
                  %s
                </tbody>
                <tfoot>
                  <tr>
                    <td colspan="3" style="padding: 10px; border-bottom: 2px solid #c9540e; text-align: right; font-weight: bold; color: #333;">Subtotal</td>
                    <td colspan="2" style="padding: 10px; border-bottom: 2px solid #c9540e; text-align: right; color: #333;">$ %s</td>
                  </tr>
                  <tr>
                    <td colspan="3" style="padding: 10px; text-align: right; font-weight: bold; color: #333;">IVA</td>
                    <td colspan="2" style="padding: 10px; text-align: right; color: #333;">$ %s</td>
                  </tr>
                  <tr>
                    <td colspan="3" style="padding: 10px; text-align: right; font-weight: bold; font-size: 18px; color: #c9540e;">Total</td>
                    <td colspan="2" style="padding: 10px; text-align: right; font-weight: bold; font-size: 18px; color: #c9540e;">$ %s</td>
                  </tr>
                </tfoot>
              </table>

              <p style="font-size: 15px; color: #555; line-height: 1.6; margin-top: 30px;">
                Le notificaremos cuando los productos tengan stock disponible.
              </p>
              <p style="font-size: 15px; color: #555; line-height: 1.6;">
                Atentamente,<br>
                <strong>Arsenal Welds</strong>
              </p>
              <p style="font-size: 14px; color: #555; margin-top: 20px;">
                <a href="https://www.arsenalwelds.com" style="color: #491212; text-decoration: none; font-weight: bold;">www.arsenalwelds.com</a>
              </p>
            </td>
          </tr>
          <tr>
            <td style="background: #f0f0f0; padding: 15px; text-align: center; font-size: 12px; color: #999;">
              Arsenal Welds — ventas@arsenalwelds.com
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`, bo.Folio, bo.Folio, bo.Fecha, bo.MetodoPago, itemsRows, bo.Subtotal, bo.Iva, bo.Total)
}

func buildOrderHTML(o OrderData) string {
	itemsRows := ""
	for _, item := range o.Items {
		// Si el item tiene información de ahorro por nivel, mostrar columna adicional
		nivelInfo := ""
		if item.AhorroNivel != "" && item.AhorroNivel != "0.00" {
			nivelInfo = fmt.Sprintf(`<br><span style="color: #e65100; font-size: 11px; font-weight: bold;">🏆 Ahorro Nivel %s: -$%s c/u</span>`, item.NivelNombre, item.AhorroNivel)
		}

		itemsRows += fmt.Sprintf(`<tr>
              <td style="padding: 10px; border-bottom: 1px solid #e0e0e0; color: #333;">%s%s</td>
              <td style="padding: 10px; border-bottom: 1px solid #e0e0e0; color: #333;">%s</td>
              <td style="padding: 10px; border-bottom: 1px solid #e0e0e0; color: #333; text-align: center;">%d</td>
              <td style="padding: 10px; border-bottom: 1px solid #e0e0e0; color: #333; text-align: right;">$ %s</td>
              <td style="padding: 10px; border-bottom: 1px solid #e0e0e0; color: #333; text-align: right;">$ %s</td>
            </tr>`, item.SKU, nivelInfo, item.Descripcion, item.Cantidad, item.Precio, item.Importe)
	}

	descuentoRow := ""
	if o.DescuentoAplicado != "" && o.DescuentoAplicado != "0.00" {
		descuentoRow = fmt.Sprintf(`
                  <tr>
                    <td colspan="3" style="padding: 10px; text-align: right; font-weight: bold; color: #c62828;">Descuento (%s%%)</td>
                    <td colspan="2" style="padding: 10px; text-align: right; color: #c62828;">- $ %s</td>
                  </tr>`, o.DescuentoPorcentaje, o.DescuentoAplicado)
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: Arial, sans-serif; margin: 0; padding: 0; background: #f5f5f5;">
  <table width="100%%" cellpadding="0" cellspacing="0" style="background: #f5f5f5; padding: 20px;">
    <tr>
      <td align="left">
        <table width="70%%" cellpadding="0" cellspacing="0" style="background: #ffffff; border-radius: 8px; overflow: hidden;">
          <tr>
            <td style="background: linear-gradient(135deg, #491212, #7a1a1a); padding: 20px; text-align: center;">
              <img src="https://kubo-producto.oss-us-east-1.aliyuncs.com/logos/arsenal_logo1t.png" alt="Arsenal Welds" style="max-height: 60px; margin-bottom: 10px;">
              <h1 style="color: #ffffff; margin: 10px 0 0; font-size: 22px;">¡Nuevo Pedido Registrado!</h1>
            </td>
          </tr>
          <tr>
            <td style="padding: 30px;">
              <p style="font-size: 16px; color: #333;">Estimado cliente,</p>
              <p style="font-size: 15px; color: #555; line-height: 1.6;">
                Hemos recibido su pedido <strong>%s</strong> y se encuentra en proceso de revisión.<br>
                A continuación se detallan los artículos solicitados:
              </p>

              <table width="100%%" cellpadding="0" cellspacing="0" style="margin: 20px 0;">
                <tr>
                  <td style="padding: 8px 10px; background: #f8f9fa; font-weight: bold; color: #333; border-bottom: 2px solid #491212;">Folio</td>
                  <td style="padding: 8px 10px; background: #f8f9fa; color: #491212; border-bottom: 2px solid #491212;">%s</td>
                </tr>
                <tr>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0; font-weight: bold; color: #333;">Fecha</td>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0; color: #555;">%s</td>
                </tr>
                <tr>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0; font-weight: bold; color: #333;">Método de Pago</td>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0; color: #555;">%s</td>
                </tr>
              </table>

              <h3 style="color: #491212; margin: 20px 0 10px;">Detalle del Pedido</h3>
              <table width="100%%" cellpadding="0" cellspacing="0" style="border-collapse: collapse;">
                <thead>
                  <tr style="background: #491212; color: #ffffff;">
                    <th style="padding: 10px; text-align: left;">CÓDIGO</th>
                    <th style="padding: 10px; text-align: left;">Producto</th>
                    <th style="padding: 10px; text-align: center;">Cant.</th>
                    <th style="padding: 10px; text-align: right;">Precio Unit.</th>
                    <th style="padding: 10px; text-align: right;">Importe</th>
                  </tr>
                </thead>
                <tbody>
                  %s
                </tbody>
                <tfoot>
                  <tr>
                    <td colspan="3" style="padding: 10px; border-bottom: 1px solid #e0e0e0; text-align: right; font-weight: bold; color: #333;">Subtotal</td>
                    <td colspan="2" style="padding: 10px; border-bottom: 1px solid #e0e0e0; text-align: right; color: #333;">$ %s</td>
                  </tr>%s
                  <tr>
                    <td colspan="3" style="padding: 10px; text-align: right; font-weight: bold; color: #333;">IVA</td>
                    <td colspan="2" style="padding: 10px; text-align: right; color: #333;">$ %s</td>
                  </tr>
                  <tr>
                    <td colspan="3" style="padding: 10px; text-align: right; font-weight: bold; font-size: 18px; color: #491212;">Total</td>
                    <td colspan="2" style="padding: 10px; text-align: right; font-weight: bold; font-size: 18px; color: #491212;">$ %s</td>
                  </tr>
                </tfoot>
              </table>

              <p style="font-size: 15px; color: #555; line-height: 1.6; margin-top: 30px;">
                Le notificaremos cuando su pedido sea procesado y enviado.
              </p>
              <p style="font-size: 15px; color: #555; line-height: 1.6;">
                Atentamente,<br>
                <strong>Arsenal Welds</strong>
              </p>
              <p style="font-size: 14px; color: #555; margin-top: 20px;">
                <a href="https://www.arsenalwelds.com" style="color: #491212; text-decoration: none; font-weight: bold;">www.arsenalwelds.com</a>
              </p>
            </td>
          </tr>
          <tr>
            <td style="background: #f0f0f0; padding: 15px; text-align: center; font-size: 12px; color: #999;">
              Arsenal Welds — ventas@arsenalwelds.com
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`, o.Folio, o.Folio, o.Fecha, o.MetodoPago, itemsRows, o.Subtotal, descuentoRow, o.Iva, o.Total)
}

func (c *Config) SendOrderStatusNotification(order OrderData, clientEmail string) {
	if c == nil {
		log.Println("Email: configuración de correo es nil, omitiendo notificación")
		return
	}
	if c.Password == "" {
		log.Println("Email: password de correo vacío, omitiendo notificación")
		return
	}

	subject := fmt.Sprintf("Pedido %s — %s", order.Status, order.Folio)
	body := buildOrderStatusHTML(order)

	recipients := []string{clientEmail}
	recipients = append(recipients, c.DefaultRecipients...)

	log.Printf("Email: enviando notificación de estado a %d destinatarios (primero: %s)", len(recipients), clientEmail)

	if err := c.SendToMultiple(recipients, subject, body); err != nil {
		log.Printf("Email: error al enviar notificación de estado del pedido %s: %v", order.Folio, err)
		return
	}
	log.Printf("Email: notificación de estado del pedido %s enviada correctamente a %s y %d copias", order.Folio, clientEmail, len(c.DefaultRecipients))
}

func buildOrderStatusHTML(o OrderData) string {
	itemsRows := ""
	for _, item := range o.Items {
		// Mostrar ahorro por nivel en cada item si aplica
		nivelInfo := ""
		if item.AhorroNivel != "" && item.AhorroNivel != "0.00" {
			nivelInfo = fmt.Sprintf(`<br><span style="color: #e65100; font-size: 11px; font-weight: bold;">🏆 Ahorro Nivel %s: -$%s c/u</span>`, item.NivelNombre, item.AhorroNivel)
		}

		itemsRows += fmt.Sprintf(`<tr>
              <td style="padding: 10px; border-bottom: 1px solid #e0e0e0; color: #333;">%s%s</td>
              <td style="padding: 10px; border-bottom: 1px solid #e0e0e0; color: #333;">%s</td>
              <td style="padding: 10px; border-bottom: 1px solid #e0e0e0; color: #333; text-align: center;">%d</td>
              <td style="padding: 10px; border-bottom: 1px solid #e0e0e0; color: #333; text-align: right;">$ %s</td>
              <td style="padding: 10px; border-bottom: 1px solid #e0e0e0; color: #333; text-align: right;">$ %s</td>
            </tr>`, item.SKU, nivelInfo, item.Descripcion, item.Cantidad, item.Precio, item.Importe)
	}

	statusColor := "#1565c0"
	statusBadgeBg := "#e3f2fd"
	switch o.Status {
	case "Entregado":
		statusColor = "#2e7d32"
		statusBadgeBg = "#e8f5e9"
	case "Cancelado":
		statusColor = "#c62828"
		statusBadgeBg = "#ffebee"
	}

	notasSection := ""
	if o.NotasAdmin != "" {
		notasHTML := strings.ReplaceAll(o.NotasAdmin, "\n", "<br>")
		notasSection = fmt.Sprintf(`
              <div style="background: #fff8e1; border: 1px solid #ffe082; border-radius: 6px; padding: 15px; margin: 20px 0;">
                <div style="font-weight: bold; color: #e65100; margin-bottom: 6px;">Nota del Administrador</div>
                <div style="color: #555; font-size: 14px; line-height: 1.5;">%s</div>
              </div>`, notasHTML)
	}

	// Fila de descuento (solo si hay descuento aplicado)
	descuentoRow := ""
	if o.DescuentoAplicado != "" && o.DescuentoAplicado != "0.00" {
		descuentoRow = fmt.Sprintf(`
                  <tr>
                    <td colspan="3" style="padding: 10px; text-align: right; font-weight: bold; color: #c62828;">Descuento (%s%%)</td>
                    <td colspan="2" style="padding: 10px; text-align: right; color: #c62828;">- $ %s</td>
                  </tr>`, o.DescuentoPorcentaje, o.DescuentoAplicado)
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: Arial, sans-serif; margin: 0; padding: 0; background: #f5f5f5;">
  <table width="100%%" cellpadding="0" cellspacing="0" style="background: #f5f5f5; padding: 20px;">
    <tr>
      <td align="left">
        <table width="70%%" cellpadding="0" cellspacing="0" style="background: #ffffff; border-radius: 8px; overflow: hidden;">
          <tr>
            <td style="background: linear-gradient(135deg, #491212, #7a1a1a); padding: 25px; text-align: center;">
              <img src="https://kubo-producto.oss-us-east-1.aliyuncs.com/logos/arsenal_logo1t.png" alt="Arsenal Welds" style="max-height: 55px; margin-bottom: 10px;">
              <h1 style="color: #ffffff; margin: 10px 0 0; font-size: 22px;">Actualización de Pedido</h1>
            </td>
          </tr>
          <tr>
            <td style="padding: 30px;">
              <p style="font-size: 16px; color: #333;">Estimado cliente,</p>
              <p style="font-size: 15px; color: #555; line-height: 1.6;">
                Le informamos que su pedido <strong>%s</strong> ha sido actualizado al siguiente estado:
              </p>

              <table width="100%%" cellpadding="0" cellspacing="0" style="margin: 20px 0;">
                <tr>
                  <td style="padding: 8px 10px; background: #f8f9fa; font-weight: bold; color: #333; border-bottom: 2px solid #491212;">Folio</td>
                  <td style="padding: 8px 10px; background: #f8f9fa; color: #491212; border-bottom: 2px solid #491212;">%s</td>
                </tr>
                <tr>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0; font-weight: bold; color: #333;">Fecha</td>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0; color: #555;">%s</td>
                </tr>
                <tr>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0; font-weight: bold; color: #333;">Método de Pago</td>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0; color: #555;">%s</td>
                </tr>
                <tr>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0; font-weight: bold; color: #333;">Estado</td>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0;"><span style="display: inline-block; background: %s; color: %s; padding: 4px 14px; border-radius: 4px; font-weight: bold; font-size: 14px;">%s</span></td>
                </tr>
                <tr>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0; font-weight: bold; color: #333;">Guía</td>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0; color: #555;">%s</td>
                </tr>
              </table>

              <h3 style="color: #491212; margin: 20px 0 10px;">Detalle del Pedido</h3>
              <table width="100%%" cellpadding="0" cellspacing="0" style="border-collapse: collapse;">
                <thead>
                  <tr style="background: #491212; color: #ffffff;">
                    <th style="padding: 10px; text-align: left;">CÓDIGO</th>
                    <th style="padding: 10px; text-align: left;">Producto</th>
                    <th style="padding: 10px; text-align: center;">Cant.</th>
                    <th style="padding: 10px; text-align: right;">Precio Unit.</th>
                    <th style="padding: 10px; text-align: right;">Importe</th>
                  </tr>
                </thead>
                <tbody>
                  %s
                </tbody>
                <tfoot>
                  <tr>
                    <td colspan="3" style="padding: 10px; border-bottom: 1px solid #e0e0e0; text-align: right; font-weight: bold; color: #333;">Subtotal</td>
                    <td colspan="2" style="padding: 10px; border-bottom: 1px solid #e0e0e0; text-align: right; color: #333;">$ %s</td>
                  </tr>%s
                  <tr>
                    <td colspan="3" style="padding: 10px; text-align: right; font-weight: bold; color: #333;">IVA</td>
                    <td colspan="2" style="padding: 10px; text-align: right; color: #333;">$ %s</td>
                  </tr>
                  <tr>
                    <td colspan="3" style="padding: 10px; text-align: right; font-weight: bold; font-size: 18px; color: #491212;">Total</td>
                    <td colspan="2" style="padding: 10px; text-align: right; font-weight: bold; font-size: 18px; color: #491212;">$ %s</td>
                  </tr>
                </tfoot>
              </table>

              %s

              <p style="font-size: 15px; color: #555; line-height: 1.6; margin-top: 20px;">
                Atentamente,<br>
                <strong>Arsenal Welds</strong>
              </p>
              <p style="font-size: 14px; color: #555; margin-top: 20px;">
                <a href="https://www.arsenalwelds.com" style="color: #491212; text-decoration: none; font-weight: bold;">www.arsenalwelds.com</a>
              </p>
            </td>
          </tr>
          <tr>
            <td style="background: #f0f0f0; padding: 15px; text-align: center; font-size: 12px; color: #999;">
              Arsenal Welds — ventas@arsenalwelds.com
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`, o.Folio, o.Folio, o.Fecha, o.MetodoPago, statusBadgeBg, statusColor, o.Status, o.Guia, itemsRows, o.Subtotal, descuentoRow, o.Iva, o.Total, notasSection)
}

type PagoFacturaItemData struct {
	NoFactura   string
	MontoPagado string
}

type PagoFacturaData struct {
	ClienteName string
	Fecha       string
	MetodoPago  string
	Terminacion string
	Total       string
	Facturas    []PagoFacturaItemData
}

func (c *Config) SendPagoFacturaNotification(pf PagoFacturaData, clientEmail string) {
	if c == nil || c.Password == "" {
		return
	}

	subject := "Confirmación de Pago de Facturas"
	body := buildPagoFacturaHTML(pf)

	recipients := []string{clientEmail}
	recipients = append(recipients, c.DefaultRecipients...)

	log.Printf("Email: enviando notificación de pago de facturas a %d destinatarios (primero: %s)", len(recipients), clientEmail)
	if err := c.SendToMultiple(recipients, subject, body); err != nil {
		log.Printf("Email: error al enviar notificación de pago de facturas: %v", err)
		return
	}
	log.Printf("Email: notificación de pago de facturas enviada correctamente a %s y %d copias", clientEmail, len(c.DefaultRecipients))
}

type DevolucionData struct {
	Folio             string
	Tipo              string
	PedidoFolio       string
	NumerosParte      string
	Cantidades        string
	NotaCliente       string
	NotaAdministrador string
	Estatus           string
	ClienteName       string
}

func (c *Config) SendDevolucionNotification(dv DevolucionData, clientEmail string) {
	if c == nil || c.Password == "" {
		return
	}

	subject := fmt.Sprintf("Solicitud de %s Recibida — %s", dv.Tipo, dv.Folio)
	body := buildDevolucionHTML(dv)

	recipients := []string{clientEmail}
	recipients = append(recipients, c.DefaultRecipients...)

	log.Printf("Email: enviando notificación de %s a %d destinatarios", dv.Tipo, len(recipients))
	if err := c.SendToMultiple(recipients, subject, body); err != nil {
		log.Printf("Email: error al enviar notificación de %s %s: %v", dv.Tipo, dv.Folio, err)
	}
}

type SolicitudDistribuidorData struct {
	NombreComercial string
	NombreContacto  string
	Tipo            string
	Folio           string
	RFC             string
	Fecha           string
}

func (c *Config) SendSolicitudDistribuidorNotification(sd SolicitudDistribuidorData, clientEmail string) {
	if c == nil || c.Password == "" {
		log.Println("Email: configuración de correo incompleta, omitiendo notificación de solicitud de registro")
		return
	}

	esDistribuidor := sd.Tipo == "Distribuidor" || sd.Tipo == "Ambos"
	subject := "Solicitud de Registro Recibida"
	if esDistribuidor {
		subject = "Solicitud de Distribuidor Recibida"
	}
	body := buildSolicitudDistribuidorHTML(sd, esDistribuidor)

	recipients := []string{clientEmail}
	recipients = append(recipients, c.DefaultRecipients...)

	log.Printf("Email: enviando notificación de solicitud de registro a %d destinatarios (primero: %s)", len(recipients), clientEmail)
	if err := c.SendToMultiple(recipients, subject, body); err != nil {
		log.Printf("Email: error al enviar notificación de solicitud de registro a %s: %v", clientEmail, err)
		return
	}
	log.Printf("Email: notificación de solicitud de registro enviada correctamente a %s y %d copias", clientEmail, len(c.DefaultRecipients))
}

type SolicitudEstatusData struct {
	NombreComercial string
	Tipo            string
	Folio           string
	RFC             string
	Fecha           string
	Estatus         string
	Observacion     string
}

func (c *Config) SendSolicitudEstatusNotification(se SolicitudEstatusData, clientEmail string) {
	if c == nil || c.Password == "" {
		log.Println("Email: configuración de correo incompleta, omitiendo notificación de estatus de solicitud")
		return
	}

	subject := fmt.Sprintf("Solicitud de Registro %s", se.Estatus)
	body := buildSolicitudEstatusHTML(se)

	recipients := []string{clientEmail}
	recipients = append(recipients, c.DefaultRecipients...)

	log.Printf("Email: enviando notificación de estatus de solicitud a %d destinatarios (primero: %s)", len(recipients), clientEmail)
	if err := c.SendToMultiple(recipients, subject, body); err != nil {
		log.Printf("Email: error al enviar notificación de estatus de solicitud a %s: %v", clientEmail, err)
		return
	}
	log.Printf("Email: notificación de estatus de solicitud enviada correctamente a %s y %d copias", clientEmail, len(c.DefaultRecipients))
}

func (c *Config) SendDevolucionStatusNotification(dv DevolucionData, clientEmail string) {
	if c == nil || c.Password == "" {
		return
	}

	subject := fmt.Sprintf("%s %s — %s", dv.Tipo, dv.Estatus, dv.Folio)
	body := buildDevolucionStatusHTML(dv)

	recipients := []string{clientEmail}
	recipients = append(recipients, c.DefaultRecipients...)

	log.Printf("Email: enviando notificación de estatus de %s a %d destinatarios", dv.Tipo, len(recipients))
	if err := c.SendToMultiple(recipients, subject, body); err != nil {
		log.Printf("Email: error al enviar notificación de estatus de %s %s: %v", dv.Tipo, dv.Folio, err)
	}
}

func buildDevolucionHTML(dv DevolucionData) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: Arial, sans-serif; margin: 0; padding: 0; background: #f5f5f5;">
  <table width="100%%" cellpadding="0" cellspacing="0" style="background: #f5f5f5; padding: 20px;">
    <tr>
      <td align="left">
        <table width="70%%" cellpadding="0" cellspacing="0" style="background: #ffffff; border-radius: 8px; overflow: hidden;">
          <tr>
            <td style="background: linear-gradient(135deg, #0d47a1, #1976d2); padding: 25px; text-align: center;">
              <img src="https://kubo-producto.oss-us-east-1.aliyuncs.com/logos/arsenal_logo1t.png" alt="Arsenal Welds" style="max-height: 55px; margin-bottom: 10px;">
              <h1 style="color: #ffffff; margin: 10px 0 0; font-size: 22px;">Solicitud de %s Recibida</h1>
            </td>
          </tr>
          <tr>
            <td style="padding: 30px;">
              <p style="font-size: 16px; color: #333;">Estimado cliente,</p>
              <p style="font-size: 15px; color: #555; line-height: 1.6;">
                Hemos recibido su solicitud de <strong>%s</strong> con folio <strong>%s</strong>.<br>
                Nuestro equipo revisará y validará la información proporcionada y le estaremos notificando el resultado.
              </p>

              <table width="100%%" cellpadding="0" cellspacing="0" style="margin: 20px 0;">
                <tr>
                  <td style="padding: 8px 10px; background: #f8f9fa; font-weight: bold; color: #333; border-bottom: 2px solid #1976d2;">Folio</td>
                  <td style="padding: 8px 10px; background: #f8f9fa; color: #1976d2; border-bottom: 2px solid #1976d2;">%s</td>
                </tr>
                <tr>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0; font-weight: bold; color: #333;">Tipo</td>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0; color: #555;">%s</td>
                </tr>
                <tr>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0; font-weight: bold; color: #333;">Pedido</td>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0; color: #555;">%s</td>
                </tr>
                <tr>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0; font-weight: bold; color: #333;">Estatus</td>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0;"><span style="display: inline-block; background: #e3f2fd; color: #1565c0; padding: 4px 14px; border-radius: 4px; font-weight: bold; font-size: 14px;">%s</span></td>
                </tr>
              </table>

              <div style="background: #e3f2fd; border: 1px solid #90caf9; border-radius: 6px; padding: 15px; margin: 20px 0;">
                <div style="font-weight: bold; color: #1565c0; margin-bottom: 6px;">Nota del Cliente</div>
                <div style="color: #555; font-size: 14px; line-height: 1.5;">%s</div>
              </div>

              <p style="font-size: 15px; color: #555; line-height: 1.6;">
                En caso de requerir información adicional, nuestro equipo se comunicará con usted.
              </p>
              <p style="font-size: 15px; color: #555; line-height: 1.6;">
                Atentamente,<br>
                <strong>Arsenal Welds</strong>
              </p>
              <p style="font-size: 14px; color: #555; margin-top: 20px;">
                <a href="https://www.arsenalwelds.com" style="color: #1976d2; text-decoration: none; font-weight: bold;">www.arsenalwelds.com</a>
              </p>
            </td>
          </tr>
          <tr>
            <td style="background: #f0f0f0; padding: 15px; text-align: center; font-size: 12px; color: #999;">
              Arsenal Welds — ventas@arsenalwelds.com
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`, dv.Tipo, dv.Tipo, dv.Folio, dv.Folio, dv.Tipo, dv.PedidoFolio, dv.Estatus, strings.ReplaceAll(dv.NotaCliente, "\n", "<br>"))
}

func buildDevolucionStatusHTML(dv DevolucionData) string {
	statusColor := "#1565c0"
	statusBadgeBg := "#e3f2fd"
	switch dv.Estatus {
	case "Aprobada":
		statusColor = "#2e7d32"
		statusBadgeBg = "#e8f5e9"
	case "Rechazada":
		statusColor = "#c62828"
		statusBadgeBg = "#ffebee"
	case "Cancelada":
		statusColor = "#616161"
		statusBadgeBg = "#eeeeee"
	}

	notaClienteHTML := strings.ReplaceAll(dv.NotaCliente, "\n", "<br>")
	notaAdminHTML := strings.ReplaceAll(dv.NotaAdministrador, "\n", "<br>")

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: Arial, sans-serif; margin: 0; padding: 0; background: #f5f5f5;">
  <table width="100%%" cellpadding="0" cellspacing="0" style="background: #f5f5f5; padding: 20px;">
    <tr>
      <td align="left">
        <table width="70%%" cellpadding="0" cellspacing="0" style="background: #ffffff; border-radius: 8px; overflow: hidden;">
          <tr>
            <td style="background: linear-gradient(135deg, #0d47a1, #1976d2); padding: 25px; text-align: center;">
              <img src="https://kubo-producto.oss-us-east-1.aliyuncs.com/logos/arsenal_logo1t.png" alt="Arsenal Welds" style="max-height: 55px; margin-bottom: 10px;">
              <h1 style="color: #ffffff; margin: 10px 0 0; font-size: 22px;">Actualización de %s</h1>
            </td>
          </tr>
          <tr>
            <td style="padding: 30px;">
              <p style="font-size: 16px; color: #333;">Estimado cliente,</p>
              <p style="font-size: 15px; color: #555; line-height: 1.6;">
                Le informamos que su solicitud de <strong>%s</strong> ha sido actualizada al siguiente estatus:
              </p>

              <table width="100%%" cellpadding="0" cellspacing="0" style="margin: 20px 0;">
                <tr>
                  <td style="padding: 8px 10px; background: #f8f9fa; font-weight: bold; color: #333; border-bottom: 2px solid #1976d2;">Folio</td>
                  <td style="padding: 8px 10px; background: #f8f9fa; color: #1976d2; border-bottom: 2px solid #1976d2;">%s</td>
                </tr>
                <tr>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0; font-weight: bold; color: #333;">Tipo</td>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0; color: #555;">%s</td>
                </tr>
                <tr>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0; font-weight: bold; color: #333;">Pedido</td>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0; color: #555;">%s</td>
                </tr>
                <tr>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0; font-weight: bold; color: #333;">Estatus</td>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0;"><span style="display: inline-block; background: %s; color: %s; padding: 4px 14px; border-radius: 4px; font-weight: bold; font-size: 14px;">%s</span></td>
                </tr>
              </table>

              <div style="background: #e3f2fd; border: 1px solid #90caf9; border-radius: 6px; padding: 15px; margin: 20px 0;">
                <div style="font-weight: bold; color: #1565c0; margin-bottom: 6px;">Nota del Cliente</div>
                <div style="color: #555; font-size: 14px; line-height: 1.5;">%s</div>
              </div>

              <div style="background: #fff8e1; border: 1px solid #ffe082; border-radius: 6px; padding: 15px; margin: 20px 0;">
                <div style="font-weight: bold; color: #e65100; margin-bottom: 6px;">Nota del Administrador</div>
                <div style="color: #555; font-size: 14px; line-height: 1.5;">%s</div>
              </div>

              <p style="font-size: 15px; color: #555; line-height: 1.6; margin-top: 20px;">
                Atentamente,<br>
                <strong>Arsenal Welds</strong>
              </p>
              <p style="font-size: 14px; color: #555; margin-top: 20px;">
                <a href="https://www.arsenalwelds.com" style="color: #1976d2; text-decoration: none; font-weight: bold;">www.arsenalwelds.com</a>
              </p>
            </td>
          </tr>
          <tr>
            <td style="background: #f0f0f0; padding: 15px; text-align: center; font-size: 12px; color: #999;">
              Arsenal Welds — ventas@arsenalwelds.com
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`, dv.Tipo, dv.Tipo, dv.Folio, dv.Tipo, dv.PedidoFolio, statusBadgeBg, statusColor, dv.Estatus, notaClienteHTML, notaAdminHTML)
}

func buildPagoFacturaHTML(pf PagoFacturaData) string {
	facturasRows := ""
	for _, f := range pf.Facturas {
		facturasRows += fmt.Sprintf(`<tr>
              <td style="padding: 10px; border-bottom: 1px solid #e0e0e0; color: #333;">%s</td>
              <td style="padding: 10px; border-bottom: 1px solid #e0e0e0; color: #333; text-align: right;">$ %s</td>
            </tr>`, f.NoFactura, f.MontoPagado)
	}

	terminacionRow := ""
	if pf.Terminacion != "" {
		terminacionRow = fmt.Sprintf(`
                <tr>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0; font-weight: bold; color: #333;">Tarjeta terminación</td>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0; color: #555;">****%s</td>
                </tr>`, pf.Terminacion)
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: Arial, sans-serif; margin: 0; padding: 0; background: #f5f5f5;">
  <table width="100%%" cellpadding="0" cellspacing="0" style="background: #f5f5f5; padding: 20px;">
    <tr>
      <td align="left">
        <table width="70%%" cellpadding="0" cellspacing="0" style="background: #ffffff; border-radius: 8px; overflow: hidden;">
          <tr>
            <td style="background: linear-gradient(135deg, #2e7d32, #43a047); padding: 25px; text-align: center;">
              <img src="https://kubo-producto.oss-us-east-1.aliyuncs.com/logos/arsenal_logo1t.png" alt="Arsenal Welds" style="max-height: 55px; margin-bottom: 10px;">
              <h1 style="color: #ffffff; margin: 10px 0 0; font-size: 22px;">Pago de Facturas Confirmado</h1>
            </td>
          </tr>
          <tr>
            <td style="padding: 30px;">
              <p style="font-size: 16px; color: #333;">Estimado cliente,</p>
              <p style="font-size: 15px; color: #555; line-height: 1.6;">
                Le confirmamos que hemos recibido el pago de las siguientes facturas:
              </p>

              <table width="100%%" cellpadding="0" cellspacing="0" style="margin: 20px 0;">
                <tr>
                  <td style="padding: 8px 10px; background: #f8f9fa; font-weight: bold; color: #333; border-bottom: 2px solid #2e7d32;">Cliente</td>
                  <td style="padding: 8px 10px; background: #f8f9fa; color: #2e7d32; border-bottom: 2px solid #2e7d32;">%s</td>
                </tr>
                <tr>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0; font-weight: bold; color: #333;">Fecha y hora</td>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0; color: #555;">%s</td>
                </tr>
                <tr>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0; font-weight: bold; color: #333;">Método de pago</td>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0; color: #555;">%s</td>
                </tr>%s
              </table>

              <h3 style="color: #2e7d32; margin: 20px 0 10px;">Facturas Pagadas</h3>
              <table width="100%%" cellpadding="0" cellspacing="0" style="border-collapse: collapse;">
                <thead>
                  <tr style="background: #2e7d32; color: #ffffff;">
                    <th style="padding: 10px; text-align: left;">No. Factura</th>
                    <th style="padding: 10px; text-align: right;">Monto Pagado</th>
                  </tr>
                </thead>
                <tbody>
                  %s
                </tbody>
                <tfoot>
                  <tr>
                    <td style="padding: 10px; border-top: 2px solid #2e7d32; text-align: right; font-weight: bold; font-size: 16px; color: #2e7d32;">Total</td>
                    <td style="padding: 10px; border-top: 2px solid #2e7d32; text-align: right; font-weight: bold; font-size: 16px; color: #2e7d32;">$ %s</td>
                  </tr>
                </tfoot>
              </table>

              <p style="font-size: 15px; color: #555; line-height: 1.6; margin-top: 30px;">
                Si tiene alguna duda o requiere una factura con los datos de su pago, no dude en contactarnos.
              </p>
              <p style="font-size: 15px; color: #555; line-height: 1.6;">
                Atentamente,<br>
                <strong>Arsenal Welds</strong>
              </p>
              <p style="font-size: 14px; color: #555; margin-top: 20px;">
                <a href="https://www.arsenalwelds.com" style="color: #2e7d32; text-decoration: none; font-weight: bold;">www.arsenalwelds.com</a>
              </p>
            </td>
          </tr>
          <tr>
            <td style="background: #f0f0f0; padding: 15px; text-align: center; font-size: 12px; color: #999;">
              Arsenal Welds — ventas@arsenalwelds.com
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`, pf.ClienteName, pf.Fecha, pf.MetodoPago, terminacionRow, facturasRows, pf.Total)
}

func buildSolicitudDistribuidorHTML(sd SolicitudDistribuidorData, esDistribuidor bool) string {
	titulo := "Solicitud de Registro Recibida"
	parrafoBienvenida := `Hemos recibido su solicitud de registro.<br>Agradecemos su interés en formar parte de Arsenal Welds y le confirmamos que su petición fue recibida correctamente.`
	if esDistribuidor {
		titulo = "Solicitud de Distribuidor Recibida"
		parrafoBienvenida = `Hemos recibido su solicitud para registrarse como <strong>Distribuidor</strong>.<br>Agradecemos su interés en formar parte de Arsenal Welds y le confirmamos que su petición fue recibida correctamente.`
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: Arial, sans-serif; margin: 0; padding: 0; background: #f5f5f5;">
  <table width="100%%" cellpadding="0" cellspacing="0" style="background: #f5f5f5; padding: 20px;">
    <tr>
      <td align="left">
        <table width="70%%" cellpadding="0" cellspacing="0" style="background: #ffffff; border-radius: 8px; overflow: hidden;">
          <tr>
            <td style="background: linear-gradient(135deg, #491212, #7a1a1a); padding: 25px; text-align: center;">
              <img src="https://kubo-producto.oss-us-east-1.aliyuncs.com/logos/arsenal_logo1t.png" alt="Arsenal Welds" style="max-height: 55px; margin-bottom: 10px;">
              <h1 style="color: #ffffff; margin: 10px 0 0; font-size: 22px;">%s</h1>
            </td>
          </tr>
          <tr>
            <td style="padding: 30px;">
              <p style="font-size: 16px; color: #333;">Estimado %s,</p>
              <p style="font-size: 15px; color: #555; line-height: 1.6;">
                %s
              </p>
              <p style="font-size: 15px; color: #555; line-height: 1.6;">
                Nuestro equipo revisará la información proporcionada y muy pronto nos pondremos en contacto con usted para
                compartir sus <strong>credenciales de acceso</strong> a la plataforma.
              </p>

              <table width="100%%" cellpadding="0" cellspacing="0" style="margin: 20px 0;">
                <tr>
                  <td style="padding: 8px 10px; background: #f8f9fa; font-weight: bold; color: #333; border-bottom: 2px solid #491212;">Folio</td>
                  <td style="padding: 8px 10px; background: #f8f9fa; color: #491212; border-bottom: 2px solid #491212;">%s</td>
                </tr>
                <tr>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0; font-weight: bold; color: #333;">Nombre Comercial</td>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0; color: #555;">%s</td>
                </tr>
                <tr>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0; font-weight: bold; color: #333;">RFC</td>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0; color: #555;">%s</td>
                </tr>
                <tr>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0; font-weight: bold; color: #333;">Fecha de solicitud</td>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0; color: #555;">%s</td>
                </tr>
              </table>

              <p style="font-size: 15px; color: #555; line-height: 1.6;">
                Si tiene alguna duda o requiere información adicional, no dude en contactarnos.
              </p>
              <p style="font-size: 15px; color: #555; line-height: 1.6;">
                Atentamente,<br>
                <strong>Arsenal Welds</strong>
              </p>
              <p style="font-size: 14px; color: #555; margin-top: 20px;">
                <a href="https://www.arsenalwelds.com" style="color: #491212; text-decoration: none; font-weight: bold;">www.arsenalwelds.com</a>
              </p>
            </td>
          </tr>
          <tr>
            <td style="background: #f0f0f0; padding: 15px; text-align: center; font-size: 12px; color: #999;">
              Arsenal Welds — ventas@arsenalwelds.com
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`, titulo, sd.NombreComercial, parrafoBienvenida, sd.Folio, sd.NombreComercial, sd.RFC, sd.Fecha)
}

func buildSolicitudEstatusHTML(se SolicitudEstatusData) string {
	aprobada := se.Estatus == "Aceptado"
	statusColor := "#2e7d32"
	statusBadgeBg := "#e8f5e9"
	statusText := "Aceptada"
	if !aprobada {
		statusColor = "#c62828"
		statusBadgeBg = "#ffebee"
		statusText = "Rechazada"
	}

	parrafo := fmt.Sprintf(`Le informamos que su solicitud de registro fue <strong>%s</strong>.<br>Solicitud presentada el %s.`, statusText, se.Fecha)

	notaSection := ""
	if se.Observacion != "" {
		notaLabel := "Nota del administrador"
		if !aprobada {
			notaLabel = "Motivo del rechazo"
		}
		notaHTML := strings.ReplaceAll(se.Observacion, "\n", "<br>")
		notaSection = fmt.Sprintf(`
              <div style="background: %s; border: 1px solid %s; border-radius: 6px; padding: 15px; margin: 20px 0;">
                <div style="font-weight: bold; color: %s; margin-bottom: 6px;">%s</div>
                <div style="color: #555; font-size: 14px; line-height: 1.5;">%s</div>
              </div>`, statusBadgeBg, statusColor, statusColor, notaLabel, notaHTML)
	}

	acceso := `<p style="font-size: 15px; color: #555; line-height: 1.6;">
                En breve nuestro equipo se pondrá en contacto con usted para compartir sus <strong>credenciales de acceso</strong> a la plataforma y pueda comenzar a realizar sus pedidos.
              </p>`
	if !aprobada {
		acceso = `<p style="font-size: 15px; color: #555; line-height: 1.6;">
                Si considera que esto es un error o desea volver a enviar su información, no dude en contactarnos.
              </p>`
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: Arial, sans-serif; margin: 0; padding: 0; background: #f5f5f5;">
  <table width="100%%" cellpadding="0" cellspacing="0" style="background: #f5f5f5; padding: 20px;">
    <tr>
      <td align="left">
        <table width="70%%" cellpadding="0" cellspacing="0" style="background: #ffffff; border-radius: 8px; overflow: hidden;">
          <tr>
            <td style="background: linear-gradient(135deg, #491212, #7a1a1a); padding: 25px; text-align: center;">
              <img src="https://kubo-producto.oss-us-east-1.aliyuncs.com/logos/arsenal_logo1t.png" alt="Arsenal Welds" style="max-height: 55px; margin-bottom: 10px;">
              <h1 style="color: #ffffff; margin: 10px 0 0; font-size: 22px;">Actualización de Solicitud de Registro</h1>
            </td>
          </tr>
          <tr>
            <td style="padding: 30px;">
              <p style="font-size: 16px; color: #333;">Estimado %s,</p>
              <p style="font-size: 15px; color: #555; line-height: 1.6;">
                %s
              </p>

              <table width="100%%" cellpadding="0" cellspacing="0" style="margin: 20px 0;">
                <tr>
                  <td style="padding: 8px 10px; background: #f8f9fa; font-weight: bold; color: #333; border-bottom: 2px solid #491212;">Folio</td>
                  <td style="padding: 8px 10px; background: #f8f9fa; color: #491212; border-bottom: 2px solid #491212;">%s</td>
                </tr>
                <tr>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0; font-weight: bold; color: #333;">Nombre Comercial</td>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0; color: #555;">%s</td>
                </tr>
                <tr>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0; font-weight: bold; color: #333;">RFC</td>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0; color: #555;">%s</td>
                </tr>
                <tr>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0; font-weight: bold; color: #333;">Estatus</td>
                  <td style="padding: 8px 10px; border-bottom: 1px solid #e0e0e0;"><span style="display: inline-block; background: %s; color: %s; padding: 4px 14px; border-radius: 4px; font-weight: bold; font-size: 14px;">%s</span></td>
                </tr>
              </table>

              %s

              %s

              <p style="font-size: 15px; color: #555; line-height: 1.6; margin-top: 20px;">
                Atentamente,<br>
                <strong>Arsenal Welds</strong>
              </p>
              <p style="font-size: 14px; color: #555; margin-top: 20px;">
                <a href="https://www.arsenalwelds.com" style="color: #491212; text-decoration: none; font-weight: bold;">www.arsenalwelds.com</a>
              </p>
            </td>
          </tr>
          <tr>
            <td style="background: #f0f0f0; padding: 15px; text-align: center; font-size: 12px; color: #999;">
              Arsenal Welds — ventas@arsenalwelds.com
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`, se.NombreComercial, parrafo, se.Folio, se.NombreComercial, se.RFC, statusBadgeBg, statusColor, statusText, notaSection, acceso)
}
