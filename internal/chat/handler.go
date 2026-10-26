package chat

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/mail"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/OrlandoHdz/kubo/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

const minRequestBody = 64 << 10

var conversationIDPattern = regexp.MustCompile(`^[a-f0-9]{32}$`)

type Handler struct {
	cfg         Config
	agent       *Agent
	contexto    ContextoStore
	auditoria   AuditoriaStore
	limiter     *RateLimiter
	readPool    *pgxpool.Pool
	auditPool   *pgxpool.Pool
	maxBodySize int64
}

func NewHandler(cfg Config, agent *Agent, contexto ContextoStore, auditoria AuditoriaStore) *Handler {
	if cfg.MaxMessageChars <= 0 {
		cfg.MaxMessageChars = DefaultMaxMessageChars
	}
	maxBody := int64(cfg.MaxHistoryMessages*cfg.MaxMessageChars + cfg.MaxMessageChars + 8192)
	if maxBody < minRequestBody {
		maxBody = minRequestBody
	}
	if maxBody > 1<<20 {
		maxBody = 1 << 20
	}
	return &Handler{
		cfg:         cfg,
		agent:       agent,
		contexto:    contexto,
		auditoria:   auditoria,
		limiter:     NewRateLimiter(cfg.RateLimitPerMinute),
		maxBodySize: maxBody,
	}
}

func (h *Handler) Close() {
	if h.readPool != nil {
		h.readPool.Close()
	}
	if h.auditPool != nil {
		h.auditPool.Close()
	}
}

type chatRequest struct {
	Message        string               `json:"message"`
	ConversationID string               `json:"conversation_id"`
	History        []chatHistoryMessage `json:"history"`
	Contact        *chatContact         `json:"contact,omitempty"`
}

type chatHistoryMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatContact struct {
	Nombre   string `json:"nombre"`
	Email    string `json:"email"`
	Telefono string `json:"telefono"`
	Motivo   string `json:"motivo"`
}

type sessionUser struct {
	ID    int32
	Email string
	Rol   string
}

// Chat implementa POST /api/chat con streaming SSE.
func (h *Handler) Chat(c *gin.Context) {
	ctx := c.Request.Context()

	limit := h.maxBodySize
	if limit <= 0 {
		limit = minRequestBody
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
	var req chatRequest
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "El mensaje es demasiado grande."})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "Solicitud inválida."})
		return
	}

	history, err := h.normalizeHistory(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	usuario, ok := optionalUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token inválido o expirado. Inicia sesión nuevamente."})
		return
	}

	var (
		usuarioID    *int32
		usuarioEmail string
	)
	if usuario != nil {
		id := usuario.ID
		usuarioID = &id
		usuarioEmail = usuario.Email
	}

	contacto, err := h.normalizeContact(&req, usuario)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rateKey := "ip:" + remoteIP(c.Request)
	if usuarioID != nil {
		rateKey = fmt.Sprintf("u:%d", *usuarioID)
	}
	if !h.limiter.Allow(rateKey) {
		c.Header("Retry-After", "60")
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Demasiadas solicitudes. Espera un momento antes de volver a intentar."})
		return
	}

	var (
		clienteID      *int32
		esDistribuidor bool
	)
	if usuarioID != nil && h.contexto != nil {
		info, err := h.contexto.ObtenerContexto(ctx, *usuarioID)
		if err != nil {
			log.Printf("chat: no se pudo resolver cliente para usuario=%d: %v", *usuarioID, err)
		} else {
			clienteID = info.ClienteID
			esDistribuidor = info.EsDistribuidor
		}
	}

	conversationID := strings.TrimSpace(req.ConversationID)
	isNewConversation := false
	if conversationID == "" {
		conversationID, err = newConversationID()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo iniciar la conversación."})
			return
		}
		isNewConversation = true
	} else if !conversationIDPattern.MatchString(conversationID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Identificador de conversación inválido."})
		return
	}

	userMessage := strings.TrimSpace(req.Message)
	h.auditUser(ctx, conversationID, isNewConversation, usuarioID, clienteID, userMessage)

	sw, err := newSSEWriter(c.Writer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "El servidor no soporta streaming."})
		return
	}

	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache, no-transform")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	if err := sw.Send("meta", map[string]any{
		"conversation_id": conversationID,
		"model":           h.cfg.Model,
	}); err != nil {
		return
	}

	stopKeepalive := startKeepalive(sw)
	defer stopKeepalive()

	toolCtx := ToolContext{
		ConversationID: conversationID,
		UsuarioID:      usuarioID,
		UsuarioEmail:   usuarioEmail,
		ClienteID:      clienteID,
		EsDistribuidor: esDistribuidor,
		Contacto:       contacto,
	}

	emit := func(ev AgentEvent) error {
		switch ev.Type {
		case "text":
			return sw.Send("text", map[string]string{"delta": ev.Text})
		case "status":
			return sw.Send("status", map[string]string{"stage": ev.Stage, "tool": ev.Tool})
		case "products":
			return sw.Send("products", map[string]any{"items": ev.Products})
		case "escalation":
			return sw.Send("escalation", ev.Escalation)
		default:
			return nil
		}
	}

	result, runErr := h.agent.Run(ctx, history, userMessage, toolCtx, emit)
	stopKeepalive()

	if runErr != nil {
		if errors.Is(runErr, context.Canceled) || ctx.Err() != nil {
			// El cliente cerró la conexión; no intentamos escribir más.
			h.auditAssistant(ctx, conversationID, usuarioID, result)
			return
		}
		log.Printf("chat: error conversation=%s: %v", conversationID, runErr)
		_ = sw.Send("error", h.errorPayload(runErr))
		h.auditAssistant(ctx, conversationID, usuarioID, result)
		return
	}

	log.Printf(
		"chat: done conversation=%s stop_reason=%s text_chars=%d tool_calls=%d",
		conversationID,
		result.StopReason,
		utf8.RuneCountInString(result.Text),
		len(result.ToolCalls),
	)

	if err := sw.Send("done", map[string]any{
		"conversation_id": conversationID,
		"stop_reason":     result.StopReason,
		"usage": map[string]int32{
			"input_tokens":  result.Usage.InputTokens,
			"output_tokens": result.Usage.OutputTokens,
		},
	}); err != nil {
		h.auditAssistant(ctx, conversationID, usuarioID, result)
		return
	}

	h.auditAssistant(ctx, conversationID, usuarioID, result)
}

