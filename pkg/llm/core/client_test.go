package core

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
)

// ═══════════════════════════════════════════════════════════════════════════
// Mock 实现
// ═══════════════════════════════════════════════════════════════════════════

// mockConfig Mock 配置实现
type mockConfig struct {
	apiKey       string
	baseURL      string
	model        string
	providerName string
}

func (m *mockConfig) Validate() error {
	if m.apiKey == "" {
		return llm.NewConfigError("API key is required", nil)
	}
	return nil
}

func (m *mockConfig) GetDefaults() (string, string, time.Duration) {
	baseURL := m.baseURL
	if baseURL == "" {
		baseURL = "https://api.example.com/v1"
	}
	model := m.model
	if model == "" {
		model = "test-model"
	}
	timeout := 30 * time.Second
	return baseURL, model, timeout
}

func (m *mockConfig) BuildHeaders() map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + m.apiKey,
		"Content-Type":  "application/json",
	}
}

func (m *mockConfig) ProviderName() string {
	return m.providerName
}

func (m *mockConfig) GetAPIKey() string {
	return m.apiKey
}

func (m *mockConfig) GetModel() string {
	return m.model
}

// mockRequestBuilder Mock 请求构建器
type mockRequestBuilder struct {
	requestBody map[string]any
}

func (m *mockRequestBuilder) BuildRequest(messages []llm.Message, opts *llm.Options, stream bool) (map[string]any, error) {
	if m.requestBody != nil {
		return m.requestBody, nil
	}

	return map[string]any{
		"model":    "test-model",
		"messages": []map[string]any{{"role": "user", "content": "Hello"}},
		"stream":   stream,
	}, nil
}

// mockAdapter Mock 协议适配器
type mockAdapter struct{}

func (m *mockAdapter) ConvertToAPI(messages []llm.Message) []map[string]any {
	result := make([]map[string]any, len(messages))
	for i, msg := range messages {
		result[i] = map[string]any{
			"role":    string(msg.Role),
			"content": msg.Content,
		}
	}
	return result
}

func (m *mockAdapter) ConvertFromAPI(apiResp map[string]any) (llm.Message, string) {
	return llm.Message{
		Role:    llm.RoleAssistant,
		Content: "Test response",
	}, "stop"
}

func (m *mockAdapter) ConvertUsage(apiResp map[string]any) *llm.TokenUsage {
	return &llm.TokenUsage{
		InputTokens:  10,
		OutputTokens: 20,
		TotalTokens:  30,
	}
}

func (m *mockAdapter) GetSystemMessageHandling() SystemMessageStrategy {
	return SystemInline
}

// mockEventHandler Mock SSE 事件处理器
type mockEventHandler struct{}

func (m *mockEventHandler) HandleEvent(eventType string, data map[string]any) ([]*llm.Event, bool) {
	return []*llm.Event{
		{
			Type:      llm.EventTypeText,
			TextDelta: "test",
		},
	}, false
}

func (m *mockEventHandler) ShouldStopOnData(data string) bool {
	return data == "[DONE]"
}

// ═══════════════════════════════════════════════════════════════════════════
// BaseClient 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestNewBaseClient(t *testing.T) {
	t.Run("成功创建 BaseClient", func(t *testing.T) {
		config := &mockConfig{
			apiKey:  "test-key",
			baseURL: "https://api.example.com/v1",
		}

		client, err := NewBaseClient(config, &mockAdapter{}, &mockEventHandler{})

		require.NoError(t, err)
		require.NotNil(t, client)
		assert.NotNil(t, client.resty)
		assert.NotNil(t, client.transformer)
		assert.NotNil(t, client.sseParser)
	})

	t.Run("配置验证失败", func(t *testing.T) {
		config := &mockConfig{apiKey: ""} // 空 API key

		client, err := NewBaseClient(config, &mockAdapter{}, &mockEventHandler{})

		require.Error(t, err)
		assert.Nil(t, client)
		assert.True(t, llm.IsConfigError(err))
	})
}

