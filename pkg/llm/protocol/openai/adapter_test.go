package openai

import (
	"encoding/json"
	"testing"

	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm/core"
)

// ═══════════════════════════════════════════════════════════════════════════
// ConvertToAPI 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestAdapter_ConvertToAPI_TextMessage(t *testing.T) {
	adapter := NewAdapter()
	messages := []llm.Message{
		{
			Role:    llm.RoleUser,
			Content: "Hello, world!",
		},
	}

	result := adapter.ConvertToAPI(messages)

	if len(result) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(result))
	}

	if result[0]["role"] != "user" {
		t.Errorf("Expected role 'user', got %v", result[0]["role"])
	}

	if result[0]["content"] != "Hello, world!" {
		t.Errorf("Expected content 'Hello, world!', got %v", result[0]["content"])
	}
}

func TestAdapter_ConvertToAPI_ToolUse(t *testing.T) {
	adapter := NewAdapter()
	messages := []llm.Message{
		{
			Role: llm.RoleAssistant,
			ContentBlocks: []llm.ContentBlock{
				&llm.TextBlock{Text: "Let me check the weather."},
				&llm.ToolCall{
					ID:   "call_123",
					Name: "get_weather",
					Input: map[string]any{
						"location": "San Francisco",
						"unit":     "celsius",
					},
				},
			},
		},
	}

	result := adapter.ConvertToAPI(messages)

	if len(result) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(result))
	}

	msg := result[0]

	// 检查文本内容
	if msg["content"] != "Let me check the weather." {
		t.Errorf("Expected text content, got %v", msg["content"])
	}

	// 检查工具调用
	toolCalls, ok := msg["tool_calls"].([]map[string]any)
	if !ok {
		t.Fatalf("Expected tool_calls array, got %T", msg["tool_calls"])
	}

	if len(toolCalls) != 1 {
		t.Fatalf("Expected 1 tool call, got %d", len(toolCalls))
	}

	tc := toolCalls[0]

	if tc["id"] != "call_123" {
		t.Errorf("Expected id 'call_123', got %v", tc["id"])
	}

	if tc["type"] != "function" {
		t.Errorf("Expected type 'function', got %v", tc["type"])
	}

	fn, ok := tc["function"].(map[string]any)
	if !ok {
		t.Fatalf("Expected function object, got %T", tc["function"])
	}

	if fn["name"] != "get_weather" {
		t.Errorf("Expected name 'get_weather', got %v", fn["name"])
	}

	// ⚠️ 关键验证：参数必须是 JSON 字符串
	argsStr, ok := fn["arguments"].(string)
	if !ok {
		t.Fatalf("Expected arguments to be string, got %T", fn["arguments"])
	}

	var args map[string]any
	if err := json.Unmarshal([]byte(argsStr), &args); err != nil {
		t.Fatalf("Failed to parse arguments JSON: %v", err)
	}

	if args["location"] != "San Francisco" {
		t.Errorf("Expected location 'San Francisco', got %v", args["location"])
	}

	if args["unit"] != "celsius" {
		t.Errorf("Expected unit 'celsius', got %v", args["unit"])
	}
}

func TestAdapter_ConvertToAPI_ToolResult(t *testing.T) {
	adapter := NewAdapter()
	messages := []llm.Message{
		{
			Role: llm.RoleUser,
			ContentBlocks: []llm.ContentBlock{
				&llm.ToolResultBlock{
					ToolUseID: "call_123",
					Content:   "Temperature: 18°C, Sunny",
				},
				&llm.ToolResultBlock{
					ToolUseID: "call_456",
					Content:   "Humidity: 65%",
				},
			},
		},
	}

	result := adapter.ConvertToAPI(messages)

	// ⚠️ 关键验证：ToolResult 必须展开为独立消息
	if len(result) != 2 {
		t.Fatalf("Expected 2 messages (expanded ToolResults), got %d", len(result))
	}

	// 第一条 ToolResult
	if result[0]["role"] != "tool" {
		t.Errorf("Expected role 'tool', got %v", result[0]["role"])
	}

	if result[0]["tool_call_id"] != "call_123" {
		t.Errorf("Expected tool_call_id 'call_123', got %v", result[0]["tool_call_id"])
	}

	if result[0]["content"] != "Temperature: 18°C, Sunny" {
		t.Errorf("Expected content 'Temperature: 18°C, Sunny', got %v", result[0]["content"])
	}

	// 第二条 ToolResult
	if result[1]["role"] != "tool" {
		t.Errorf("Expected role 'tool', got %v", result[1]["role"])
	}

	if result[1]["tool_call_id"] != "call_456" {
		t.Errorf("Expected tool_call_id 'call_456', got %v", result[1]["tool_call_id"])
	}

	if result[1]["content"] != "Humidity: 65%" {
		t.Errorf("Expected content 'Humidity: 65%%', got %v", result[1]["content"])
	}
}

