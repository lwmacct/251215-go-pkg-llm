package anthropic

import (
	"testing"

	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm/core"
	"github.com/stretchr/testify/require"
)

// ═══════════════════════════════════════════════════════════════════════════
// ConvertToAPI 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestAdapter_ConvertToAPI_TextMessage(t *testing.T) {
	adapter := NewAdapter()
	messages := []llm.Message{
		{
			Role:    llm.RoleUser,
			Content: "Hello, Claude!",
		},
	}

	result := adapter.ConvertToAPI(messages)

	if len(result) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(result))
	}

	if result[0]["role"] != "user" {
		t.Errorf("Expected role 'user', got %v", result[0]["role"])
	}

	content, ok := result[0]["content"].([]map[string]any)
	if !ok {
		t.Fatalf("Expected content array, got %T", result[0]["content"])
	}

	if len(content) != 1 {
		t.Fatalf("Expected 1 content block, got %d", len(content))
	}

	if content[0]["type"] != "text" {
		t.Errorf("Expected type 'text', got %v", content[0]["type"])
	}

	if content[0]["text"] != "Hello, Claude!" {
		t.Errorf("Expected text 'Hello, Claude!', got %v", content[0]["text"])
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
					ID:   "toolu_123",
					Name: "get_weather",
					Input: map[string]any{
						"location": "Paris",
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

	content, ok := result[0]["content"].([]map[string]any)
	if !ok {
		t.Fatalf("Expected content array, got %T", result[0]["content"])
	}

	if len(content) != 2 {
		t.Fatalf("Expected 2 content blocks (text + tool_use), got %d", len(content))
	}

	// 第一个 block: TextBlock
	if content[0]["type"] != "text" {
		t.Errorf("Expected first block type 'text', got %v", content[0]["type"])
	}

	if content[0]["text"] != "Let me check the weather." {
		t.Errorf("Expected text, got %v", content[0]["text"])
	}

	// 第二个 block: ToolCall
	if content[1]["type"] != "tool_use" {
		t.Errorf("Expected second block type 'tool_use', got %v", content[1]["type"])
	}

	if content[1]["id"] != "toolu_123" {
		t.Errorf("Expected id 'toolu_123', got %v", content[1]["id"])
	}

	if content[1]["name"] != "get_weather" {
		t.Errorf("Expected name 'get_weather', got %v", content[1]["name"])
	}

	// ⚠️ 关键验证：参数必须是对象，不是 JSON 字符串
	input, ok := content[1]["input"].(map[string]any)
	if !ok {
		t.Fatalf("Expected input to be map[string]any, got %T", content[1]["input"])
	}

	if input["location"] != "Paris" {
		t.Errorf("Expected location 'Paris', got %v", input["location"])
	}

	if input["unit"] != "celsius" {
		t.Errorf("Expected unit 'celsius', got %v", input["unit"])
	}
}

func TestAdapter_ConvertToAPI_ToolResult(t *testing.T) {
	adapter := NewAdapter()
	messages := []llm.Message{
		{
			Role: llm.RoleUser,
			ContentBlocks: []llm.ContentBlock{
				&llm.ToolResultBlock{
					ToolUseID: "toolu_123",
					Content:   "Temperature: 22°C, Partly cloudy",
				},
			},
		},
	}

	result := adapter.ConvertToAPI(messages)

	if len(result) != 1 {
		t.Fatalf("Expected 1 message (ToolResult inlined), got %d", len(result))
	}

	if result[0]["role"] != "user" {
		t.Errorf("Expected role 'user', got %v", result[0]["role"])
	}

	content, ok := result[0]["content"].([]map[string]any)
	if !ok {
		t.Fatalf("Expected content array, got %T", result[0]["content"])
	}

	if len(content) != 1 {
		t.Fatalf("Expected 1 content block, got %d", len(content))
	}

	// ⚠️ 关键验证：ToolResult 内联在 content 数组中
	if content[0]["type"] != "tool_result" {
		t.Errorf("Expected type 'tool_result', got %v", content[0]["type"])
	}

	if content[0]["tool_use_id"] != "toolu_123" {
		t.Errorf("Expected tool_use_id 'toolu_123', got %v", content[0]["tool_use_id"])
	}

	if content[0]["content"] != "Temperature: 22°C, Partly cloudy" {
		t.Errorf("Expected content, got %v", content[0]["content"])
	}
}

func TestAdapter_ConvertToAPI_MultipleToolResults(t *testing.T) {
	adapter := NewAdapter()
	messages := []llm.Message{
		{
			Role: llm.RoleUser,
			ContentBlocks: []llm.ContentBlock{
				&llm.ToolResultBlock{
					ToolUseID: "toolu_1",
					Content:   "Result 1",
				},
				&llm.ToolResultBlock{
					ToolUseID: "toolu_2",
					Content:   "Result 2",
				},
			},
		},
	}

	result := adapter.ConvertToAPI(messages)

	if len(result) != 1 {
		t.Fatalf("Expected 1 message (multiple ToolResults inlined), got %d", len(result))
	}

	content, ok := result[0]["content"].([]map[string]any)
	if !ok {
		t.Fatalf("Expected content array, got %T", result[0]["content"])
	}

	// 两个 ToolResult 应该都在同一个消息的 content 数组中
	if len(content) != 2 {
		t.Fatalf("Expected 2 content blocks, got %d", len(content))
	}

	if content[0]["tool_use_id"] != "toolu_1" {
		t.Errorf("Expected first tool_use_id 'toolu_1', got %v", content[0]["tool_use_id"])
	}

	if content[1]["tool_use_id"] != "toolu_2" {
		t.Errorf("Expected second tool_use_id 'toolu_2', got %v", content[1]["tool_use_id"])
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
			Role:    llm.RoleUser,
			Content: "",
		},
	}

	result := adapter.ConvertToAPI(messages)

	// 空内容的消息应该被跳过
	if len(result) != 0 {
		t.Errorf("Expected 0 messages (empty content skipped), got %d", len(result))
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// ConvertFromAPI 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestAdapter_ConvertFromAPI_TextResponse(t *testing.T) {
	adapter := NewAdapter()
	apiResp := map[string]any{
		"content": []any{
			map[string]any{
				"type": "text",
				"text": "Hello! How can I assist you today?",
			},
		},
		"stop_reason": "end_turn",
	}

	msg, finishReason := adapter.ConvertFromAPI(apiResp)

	if msg.Role != llm.RoleAssistant {
		t.Errorf("Expected role assistant, got %v", msg.Role)
	}

	if msg.Content != "Hello! How can I assist you today?" {
		t.Errorf("Expected content, got %v", msg.Content)
	}

	if finishReason != "stop" {
		t.Errorf("Expected finish_reason 'stop' (converted from end_turn), got %v", finishReason)
	}
}

func TestAdapter_ConvertFromAPI_ToolUseResponse(t *testing.T) {
	adapter := NewAdapter()
	apiResp := map[string]any{
		"content": []any{
			map[string]any{
				"type": "text",
				"text": "Let me check that for you.",
			},
			map[string]any{
				"type": "tool_use",
				"id":   "toolu_xyz",
				"name": "get_weather",
				// ⚠️ 关键测试：Anthropic 返回直接对象
				"input": map[string]any{
					"location": "London",
					"unit":     "fahrenheit",
				},
			},
		},
		"stop_reason": "tool_use",
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

	// 第二个 block 应该是 ToolCall
	toolBlock, ok := msg.ContentBlocks[1].(*llm.ToolCall)
	if !ok {
		t.Fatalf("Expected ToolCall, got %T", msg.ContentBlocks[1])
	}

	if toolBlock.ID != "toolu_xyz" {
		t.Errorf("Expected ID 'toolu_xyz', got %v", toolBlock.ID)
	}

	if toolBlock.Name != "get_weather" {
		t.Errorf("Expected name 'get_weather', got %v", toolBlock.Name)
	}

	// ⚠️ 关键验证：参数应该是直接对象（无需反序列化）
	if toolBlock.Input["location"] != "London" {
		t.Errorf("Expected location 'London', got %v", toolBlock.Input["location"])
	}

	if toolBlock.Input["unit"] != "fahrenheit" {
		t.Errorf("Expected unit 'fahrenheit', got %v", toolBlock.Input["unit"])
	}

	if finishReason != "tool_calls" {
		t.Errorf("Expected finish_reason 'tool_calls' (converted from tool_use), got %v", finishReason)
	}

	// Content 字段应该被清空（多个 blocks）
	if msg.Content != "" {
		t.Errorf("Expected empty Content when using ContentBlocks, got %v", msg.Content)
	}
}

func TestAdapter_ConvertFromAPI_StopReasonMapping(t *testing.T) {
	adapter := NewAdapter()

	testCases := []struct {
		stopReason     string
		expectedFinish string
	}{
		{"end_turn", "stop"},
		{"max_tokens", "length"},
		{"tool_use", "tool_calls"},
		{"stop_sequence", "stop"},
		{"unknown_reason", "unknown_reason"},
	}

	for _, tc := range testCases {
		apiResp := map[string]any{
			"content": []any{
				map[string]any{
					"type": "text",
					"text": "Test",
				},
			},
			"stop_reason": tc.stopReason,
		}

		_, finishReason := adapter.ConvertFromAPI(apiResp)

		if finishReason != tc.expectedFinish {
			t.Errorf("Expected stop_reason %q to map to %q, got %q",
				tc.stopReason, tc.expectedFinish, finishReason)
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// ConvertUsage 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestAdapter_ConvertUsage_Basic(t *testing.T) {
	adapter := NewAdapter()
	apiResp := map[string]any{
		"usage": map[string]any{
			"input_tokens":  float64(100),
			"output_tokens": float64(50),
		},
	}

	usage := adapter.ConvertUsage(apiResp)

	require.NotNil(t, usage, "Expected usage, got nil")

	if usage.InputTokens != 100 {
		t.Errorf("Expected InputTokens 100, got %d", usage.InputTokens)
	}

	if usage.OutputTokens != 50 {
		t.Errorf("Expected OutputTokens 50, got %d", usage.OutputTokens)
	}

	// TotalTokens 应该被自动计算
	if usage.TotalTokens != 150 {
		t.Errorf("Expected TotalTokens 150 (auto-calculated), got %d", usage.TotalTokens)
	}
}

func TestAdapter_ConvertUsage_WithCachedTokens(t *testing.T) {
	adapter := NewAdapter()
	apiResp := map[string]any{
		"usage": map[string]any{
			"input_tokens":            float64(100),
			"output_tokens":           float64(50),
			"cache_read_input_tokens": float64(80),
		},
	}

	usage := adapter.ConvertUsage(apiResp)

	require.NotNil(t, usage, "Expected usage, got nil")

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

	if strategy != core.SystemSeparate {
		t.Errorf("Expected SystemSeparate, got %v", strategy)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// 接口实现验证
// ═══════════════════════════════════════════════════════════════════════════

func TestAdapter_ImplementsProtocolAdapter(t *testing.T) {
	var _ core.ProtocolAdapter = (*Adapter)(nil)
}
