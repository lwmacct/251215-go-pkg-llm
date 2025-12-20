package core_test

import (
	"testing"

	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm/core"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm/protocol/anthropic"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm/protocol/openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ═══════════════════════════════════════════════════════════════════════════
// BuildAPIMessages 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestTransformer_BuildAPIMessages_SystemInline(t *testing.T) {
	// 使用 OpenAI Adapter (SystemInline 策略)
	adapter := openai.NewAdapter()
	transformer := core.NewTransformer(adapter)

	messages := []llm.Message{
		{Role: llm.RoleUser, Content: "Hello!"},
	}
	systemPrompt := "You are a helpful assistant."

	result := transformer.BuildAPIMessages(messages, systemPrompt)

	// ⚠️ 关键验证：systemPrompt 被插入消息数组开头
	require.Len(t, result, 2, "Expected 2 messages (system + user)")

	// 第一条应该是系统消息
	assert.Equal(t, "system", result[0]["role"], "First message should be system")
	assert.Equal(t, systemPrompt, result[0]["content"], "System content should match")

	// 第二条应该是用户消息
	assert.Equal(t, "user", result[1]["role"], "Second message should be user")
	assert.Equal(t, "Hello!", result[1]["content"], "User content should match")
}

func TestTransformer_BuildAPIMessages_SystemSeparate(t *testing.T) {
	// 使用 Anthropic Adapter (SystemSeparate 策略)
	adapter := anthropic.NewAdapter()
	transformer := core.NewTransformer(adapter)

	messages := []llm.Message{
		{Role: llm.RoleUser, Content: "Hello!"},
	}
	systemPrompt := "You are a helpful assistant."

	result := transformer.BuildAPIMessages(messages, systemPrompt)

	// ⚠️ 关键验证：systemPrompt 不被插入消息数组（由调用方作为独立参数传递）
	require.Len(t, result, 1, "Expected 1 message (system NOT inlined)")

	// 应该只有用户消息
	assert.Equal(t, "user", result[0]["role"], "Only message should be user")
}

func TestTransformer_BuildAPIMessages_FilterSystemMessages(t *testing.T) {
	adapter := openai.NewAdapter()
	transformer := core.NewTransformer(adapter)

	// 输入包含 RoleSystem 的消息
	messages := []llm.Message{
		{Role: llm.RoleSystem, Content: "Old system message"},
		{Role: llm.RoleUser, Content: "Hello!"},
		{Role: llm.RoleAssistant, Content: "Hi there!"},
	}
	systemPrompt := "New system prompt"

	result := transformer.BuildAPIMessages(messages, systemPrompt)

	// ⚠️ 关键验证：messages 中的系统消息被过滤
	// 期望：systemPrompt (新) + user + assistant = 3 条消息
	require.Len(t, result, 3, "Expected 3 messages (old system filtered)")

	// 验证消息顺序
	assert.Equal(t, "system", result[0]["role"], "First should be new system")
	assert.Equal(t, systemPrompt, result[0]["content"], "System content from systemPrompt")
	assert.Equal(t, "user", result[1]["role"], "Second should be user")
	assert.Equal(t, "assistant", result[2]["role"], "Third should be assistant")
}

func TestTransformer_BuildAPIMessages_EmptySystemPrompt(t *testing.T) {
	adapter := openai.NewAdapter()
	transformer := core.NewTransformer(adapter)

	messages := []llm.Message{
		{Role: llm.RoleUser, Content: "Hello!"},
	}
	systemPrompt := "" // 空系统提示

	result := transformer.BuildAPIMessages(messages, systemPrompt)

	// ⚠️ 关键验证：无系统消息插入
	require.Len(t, result, 1, "Expected 1 message (no system)")
	assert.Equal(t, "user", result[0]["role"])
}

func TestTransformer_BuildAPIMessages_EmptyMessages(t *testing.T) {
	adapter := openai.NewAdapter()
	transformer := core.NewTransformer(adapter)

	messages := []llm.Message{}
	systemPrompt := "You are helpful."

	result := transformer.BuildAPIMessages(messages, systemPrompt)

	// 只有系统消息
	require.Len(t, result, 1, "Expected 1 message (only system)")
	assert.Equal(t, "system", result[0]["role"])
}

