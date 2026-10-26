package chat

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
)

// Streamer abstrae el cliente de Claude para poder simularlo en tests.
type Streamer interface {
	Stream(ctx context.Context, req StreamRequest, onText func(string) error) (StreamResult, error)
}

type StreamRequest struct {
	System   string
	Messages []Message
	Tools    []ToolDefinition
}

type StreamResult struct {
	Content    []ContentBlock
	StopReason string
	Usage      Usage
}

// APIError representa un error HTTP/SSE de la API de Anthropic.
type APIError struct {
	StatusCode int
	Type       string
	Message    string
	Retryable  bool
}

func (e *APIError) Error() string {
	if e.Type != "" {
		return fmt.Sprintf("anthropic: HTTP %d (%s): %s", e.StatusCode, e.Type, e.Message)
	}
	return fmt.Sprintf("anthropic: HTTP %d: %s", e.StatusCode, e.Message)
}

type Client struct {
	apiKey     string
	model      string
	maxTokens  int
	baseURL    string
	httpClient *http.Client
}

func NewClient(cfg Config) *Client {
	model := cfg.Model
	if model == "" {
		model = DefaultModel
	}
	maxTokens := cfg.MaxTokens
	if maxTokens <= 0 {
		maxTokens = DefaultMaxTokens
	}
	baseURL := strings.TrimRight(cfg.AnthropicBaseURL, "/")
	if baseURL == "" {
		baseURL = DefaultAnthropicBaseURL
	}
	return &Client{
		apiKey:    cfg.APIKey,
		model:     model,
		maxTokens: maxTokens,
		baseURL:   baseURL,
		httpClient: &http.Client{
			// El timeout se controla con el contexto de cada llamada.
			Timeout: 0,
		},
	}
}

type apiMessage struct {
	Role    string       `json:"role"`
	Content []apiContent `json:"content"`
}

type apiContent struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   string          `json:"content,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
	Thinking  string          `json:"thinking,omitempty"`
	Signature string          `json:"signature,omitempty"`
	Data      string          `json:"data,omitempty"`
}

// MarshalJSON fuerza los campos obligatorios de los bloques thinking y
// redacted_thinking. La API exige "thinking" y "signature" incluso cuando el
// texto del pensamiento viene vacío (display: "omitted").
func (c apiContent) MarshalJSON() ([]byte, error) {
	switch c.Type {
	case "thinking":
		return json.Marshal(struct {
			Type      string `json:"type"`
			Thinking  string `json:"thinking"`
			Signature string `json:"signature"`
		}{
			Type:      c.Type,
			Thinking:  c.Thinking,
			Signature: c.Signature,
		})
	case "redacted_thinking":
		return json.Marshal(struct {
			Type string `json:"type"`
			Data string `json:"data"`
		}{
			Type: c.Type,
			Data: c.Data,
		})
	default:
		type plain apiContent
		return json.Marshal(plain(c))
	}
}

type apiTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema"`
}

type apiToolChoice struct {
	Type                   string `json:"type"`
	DisableParallelToolUse bool   `json:"disable_parallel_tool_use,omitempty"`
}

type apiRequest struct {
	Model      string         `json:"model"`
	MaxTokens  int            `json:"max_tokens"`
	System     string         `json:"system,omitempty"`
	Messages   []apiMessage   `json:"messages"`
	Tools      []apiTool      `json:"tools,omitempty"`
	ToolChoice *apiToolChoice `json:"tool_choice,omitempty"`
	Stream     bool           `json:"stream"`
}

type apiUsage struct {
	InputTokens  int32 `json:"input_tokens"`
	OutputTokens int32 `json:"output_tokens"`
}

