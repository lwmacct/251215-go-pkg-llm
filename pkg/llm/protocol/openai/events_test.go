package openai

import (
	"testing"

	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
)

// ═══════════════════════════════════════════════════════════════════════════
// HandleEvent 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestEventHandler_HandleEvent_TextDelta(t *testing.T) {
	handler := NewEventHandler()
	data := map[string]any{
		"choices": []any{
			map[string]any{
				"delta": map[string]any{
					"content": "Hello, world!",
				},
			},
		},
	}

	chunks, stop := handler.HandleEvent("", data)

	if stop {
		t.Error("Expected stop=false for text delta")
	}

	if len(chunks) != 1 {
		t.Fatalf("Expected 1 chunk, got %d", len(chunks))
	}

	chunk := chunks[0]

	if chunk.Type != "text" {
		t.Errorf("Expected type 'text', got %v", chunk.Type)
	}

	if chunk.TextDelta != "Hello, world!" {
		t.Errorf("Expected TextDelta 'Hello, world!', got %v", chunk.TextDelta)
	}
}

func TestEventHandler_HandleEvent_ReasoningContent(t *testing.T) {
	handler := NewEventHandler()
	data := map[string]any{
		"choices": []any{
			map[string]any{
				"delta": map[string]any{
					"reasoning_content": "Let me think about this...",
				},
			},
		},
	}

	chunks, stop := handler.HandleEvent("", data)

	if stop {
		t.Error("Expected stop=false for reasoning delta")
	}

	if len(chunks) != 1 {
		t.Fatalf("Expected 1 chunk, got %d", len(chunks))
	}

	chunk := chunks[0]

	if chunk.Type != "reasoning" {
		t.Errorf("Expected type 'reasoning', got %v", chunk.Type)
	}

	if chunk.Reasoning == nil {
		t.Fatal("Expected Reasoning to be non-nil")
	}

	if chunk.Reasoning.ThoughtDelta != "Let me think about this..." {
		t.Errorf("Expected ThoughtDelta, got %v", chunk.Reasoning.ThoughtDelta)
	}
}

func TestEventHandler_HandleEvent_ToolCallDelta(t *testing.T) {
	handler := NewEventHandler()
	data := map[string]any{
		"choices": []any{
			map[string]any{
				"delta": map[string]any{
					"tool_calls": []any{
						map[string]any{
							"index": float64(0),
							"id":    "call_123",
							"function": map[string]any{
								"name":      "get_weather",
								"arguments": `{"location":"`,
							},
						},
					},
				},
			},
		},
	}

	chunks, stop := handler.HandleEvent("", data)

	if stop {
		t.Error("Expected stop=false for tool call delta")
	}

	if len(chunks) != 1 {
		t.Fatalf("Expected 1 chunk, got %d", len(chunks))
	}

	chunk := chunks[0]

	if chunk.Type != "tool_call" {
		t.Errorf("Expected type 'tool_call', got %v", chunk.Type)
	}

	if chunk.ToolCall == nil {
		t.Fatal("Expected ToolCall to be non-nil")
	}

	if chunk.ToolCall.Index != 0 {
		t.Errorf("Expected Index 0, got %d", chunk.ToolCall.Index)
	}

	if chunk.ToolCall.ID != "call_123" {
		t.Errorf("Expected ID 'call_123', got %v", chunk.ToolCall.ID)
	}

	if chunk.ToolCall.Name != "get_weather" {
		t.Errorf("Expected Name 'get_weather', got %v", chunk.ToolCall.Name)
	}

	if chunk.ToolCall.ArgumentsDelta != `{"location":"` {
		t.Errorf("Expected ArgumentsDelta, got %v", chunk.ToolCall.ArgumentsDelta)
	}
}

func TestEventHandler_HandleEvent_MultipleChunks(t *testing.T) {
	handler := NewEventHandler()
	data := map[string]any{
		"choices": []any{
			map[string]any{
				"delta": map[string]any{
					"content":           "Hello",
					"reasoning_content": "Thinking...",
				},
			},
		},
	}

	chunks, stop := handler.HandleEvent("", data)

	if stop {
		t.Error("Expected stop=false")
	}

	// 应该产生 2 个 chunks（文本 + 推理）
	if len(chunks) != 2 {
		t.Fatalf("Expected 2 chunks, got %d", len(chunks))
	}

	// 第一个应该是文本
	if chunks[0].Type != "text" {
		t.Errorf("Expected first chunk type 'text', got %v", chunks[0].Type)
	}

	// 第二个应该是推理
	if chunks[1].Type != "reasoning" {
		t.Errorf("Expected second chunk type 'reasoning', got %v", chunks[1].Type)
	}
}

func TestEventHandler_HandleEvent_FinishReason(t *testing.T) {
	handler := NewEventHandler()
	data := map[string]any{
		"choices": []any{
			map[string]any{
				"finish_reason": "stop",
			},
		},
	}

	chunks, stop := handler.HandleEvent("", data)

	if stop {
		t.Error("Expected stop=false (finish_reason doesn't stop parsing)")
	}

	if len(chunks) != 1 {
		t.Fatalf("Expected 1 chunk, got %d", len(chunks))
	}

	chunk := chunks[0]

	if chunk.Type != "done" {
		t.Errorf("Expected type 'done', got %v", chunk.Type)
	}

	if chunk.FinishReason != "stop" {
		t.Errorf("Expected FinishReason 'stop', got %v", chunk.FinishReason)
	}
}

