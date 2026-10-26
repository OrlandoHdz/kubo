package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

type mockStreamer struct {
	calls     int
	requests  []StreamRequest
	responses []StreamResult
	deltas    [][]string
	errs      []error
}

func (m *mockStreamer) Stream(ctx context.Context, req StreamRequest, onText func(string) error) (StreamResult, error) {
	idx := m.calls
	m.calls++
	m.requests = append(m.requests, req)

	if idx < len(m.errs) && m.errs[idx] != nil {
		return StreamResult{}, m.errs[idx]
	}

	if idx < len(m.deltas) {
		for _, delta := range m.deltas[idx] {
			if err := onText(delta); err != nil {
				return StreamResult{}, err
			}
		}
	}

	if len(m.responses) == 0 {
		return StreamResult{}, fmt.Errorf("mock sin respuestas")
	}
	if idx >= len(m.responses) {
		return m.responses[len(m.responses)-1], nil
	}
	return m.responses[idx], nil
}

func testToolRegistry(handler func(context.Context, json.RawMessage, ToolContext) (ToolOutput, error)) *ToolRegistry {
	return &ToolRegistry{
		tools: map[string]Tool{
			"buscar_productos": {
				Definition: ToolDefinition{Name: "buscar_productos"},
				Handler:    handler,
			},
		},
		order: []string{"buscar_productos"},
	}
}

func TestAgentTextOnly(t *testing.T) {
	streamer := &mockStreamer{
		deltas: [][]string{{"Hola", " mundo"}},
		responses: []StreamResult{{
			Content:    []ContentBlock{{Type: "text", Text: "Hola mundo"}},
			StopReason: "end_turn",
			Usage:      Usage{InputTokens: 12, OutputTokens: 3},
		}},
	}

	agent := NewAgent(streamer, &ToolRegistry{tools: map[string]Tool{}}, "prompt", Config{
		MaxIterations: 6,
		LLMTimeout:    time.Second,
	})

	var events []AgentEvent
	result, err := agent.Run(context.Background(), nil, "Hola", ToolContext{}, func(ev AgentEvent) error {
		events = append(events, ev)
		return nil
	})
	if err != nil {
		t.Fatalf("Run devolvió error: %v", err)
	}
	if streamer.calls != 1 {
		t.Fatalf("llamadas = %d, esperado 1", streamer.calls)
	}
	if result.Text != "Hola mundo" {
		t.Fatalf("texto = %q", result.Text)
	}
	if result.Usage.InputTokens != 12 || result.Usage.OutputTokens != 3 {
		t.Fatalf("usage = %+v", result.Usage)
	}
	if len(events) != 2 || events[0].Type != "text" || events[1].Type != "text" {
		t.Fatalf("eventos = %+v", events)
	}
	if got := streamer.requests[0].Messages; len(got) != 1 || got[0].Role != "user" {
		t.Fatalf("mensajes iniciales = %+v", got)
	}
}

func TestAgentToolUseThenText(t *testing.T) {
	toolCalls := 0
	registry := testToolRegistry(func(ctx context.Context, args json.RawMessage, tc ToolContext) (ToolOutput, error) {
		toolCalls++
		if tc.ConversationID != "conv-1" {
			t.Fatalf("conversation id = %q", tc.ConversationID)
		}
		return ToolOutput{
			Content:  map[string]any{"productos": []string{"6013"}},
			Products: []ProductCard{{ID: 1, SKU: "E6013-18"}},
		}, nil
	})

	streamer := &mockStreamer{
		deltas: [][]string{{}, {"Encontré un producto."}},
		responses: []StreamResult{
			{
				Content: []ContentBlock{{
					Type:  "tool_use",
					ID:    "tu_1",
					Name:  "buscar_productos",
					Input: json.RawMessage(`{"consulta":"6013"}`),
				}},
				StopReason: "tool_use",
				Usage:      Usage{InputTokens: 20, OutputTokens: 5},
			},
			{
				Content:    []ContentBlock{{Type: "text", Text: "Encontré un producto."}},
				StopReason: "end_turn",
				Usage:      Usage{InputTokens: 30, OutputTokens: 7},
			},
		},
	}

	agent := NewAgent(streamer, registry, "prompt", Config{MaxIterations: 6, LLMTimeout: time.Second})
	var events []AgentEvent
	result, err := agent.Run(context.Background(), nil, "Busca 6013", ToolContext{ConversationID: "conv-1"}, func(ev AgentEvent) error {
		events = append(events, ev)
		return nil
	})
	if err != nil {
		t.Fatalf("Run devolvió error: %v", err)
	}
	if toolCalls != 1 {
		t.Fatalf("tool calls = %d", toolCalls)
	}
	if streamer.calls != 2 {
		t.Fatalf("llamadas = %d, esperado 2", streamer.calls)
	}
	if len(result.ToolCalls) != 1 || result.ToolCalls[0].Name != "buscar_productos" || result.ToolCalls[0].IsError {
		t.Fatalf("tool calls = %+v", result.ToolCalls)
	}
	if result.Usage.InputTokens != 50 || result.Usage.OutputTokens != 12 {
		t.Fatalf("usage = %+v", result.Usage)
	}

	foundProducts := false
	for _, ev := range events {
		if ev.Type == "products" && len(ev.Products) == 1 {
			foundProducts = true
		}
	}
	if !foundProducts {
		t.Fatalf("no se emitió evento products: %+v", events)
	}

	second := streamer.requests[1].Messages
	if len(second) != 3 {
		t.Fatalf("mensajes segunda llamada = %d", len(second))
	}
	last := second[2]
	if last.Role != "user" || len(last.Content) != 1 || last.Content[0].Type != "tool_result" {
		t.Fatalf("tool_result = %+v", last)
	}
	if last.Content[0].ToolUseID != "tu_1" || last.Content[0].IsError {
		t.Fatalf("tool_result contenido = %+v", last.Content[0])
	}
}

