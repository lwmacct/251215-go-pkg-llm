package anthropic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ═══════════════════════════════════════════════════════════════════════════
// New 函数测试
// ═══════════════════════════════════════════════════════════════════════════

func TestNew_NilConfig(t *testing.T) {
	client, err := New(nil)

	assert.Nil(t, client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "config is required")
}

func TestNew_MissingAPIKey(t *testing.T) {
	client, err := New(&Config{})

	assert.Nil(t, client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API key is required")
}

func TestNew_Success(t *testing.T) {
	client, err := New(&Config{
		APIKey: "test-key",
	})

	require.NoError(t, err)
	require.NotNil(t, client)
	assert.NoError(t, client.Close())
}

func TestNew_DefaultValues(t *testing.T) {
	client, err := New(&Config{
		APIKey: "test-key",
	})

	require.NoError(t, err)
	require.NotNil(t, client)

	// 验证默认值
	assert.Equal(t, "claude-3-5-haiku-latest", client.config.Model)
	assert.Contains(t, client.config.BaseURL, "api.anthropic.com")
}

func TestNew_CustomValues(t *testing.T) {
	client, err := New(&Config{
		APIKey:           "test-key",
		BaseURL:          "https://custom.api.example.com/v1",
		Model:            "claude-3-opus",
		Timeout:          30 * time.Second,
		AnthropicVersion: "2024-01-01",
		Headers: map[string]string{
			"X-Custom-Header": "custom-value",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Equal(t, "claude-3-opus", client.config.Model)
	assert.Equal(t, "https://custom.api.example.com/v1", client.config.BaseURL)
}

// ═══════════════════════════════════════════════════════════════════════════
// Complete 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestClient_Complete_Success(t *testing.T) {
	// Mock HTTP Server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/messages", r.URL.Path)
		assert.NotEmpty(t, r.Header.Get("X-Api-Key"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// 验证请求体
		var reqBody map[string]any
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		assert.NoError(t, err)
		assert.Equal(t, "claude-3-5-haiku-latest", reqBody["model"])
		assert.NotNil(t, reqBody["messages"])

		// 返回模拟响应
		resp := map[string]any{
			"content": []any{
				map[string]any{
					"type": "text",
					"text": "Hello! I'm Claude.",
				},
			},
			"model":       "claude-3-5-haiku-latest",
			"stop_reason": "end_turn",
			"usage": map[string]any{
				"input_tokens":  float64(10),
				"output_tokens": float64(5),
			},
		}

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(resp)
		assert.NoError(t, err)
	}))
	defer server.Close()

	client, err := New(&Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})
	require.NoError(t, err)
	defer func() { _ = client.Close() }()

	messages := []llm.Message{
		{Role: llm.RoleUser, Content: "Hello!"},
	}

	resp, err := client.Complete(context.Background(), messages, nil)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, llm.RoleAssistant, resp.Message.Role)
	assert.Equal(t, "Hello! I'm Claude.", resp.Message.Content)
	assert.Equal(t, "stop", resp.FinishReason)
	assert.Equal(t, "claude-3-5-haiku-latest", resp.Model)
	require.NotNil(t, resp.Usage)
	assert.Equal(t, int64(10), resp.Usage.InputTokens)
	assert.Equal(t, int64(5), resp.Usage.OutputTokens)
}

func TestClient_Complete_WithToolCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"content": []any{
				map[string]any{
					"type": "text",
					"text": "Let me check the weather.",
				},
				map[string]any{
					"type": "tool_use",
					"id":   "toolu_123",
					"name": "get_weather",
					"input": map[string]any{
						"city": "Tokyo",
					},
				},
			},
			"model":       "claude-3-5-haiku-latest",
			"stop_reason": "tool_use",
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := New(&Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})
	require.NoError(t, err)
	defer func() { _ = client.Close() }()

	resp, err := client.Complete(context.Background(), []llm.Message{
		{Role: llm.RoleUser, Content: "What's the weather?"},
	}, nil)

	require.NoError(t, err)
	assert.Equal(t, "tool_calls", resp.FinishReason)
	assert.True(t, resp.Message.HasToolCalls())

	toolCalls := resp.Message.GetToolCalls()
	require.Len(t, toolCalls, 1)
	assert.Equal(t, "toolu_123", toolCalls[0].ID)
	assert.Equal(t, "get_weather", toolCalls[0].Name)
}

func TestClient_Complete_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": {"message": "Invalid API key"}}`))
	}))
	defer server.Close()

	client, err := New(&Config{
		APIKey:  "invalid-key",
		BaseURL: server.URL,
	})
	require.NoError(t, err)
	defer func() { _ = client.Close() }()

	resp, err := client.Complete(context.Background(), []llm.Message{
		{Role: llm.RoleUser, Content: "Hello"},
	}, nil)

	assert.Nil(t, resp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API error: 401")
}

func TestClient_Complete_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 延迟响应
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := New(&Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})
	require.NoError(t, err)
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	resp, err := client.Complete(ctx, []llm.Message{
		{Role: llm.RoleUser, Content: "Hello"},
	}, nil)

	assert.Nil(t, resp)
	require.Error(t, err)
}