func TestAdapter_ConvertToAPI_SkipSystemMessage(t *testing.T) {
	adapter := NewAdapter()
	messages := []llm.Message{
		{
			Role:    llm.RoleSystem,
			Content: "You are a helpful assistant.",
		},
		{
			Role:    llm.RoleUser,
			Content: "Hello",
		},
	}

	result := adapter.ConvertToAPI(messages)

	// 系统消息应该被跳过（由 Transformer 统一处理）
	if len(result) != 1 {
		t.Fatalf("Expected 1 message (system skipped), got %d", len(result))
	}

	if result[0]["role"] != "user" {
		t.Errorf("Expected role 'user', got %v", result[0]["role"])
	}
}

func TestAdapter_ConvertToAPI_EmptyContent(t *testing.T) {
	adapter := NewAdapter()
	messages := []llm.Message{
		{
			Role: llm.RoleAssistant,
			ContentBlocks: []llm.ContentBlock{
				&llm.ToolCall{
					ID:    "call_123",
					Name:  "get_time",
					Input: map[string]any{},
				},
			},
		},
	}

	result := adapter.ConvertToAPI(messages)

	if len(result) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(result))
	}

	// ⚠️ 关键验证：OpenAI 要求有 content 字段（即使为空）
	if _, exists := result[0]["content"]; !exists {
		t.Error("Expected content field to exist (even if empty)")
	}

	if result[0]["content"] != "" {
		t.Errorf("Expected empty content, got %v", result[0]["content"])
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// ConvertFromAPI 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestAdapter_ConvertFromAPI_TextResponse(t *testing.T) {
	adapter := NewAdapter()
	apiResp := map[string]any{
		"choices": []any{
			map[string]any{
				"message": map[string]any{
					"content": "Hello! How can I help you?",
				},
				"finish_reason": "stop",
			},
		},
	}

	msg, finishReason := adapter.ConvertFromAPI(apiResp)

	if msg.Role != llm.RoleAssistant {
		t.Errorf("Expected role assistant, got %v", msg.Role)
	}

	if msg.Content != "Hello! How can I help you?" {
		t.Errorf("Expected content, got %v", msg.Content)
	}

	if finishReason != "stop" {
		t.Errorf("Expected finish_reason 'stop', got %v", finishReason)
	}
}

func TestAdapter_ConvertFromAPI_ToolCallResponse(t *testing.T) {
	adapter := NewAdapter()
	apiResp := map[string]any{
		"choices": []any{
			map[string]any{
				"message": map[string]any{
					"content": "Let me check that for you.",
					"tool_calls": []any{
						map[string]any{
							"id":   "call_abc",
							"type": "function",
							"function": map[string]any{
								"name": "get_weather",
								// ⚠️ 关键测试：OpenAI 返回 JSON 字符串
								"arguments": `{"location":"Tokyo","unit":"celsius"}`,
							},
						},
					},
				},
				"finish_reason": "tool_calls",
			},
		},
	}

	msg, finishReason := adapter.ConvertFromAPI(apiResp)

	if msg.Role != llm.RoleAssistant {
		t.Errorf("Expected role assistant, got %v", msg.Role)
	}

	if len(msg.ContentBlocks) != 2 {
		t.Fatalf("Expected 2 content blocks (text + tool_use), got %d", len(msg.ContentBlocks))
	}

	// 第一个 block 应该是 TextBlock
	textBlock, ok := msg.ContentBlocks[0].(*llm.TextBlock)
	if !ok {
		t.Fatalf("Expected TextBlock, got %T", msg.ContentBlocks[0])
	}

	if textBlock.Text != "Let me check that for you." {
		t.Errorf("Expected text content, got %v", textBlock.Text)
	}

	// 第二个 block 应该是 ToolUseBlock
	toolBlock, ok := msg.ContentBlocks[1].(*llm.ToolCall)
	if !ok {
		t.Fatalf("Expected ToolUseBlock, got %T", msg.ContentBlocks[1])
	}

	if toolBlock.ID != "call_abc" {
		t.Errorf("Expected ID 'call_abc', got %v", toolBlock.ID)
	}

	if toolBlock.Name != "get_weather" {
		t.Errorf("Expected name 'get_weather', got %v", toolBlock.Name)
	}

	// ⚠️ 关键验证：参数应该被反序列化为 map
	if toolBlock.Input["location"] != "Tokyo" {
		t.Errorf("Expected location 'Tokyo', got %v", toolBlock.Input["location"])
	}

	if toolBlock.Input["unit"] != "celsius" {
		t.Errorf("Expected unit 'celsius', got %v", toolBlock.Input["unit"])
	}

	if finishReason != "tool_calls" {
		t.Errorf("Expected finish_reason 'tool_calls', got %v", finishReason)
	}

	// Content 字段应该被清空（使用 ContentBlocks）
	if msg.Content != "" {
		t.Errorf("Expected empty Content when using ContentBlocks, got %v", msg.Content)
	}
}