func TestBaseClient_Complete(t *testing.T) {
	t.Run("成功的 Complete 请求", func(t *testing.T) {
		// Mock HTTP Server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 验证请求
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "/chat/completions", r.URL.Path)
			assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

			// 返回模拟响应
			response := map[string]any{
				"id":     "test-id",
				"object": "chat.completion",
				"model":  "test-model",
				"choices": []any{
					map[string]any{
						"message": map[string]any{
							"role":    "assistant",
							"content": "Test response",
						},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]any{
					"prompt_tokens":     10,
					"completion_tokens": 20,
					"total_tokens":      30,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		// 创建客户端
		config := &mockConfig{
			apiKey:  "test-key",
			baseURL: server.URL,
			model:   "test-model",
		}
		client, err := NewBaseClient(config, &mockAdapter{}, &mockEventHandler{})
		require.NoError(t, err)

		// 调用 Complete
		messages := []llm.Message{
			{Role: llm.RoleUser, Content: "Hello"},
		}
		requestBuilder := &mockRequestBuilder{}

		resp, err := client.Complete(context.Background(), messages, nil, requestBuilder)

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, llm.RoleAssistant, resp.Message.Role)
		assert.Equal(t, "Test response", resp.Message.Content)
		assert.Equal(t, "stop", resp.FinishReason)
		assert.Equal(t, "test-model", resp.Model)
		assert.NotNil(t, resp.Usage)
		assert.Equal(t, int64(30), resp.Usage.TotalTokens)
	})

	t.Run("API 返回错误 (401)", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Request-ID", "req-123")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error": {"message": "Invalid API key"}}`))
		}))
		defer server.Close()

		config := &mockConfig{
			apiKey:       "invalid-key",
			baseURL:      server.URL,
			providerName: "test-provider",
		}
		client, err := NewBaseClient(config, &mockAdapter{}, &mockEventHandler{})
		require.NoError(t, err)

		messages := []llm.Message{{Role: llm.RoleUser, Content: "Hello"}}
		requestBuilder := &mockRequestBuilder{}

		resp, err := client.Complete(context.Background(), messages, nil, requestBuilder)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.True(t, llm.IsAPIError(err))

		apiErr, ok := llm.GetAPIError(err)
		require.True(t, ok)
		assert.Equal(t, 401, apiErr.StatusCode)
		assert.Equal(t, "test-provider", apiErr.Provider)
		assert.Equal(t, "req-123", apiErr.RequestID)
		assert.Contains(t, apiErr.Response, "Invalid API key")
	})

	t.Run("网络错误", func(t *testing.T) {
		// 使用无效 URL 模拟网络错误
		config := &mockConfig{
			apiKey:  "test-key",
			baseURL: "http://invalid-host-12345:9999",
		}
		client, err := NewBaseClient(config, &mockAdapter{}, &mockEventHandler{})
		require.NoError(t, err)

		messages := []llm.Message{{Role: llm.RoleUser, Content: "Hello"}}
		requestBuilder := &mockRequestBuilder{}

		resp, err := client.Complete(context.Background(), messages, nil, requestBuilder)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.True(t, llm.IsHTTPError(err))
	})
}

func TestBaseClient_Stream(t *testing.T) {
	t.Run("成功的 Stream 请求", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "/chat/completions", r.URL.Path)

			// 发送 SSE 流
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")

			// 发送一些事件
			_, _ = fmt.Fprint(w, "data: {\"content\": \"Hello\"}\n\n")
			_, _ = fmt.Fprint(w, "data: {\"content\": \" World\"}\n\n")
			_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
		}))
		defer server.Close()

		config := &mockConfig{
			apiKey:  "test-key",
			baseURL: server.URL,
		}
		client, err := NewBaseClient(config, &mockAdapter{}, &mockEventHandler{})
		require.NoError(t, err)

		messages := []llm.Message{{Role: llm.RoleUser, Content: "Hello"}}
		requestBuilder := &mockRequestBuilder{}

		events, err := client.Stream(context.Background(), messages, nil, requestBuilder)

		require.NoError(t, err)
		require.NotNil(t, events)

		// 接收事件
		eventCount := 0
		for range events {
			eventCount++
		}

		assert.Positive(t, eventCount)
	})

	t.Run("Stream 返回错误", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error": "Rate limit exceeded"}`))
		}))
		defer server.Close()

		config := &mockConfig{
			apiKey:       "test-key",
			baseURL:      server.URL,
			providerName: "test-provider",
		}
		client, err := NewBaseClient(config, &mockAdapter{}, &mockEventHandler{})
		require.NoError(t, err)

		messages := []llm.Message{{Role: llm.RoleUser, Content: "Hello"}}
		requestBuilder := &mockRequestBuilder{}

		events, err := client.Stream(context.Background(), messages, nil, requestBuilder)

		require.Error(t, err)
		assert.Nil(t, events)
		assert.True(t, llm.IsAPIError(err))

		apiErr, ok := llm.GetAPIError(err)
		require.True(t, ok)
		assert.Equal(t, 429, apiErr.StatusCode)
		assert.True(t, apiErr.IsRetryable())
	})
}

