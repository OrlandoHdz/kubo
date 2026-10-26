package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
)

type AgentEvent struct {
	Type       string
	Text       string
	Stage      string
	Tool       string
	Products   []ProductCard
	Escalation *EscalationEvent
}

type Emitter func(AgentEvent) error

type RunResult struct {
	Text       string
	StopReason string
	Usage      Usage
	ToolCalls  []ToolCallRecord
}

type Agent struct {
	streamer      Streamer
	registry      *ToolRegistry
	systemPrompt  string
	maxIterations int
	llmTimeout    time.Duration
}

func NewAgent(streamer Streamer, registry *ToolRegistry, systemPrompt string, cfg Config) *Agent {
	maxIterations := cfg.MaxIterations
	if maxIterations <= 0 {
		maxIterations = DefaultMaxIterations
	}
	llmTimeout := cfg.LLMTimeout
	if llmTimeout <= 0 {
		llmTimeout = DefaultLLMTimeout
	}
	return &Agent{
		streamer:      streamer,
		registry:      registry,
		systemPrompt:  systemPrompt,
		maxIterations: maxIterations,
		llmTimeout:    llmTimeout,
	}
}

// Run ejecuta el loop de tool use. Envía el texto a medida que llega y devuelve
// el texto completo, el uso de tokens y el registro de herramientas.
func (a *Agent) Run(
	ctx context.Context,
	history []Message,
	userMessage string,
	toolCtx ToolContext,
	emit Emitter,
) (RunResult, error) {
	if emit == nil {
		emit = func(AgentEvent) error { return nil }
	}

	messages := append([]Message(nil), history...)
	messages = appendMessage(messages, Message{
		Role:    "user",
		Content: []ContentBlock{{Type: "text", Text: userMessage}},
	})

	var (
		result RunResult
		text   strings.Builder
	)

	onText := func(delta string) error {
		text.WriteString(delta)
		result.Text = text.String()
		return emit(AgentEvent{Type: "text", Text: delta})
	}

	for iteration := 0; iteration < a.maxIterations; iteration++ {
		callCtx, cancel := context.WithTimeout(ctx, a.llmTimeout)
		res, err := a.streamer.Stream(callCtx, StreamRequest{
			System:   a.systemPrompt,
			Messages: messages,
			Tools:    a.registry.Definitions(),
		}, onText)
		cancel()

		if err != nil {
			return result, err
		}

		result.Usage.InputTokens += res.Usage.InputTokens
		result.Usage.OutputTokens += res.Usage.OutputTokens

		if len(res.Content) > 0 {
			messages = appendMessage(messages, Message{
				Role:    "assistant",
				Content: res.Content,
			})
		}

		if res.StopReason != "tool_use" {
			result.StopReason = res.StopReason
			if strings.TrimSpace(result.Text) == "" && len(result.ToolCalls) == 0 {
				return result, ErrEmptyResponse
			}
			return result, nil
		}

		var toolResults []ContentBlock
		for _, block := range res.Content {
			if block.Type != "tool_use" {
				continue
			}

			if err := emit(AgentEvent{Type: "status", Stage: "tool_running", Tool: block.Name}); err != nil {
				return result, err
			}

			started := time.Now()
			out, toolErr := a.registry.Execute(ctx, block.Name, block.Input, toolCtx)
			duration := time.Since(started)

			isError := toolErr != nil
			var content any
			if isError {
				userMessage := "La herramienta no pudo completar la consulta."
				var te *ToolError
				if errors.As(toolErr, &te) && te.UserMessage != "" {
					userMessage = te.UserMessage
				}
				content = map[string]any{
					"error":   "tool_error",
					"mensaje": userMessage,
				}
				log.Printf(
					"chat: tool=%s conversation=%s is_error=true duration_ms=%d error=%v",
					block.Name, toolCtx.ConversationID, duration.Milliseconds(), toolErr,
				)
			} else {
				log.Printf(
					"chat: tool=%s conversation=%s is_error=false duration_ms=%d",
					block.Name, toolCtx.ConversationID, duration.Milliseconds(),
				)
				if out.Content == nil {
					out.Content = map[string]any{"ok": true}
				}
				content = out.Content

				if len(out.Products) > 0 {
					if err := emit(AgentEvent{Type: "products", Products: out.Products}); err != nil {
						return result, err
					}
				}
				if out.Escalation != nil {
					if err := emit(AgentEvent{Type: "escalation", Escalation: out.Escalation}); err != nil {
						return result, err
					}
				}
			}

			if err := emit(AgentEvent{Type: "status", Stage: "tool_done", Tool: block.Name}); err != nil {
				return result, err
			}

			payload, err := json.Marshal(content)
			if err != nil {
				payload = []byte(`{"error":"tool_error","mensaje":"No se pudo serializar el resultado."}`)
				isError = true
			}

			result.ToolCalls = append(result.ToolCalls, ToolCallRecord{
				Name:       block.Name,
				Input:      block.Input,
				IsError:    isError,
				DurationMS: duration.Milliseconds(),
			})

			toolResults = append(toolResults, ContentBlock{
				Type:        "tool_result",
				ToolUseID:   block.ID,
				ToolContent: string(payload),
				IsError:     isError,
			})
		}

		if len(toolResults) == 0 {
			return result, fmt.Errorf("Claude pidió tool_use sin bloques de herramienta")
		}

		messages = appendMessage(messages, Message{
			Role:    "user",
			Content: toolResults,
		})
	}

	result.StopReason = "max_iterations"
	return result, ErrMaxIterations
}

func appendMessage(messages []Message, msg Message) []Message {
	if len(messages) == 0 || messages[len(messages)-1].Role != msg.Role {
		return append(messages, msg)
	}

	last := messages[len(messages)-1]
	merged := make([]ContentBlock, 0, len(last.Content)+len(msg.Content))
	merged = append(merged, last.Content...)
	merged = append(merged, msg.Content...)

	out := append([]Message(nil), messages...)
	out[len(out)-1] = Message{Role: msg.Role, Content: merged}
	return out
}