func TestAdapter_ConvertFromAPI_EmptyChoices(t *testing.T) {
	adapter := NewAdapter()
	apiResp := map[string]any{
		"choices": []any{},
	}

	msg, finishReason := adapter.ConvertFromAPI(apiResp)

	if msg.Role != llm.RoleAssistant {
		t.Errorf("Expected role assistant, got %v", msg.Role)
	}

	if finishReason != "" {
		t.Errorf("Expected empty finish_reason, got %v", finishReason)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// ConvertUsage 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestAdapter_ConvertUsage_Basic(t *testing.T) {
	adapter := NewAdapter()
	apiResp := map[string]any{
		"usage": map[string]any{
			"prompt_tokens":     float64(100),
			"completion_tokens": float64(50),
			"total_tokens":      float64(150),
		},
	}

	usage := adapter.ConvertUsage(apiResp)

	if usage == nil {
		t.Fatal("Expected usage, got nil")
	}

	if usage.InputTokens != 100 {
		t.Errorf("Expected InputTokens 100, got %d", usage.InputTokens)
	}

	if usage.OutputTokens != 50 {
		t.Errorf("Expected OutputTokens 50, got %d", usage.OutputTokens)
	}

	if usage.TotalTokens != 150 {
		t.Errorf("Expected TotalTokens 150, got %d", usage.TotalTokens)
	}
}

func TestAdapter_ConvertUsage_WithReasoningTokens(t *testing.T) {
	adapter := NewAdapter()
	apiResp := map[string]any{
		"usage": map[string]any{
			"prompt_tokens":     float64(100),
			"completion_tokens": float64(50),
			"total_tokens":      float64(200),
			"completion_tokens_details": map[string]any{
				"reasoning_tokens": float64(150),
			},
		},
	}

	usage := adapter.ConvertUsage(apiResp)

	if usage == nil {
		t.Fatal("Expected usage, got nil")
	}

	if usage.ReasoningTokens != 150 {
		t.Errorf("Expected ReasoningTokens 150, got %d", usage.ReasoningTokens)
	}
}

func TestAdapter_ConvertUsage_WithCachedTokens(t *testing.T) {
	adapter := NewAdapter()
	apiResp := map[string]any{
		"usage": map[string]any{
			"prompt_tokens":     float64(100),
			"completion_tokens": float64(50),
			"total_tokens":      float64(150),
			"prompt_tokens_details": map[string]any{
				"cached_tokens": float64(80),
			},
		},
	}

	usage := adapter.ConvertUsage(apiResp)

	if usage == nil {
		t.Fatal("Expected usage, got nil")
	}

	if usage.CachedTokens != 80 {
		t.Errorf("Expected CachedTokens 80, got %d", usage.CachedTokens)
	}
}

func TestAdapter_ConvertUsage_NoUsage(t *testing.T) {
	adapter := NewAdapter()
	apiResp := map[string]any{}

	usage := adapter.ConvertUsage(apiResp)

	if usage != nil {
		t.Errorf("Expected nil usage, got %+v", usage)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// GetSystemMessageHandling 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestAdapter_GetSystemMessageHandling(t *testing.T) {
	adapter := NewAdapter()

	strategy := adapter.GetSystemMessageHandling()

	if strategy != core.SystemInline {
		t.Errorf("Expected SystemInline, got %v", strategy)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// 接口实现验证
// ═══════════════════════════════════════════════════════════════════════════

func TestAdapter_ImplementsProtocolAdapter(t *testing.T) {
	var _ core.ProtocolAdapter = (*Adapter)(nil)
}