func TestBaseClient_EndpointBuilder(t *testing.T) {
	t.Run("使用自定义端点构建器", func(t *testing.T) {
		mockBuilder := &mockEndpointBuilder{
			completeEndpoint: "/v1/chat",
			streamEndpoint:   "/v1/chat/stream",
		}

		config := &mockConfig{apiKey: "test-key"}
		client, err := NewBaseClient(config, &mockAdapter{}, &mockEventHandler{})
		require.NoError(t, err)

		client.SetEndpointBuilder(mockBuilder)

		assert.Equal(t, "/v1/chat", client.getCompleteEndpoint())
		assert.Equal(t, "/v1/chat/stream", client.getStreamEndpoint())
	})

	t.Run("使用默认端点", func(t *testing.T) {
		config := &mockConfig{apiKey: "test-key"}
		client, err := NewBaseClient(config, &mockAdapter{}, &mockEventHandler{})
		require.NoError(t, err)

		assert.Equal(t, "/chat/completions", client.getCompleteEndpoint())
		assert.Equal(t, "/chat/completions", client.getStreamEndpoint())
	})
}

// ═══════════════════════════════════════════════════════════════════════════
// Mock EndpointBuilder
// ═══════════════════════════════════════════════════════════════════════════

type mockEndpointBuilder struct {
	completeEndpoint string
	streamEndpoint   string
}

func (m *mockEndpointBuilder) BuildCompleteEndpoint() string {
	return m.completeEndpoint
}

func (m *mockEndpointBuilder) BuildStreamEndpoint() string {
	return m.streamEndpoint
}

// ═══════════════════════════════════════════════════════════════════════════
// 辅助函数测试
// ═══════════════════════════════════════════════════════════════════════════

func TestGetDefaultTimeout(t *testing.T) {
	t.Run("零超时返回默认值", func(t *testing.T) {
		timeout := GetDefaultTimeout(0)
		assert.Equal(t, 120*time.Second, timeout)
	})

	t.Run("非零超时保持不变", func(t *testing.T) {
		timeout := GetDefaultTimeout(30 * time.Second)
		assert.Equal(t, 30*time.Second, timeout)
	})
}

func TestNewInvalidConfigError(t *testing.T) {
	err := NewInvalidConfigError("model")

	assert.True(t, llm.IsConfigError(err))
	assert.Contains(t, err.Error(), "model")
}

func TestNewMissingAPIKeyError(t *testing.T) {
	err := NewMissingAPIKeyError()

	assert.True(t, llm.IsConfigError(err))
	assert.Contains(t, err.Error(), "API key")
}