type apiErrPayload struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

func (h *Handler) errorPayload(err error) apiErrPayload {
	if errors.Is(err, ErrMaxIterations) {
		return apiErrPayload{
			Code:      "max_iterations",
			Message:   "No pude completar la consulta en este momento. ¿Quieres que te pase con un asesor humano?",
			Retryable: false,
		}
	}

	if errors.Is(err, ErrEmptyResponse) {
		return apiErrPayload{
			Code:      "empty_response",
			Message:   "No recibí una respuesta del asesor. Intenta de nuevo.",
			Retryable: true,
		}
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) {
		switch {
		case apiErr.StatusCode == http.StatusTooManyRequests || apiErr.StatusCode == 529 ||
			apiErr.Type == "overloaded_error" || apiErr.Type == "rate_limit_error":
			return apiErrPayload{
				Code:      "upstream_busy",
				Message:   "Estoy recibiendo muchas consultas. Intenta de nuevo en unos segundos.",
				Retryable: true,
			}
		case apiErr.StatusCode >= 500:
			return apiErrPayload{
				Code:      "upstream_error",
				Message:   "El servicio de asesor está tardando más de lo normal. Intenta de nuevo.",
				Retryable: true,
			}
		default:
			return apiErrPayload{
				Code:      "upstream_rejected",
				Message:   "No pude procesar la consulta en este momento. Intenta de nuevo o habla con un asesor.",
				Retryable: false,
			}
		}
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return apiErrPayload{
			Code:      "timeout",
			Message:   "La consulta tardó demasiado. Intenta de nuevo.",
			Retryable: true,
		}
	}

	return apiErrPayload{
		Code:      "internal_error",
		Message:   "Ocurrió un error al procesar tu mensaje. Intenta de nuevo o habla con un asesor.",
		Retryable: true,
	}
}

func (h *Handler) normalizeHistory(req chatRequest) ([]Message, error) {
	message := strings.TrimSpace(req.Message)
	if message == "" {
		return nil, errors.New("Escribe un mensaje.")
	}
	if utf8.RuneCountInString(message) > h.cfg.MaxMessageChars {
		return nil, fmt.Errorf("El mensaje no puede superar %d caracteres.", h.cfg.MaxMessageChars)
	}

	if h.cfg.MaxHistoryMessages == 0 {
		return nil, nil
	}

	var (
		selected []chatHistoryMessage
		chars    int
	)
	for i := len(req.History) - 1; i >= 0 && len(selected) < h.cfg.MaxHistoryMessages; i-- {
		item := req.History[i]
		role := strings.ToLower(strings.TrimSpace(item.Role))
		if role != "user" && role != "assistant" {
			return nil, errors.New("El historial contiene un rol inválido.")
		}
		content := strings.TrimSpace(item.Content)
		if content == "" {
			continue
		}
		content = trimToRunes(content, h.cfg.MaxMessageChars)
		if chars+utf8.RuneCountInString(content) > h.cfg.MaxHistoryChars {
			break
		}
		chars += utf8.RuneCountInString(content)
		selected = append(selected, chatHistoryMessage{Role: role, Content: content})
	}

	// Se recorre al revés para conservar lo más reciente; se voltea al final.
	for left, right := 0, len(selected)-1; left < right; left, right = left+1, right-1 {
		selected[left], selected[right] = selected[right], selected[left]
	}

	// Si el cliente incluyó el mensaje actual en el historial, se elimina para
	// no enviarlo dos veces.
	if n := len(selected); n > 0 && selected[n-1].Role == "user" && selected[n-1].Content == message {
		selected = selected[:n-1]
	}

	out := make([]Message, 0, len(selected))
	for _, item := range selected {
		out = append(out, Message{
			Role:    item.Role,
			Content: []ContentBlock{{Type: "text", Text: item.Content}},
		})
	}
	return out, nil
}

