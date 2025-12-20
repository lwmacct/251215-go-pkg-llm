package gemini

import (
	"testing"

	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm/core"
	"github.com/stretchr/testify/assert"
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
			Content: "Hello, Gemini!",
		},
	}

	result := adapter.ConvertToAPI(messages)

	require.Len(t, result, 1, "Expected 1 message")

	// ⚠️ Gemini 使用 Content{Role, Parts[]} 格式
	assert.Equal(t, "user", result[0]["role"], "Role should be 'user'")

	parts, ok := result[0]["parts"].([]map[string]any)
	require.True(t, ok, "Expected parts array")
	require.Len(t, parts, 1, "Expected 1 part")

	assert.Equal(t, "Hello, Gemini!", parts[0]["text"], "Text content should match")
}

func TestAdapter_ConvertToAPI_RoleMapping(t *testing.T) {
	adapter := NewAdapter()

	testCases := []struct {
		inputRole    llm.Role
		expectedRole string
	}{
		{llm.RoleUser, "user"},
		{llm.RoleAssistant, "model"}, // ⚠️ Gemini 使用 "model" 而非 "assistant"
	}

	for _, tc := range testCases {
		messages := []llm.Message{
			{Role: tc.inputRole, Content: "test"},
		}
		result := adapter.ConvertToAPI(messages)
		assert.Equal(t, tc.expectedRole, result[0]["role"],
			"Role %s should map to %s", tc.inputRole, tc.expectedRole)
	}
}

func TestAdapter_ConvertToAPI_ToolCall(t *testing.T) {
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
						"location": "Tokyo",
						"unit":     "celsius",
					},
				},
			},
		},
	}

	result := adapter.ConvertToAPI(messages)

	require.Len(t, result, 1)
	assert.Equal(t, "model", result[0]["role"])

	parts, ok := result[0]["parts"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, parts, 2, "Expected 2 parts (text + functionCall)")

	// 第一个 Part: 文本
	assert.Equal(t, "Let me check the weather.", parts[0]["text"])

	// 第二个 Part: functionCall
	fc, ok := parts[1]["functionCall"].(map[string]any)
	require.True(t, ok, "Expected functionCall part")

	assert.Equal(t, "get_weather", fc["name"])

	// ⚠️ 关键验证：参数是直接对象（不是 JSON 字符串）
	args, ok := fc["args"].(map[string]any)
	require.True(t, ok, "args should be map[string]any, not JSON string")
	assert.Equal(t, "Tokyo", args["location"])
	assert.Equal(t, "celsius", args["unit"])
}

func TestAdapter_ConvertToAPI_ToolResult(t *testing.T) {
	adapter := NewAdapter()
	messages := []llm.Message{
		{
			Role: llm.RoleUser,
			ContentBlocks: []llm.ContentBlock{
				&llm.ToolResultBlock{
					ToolUseID: "get_weather",
					Content:   "Temperature: 25°C, Sunny",
				},
			},
		},
	}

	result := adapter.ConvertToAPI(messages)

	require.Len(t, result, 1)

	parts, ok := result[0]["parts"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, parts, 1)

	// ⚠️ Gemini 使用 functionResponse 格式
	fr, ok := parts[0]["functionResponse"].(map[string]any)
	require.True(t, ok, "Expected functionResponse part")

	assert.Equal(t, "get_weather", fr["name"]) // ToolUseID 作为函数名

	response, ok := fr["response"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Temperature: 25°C, Sunny", response["content"])
}

func TestAdapter_ConvertToAPI_ThinkingBlock(t *testing.T) {
	adapter := NewAdapter()
	messages := []llm.Message{
		{
			Role: llm.RoleAssistant,
			ContentBlocks: []llm.ContentBlock{
				&llm.ThinkingBlock{Thinking: "Let me think about this..."},
			},
		},
	}

	result := adapter.ConvertToAPI(messages)

	require.Len(t, result, 1)

	parts, ok := result[0]["parts"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, parts, 1)

	// ⚠️ Gemini 的 thinking 内容标记为 thought: true
	assert.Equal(t, "Let me think about this...", parts[0]["text"])
	assert.Equal(t, true, parts[0]["thought"])
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

	// 系统消息应该被跳过（由 Transformer 统一处理，传递到 systemInstruction）
	require.Len(t, result, 1, "System message should be skipped")
	assert.Equal(t, "user", result[0]["role"])
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

	require.Len(t, result, 1)
	// 空内容消息应该没有 parts 或 parts 为空
	parts, _ := result[0]["parts"].([]map[string]any)
	assert.Empty(t, parts, "Empty content should have no parts")
}

// ═══════════════════════════════════════════════════════════════════════════
// ConvertFromAPI 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestAdapter_ConvertFromAPI_TextResponse(t *testing.T) {
	adapter := NewAdapter()
	apiResp := map[string]any{
		"candidates": []any{
			map[string]any{
				"content": map[string]any{
					"role": "model",
					"parts": []any{
						map[string]any{
							"text": "Hello! How can I help you today?",
						},
					},
				},
				"finishReason": "STOP",
			},
		},
	}

	msg, finishReason := adapter.ConvertFromAPI(apiResp)

	assert.Equal(t, llm.RoleAssistant, msg.Role)
	assert.Equal(t, "Hello! How can I help you today?", msg.Content)
	assert.Equal(t, "stop", finishReason) // STOP -> stop
}

