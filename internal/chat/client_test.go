package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientStreamParsesTextToolUseAndUsage(t *testing.T) {
	sse := strings.Join([]string{
		"event: message_start",
		`data: {"type":"message_start","message":{"usage":{"input_tokens":11,"output_tokens":1}}}`,
		"",
		"event: content_block_start",
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
		"",
		"event: content_block_delta",
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hola"}}`,
		"",
		"event: content_block_stop",
		`data: {"type":"content_block_stop","index":0}`,
		"",
		"event: content_block_start",
		`data: {"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"tu_1","name":"buscar_productos","input":{}}}`,
		"",
		"event: content_block_delta",
		`data: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"consulta\":"}}`,
		"",
		"event: content_block_delta",
		`data: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"\"6013\"}"}}`,
		"",
		"event: content_block_stop",
		`data: {"type":"content_block_stop","index":1}`,
		"",
		"event: message_delta",
		`data: {"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":9}}`,
		"",
		"event: message_stop",
		`data: {"type":"message_stop"}`,
		"",
	}, "\n")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("x-api-key = %q", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") != DefaultAnthropicVersion {
			t.Errorf("anthropic-version = %q", r.Header.Get("anthropic-version"))
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, sse)
	}))
	defer server.Close()

	client := NewClient(Config{
		APIKey:           "test-key",
		Model:            "claude-test",
		MaxTokens:        100,
		AnthropicBaseURL: server.URL,
	})

	var text strings.Builder
	result, err := client.Stream(context.Background(), StreamRequest{
		System:   "prompt",
		Messages: []Message{{Role: "user", Content: []ContentBlock{{Type: "text", Text: "Hola"}}}},
	}, func(delta string) error {
		text.WriteString(delta)
		return nil
	})
	if err != nil {
		t.Fatalf("Stream devolvió error: %v", err)
	}
	if text.String() != "Hola" {
		t.Fatalf("texto = %q", text.String())
	}
	if result.StopReason != "tool_use" {
		t.Fatalf("stop_reason = %q", result.StopReason)
	}
	if result.Usage.InputTokens != 11 || result.Usage.OutputTokens != 9 {
		t.Fatalf("usage = %+v", result.Usage)
	}
	if len(result.Content) != 2 {
		t.Fatalf("bloques = %d: %+v", len(result.Content), result.Content)
	}
	if result.Content[0].Type != "text" || result.Content[0].Text != "Hola" {
		t.Fatalf("bloque texto = %+v", result.Content[0])
	}
	if result.Content[1].Type != "tool_use" || result.Content[1].Name != "buscar_productos" || result.Content[1].ID != "tu_1" {
		t.Fatalf("bloque tool_use = %+v", result.Content[1])
	}
	var input map[string]string
	if err := json.Unmarshal(result.Content[1].Input, &input); err != nil {
		t.Fatalf("input no es JSON: %v (%s)", err, result.Content[1].Input)
	}
	if input["consulta"] != "6013" {
		t.Fatalf("input = %+v", input)
	}
}

func TestAPIContentThinkingIncludesRequiredFields(t *testing.T) {
	messages := toAPIMessages([]Message{{
		Role: "assistant",
		Content: []ContentBlock{{
			Type:      "thinking",
			Thinking:  "",
			Signature: "firma",
		}},
	}})
	raw, err := json.Marshal(messages)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"thinking":""`) {
		t.Fatalf("falta thinking vacío obligatorio: %s", raw)
	}
	if !strings.Contains(string(raw), `"signature":"firma"`) {
		t.Fatalf("falta signature: %s", raw)
	}

	textMessages := toAPIMessages([]Message{{
		Role:    "user",
		Content: []ContentBlock{{Type: "text", Text: "hola"}},
	}})
	textRaw, err := json.Marshal(textMessages)
	if err != nil {
		t.Fatalf("marshal texto: %v", err)
	}
	if strings.Contains(string(textRaw), `"thinking"`) {
		t.Fatalf("un bloque de texto no debe incluir thinking: %s", textRaw)
	}
}

func TestClientStreamParsesThinking(t *testing.T) {
	sse := strings.Join([]string{
		"event: message_start",
		`data: {"type":"message_start","message":{"usage":{"input_tokens":5}}}`,
		"",
		"event: content_block_start",
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":""}}`,
		"",
		"event: content_block_delta",
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"Analizando"}}`,
		"",
		"event: content_block_delta",
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"signature_delta","signature":"firma"}}`,
		"",
		"event: content_block_stop",
		`data: {"type":"content_block_stop","index":0}`,
		"",
		"event: message_delta",
		`data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":2}}`,
		"",
		"event: message_stop",
		`data: {"type":"message_stop"}`,
		"",
	}, "\n")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, sse)
	}))
	defer server.Close()

	client := NewClient(Config{APIKey: "k", Model: "m", MaxTokens: 10, AnthropicBaseURL: server.URL})
	result, err := client.Stream(context.Background(), StreamRequest{}, func(string) error { return nil })
	if err != nil {
		t.Fatalf("Stream devolvió error: %v", err)
	}
	if len(result.Content) != 1 || result.Content[0].Type != "thinking" {
		t.Fatalf("bloques = %+v", result.Content)
	}
	if result.Content[0].Thinking != "Analizando" || result.Content[0].Signature != "firma" {
		t.Fatalf("thinking = %+v", result.Content[0])
	}
}

func TestClientStreamAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = fmt.Fprint(w, `{"type":"error","error":{"type":"rate_limit_error","message":"slow down"}}`)
	}))
	defer server.Close()

	client := NewClient(Config{APIKey: "k", Model: "m", MaxTokens: 10, AnthropicBaseURL: server.URL})
	_, err := client.Stream(context.Background(), StreamRequest{}, func(string) error { return nil })
	if err == nil {
		t.Fatal("se esperaba error")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %T %v", err, err)
	}
	if !apiErr.Retryable || apiErr.Type != "rate_limit_error" || apiErr.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("api error = %+v", apiErr)
	}
}