func TestClient_Complete_WithOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]any
		_ = json.NewDecoder(r.Body).Decode(&reqBody)

		// 验证选项被传递
		assert.InDelta(t, 1000, reqBody["max_tokens"], 0.001)
		assert.InDelta(t, 0.7, reqBody["temperature"], 0.001)
		assert.Equal(t, "You are helpful.", reqBody["system"])

		resp := map[string]any{
			"content": []any{
				map[string]any{"type": "text", "text": "Response"},
			},
			"stop_reason": "end_turn",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := New(&Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})
	require.NoError(t, err)
	defer func() { _ = client.Close() }()

	opts := &llm.Options{
		MaxTokens:   1000,
		Temperature: 0.7,
		System:      "You are helpful.",
	}

	resp, err := client.Complete(context.Background(), []llm.Message{
		{Role: llm.RoleUser, Content: "Hello"},
	}, opts)

	require.NoError(t, err)
	require.NotNil(t, resp)
}

// ═══════════════════════════════════════════════════════════════════════════
// Stream 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestClient_Stream_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]any
		_ = json.NewDecoder(r.Body).Decode(&reqBody)

		// 验证 stream 参数
		assert.Equal(t, true, reqBody["stream"])

		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		assert.True(t, ok)
		if !ok {
			return
		}

		// 发送 SSE 事件
		events := []string{
			`event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}

`,
			`event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" World"}}

`,
			`event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn"}}

`,
			`event: message_stop
data: {"type":"message_stop"}

`,
		}

		for _, event := range events {
			_, _ = w.Write([]byte(event))
			flusher.Flush()
		}
	}))
	defer server.Close()

	client, err := New(&Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})
	require.NoError(t, err)
	defer func() { _ = client.Close() }()

	stream, err := client.Stream(context.Background(), []llm.Message{
		{Role: llm.RoleUser, Content: "Hello"},
	}, nil)

	require.NoError(t, err)
	require.NotNil(t, stream)

	var events []*llm.Event //nolint:prealloc // channel 收集数量未知
	for e := range stream {
		events = append(events, e)
	}

	// 验证收到了事件
	assert.NotEmpty(t, events)

	// 检查有文本事件
	hasText := false
	for _, e := range events {
		if e.Type == llm.EventTypeText && e.TextDelta != "" {
			hasText = true
			break
		}
	}
	assert.True(t, hasText, "Should have text events")
}

func TestClient_Stream_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	client, err := New(&Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})
	require.NoError(t, err)
	defer func() { _ = client.Close() }()

	stream, err := client.Stream(context.Background(), []llm.Message{
		{Role: llm.RoleUser, Content: "Hello"},
	}, nil)

	assert.Nil(t, stream)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API error: 500")
}

// ═══════════════════════════════════════════════════════════════════════════
// Close 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestClient_Close(t *testing.T) {
	client, err := New(&Config{
		APIKey: "test-key",
	})
	require.NoError(t, err)

	err = client.Close()
	assert.NoError(t, err)
}

// ═══════════════════════════════════════════════════════════════════════════
// buildRequest 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestClient_BuildRequest_WithTools(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]any
		_ = json.NewDecoder(r.Body).Decode(&reqBody)

		// 验证 tools 被传递
		tools, ok := reqBody["tools"].([]any)
		assert.True(t, ok)
		assert.Len(t, tools, 1)

		if len(tools) > 0 {
			tool, ok := tools[0].(map[string]any)
			assert.True(t, ok)
			assert.Equal(t, "get_weather", tool["name"])
		}

		resp := map[string]any{
			"content":     []any{map[string]any{"type": "text", "text": "Ok"}},
			"stop_reason": "end_turn",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := New(&Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})
	require.NoError(t, err)
	defer func() { _ = client.Close() }()

	opts := &llm.Options{
		Tools: []llm.ToolSchema{
			{
				Name:        "get_weather",
				Description: "Get weather info",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"city": map[string]any{"type": "string"},
					},
				},
			},
		},
	}

	resp, err := client.Complete(context.Background(), []llm.Message{
		{Role: llm.RoleUser, Content: "Hello"},
	}, opts)

	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestClient_BuildRequest_WithThinking(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]any
		_ = json.NewDecoder(r.Body).Decode(&reqBody)

		// 验证 thinking 配置
		thinking, ok := reqBody["thinking"].(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, "enabled", thinking["type"])
		assert.InDelta(t, 10000, thinking["budget"], 0.001)

		resp := map[string]any{
			"content":     []any{map[string]any{"type": "text", "text": "Response"}},
			"stop_reason": "end_turn",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := New(&Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})
	require.NoError(t, err)
	defer func() { _ = client.Close() }()

	opts := &llm.Options{
		EnableReasoning: true,
		ReasoningBudget: 10000,
	}

	resp, err := client.Complete(context.Background(), []llm.Message{
		{Role: llm.RoleUser, Content: "Think about this"},
	}, opts)

	require.NoError(t, err)
	require.NotNil(t, resp)
}

// ═══════════════════════════════════════════════════════════════════════════
// 接口实现验证
// ═══════════════════════════════════════════════════════════════════════════

func TestClient_ImplementsProvider(t *testing.T) {
	var _ llm.Provider = (*Client)(nil)
}