func TestAdapter_ConvertFromAPI_ToolCallResponse(t *testing.T) {
	adapter := NewAdapter()
	apiResp := map[string]any{
		"candidates": []any{
			map[string]any{
				"content": map[string]any{
					"role": "model",
					"parts": []any{
						map[string]any{
							"text": "I'll check the weather for you.",
						},
						map[string]any{
							"functionCall": map[string]any{
								"name": "get_weather",
								"args": map[string]any{
									"city": "London",
								},
							},
						},
					},
				},
				"finishReason": "STOP",
			},
		},
	}

	msg, _ := adapter.ConvertFromAPI(apiResp)

	assert.Equal(t, llm.RoleAssistant, msg.Role)
	require.Len(t, msg.ContentBlocks, 2, "Expected text + tool_call")

	// 第一个 block 应该是 TextBlock
	textBlock, ok := msg.ContentBlocks[0].(*llm.TextBlock)
	require.True(t, ok)
	assert.Equal(t, "I'll check the weather for you.", textBlock.Text)

	// 第二个 block 应该是 ToolCall
	toolCall, ok := msg.ContentBlocks[1].(*llm.ToolCall)
	require.True(t, ok)
	assert.Equal(t, "get_weather", toolCall.Name)
	assert.Equal(t, "London", toolCall.Input["city"])
	// Gemini 不返回 ID，应该是生成的
	assert.NotEmpty(t, toolCall.ID)
}

func TestAdapter_ConvertFromAPI_ThinkingResponse(t *testing.T) {
	adapter := NewAdapter()
	apiResp := map[string]any{
		"candidates": []any{
			map[string]any{
				"content": map[string]any{
					"role": "model",
					"parts": []any{
						map[string]any{
							"text":    "Analyzing the problem...",
							"thought": true,
						},
						map[string]any{
							"text": "The answer is 42.",
						},
					},
				},
				"finishReason": "STOP",
			},
		},
	}

	msg, _ := adapter.ConvertFromAPI(apiResp)

	require.Len(t, msg.ContentBlocks, 2)

	// 第一个应该是 ThinkingBlock
	thinkingBlock, ok := msg.ContentBlocks[0].(*llm.ThinkingBlock)
	require.True(t, ok, "First block should be ThinkingBlock")
	assert.Equal(t, "Analyzing the problem...", thinkingBlock.Thinking)

	// 第二个应该是 TextBlock
	textBlock, ok := msg.ContentBlocks[1].(*llm.TextBlock)
	require.True(t, ok)
	assert.Equal(t, "The answer is 42.", textBlock.Text)
}