func optionalUser(c *gin.Context) (*sessionUser, bool) {
	header := strings.TrimSpace(c.GetHeader("Authorization"))
	if header == "" {
		return nil, true
	}
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return nil, false
	}
	claims, err := auth.ValidarToken(parts[1])
	if err != nil {
		return nil, false
	}
	return &sessionUser{
		ID:    claims.UsuarioID,
		Email: claims.Email,
		Rol:   claims.Rol,
	}, true
}

func (h *Handler) normalizeContact(req *chatRequest, usuario *sessionUser) (*ContactoEscalamiento, error) {
	if req.Contact == nil {
		return nil, nil
	}

	nombre := strings.TrimSpace(req.Contact.Nombre)
	email := strings.TrimSpace(req.Contact.Email)
	telefono := strings.TrimSpace(req.Contact.Telefono)
	motivo := strings.TrimSpace(req.Contact.Motivo)

	if nombre == "" {
		return nil, errors.New("El nombre es requerido para solicitar un asesor.")
	}
	if utf8.RuneCountInString(nombre) > 120 {
		return nil, errors.New("El nombre no puede superar 120 caracteres.")
	}
	if email != "" {
		if utf8.RuneCountInString(email) > 160 {
			return nil, errors.New("El correo no puede superar 160 caracteres.")
		}
		addr, err := mail.ParseAddress(email)
		if err != nil {
			return nil, errors.New("El correo de contacto no es válido.")
		}
		email = addr.Address
	}
	if telefono != "" && utf8.RuneCountInString(telefono) > 30 {
		return nil, errors.New("El teléfono no puede superar 30 caracteres.")
	}
	if motivo != "" && utf8.RuneCountInString(motivo) > 500 {
		return nil, errors.New("El motivo no puede superar 500 caracteres.")
	}
	if email == "" && usuario != nil {
		email = strings.TrimSpace(usuario.Email)
	}
	if email == "" && telefono == "" {
		return nil, errors.New("Comparte un correo o un teléfono para que un asesor pueda contactarte.")
	}

	return &ContactoEscalamiento{
		Nombre:   nombre,
		Email:    email,
		Telefono: telefono,
		Motivo:   motivo,
	}, nil
}

func (h *Handler) auditUser(ctx context.Context, conversationID string, isNew bool, usuarioID, clienteID *int32, message string) {
	if h.auditoria == nil {
		return
	}
	auditCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	if isNew {
		if err := h.auditoria.CrearConversacion(auditCtx, conversationID, usuarioID, clienteID); err != nil {
			log.Printf("chat: no se pudo auditar conversación=%s: %v", conversationID, err)
		}
	}
	if err := h.auditoria.CrearMensaje(auditCtx, conversationID, usuarioID, "user", message, nil, 0, 0); err != nil {
		log.Printf("chat: no se pudo auditar mensaje de usuario conversation=%s: %v", conversationID, err)
	}
}

func (h *Handler) auditAssistant(ctx context.Context, conversationID string, usuarioID *int32, result RunResult) {
	if h.auditoria == nil {
		return
	}
	auditCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	var toolCalls json.RawMessage
	if len(result.ToolCalls) > 0 {
		if encoded, err := json.Marshal(result.ToolCalls); err == nil {
			toolCalls = encoded
		}
	}
	if err := h.auditoria.CrearMensaje(
		auditCtx,
		conversationID,
		usuarioID,
		"assistant",
		result.Text,
		toolCalls,
		result.Usage.InputTokens,
		result.Usage.OutputTokens,
	); err != nil {
		log.Printf("chat: no se pudo auditar respuesta conversation=%s: %v", conversationID, err)
	}
}

func remoteIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func newConversationID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func trimToRunes(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max])
}

func startKeepalive(sw *sseWriter) func() {
	done := make(chan struct{})
	var once sync.Once
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if err := sw.Comment("keepalive"); err != nil {
					return
				}
			}
		}
	}()

	return func() {
		once.Do(func() {
			close(done)
			wg.Wait()
		})
	}
}
