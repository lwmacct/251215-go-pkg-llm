package llm_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm/core"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm/protocol/anthropic"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm/protocol/openai"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ═══════════════════════════════════════════════════════════════════════════
// Message Roundtrip 测试 - 验证消息转换的完整性
// ═══════════════════════════════════════════════════════════════════════════

// TestIntegration_MessageRoundTrip_OpenAI 测试 OpenAI 格式的完整消息转换流程
//
// 流程：Message -> Transformer.BuildAPIMessages -> API 格式 ->
//
//	模拟响应 -> Transformer.ParseAPIResponse -> Message
//
// 验证点：数据完整性、角色映射正确性、工具调用转换
func TestIntegration_MessageRoundTrip_OpenAI(t *testing.T) {
	adapter := openai.NewAdapter()
	transformer := core.NewTransformer(adapter)

	// 原始消息
	originalMessages := []llm.Message{
		{Role: llm.RoleUser, Content: "What's the weather in Tokyo?"},
		{
			Role: llm.RoleAssistant,
			ContentBlocks: []llm.ContentBlock{
				&llm.TextBlock{Text: "Let me check the weather for you."},
				&llm.ToolCall{
					ID:    "call_123",
					Name:  "get_weather",
					Input: map[string]any{"city": "Tokyo"},
				},
			},
		},
		{
			Role: llm.RoleUser,
			ContentBlocks: []llm.ContentBlock{
				&llm.ToolResultBlock{
					ToolUseID: "call_123",
					Content:   "Tokyo: 25°C, Sunny",
				},
			},
		},
	}

	// 转换到 API 格式
	apiMessages := transformer.BuildAPIMessages(originalMessages, "")

	// 验证转换结果结构
	require.Len(t, apiMessages, 3, "Should have 3 messages")

	// 验证第一条消息
	assert.Equal(t, "user", apiMessages[0]["role"])
	assert.Equal(t, "What's the weather in Tokyo?", apiMessages[0]["content"])

	// 验证第二条消息（带工具调用）
	assert.Equal(t, "assistant", apiMessages[1]["role"])
	toolCalls, ok := apiMessages[1]["tool_calls"].([]map[string]any)
	require.True(t, ok, "Should have tool_calls")
	require.Len(t, toolCalls, 1)
	assert.Equal(t, "call_123", toolCalls[0]["id"])

	// 模拟 API 响应
	apiResponse := map[string]any{
		"choices": []any{
			map[string]any{
				"message": map[string]any{
					"role":    "assistant",
					"content": "The weather in Tokyo is 25°C and sunny.",
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]any{
			"prompt_tokens":     float64(100),
			"completion_tokens": float64(20),
			"total_tokens":      float64(120),
		},
	}

	// 解析响应
	msg, finishReason, usage := transformer.ParseAPIResponse(apiResponse)

	// 验证解析结果
	assert.Equal(t, llm.RoleAssistant, msg.Role)
	assert.Equal(t, "The weather in Tokyo is 25°C and sunny.", msg.Content)
	assert.Equal(t, "stop", finishReason)
	require.NotNil(t, usage)
	assert.Equal(t, int64(100), usage.InputTokens)
	assert.Equal(t, int64(20), usage.OutputTokens)
}

// TestIntegration_MessageRoundTrip_Anthropic 测试 Anthropic 格式的完整消息转换流程
func TestIntegration_MessageRoundTrip_Anthropic(t *testing.T) {
	adapter := anthropic.NewAdapter()
	transformer := core.NewTransformer(adapter)

	// 原始消息（注意：ThinkingBlock 是响应专用，不会被转换到 API 格式）
	originalMessages := []llm.Message{
		{Role: llm.RoleUser, Content: "Explain quantum computing"},
		{
			Role: llm.RoleAssistant,
			ContentBlocks: []llm.ContentBlock{
				&llm.TextBlock{Text: "Quantum computing uses qubits."},
				&llm.ToolCall{
					ID:    "call_456",
					Name:  "search",
					Input: map[string]any{"query": "quantum computing basics"},
				},
			},
		},
	}

	// 转换到 API 格式
	apiMessages := transformer.BuildAPIMessages(originalMessages, "You are a physics teacher.")

	// Anthropic 使用 SystemSeparate 策略，系统消息不在数组中
	require.Len(t, apiMessages, 2, "Should have 2 messages (system excluded)")

	// 验证第一条消息
	assert.Equal(t, "user", apiMessages[0]["role"])

	// 验证第二条消息（带工具调用）
	content, ok := apiMessages[1]["content"].([]map[string]any)
	require.True(t, ok, "Anthropic uses content array")
	require.Len(t, content, 2)

	// text block
	assert.Equal(t, "text", content[0]["type"])
	assert.Equal(t, "Quantum computing uses qubits.", content[0]["text"])

	// tool_use block (注意：Anthropic 使用 "tool_use" 而非 "tool_call")
	assert.Equal(t, "tool_use", content[1]["type"])
	assert.Equal(t, "call_456", content[1]["id"])
	assert.Equal(t, "search", content[1]["name"])
	// ⚠️ 关键差异：Anthropic 参数是直接对象，不是 JSON 字符串
	input, ok := content[1]["input"].(map[string]any)
	require.True(t, ok, "input should be map, not JSON string")
	assert.Equal(t, "quantum computing basics", input["query"])

	// 模拟 API 响应
	apiResponse := map[string]any{
		"content": []any{
			map[string]any{
				"type":     "thinking",
				"thinking": "Analyzing the concept...",
			},
			map[string]any{
				"type": "text",
				"text": "Here's my explanation...",
			},
		},
		"stop_reason": "end_turn",
		"usage": map[string]any{
			"input_tokens":  float64(50),
			"output_tokens": float64(100),
		},
	}

	// 解析响应
	msg, finishReason, usage := transformer.ParseAPIResponse(apiResponse)

	// 验证解析结果
	assert.Equal(t, llm.RoleAssistant, msg.Role)
	assert.Equal(t, "stop", finishReason)
	require.NotNil(t, usage)
	assert.Equal(t, int64(50), usage.InputTokens)
	assert.Equal(t, int64(100), usage.OutputTokens)
}

// ═══════════════════════════════════════════════════════════════════════════
// SSE 完整流测试 - 验证流式解析的完整性
// ═══════════════════════════════════════════════════════════════════════════

// TestIntegration_SSE_FullStream_OpenAI 测试 OpenAI 格式的完整 SSE 流解析
func TestIntegration_SSE_FullStream_OpenAI(t *testing.T) {
	handler := openai.NewEventHandler()
	parser := core.NewSSEParser(handler)

	// 模拟完整的 OpenAI SSE 流
	sseData := `data: {"choices":[{"delta":{"role":"assistant"}}]}

data: {"choices":[{"delta":{"content":"Hello"}}]}

data: {"choices":[{"delta":{"content":" World"}}]}

data: {"choices":[{"delta":{"content":"!"}}]}

data: {"choices":[{"delta":{},"finish_reason":"stop"}]}

data: [DONE]
`
	reader := io.NopCloser(strings.NewReader(sseData))
	events := make(chan *llm.Event, 20)

	go parser.Parse(reader, events)

	// 收集所有事件
	var collected []*llm.Event
	timeout := time.After(1 * time.Second)

loop:
	for {
		select {
		case e, ok := <-events:
			if !ok {
				break loop
			}
			collected = append(collected, e)
		case <-timeout:
			t.Fatal("Test timed out")
		}
	}

	// 验证事件序列
	require.NotEmpty(t, collected)

	// 收集文本内容
	var fullText strings.Builder
	for _, e := range collected {
		if e.Type == llm.EventTypeText && e.TextDelta != "" {
			fullText.WriteString(e.TextDelta)
		}
	}

	// 验证完整文本
	assert.Equal(t, "Hello World!", fullText.String())

	// 验证有完成事件
	var hasDone bool
	for _, e := range collected {
		if e.Type == llm.EventTypeDone {
			hasDone = true
			break
		}
	}
	assert.True(t, hasDone, "Should have done event")
}

// TestIntegration_SSE_ToolCallStream_OpenAI 测试 OpenAI 工具调用的流式解析
func TestIntegration_SSE_ToolCallStream_OpenAI(t *testing.T) {
	handler := openai.NewEventHandler()
	parser := core.NewSSEParser(handler)

	// 模拟工具调用的 SSE 流
	sseData := `data: {"choices":[{"delta":{"role":"assistant"}}]}

data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_abc","function":{"name":"get_weather","arguments":""}}]}}]}

data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"city\":"}}]}}]}

data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"Tokyo\"}"}}]}}]}

data: {"choices":[{"delta":{},"finish_reason":"tool_calls"}]}

data: [DONE]
`
	reader := io.NopCloser(strings.NewReader(sseData))
	events := make(chan *llm.Event, 20)

	go parser.Parse(reader, events)

	// 收集所有事件
	var collected []*llm.Event //nolint:prealloc // channel 收集数量未知
	for e := range events {
		collected = append(collected, e)
	}

	// 验证有工具调用事件
	var toolCallEvents []*llm.Event
	for _, e := range collected {
		if e.Type == llm.EventTypeToolCall {
			toolCallEvents = append(toolCallEvents, e)
		}
	}

	require.NotEmpty(t, toolCallEvents, "Should have tool call events")

	// 第一个工具调用事件应该有 ID 和 Name
	assert.Equal(t, "call_abc", toolCallEvents[0].ToolCall.ID)
	assert.Equal(t, "get_weather", toolCallEvents[0].ToolCall.Name)
}

// TestIntegration_SSE_FullStream_Anthropic 测试 Anthropic 格式的完整 SSE 流解析
func TestIntegration_SSE_FullStream_Anthropic(t *testing.T) {
	handler := anthropic.NewEventHandler()
	parser := core.NewSSEParser(handler)

	// 模拟完整的 Anthropic SSE 流
	sseData := `event: message_start
data: {"type":"message_start","message":{"id":"msg_123","model":"claude-3"}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"The answer"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" is 42."}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn"}}

event: message_stop
data: {"type":"message_stop"}
`
	reader := io.NopCloser(strings.NewReader(sseData))
	events := make(chan *llm.Event, 20)

	go parser.Parse(reader, events)

	// 收集所有事件
	var collected []*llm.Event //nolint:prealloc // channel 收集数量未知
	for e := range events {
		collected = append(collected, e)
	}

	// 收集文本内容
	var fullText strings.Builder
	for _, e := range collected {
		if e.Type == llm.EventTypeText && e.TextDelta != "" {
			fullText.WriteString(e.TextDelta)
		}
	}

	// 验证完整文本
	assert.Equal(t, "The answer is 42.", fullText.String())
}

// ═══════════════════════════════════════════════════════════════════════════
// Provider 完整测试 - 使用 Mock HTTP 验证 Provider 行为
// ═══════════════════════════════════════════════════════════════════════════

// TestIntegration_Provider_Complete_OpenAI 测试 OpenAI Provider 的完整请求流程
func TestIntegration_Provider_Complete_OpenAI(t *testing.T) {
	// 创建 Mock HTTP Server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/chat/completions", r.URL.Path)
		assert.Contains(t, r.Header.Get("Authorization"), "Bearer test-key")
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// 验证请求体
		var reqBody map[string]any
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		assert.NoError(t, err)

		assert.Equal(t, "gpt-4", reqBody["model"])
		assert.NotNil(t, reqBody["messages"])

		messages, ok := reqBody["messages"].([]any)
		assert.True(t, ok, "messages should be []any")
		assert.Len(t, messages, 2, "Should have system + user message")

		// 验证系统消息
		if len(messages) > 0 {
			systemMsg, ok := messages[0].(map[string]any)
			assert.True(t, ok)
			assert.Equal(t, "system", systemMsg["role"])
			assert.Equal(t, "You are helpful.", systemMsg["content"])
		}

		// 返回响应
		resp := map[string]any{
			"id":    "chatcmpl-123",
			"model": "gpt-4",
			"choices": []any{
				map[string]any{
					"message": map[string]any{
						"role":    "assistant",
						"content": "Hello! How can I help you?",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     float64(20),
				"completion_tokens": float64(10),
				"total_tokens":      float64(30),
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 创建 Provider
	p, err := provider.New(&llm.Config{
		Type:    llm.ProviderTypeOpenAI,
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gpt-4",
	})
	require.NoError(t, err)
	defer func() { _ = p.Close() }()

	// 发送请求
	resp, err := p.Complete(context.Background(), []llm.Message{
		{Role: llm.RoleUser, Content: "Hello"},
	}, &llm.Options{
		System: "You are helpful.",
	})

	// 验证响应
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, llm.RoleAssistant, resp.Message.Role)
	assert.Equal(t, "Hello! How can I help you?", resp.Message.Content)
	assert.Equal(t, "stop", resp.FinishReason)
	require.NotNil(t, resp.Usage)
	assert.Equal(t, int64(20), resp.Usage.InputTokens)
	assert.Equal(t, int64(10), resp.Usage.OutputTokens)
}

// TestIntegration_Provider_Stream_OpenAI 测试 OpenAI Provider 的流式请求
func TestIntegration_Provider_Stream_OpenAI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证 stream 参数
		var reqBody map[string]any
		_ = json.NewDecoder(r.Body).Decode(&reqBody)
		assert.Equal(t, true, reqBody["stream"])

		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		assert.True(t, ok)
		if !ok {
			return
		}

		// 发送 SSE 事件
		events := []string{
			`data: {"choices":[{"delta":{"role":"assistant"}}]}

`,
			`data: {"choices":[{"delta":{"content":"Hi"}}]}

`,
			`data: {"choices":[{"delta":{"content":"!"}}]}

`,
			`data: {"choices":[{"delta":{},"finish_reason":"stop"}]}

`,
			`data: [DONE]

`,
		}

		for _, event := range events {
			_, _ = w.Write([]byte(event))
			flusher.Flush()
		}
	}))
	defer server.Close()

	p, err := provider.New(&llm.Config{
		Type:    llm.ProviderTypeOpenAI,
		APIKey:  "test-key",
		BaseURL: server.URL,
	})
	require.NoError(t, err)
	defer func() { _ = p.Close() }()

	stream, err := p.Stream(context.Background(), []llm.Message{
		{Role: llm.RoleUser, Content: "Hi"},
	}, nil)

	require.NoError(t, err)
	require.NotNil(t, stream)

	// 收集文本
	var fullText strings.Builder
	for e := range stream {
		if e.Type == llm.EventTypeText {
			fullText.WriteString(e.TextDelta)
		}
	}

	assert.Equal(t, "Hi!", fullText.String())
}

// TestIntegration_Provider_ToolCall_OpenAI 测试 OpenAI Provider 的工具调用
func TestIntegration_Provider_ToolCall_OpenAI(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		var reqBody map[string]any
		_ = json.NewDecoder(r.Body).Decode(&reqBody)

		var resp map[string]any

		if callCount == 1 {
			// 第一次调用：验证工具定义
			tools, ok := reqBody["tools"].([]any)
			assert.True(t, ok, "Should have tools")
			assert.Len(t, tools, 1)

			// 返回工具调用
			resp = map[string]any{
				"choices": []any{
					map[string]any{
						"message": map[string]any{
							"role":    "assistant",
							"content": nil,
							"tool_calls": []any{
								map[string]any{
									"id":   "call_abc",
									"type": "function",
									"function": map[string]any{
										"name":      "get_weather",
										"arguments": `{"city":"Tokyo"}`,
									},
								},
							},
						},
						"finish_reason": "tool_calls",
					},
				},
			}
		} else {
			// 第二次调用：包含工具结果
			messages, ok := reqBody["messages"].([]any)
			assert.True(t, ok)
			if len(messages) > 0 {
				lastMsg, ok := messages[len(messages)-1].(map[string]any)
				assert.True(t, ok)
				assert.Equal(t, "tool", lastMsg["role"])
				assert.Equal(t, "call_abc", lastMsg["tool_call_id"])
			}

			// 返回最终响应
			resp = map[string]any{
				"choices": []any{
					map[string]any{
						"message": map[string]any{
							"role":    "assistant",
							"content": "The weather in Tokyo is sunny.",
						},
						"finish_reason": "stop",
					},
				},
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p, err := provider.New(&llm.Config{
		Type:    llm.ProviderTypeOpenAI,
		APIKey:  "test-key",
		BaseURL: server.URL,
	})
	require.NoError(t, err)
	defer func() { _ = p.Close() }()

	// 第一次调用：带工具定义
	resp1, err := p.Complete(context.Background(), []llm.Message{
		{Role: llm.RoleUser, Content: "What's the weather in Tokyo?"},
	}, &llm.Options{
		Tools: []llm.ToolSchema{
			{
				Name:        "get_weather",
				Description: "Get weather information",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"city": map[string]any{"type": "string"},
					},
				},
			},
		},
	})

	require.NoError(t, err)
	assert.True(t, resp1.Message.HasToolCalls())

	toolCalls := resp1.Message.GetToolCalls()
	require.Len(t, toolCalls, 1)
	assert.Equal(t, "call_abc", toolCalls[0].ID)
	assert.Equal(t, "get_weather", toolCalls[0].Name)

	// 第二次调用：带工具结果
	resp2, err := p.Complete(context.Background(), []llm.Message{
		{Role: llm.RoleUser, Content: "What's the weather in Tokyo?"},
		{
			Role: llm.RoleAssistant,
			ContentBlocks: []llm.ContentBlock{
				&llm.ToolCall{
					ID:    "call_abc",
					Name:  "get_weather",
					Input: map[string]any{"city": "Tokyo"},
				},
			},
		},
		{
			Role: llm.RoleUser,
			ContentBlocks: []llm.ContentBlock{
				&llm.ToolResultBlock{
					ToolUseID: "call_abc",
					Content:   "Tokyo: 25°C, Sunny",
				},
			},
		},
	}, nil)

	require.NoError(t, err)
	assert.Equal(t, "The weather in Tokyo is sunny.", resp2.Message.Content)
}

// ═══════════════════════════════════════════════════════════════════════════
// Mock Provider 测试 - 验证 Mock 功能正常
// ═══════════════════════════════════════════════════════════════════════════

// TestIntegration_MockProvider_Complete 测试 Mock Provider 的完整功能
func TestIntegration_MockProvider_Complete(t *testing.T) {
	p := provider.Mock()
	defer func() { _ = p.Close() }()

	resp, err := p.Complete(context.Background(), []llm.Message{
		{Role: llm.RoleUser, Content: "Hello"},
	}, nil)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, llm.RoleAssistant, resp.Message.Role)
	assert.NotEmpty(t, resp.Message.Content)
}

// TestIntegration_MockProvider_Stream 测试 Mock Provider 的流式功能
func TestIntegration_MockProvider_Stream(t *testing.T) {
	p := provider.Mock()
	defer func() { _ = p.Close() }()

	stream, err := p.Stream(context.Background(), []llm.Message{
		{Role: llm.RoleUser, Content: "Hello"},
	}, nil)

	require.NoError(t, err)
	require.NotNil(t, stream)

	var eventCount int
	for range stream {
		eventCount++
	}

	assert.Positive(t, eventCount, "Should receive events from mock")
}