func TestAdapter_ConvertFromAPI_FinishReasonMapping(t *testing.T) {
	adapter := NewAdapter()

	testCases := []struct {
		geminiReason   string
		expectedReason string
	}{
		{"STOP", "stop"},
		{"MAX_TOKENS", "length"},
		{"SAFETY", "content_filter"},
		{"RECITATION", "content_filter"},
		{"OTHER", "stop"},
		{"UNKNOWN", "UNKNOWN"}, // 未知原因保持原样
	}

	for _, tc := range testCases {
		apiResp := map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{
							map[string]any{"text": "test"},
						},
					},
					"finishReason": tc.geminiReason,
				},
			},
		}

		_, finishReason := adapter.ConvertFromAPI(apiResp)

		assert.Equal(t, tc.expectedReason, finishReason,
			"Gemini reason %q should map to %q", tc.geminiReason, tc.expectedReason)
	}
}

func TestAdapter_ConvertFromAPI_EmptyCandidates(t *testing.T) {
	adapter := NewAdapter()
	apiResp := map[string]any{
		"candidates": []any{},
	}

	msg, finishReason := adapter.ConvertFromAPI(apiResp)

	assert.Equal(t, llm.RoleAssistant, msg.Role)
	assert.Empty(t, msg.Content)
	assert.Empty(t, finishReason)
}

// ═══════════════════════════════════════════════════════════════════════════
// ConvertUsage 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestAdapter_ConvertUsage_Basic(t *testing.T) {
	adapter := NewAdapter()
	apiResp := map[string]any{
		"usageMetadata": map[string]any{
			"promptTokenCount":     float64(100),
			"candidatesTokenCount": float64(50),
			"totalTokenCount":      float64(150),
		},
	}

	usage := adapter.ConvertUsage(apiResp)

	require.NotNil(t, usage)
	assert.Equal(t, int64(100), usage.InputTokens)
	assert.Equal(t, int64(50), usage.OutputTokens)
	assert.Equal(t, int64(150), usage.TotalTokens)
}

func TestAdapter_ConvertUsage_WithThinkingTokens(t *testing.T) {
	adapter := NewAdapter()
	apiResp := map[string]any{
		"usageMetadata": map[string]any{
			"promptTokenCount":     float64(100),
			"candidatesTokenCount": float64(200),
			"totalTokenCount":      float64(300),
			"thoughtsTokenCount":   float64(150), // Gemini 2.5 thinking tokens
		},
	}

	usage := adapter.ConvertUsage(apiResp)

	require.NotNil(t, usage)
	assert.Equal(t, int64(150), usage.ReasoningTokens)
}

func TestAdapter_ConvertUsage_WithCachedTokens(t *testing.T) {
	adapter := NewAdapter()
	apiResp := map[string]any{
		"usageMetadata": map[string]any{
			"promptTokenCount":        float64(100),
			"candidatesTokenCount":    float64(50),
			"totalTokenCount":         float64(150),
			"cachedContentTokenCount": float64(80), // Prompt caching
		},
	}

	usage := adapter.ConvertUsage(apiResp)

	require.NotNil(t, usage)
	assert.Equal(t, int64(80), usage.CachedTokens)
}

func TestAdapter_ConvertUsage_NoUsage(t *testing.T) {
	adapter := NewAdapter()
	apiResp := map[string]any{}

	usage := adapter.ConvertUsage(apiResp)

	assert.Nil(t, usage, "Expected nil usage when not present")
}

// ═══════════════════════════════════════════════════════════════════════════
// GetSystemMessageHandling 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestAdapter_GetSystemMessageHandling(t *testing.T) {
	adapter := NewAdapter()

	strategy := adapter.GetSystemMessageHandling()

	// Gemini 使用 SystemSeparate：系统消息作为独立的 systemInstruction 参数
	assert.Equal(t, core.SystemSeparate, strategy)
}

// ═══════════════════════════════════════════════════════════════════════════
// 接口实现验证
// ═══════════════════════════════════════════════════════════════════════════

func TestAdapter_ImplementsProtocolAdapter(t *testing.T) {
	var _ core.ProtocolAdapter = (*Adapter)(nil)
}