func TestEventHandler_HandleEvent_EmptyChoices(t *testing.T) {
	handler := NewEventHandler()
	data := map[string]any{
		"choices": []any{},
	}

	chunks, stop := handler.HandleEvent("", data)

	if stop {
		t.Error("Expected stop=false")
	}

	if len(chunks) != 0 {
		t.Errorf("Expected 0 chunks for empty choices, got %d", len(chunks))
	}
}

func TestEventHandler_HandleEvent_EmptyDelta(t *testing.T) {
	handler := NewEventHandler()
	data := map[string]any{
		"choices": []any{
			map[string]any{
				"delta": map[string]any{},
			},
		},
	}

	chunks, stop := handler.HandleEvent("", data)

	if stop {
		t.Error("Expected stop=false")
	}

	if len(chunks) != 0 {
		t.Errorf("Expected 0 chunks for empty delta, got %d", len(chunks))
	}
}

func TestEventHandler_HandleEvent_EmptyContent(t *testing.T) {
	handler := NewEventHandler()
	data := map[string]any{
		"choices": []any{
			map[string]any{
				"delta": map[string]any{
					"content": "",
				},
			},
		},
	}

	chunks, stop := handler.HandleEvent("", data)

	if stop {
		t.Error("Expected stop=false")
	}

	// 空字符串应该被忽略
	if len(chunks) != 0 {
		t.Errorf("Expected 0 chunks for empty content, got %d", len(chunks))
	}
}

func TestEventHandler_HandleEvent_MultipleToolCalls(t *testing.T) {
	handler := NewEventHandler()
	data := map[string]any{
		"choices": []any{
			map[string]any{
				"delta": map[string]any{
					"tool_calls": []any{
						map[string]any{
							"index": float64(0),
							"id":    "call_1",
							"function": map[string]any{
								"name": "tool_1",
							},
						},
						map[string]any{
							"index": float64(1),
							"id":    "call_2",
							"function": map[string]any{
								"name": "tool_2",
							},
						},
					},
				},
			},
		},
	}

	chunks, stop := handler.HandleEvent("", data)

	if stop {
		t.Error("Expected stop=false")
	}

	if len(chunks) != 2 {
		t.Fatalf("Expected 2 chunks (2 tool calls), got %d", len(chunks))
	}

	// 验证第一个工具调用
	if chunks[0].ToolCall.Index != 0 {
		t.Errorf("Expected first tool call index 0, got %d", chunks[0].ToolCall.Index)
	}

	if chunks[0].ToolCall.ID != "call_1" {
		t.Errorf("Expected first tool call ID 'call_1', got %v", chunks[0].ToolCall.ID)
	}

	// 验证第二个工具调用
	if chunks[1].ToolCall.Index != 1 {
		t.Errorf("Expected second tool call index 1, got %d", chunks[1].ToolCall.Index)
	}

	if chunks[1].ToolCall.ID != "call_2" {
		t.Errorf("Expected second tool call ID 'call_2', got %v", chunks[1].ToolCall.ID)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// ShouldStopOnData 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestEventHandler_ShouldStopOnData_DONE(t *testing.T) {
	handler := NewEventHandler()

	if !handler.ShouldStopOnData("[DONE]") {
		t.Error("Expected ShouldStopOnData to return true for '[DONE]'")
	}
}

func TestEventHandler_ShouldStopOnData_NotDONE(t *testing.T) {
	handler := NewEventHandler()

	testCases := []string{
		"",
		"{}",
		"[DONE",
		"DONE]",
		"done",
		"[done]",
	}

	for _, tc := range testCases {
		if handler.ShouldStopOnData(tc) {
			t.Errorf("Expected ShouldStopOnData to return false for %q", tc)
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// 边界情况测试
// ═══════════════════════════════════════════════════════════════════════════

func TestEventHandler_HandleEvent_MissingFunction(t *testing.T) {
	handler := NewEventHandler()
	data := map[string]any{
		"choices": []any{
			map[string]any{
				"delta": map[string]any{
					"tool_calls": []any{
						map[string]any{
							"index": float64(0),
							"id":    "call_123",
							// 缺少 function 字段
						},
					},
				},
			},
		},
	}

	chunks, stop := handler.HandleEvent("", data)

	if stop {
		t.Error("Expected stop=false")
	}

	// 即使缺少 function 字段，也应该产生一个 chunk
	if len(chunks) != 1 {
		t.Fatalf("Expected 1 chunk, got %d", len(chunks))
	}

	if chunks[0].ToolCall.ID != "call_123" {
		t.Errorf("Expected ID 'call_123', got %v", chunks[0].ToolCall.ID)
	}

	// Name 应该为空
	if chunks[0].ToolCall.Name != "" {
		t.Errorf("Expected empty Name, got %v", chunks[0].ToolCall.Name)
	}
}

func TestEventHandler_HandleEvent_EventTypeIgnored(t *testing.T) {
	handler := NewEventHandler()
	data := map[string]any{
		"choices": []any{
			map[string]any{
				"delta": map[string]any{
					"content": "Test",
				},
			},
		},
	}

	// OpenAI 不使用 eventType，应该被忽略
	chunks1, _ := handler.HandleEvent("", data)
	chunks2, _ := handler.HandleEvent("some_event_type", data)

	if len(chunks1) != len(chunks2) {
		t.Error("Expected eventType to be ignored")
	}

	if chunks1[0].TextDelta != chunks2[0].TextDelta {
		t.Error("Expected same result regardless of eventType")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// 接口实现验证
// ═══════════════════════════════════════════════════════════════════════════

func TestEventHandler_ImplementsEventHandler(t *testing.T) {
	var _ llm.Event // 确保类型存在
}