type apiStreamEvent struct {
	Type    string `json:"type"`
	Index   int    `json:"index"`
	Message *struct {
		Usage *apiUsage `json:"usage"`
	} `json:"message"`
	ContentBlock *struct {
		Type      string          `json:"type"`
		Text      string          `json:"text"`
		ID        string          `json:"id"`
		Name      string          `json:"name"`
		Input     json.RawMessage `json:"input"`
		Thinking  string          `json:"thinking"`
		Signature string          `json:"signature"`
		Data      string          `json:"data"`
	} `json:"content_block"`
	Delta *struct {
		Type        string `json:"type"`
		Text        string `json:"text"`
		PartialJSON string `json:"partial_json"`
		Thinking    string `json:"thinking"`
		Signature   string `json:"signature"`
		StopReason  string `json:"stop_reason"`
	} `json:"delta"`
	Usage *apiUsage `json:"usage"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

type apiErrorEnvelope struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

type blockState struct {
	typ       string
	text      strings.Builder
	thinking  strings.Builder
	signature string
	data      string
	id        string
	name      string
	input     json.RawMessage
	partial   strings.Builder
}

// Stream llama a POST /v1/messages con stream=true y va entregando los deltas
// de texto. Los bloques thinking/tool_use se acumulan para el loop del agente.
func (c *Client) Stream(ctx context.Context, req StreamRequest, onText func(string) error) (StreamResult, error) {
	body := apiRequest{
		Model:     c.model,
		MaxTokens: c.maxTokens,
		System:    req.System,
		Messages:  toAPIMessages(req.Messages),
		Tools:     toAPITools(req.Tools),
		Stream:    true,
	}
	if len(body.Tools) > 0 {
		body.ToolChoice = &apiToolChoice{Type: "auto", DisableParallelToolUse: true}
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return StreamResult{}, fmt.Errorf("serializando request de Claude: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/messages", bytes.NewReader(payload))
	if err != nil {
		return StreamResult{}, err
	}
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", DefaultAnthropicVersion)
	httpReq.Header.Set("content-type", "application/json")
	httpReq.Header.Set("accept", "text/event-stream")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return StreamResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return StreamResult{}, parseAPIError(resp)
	}

	return readSSE(resp.Body, onText)
}

func parseAPIError(resp *http.Response) error {
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
	var envelope apiErrorEnvelope
	_ = json.Unmarshal(raw, &envelope)

	message := strings.TrimSpace(envelope.Error.Message)
	if message == "" {
		message = http.StatusText(resp.StatusCode)
	}
	typ := envelope.Error.Type
	retryable := resp.StatusCode == http.StatusTooManyRequests ||
		resp.StatusCode == http.StatusInternalServerError ||
		resp.StatusCode == http.StatusBadGateway ||
		resp.StatusCode == http.StatusServiceUnavailable ||
		resp.StatusCode == http.StatusGatewayTimeout ||
		resp.StatusCode == 529 ||
		typ == "overloaded_error" ||
		typ == "rate_limit_error"

	return &APIError{
		StatusCode: resp.StatusCode,
		Type:       typ,
		Message:    message,
		Retryable:  retryable,
	}
}

func readSSE(body io.Reader, onText func(string) error) (StreamResult, error) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 64*1024), 2*1024*1024)

	var (
		result       StreamResult
		states       = map[int]*blockState{}
		finished     = map[int]ContentBlock{}
		eventName    string
		dataLines    []string
		stopReason   string
		inputTokens  int32
		outputTokens int32
		gotStop      bool
		totalBytes   int
	)

	dispatch := func() error {
		if len(dataLines) == 0 {
			eventName = ""
			return nil
		}
		data := strings.Join(dataLines, "\n")
		currentEvent := eventName
		eventName, dataLines = "", nil

		var ev apiStreamEvent
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			return fmt.Errorf("evento SSE inválido: %w", err)
		}
		if ev.Type == "" {
			ev.Type = currentEvent
		}

		switch ev.Type {
		case "message_start":
			if ev.Message != nil && ev.Message.Usage != nil {
				if ev.Message.Usage.InputTokens > 0 {
					inputTokens = ev.Message.Usage.InputTokens
				}
				if ev.Message.Usage.OutputTokens > 0 {
					outputTokens = ev.Message.Usage.OutputTokens
				}
			}
		case "content_block_start":
			if ev.ContentBlock == nil {
				return nil
			}
			st := &blockState{
				typ:       ev.ContentBlock.Type,
				id:        ev.ContentBlock.ID,
				name:      ev.ContentBlock.Name,
				input:     ev.ContentBlock.Input,
				signature: ev.ContentBlock.Signature,
				data:      ev.ContentBlock.Data,
			}
			st.text.WriteString(ev.ContentBlock.Text)
			st.thinking.WriteString(ev.ContentBlock.Thinking)
			states[ev.Index] = st
		case "content_block_delta":
			st, ok := states[ev.Index]
			if !ok || ev.Delta == nil {
				return nil
			}
			switch ev.Delta.Type {
			case "text_delta":
				st.text.WriteString(ev.Delta.Text)
				if onText != nil && ev.Delta.Text != "" {
					if err := onText(ev.Delta.Text); err != nil {
						return err
					}
				}
			case "input_json_delta":
				st.partial.WriteString(ev.Delta.PartialJSON)
			case "thinking_delta":
				st.thinking.WriteString(ev.Delta.Thinking)
			case "signature_delta":
				st.signature = ev.Delta.Signature
			}
		case "content_block_stop":
			st, ok := states[ev.Index]
			if !ok {
				return nil
			}
			block, err := finalizeBlock(st)
			if err != nil {
				return err
			}
			if block != nil {
				finished[ev.Index] = *block
			}
			delete(states, ev.Index)
		case "message_delta":
			if ev.Delta != nil && ev.Delta.StopReason != "" {
				stopReason = ev.Delta.StopReason
			}
			if ev.Usage != nil {
				if ev.Usage.InputTokens > 0 {
					inputTokens = ev.Usage.InputTokens
				}
				if ev.Usage.OutputTokens > 0 {
					outputTokens = ev.Usage.OutputTokens
				}
			}
		case "message_stop":
			gotStop = true
		case "error":
			if ev.Error != nil {
				retryable := ev.Error.Type == "overloaded_error" || ev.Error.Type == "rate_limit_error"
				return &APIError{
					StatusCode: 0,
					Type:       ev.Error.Type,
					Message:    ev.Error.Message,
					Retryable:  retryable,
				}
			}
			return errors.New("error SSE de Anthropic sin detalle")
		case "ping":
			// ignorado
		}
		return nil
	}

	for scanner.Scan() {
		line := scanner.Text()
		totalBytes += len(line) + 1
		if totalBytes > 8*1024*1024 {
			return StreamResult{}, errors.New("respuesta de Claude demasiado grande")
		}
		line = strings.TrimSuffix(line, "\r")

		if line == "" {
			if err := dispatch(); err != nil {
				return StreamResult{}, err
			}
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue
		}
		if strings.HasPrefix(line, "event:") {
			eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}
		if strings.HasPrefix(line, "data:") {
			data := strings.TrimPrefix(line, "data:")
			data = strings.TrimPrefix(data, " ")
			dataLines = append(dataLines, data)
			continue
		}
		// Línea desconocida: se ignora, según la política de versionado de SSE.
	}
	if err := scanner.Err(); err != nil {
		return StreamResult{}, err
	}
	if err := dispatch(); err != nil {
		return StreamResult{}, err
	}
	if !gotStop {
		return StreamResult{}, errors.New("el stream de Claude terminó sin message_stop")
	}
	if len(states) > 0 {
		return StreamResult{}, errors.New("el stream de Claude terminó con bloques incompletos")
	}

	indexes := make([]int, 0, len(finished))
	for idx := range finished {
		indexes = append(indexes, idx)
	}
	sort.Ints(indexes)
	for _, idx := range indexes {
		result.Content = append(result.Content, finished[idx])
	}
	if stopReason == "" {
		stopReason = "end_turn"
	}
	result.StopReason = stopReason
	result.Usage = Usage{InputTokens: inputTokens, OutputTokens: outputTokens}
	return result, nil
}

func finalizeBlock(st *blockState) (*ContentBlock, error) {
	switch st.typ {
	case "text":
		if st.text.Len() == 0 {
			return nil, nil
		}
		return &ContentBlock{Type: "text", Text: st.text.String()}, nil
	case "tool_use":
		input := st.input
		if st.partial.Len() > 0 {
			var parsed any
			if err := json.Unmarshal([]byte(st.partial.String()), &parsed); err != nil {
				return nil, fmt.Errorf("JSON de herramienta inválido: %w", err)
			}
			normalized, err := json.Marshal(parsed)
			if err != nil {
				return nil, err
			}
			input = normalized
		}
		if len(input) == 0 {
			input = json.RawMessage(`{}`)
		}
		return &ContentBlock{
			Type:  "tool_use",
			ID:    st.id,
			Name:  st.name,
			Input: input,
		}, nil
	case "thinking":
		if st.thinking.Len() == 0 && st.signature == "" {
			return nil, nil
		}
		return &ContentBlock{
			Type:      "thinking",
			Thinking:  st.thinking.String(),
			Signature: st.signature,
		}, nil
	case "redacted_thinking":
		if st.data == "" {
			return nil, nil
		}
		return &ContentBlock{Type: "redacted_thinking", Data: st.data}, nil
	default:
		// No usamos herramientas de servidor; se ignoran bloques desconocidos.
		return nil, nil
	}
}

func toAPIMessages(messages []Message) []apiMessage {
	out := make([]apiMessage, 0, len(messages))
	for _, msg := range messages {
		blocks := make([]apiContent, 0, len(msg.Content))
		for _, block := range msg.Content {
			switch block.Type {
			case "text":
				if block.Text == "" {
					continue
				}
				blocks = append(blocks, apiContent{Type: "text", Text: block.Text})
			case "tool_use":
				input := block.Input
				if len(input) == 0 {
					input = json.RawMessage(`{}`)
				}
				blocks = append(blocks, apiContent{
					Type:  "tool_use",
					ID:    block.ID,
					Name:  block.Name,
					Input: input,
				})
			case "tool_result":
				blocks = append(blocks, apiContent{
					Type:      "tool_result",
					ToolUseID: block.ToolUseID,
					Content:   block.ToolContent,
					IsError:   block.IsError,
				})
			case "thinking":
				if block.Signature == "" {
					continue
				}
				blocks = append(blocks, apiContent{
					Type:      "thinking",
					Thinking:  block.Thinking,
					Signature: block.Signature,
				})
			case "redacted_thinking":
				if block.Data == "" {
					continue
				}
				blocks = append(blocks, apiContent{
					Type: "redacted_thinking",
					Data: block.Data,
				})
			}
		}
		if len(blocks) == 0 {
			continue
		}
		out = append(out, apiMessage{Role: msg.Role, Content: blocks})
	}
	return out
}

func toAPITools(tools []ToolDefinition) []apiTool {
	out := make([]apiTool, 0, len(tools))
	for _, tool := range tools {
		schema := tool.InputSchema
		if len(schema) == 0 {
			schema = json.RawMessage(`{"type":"object","properties":{}}`)
		}
		out = append(out, apiTool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: schema,
		})
	}
	return out
}
