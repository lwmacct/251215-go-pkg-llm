package anthropic

import (
	"testing"

	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm/core"
)

// ═══════════════════════════════════════════════════════════════════════════
// HandleEvent 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestEventHandler_HandleEvent_ContentBlockStart_ToolUse(t *testing.T) {
	handler := NewEventHandler()
	data := map[string]any{
		"index": float64(0),
		"content_block": map[string]any{
			"type": "tool_use",
			"id":   "toolu_123",
			"name": "get_weather",
		},
	}

	chunks, stop := handler.HandleEvent("content_block_start", data)

	if stop {
		t.Error("Expected stop=false for content_block_start")
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

	if chunk.ToolCall.ID != "toolu_123" {
		t.Errorf("Expected ID 'toolu_123', got %v", chunk.ToolCall.ID)
	}

	if chunk.ToolCall.Name != "get_weather" {
		t.Errorf("Expected Name 'get_weather', got %v", chunk.ToolCall.Name)
	}
}

func TestEventHandler_HandleEvent_ContentBlockDelta_TextDelta(t *testing.T) {
	handler := NewEventHandler()
	data := map[string]any{
		"delta": map[string]any{
			"type": "text_delta",
			"text": "Hello, world!",
		},
	}

	chunks, stop := handler.HandleEvent("content_block_delta", data)

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

func TestEventHandler_HandleEvent_ContentBlockDelta_InputJSONDelta(t *testing.T) {
	handler := NewEventHandler()
	data := map[string]any{
		"index": float64(0),
		"delta": map[string]any{
			"type":         "input_json_delta",
			"partial_json": `{"location":"`,
		},
	}

	chunks, stop := handler.HandleEvent("content_block_delta", data)

	if stop {
		t.Error("Expected stop=false for input_json_delta")
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

	if chunk.ToolCall.ArgumentsDelta != `{"location":"` {
		t.Errorf("Expected ArgumentsDelta, got %v", chunk.ToolCall.ArgumentsDelta)
	}
}

func TestEventHandler_HandleEvent_ContentBlockDelta_ThinkingDelta(t *testing.T) {
	handler := NewEventHandler()
	data := map[string]any{
		"delta": map[string]any{
			"type":     "thinking_delta",
			"thinking": "Let me analyze this step by step...",
		},
	}

	chunks, stop := handler.HandleEvent("content_block_delta", data)

	if stop {
		t.Error("Expected stop=false for thinking_delta")
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

	if chunk.Reasoning.ThoughtDelta != "Let me analyze this step by step..." {
		t.Errorf("Expected ThoughtDelta, got %v", chunk.Reasoning.ThoughtDelta)
	}
}

func TestEventHandler_HandleEvent_MessageDelta(t *testing.T) {
	handler := NewEventHandler()
	data := map[string]any{
		"delta": map[string]any{
			"stop_reason": "end_turn",
		},
	}

	chunks, stop := handler.HandleEvent("message_delta", data)

	if stop {
		t.Error("Expected stop=false for message_delta")
	}

	if len(chunks) != 1 {
		t.Fatalf("Expected 1 chunk, got %d", len(chunks))
	}

	chunk := chunks[0]

	if chunk.Type != "done" {
		t.Errorf("Expected type 'done', got %v", chunk.Type)
	}

	// stop_reason 应该被转换为标准 finish_reason
	if chunk.FinishReason != "stop" {
		t.Errorf("Expected FinishReason 'stop' (converted from end_turn), got %v", chunk.FinishReason)
	}
}

func TestEventHandler_HandleEvent_MessageStop(t *testing.T) {
	handler := NewEventHandler()
	data := map[string]any{}

	chunks, stop := handler.HandleEvent("message_stop", data)

	if stop {
		t.Error("Expected stop=false for message_stop")
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

func TestEventHandler_HandleEvent_IgnoredEvents(t *testing.T) {
	handler := NewEventHandler()

	ignoredEvents := []string{
		"message_start",
		"content_block_stop",
		"ping",
	}

	for _, event := range ignoredEvents {
		chunks, stop := handler.HandleEvent(event, map[string]any{})

		if stop {
			t.Errorf("Expected stop=false for %q", event)
		}

		if len(chunks) != 0 {
			t.Errorf("Expected 0 chunks for %q, got %d", event, len(chunks))
		}
	}
}

func TestEventHandler_HandleEvent_UnknownEvent(t *testing.T) {
	handler := NewEventHandler()
	data := map[string]any{
		"some_data": "value",
	}

	chunks, stop := handler.HandleEvent("unknown_event_type", data)

	if stop {
		t.Error("Expected stop=false for unknown event")
	}

	if len(chunks) != 0 {
		t.Errorf("Expected 0 chunks for unknown event, got %d", len(chunks))
	}
}

func TestEventHandler_HandleEvent_EmptyDelta(t *testing.T) {
	handler := NewEventHandler()
	data := map[string]any{
		"delta": map[string]any{},
	}

	chunks, stop := handler.HandleEvent("content_block_delta", data)

	if stop {
		t.Error("Expected stop=false")
	}

	if len(chunks) != 0 {
		t.Errorf("Expected 0 chunks for empty delta, got %d", len(chunks))
	}
}

func TestEventHandler_HandleEvent_EmptyText(t *testing.T) {
	handler := NewEventHandler()
	data := map[string]any{
		"delta": map[string]any{
			"type": "text_delta",
			"text": "",
		},
	}

	chunks, stop := handler.HandleEvent("content_block_delta", data)

	if stop {
		t.Error("Expected stop=false")
	}

	// 空字符串应该被忽略
	if len(chunks) != 0 {
		t.Errorf("Expected 0 chunks for empty text, got %d", len(chunks))
	}
}

func TestEventHandler_HandleEvent_EmptyPartialJSON(t *testing.T) {
	handler := NewEventHandler()
	data := map[string]any{
		"index": float64(0),
		"delta": map[string]any{
			"type":         "input_json_delta",
			"partial_json": "",
		},
	}

	chunks, stop := handler.HandleEvent("content_block_delta", data)

	if stop {
		t.Error("Expected stop=false")
	}

	// 空 JSON 应该被忽略
	if len(chunks) != 0 {
		t.Errorf("Expected 0 chunks for empty partial_json, got %d", len(chunks))
	}
}

func TestEventHandler_HandleEvent_EmptyThinking(t *testing.T) {
	handler := NewEventHandler()
	data := map[string]any{
		"delta": map[string]any{
			"type":     "thinking_delta",
			"thinking": "",
		},
	}

	chunks, stop := handler.HandleEvent("content_block_delta", data)

	if stop {
		t.Error("Expected stop=false")
	}

	// 空 thinking 应该被忽略
	if len(chunks) != 0 {
		t.Errorf("Expected 0 chunks for empty thinking, got %d", len(chunks))
	}
}

func TestEventHandler_HandleEvent_MessageDeltaNoStopReason(t *testing.T) {
	handler := NewEventHandler()
	data := map[string]any{
		"delta": map[string]any{
			"stop_reason": "",
		},
	}

	chunks, stop := handler.HandleEvent("message_delta", data)

	if stop {
		t.Error("Expected stop=false")
	}

	// 空 stop_reason 应该被忽略
	if len(chunks) != 0 {
		t.Errorf("Expected 0 chunks for empty stop_reason, got %d", len(chunks))
	}
}

func TestEventHandler_HandleEvent_ContentBlockStartNonToolUse(t *testing.T) {
	handler := NewEventHandler()
	data := map[string]any{
		"index": float64(0),
		"content_block": map[string]any{
			"type": "text",
			"text": "Some text",
		},
	}

	chunks, stop := handler.HandleEvent("content_block_start", data)

	if stop {
		t.Error("Expected stop=false")
	}

	// 非 tool_use 的 content_block_start 应该被忽略
	if len(chunks) != 0 {
		t.Errorf("Expected 0 chunks for non-tool_use content_block_start, got %d", len(chunks))
	}
}

func TestEventHandler_HandleEvent_MultipleIndexes(t *testing.T) {
	handler := NewEventHandler()

	// 第一个工具调用
	data1 := map[string]any{
		"index": float64(0),
		"content_block": map[string]any{
			"type": "tool_use",
			"id":   "toolu_1",
			"name": "tool_1",
		},
	}

	chunks1, _ := handler.HandleEvent("content_block_start", data1)

	if len(chunks1) != 1 {
		t.Fatalf("Expected 1 chunk for first tool call, got %d", len(chunks1))
	}

	if chunks1[0].ToolCall.Index != 0 {
		t.Errorf("Expected Index 0 for first tool call, got %d", chunks1[0].ToolCall.Index)
	}

	// 第二个工具调用
	data2 := map[string]any{
		"index": float64(1),
		"content_block": map[string]any{
			"type": "tool_use",
			"id":   "toolu_2",
			"name": "tool_2",
		},
	}

	chunks2, _ := handler.HandleEvent("content_block_start", data2)

	if len(chunks2) != 1 {
		t.Fatalf("Expected 1 chunk for second tool call, got %d", len(chunks2))
	}

	if chunks2[0].ToolCall.Index != 1 {
		t.Errorf("Expected Index 1 for second tool call, got %d", chunks2[0].ToolCall.Index)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// ShouldStopOnData 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestEventHandler_ShouldStopOnData(t *testing.T) {
	handler := NewEventHandler()

	testCases := []string{
		"",
		"[DONE]",
		"{}",
		"some data",
	}

	for _, tc := range testCases {
		if handler.ShouldStopOnData(tc) {
			t.Errorf("Expected ShouldStopOnData to always return false, got true for %q", tc)
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// 接口实现验证
// ═══════════════════════════════════════════════════════════════════════════

func TestEventHandler_ImplementsEventHandler(t *testing.T) {
	var _ core.EventHandler = (*EventHandler)(nil)
}