func TestTransformer_BuildAPIMessages_WithToolCall(t *testing.T) {
	adapter := openai.NewAdapter()
	transformer := core.NewTransformer(adapter)

	messages := []llm.Message{
		{Role: llm.RoleUser, Content: "What's the weather?"},
		{
			Role: llm.RoleAssistant,
			ContentBlocks: []llm.ContentBlock{
				&llm.TextBlock{Text: "Let me check."},
				&llm.ToolCall{
					ID:    "call_123",
					Name:  "get_weather",
					Input: map[string]any{"city": "Tokyo"},
				},
			},
		},
	}

	result := transformer.BuildAPIMessages(messages, "")

	require.Len(t, result, 2, "Expected 2 messages")

	// 验证工具调用消息结构
	toolMsg := result[1]
	assert.Equal(t, "assistant", toolMsg["role"])
	assert.Equal(t, "Let me check.", toolMsg["content"])

	toolCalls, ok := toolMsg["tool_calls"].([]map[string]any)
	require.True(t, ok, "Expected tool_calls array")
	require.Len(t, toolCalls, 1)
	assert.Equal(t, "call_123", toolCalls[0]["id"])
}

// ═══════════════════════════════════════════════════════════════════════════
// ParseAPIResponse 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestTransformer_ParseAPIResponse_OpenAI(t *testing.T) {
	adapter := openai.NewAdapter()
	transformer := core.NewTransformer(adapter)

	// 模拟 OpenAI API 响应格式
	apiResp := map[string]any{
		"choices": []any{
			map[string]any{
				"message": map[string]any{
					"content": "Hello! How can I help you?",
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]any{
			"prompt_tokens":     float64(10),
			"completion_tokens": float64(8),
			"total_tokens":      float64(18),
		},
	}

	msg, finishReason, usage := transformer.ParseAPIResponse(apiResp)

	// 验证消息
	assert.Equal(t, llm.RoleAssistant, msg.Role)
	assert.Equal(t, "Hello! How can I help you?", msg.Content)

	// 验证完成原因
	assert.Equal(t, "stop", finishReason)

	// 验证 Token 使用量
	require.NotNil(t, usage)
	assert.Equal(t, int64(10), usage.InputTokens)
	assert.Equal(t, int64(8), usage.OutputTokens)
	assert.Equal(t, int64(18), usage.TotalTokens)
}

func TestTransformer_ParseAPIResponse_Anthropic(t *testing.T) {
	adapter := anthropic.NewAdapter()
	transformer := core.NewTransformer(adapter)

	// 模拟 Anthropic API 响应格式
	apiResp := map[string]any{
		"content": []any{
			map[string]any{
				"type": "text",
				"text": "Hello! I'm Claude.",
			},
		},
		"stop_reason": "end_turn",
		"usage": map[string]any{
			"input_tokens":  float64(15),
			"output_tokens": float64(5),
		},
	}

	msg, finishReason, usage := transformer.ParseAPIResponse(apiResp)

	// 验证消息
	assert.Equal(t, llm.RoleAssistant, msg.Role)
	assert.Equal(t, "Hello! I'm Claude.", msg.Content)

	// 验证完成原因（end_turn -> stop）
	assert.Equal(t, "stop", finishReason)

	// 验证 Token 使用量
	require.NotNil(t, usage)
	assert.Equal(t, int64(15), usage.InputTokens)
	assert.Equal(t, int64(5), usage.OutputTokens)
	// Anthropic TotalTokens 是自动计算的
	assert.Equal(t, int64(20), usage.TotalTokens)
}

func TestTransformer_ParseAPIResponse_WithToolCall_OpenAI(t *testing.T) {
	adapter := openai.NewAdapter()
	transformer := core.NewTransformer(adapter)

	apiResp := map[string]any{
		"choices": []any{
			map[string]any{
				"message": map[string]any{
					"content": "Let me check.",
					"tool_calls": []any{
						map[string]any{
							"id":   "call_abc",
							"type": "function",
							"function": map[string]any{
								"name":      "get_weather",
								"arguments": `{"city":"London"}`,
							},
						},
					},
				},
				"finish_reason": "tool_calls",
			},
		},
	}

	msg, finishReason, _ := transformer.ParseAPIResponse(apiResp)

	assert.Equal(t, llm.RoleAssistant, msg.Role)
	assert.Equal(t, "tool_calls", finishReason)

	// 验证 ContentBlocks
	require.Len(t, msg.ContentBlocks, 2, "Expected text + tool_call")

	textBlock, ok := msg.ContentBlocks[0].(*llm.TextBlock)
	require.True(t, ok)
	assert.Equal(t, "Let me check.", textBlock.Text)

	toolCall, ok := msg.ContentBlocks[1].(*llm.ToolCall)
	require.True(t, ok)
	assert.Equal(t, "call_abc", toolCall.ID)
	assert.Equal(t, "get_weather", toolCall.Name)
	assert.Equal(t, "London", toolCall.Input["city"])
}

func TestTransformer_ParseAPIResponse_WithToolCall_Anthropic(t *testing.T) {
	adapter := anthropic.NewAdapter()
	transformer := core.NewTransformer(adapter)

	apiResp := map[string]any{
		"content": []any{
			map[string]any{
				"type": "text",
				"text": "I'll look that up.",
			},
			map[string]any{
				"type": "tool_use",
				"id":   "toolu_xyz",
				"name": "search",
				"input": map[string]any{
					"query": "weather",
				},
			},
		},
		"stop_reason": "tool_use",
	}

	msg, finishReason, _ := transformer.ParseAPIResponse(apiResp)

	assert.Equal(t, llm.RoleAssistant, msg.Role)
	assert.Equal(t, "tool_calls", finishReason) // tool_use -> tool_calls

	require.Len(t, msg.ContentBlocks, 2)

	toolCall, ok := msg.ContentBlocks[1].(*llm.ToolCall)
	require.True(t, ok)
	assert.Equal(t, "toolu_xyz", toolCall.ID)
	assert.Equal(t, "search", toolCall.Name)
	assert.Equal(t, "weather", toolCall.Input["query"])
}

func TestTransformer_ParseAPIResponse_NoUsage(t *testing.T) {
	adapter := openai.NewAdapter()
	transformer := core.NewTransformer(adapter)

	apiResp := map[string]any{
		"choices": []any{
			map[string]any{
				"message": map[string]any{
					"content": "Hello",
				},
				"finish_reason": "stop",
			},
		},
		// 没有 usage 字段
	}

	_, _, usage := transformer.ParseAPIResponse(apiResp)

	assert.Nil(t, usage, "Expected nil usage when not present")
}

// ═══════════════════════════════════════════════════════════════════════════
// 联合测试：完整消息往返
// ═══════════════════════════════════════════════════════════════════════════

func TestTransformer_Integration_MessageRoundTrip_OpenAI(t *testing.T) {
	adapter := openai.NewAdapter()
	transformer := core.NewTransformer(adapter)

	// 原始消息
	originalMessages := []llm.Message{
		{Role: llm.RoleUser, Content: "What's 2+2?"},
	}

	// 构建 API 请求
	apiMessages := transformer.BuildAPIMessages(originalMessages, "You are a math tutor.")

	// 验证转换后的结构
	require.Len(t, apiMessages, 2)
	assert.Equal(t, "system", apiMessages[0]["role"])
	assert.Equal(t, "user", apiMessages[1]["role"])

	// 模拟 API 响应并解析
	apiResp := map[string]any{
		"choices": []any{
			map[string]any{
				"message": map[string]any{
					"content": "2 + 2 = 4",
				},
				"finish_reason": "stop",
			},
		},
	}

	msg, reason, _ := transformer.ParseAPIResponse(apiResp)

	// 验证往返完整性
	assert.Equal(t, llm.RoleAssistant, msg.Role)
	assert.Equal(t, "2 + 2 = 4", msg.Content)
	assert.Equal(t, "stop", reason)
}

func TestTransformer_Integration_MessageRoundTrip_Anthropic(t *testing.T) {
	adapter := anthropic.NewAdapter()
	transformer := core.NewTransformer(adapter)

	// 原始消息
	originalMessages := []llm.Message{
		{Role: llm.RoleUser, Content: "Tell me a joke"},
	}

	// 构建 API 请求
	apiMessages := transformer.BuildAPIMessages(originalMessages, "You are a comedian.")

	// Anthropic: systemPrompt 不插入消息数组
	require.Len(t, apiMessages, 1)
	assert.Equal(t, "user", apiMessages[0]["role"])

	// 模拟 API 响应
	apiResp := map[string]any{
		"content": []any{
			map[string]any{
				"type": "text",
				"text": "Why did the chicken cross the road?",
			},
		},
		"stop_reason": "end_turn",
	}

	msg, reason, _ := transformer.ParseAPIResponse(apiResp)

	assert.Equal(t, llm.RoleAssistant, msg.Role)
	assert.Equal(t, "Why did the chicken cross the road?", msg.Content)
	assert.Equal(t, "stop", reason)
}