func TestAgentToolErrorContinues(t *testing.T) {
	registry := testToolRegistry(func(ctx context.Context, args json.RawMessage, tc ToolContext) (ToolOutput, error) {
		return ToolOutput{}, &ToolError{UserMessage: "No encontré el producto."}
	})

	streamer := &mockStreamer{
		responses: []StreamResult{
			{
				Content:    []ContentBlock{{Type: "tool_use", ID: "tu_err", Name: "buscar_productos", Input: json.RawMessage(`{}`)}},
				StopReason: "tool_use",
			},
			{
				Content:    []ContentBlock{{Type: "text", Text: "No lo encontré, ¿me das más datos?"}},
				StopReason: "end_turn",
			},
		},
	}

	agent := NewAgent(streamer, registry, "prompt", Config{MaxIterations: 6, LLMTimeout: time.Second})
	result, err := agent.Run(context.Background(), nil, "Busca algo", ToolContext{}, func(AgentEvent) error { return nil })
	if err != nil {
		t.Fatalf("Run devolvió error: %v", err)
	}
	if len(result.ToolCalls) != 1 || !result.ToolCalls[0].IsError {
		t.Fatalf("tool calls = %+v", result.ToolCalls)
	}
	second := streamer.requests[1].Messages
	last := second[len(second)-1]
	if !last.Content[0].IsError {
		t.Fatalf("se esperaba is_error=true: %+v", last.Content[0])
	}
	if !strings.Contains(last.Content[0].ToolContent, "No encontré") {
		t.Fatalf("contenido = %q", last.Content[0].ToolContent)
	}
}

func TestAgentMaxIterations(t *testing.T) {
	registry := testToolRegistry(func(ctx context.Context, args json.RawMessage, tc ToolContext) (ToolOutput, error) {
		return ToolOutput{Content: map[string]any{"ok": true}}, nil
	})

	streamer := &mockStreamer{
		responses: []StreamResult{{
			Content:    []ContentBlock{{Type: "tool_use", ID: "tu_loop", Name: "buscar_productos", Input: json.RawMessage(`{}`)}},
			StopReason: "tool_use",
		}},
	}

	agent := NewAgent(streamer, registry, "prompt", Config{MaxIterations: 6, LLMTimeout: time.Second})
	_, err := agent.Run(context.Background(), nil, "Loop", ToolContext{}, func(AgentEvent) error { return nil })
	if !errors.Is(err, ErrMaxIterations) {
		t.Fatalf("error = %v, esperado ErrMaxIterations", err)
	}
	if streamer.calls != 6 {
		t.Fatalf("llamadas = %d, esperado 6", streamer.calls)
	}
}

func TestAgentEmptyResponse(t *testing.T) {
	streamer := &mockStreamer{
		responses: []StreamResult{{
			Content:    []ContentBlock{{Type: "thinking", Thinking: "", Signature: "firma"}},
			StopReason: "end_turn",
		}},
	}
	agent := NewAgent(streamer, &ToolRegistry{tools: map[string]Tool{}}, "prompt", Config{
		MaxIterations: 6,
		LLMTimeout:    time.Second,
	})
	_, err := agent.Run(context.Background(), nil, "Hola", ToolContext{}, func(AgentEvent) error { return nil })
	if !errors.Is(err, ErrEmptyResponse) {
		t.Fatalf("error = %v, esperado ErrEmptyResponse", err)
	}
}

func TestAgentEmitterErrorStops(t *testing.T) {
	streamer := &mockStreamer{
		deltas:    [][]string{{"Hola"}},
		responses: []StreamResult{{Content: []ContentBlock{{Type: "text", Text: "Hola"}}, StopReason: "end_turn"}},
	}
	agent := NewAgent(streamer, &ToolRegistry{tools: map[string]Tool{}}, "prompt", Config{MaxIterations: 6, LLMTimeout: time.Second})
	expected := errors.New("cliente desconectado")
	_, err := agent.Run(context.Background(), nil, "Hola", ToolContext{}, func(AgentEvent) error {
		return expected
	})
	if !errors.Is(err, expected) {
		t.Fatalf("error = %v", err)
	}
}
