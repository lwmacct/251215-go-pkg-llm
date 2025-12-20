package gemini

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
	assert.Equal(t, "gemini-1.5-flash", client.config.Model)
	assert.Contains(t, client.config.BaseURL, "generativelanguage.googleapis.com")
}

func TestNew_CustomValues(t *testing.T) {
	client, err := New(&Config{
		APIKey:  "test-key",
		BaseURL: "https://custom.api.example.com/v1",
		Model:   "gemini-2.5-pro",
		Timeout: 30 * time.Second,
		Headers: map[string]string{
			"X-Custom-Header": "custom-value",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Equal(t, "gemini-2.5-pro", client.config.Model)
	assert.Equal(t, "https://custom.api.example.com/v1", client.config.BaseURL)
}

func TestNew_VertexAI_NoAPIKeyRequired(t *testing.T) {
	// Vertex AI 模式不需要 API key
	client, err := New(&Config{
		VertexProject:  "my-project",
		VertexLocation: "us-central1",
	})

	require.NoError(t, err)
	require.NotNil(t, client)
	assert.True(t, client.useVertexAI)
	assert.Contains(t, client.config.BaseURL, "aiplatform.googleapis.com")
}

func TestNew_VertexAI_DefaultLocation(t *testing.T) {
	client, err := New(&Config{
		VertexProject: "my-project",
		// 不指定 VertexLocation，应该使用默认值
	})

	require.NoError(t, err)
	require.NotNil(t, client)
	// 默认 location 是 us-central1
	assert.Contains(t, client.config.BaseURL, "us-central1-aiplatform.googleapis.com")
}

// ═══════════════════════════════════════════════════════════════════════════
// Complete 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestClient_Complete_Success(t *testing.T) {
	// Mock HTTP Server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/models/gemini-1.5-flash:generateContent")
		assert.Contains(t, r.URL.RawQuery, "key=test-key")
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// 验证请求体
		var reqBody map[string]any
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		assert.NoError(t, err)
		assert.NotNil(t, reqBody["contents"])

		// 返回模拟响应
		resp := map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"role": "model",
						"parts": []any{
							map[string]any{
								"text": "Hello! I'm Gemini.",
							},
						},
					},
					"finishReason": "STOP",
				},
			},
			"usageMetadata": map[string]any{
				"promptTokenCount":     float64(10),
				"candidatesTokenCount": float64(5),
				"totalTokenCount":      float64(15),
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
	assert.Equal(t, "Hello! I'm Gemini.", resp.Message.Content)
	assert.Equal(t, "stop", resp.FinishReason)
	assert.Equal(t, "gemini-1.5-flash", resp.Model)
	require.NotNil(t, resp.Usage)
	assert.Equal(t, int64(10), resp.Usage.InputTokens)
	assert.Equal(t, int64(5), resp.Usage.OutputTokens)
}

func TestClient_Complete_WithToolCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"role": "model",
						"parts": []any{
							map[string]any{
								"text": "Let me check the weather.",
							},
							map[string]any{
								"functionCall": map[string]any{
									"name": "get_weather",
									"args": map[string]any{
										"city": "Tokyo",
									},
								},
							},
						},
					},
					"finishReason": "STOP",
				},
			},
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
	assert.True(t, resp.Message.HasToolCalls())

	toolCalls := resp.Message.GetToolCalls()
	require.Len(t, toolCalls, 1)
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

		// 验证系统指令
		systemInstr, ok := reqBody["systemInstruction"].(map[string]any)
		assert.True(t, ok, "Expected systemInstruction")
		if ok {
			parts, partsOK := systemInstr["parts"].([]any)
			assert.True(t, partsOK)
			if partsOK && len(parts) > 0 {
				firstPart, partOK := parts[0].(map[string]any)
				assert.True(t, partOK)
				assert.Equal(t, "You are helpful.", firstPart["text"])
			}
		}

		// 验证生成配置
		genConfig, ok := reqBody["generationConfig"].(map[string]any)
		assert.True(t, ok, "Expected generationConfig")
		assert.InDelta(t, 1000, genConfig["maxOutputTokens"], 0.001)
		assert.InDelta(t, 0.7, genConfig["temperature"], 0.001)

		resp := map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{map[string]any{"text": "Response"}},
					},
					"finishReason": "STOP",
				},
			},
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
		// 验证端点是 streamGenerateContent
		assert.Contains(t, r.URL.Path, "streamGenerateContent")

		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		assert.True(t, ok)
		if !ok {
			return
		}

		// 发送 SSE 事件 (Gemini 格式)
		events := []string{
			`data: {"candidates":[{"content":{"parts":[{"text":"Hello"}]}}]}

`,
			`data: {"candidates":[{"content":{"parts":[{"text":" World"}]}}]}

`,
			`data: {"candidates":[{"finishReason":"STOP"}]}

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
// buildEndpoint 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestClient_BuildEndpoint_GeminiAPI(t *testing.T) {
	client, err := New(&Config{
		APIKey: "test-key",
		Model:  "gemini-1.5-pro",
	})
	require.NoError(t, err)

	// Complete 端点
	endpoint := client.buildEndpoint(false)
	assert.Contains(t, endpoint, "/models/gemini-1.5-pro:generateContent")
	assert.Contains(t, endpoint, "key=test-key")

	// Stream 端点
	streamEndpoint := client.buildEndpoint(true)
	assert.Contains(t, streamEndpoint, "/models/gemini-1.5-pro:streamGenerateContent")
}

func TestClient_BuildEndpoint_VertexAI(t *testing.T) {
	client, err := New(&Config{
		VertexProject:  "my-project",
		VertexLocation: "asia-northeast1",
		Model:          "gemini-1.5-pro",
	})
	require.NoError(t, err)

	// Complete 端点
	endpoint := client.buildEndpoint(false)
	assert.Contains(t, endpoint, "/projects/my-project/locations/asia-northeast1")
	assert.Contains(t, endpoint, "/publishers/google/models/gemini-1.5-pro:generateContent")

	// Stream 端点
	streamEndpoint := client.buildEndpoint(true)
	assert.Contains(t, streamEndpoint, ":streamGenerateContent")
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
			toolWrapper, ok := tools[0].(map[string]any)
			assert.True(t, ok)
			functionDecls, ok := toolWrapper["functionDeclarations"].([]any)
			assert.True(t, ok)
			assert.Len(t, functionDecls, 1)

			if len(functionDecls) > 0 {
				funcDecl, ok := functionDecls[0].(map[string]any)
				assert.True(t, ok)
				assert.Equal(t, "get_weather", funcDecl["name"])
			}
		}

		resp := map[string]any{
			"candidates": []any{
				map[string]any{
					"content":      map[string]any{"parts": []any{map[string]any{"text": "Ok"}}},
					"finishReason": "STOP",
				},
			},
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
		thinkingConfig, ok := reqBody["thinkingConfig"].(map[string]any)
		assert.True(t, ok, "Expected thinkingConfig for Gemini 2.5 model")
		assert.Equal(t, true, thinkingConfig["includeThoughts"])
		assert.InDelta(t, 10000, thinkingConfig["thinkingBudget"], 0.001)

		resp := map[string]any{
			"candidates": []any{
				map[string]any{
					"content":      map[string]any{"parts": []any{map[string]any{"text": "Response"}}},
					"finishReason": "STOP",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 必须使用支持 thinking 的模型
	client, err := New(&Config{
		APIKey:         "test-key",
		BaseURL:        server.URL,
		Model:          "gemini-2.5-pro", // 支持 thinking 的模型
		EnableThinking: true,
		ThinkingBudget: 10000,
	})
	require.NoError(t, err)
	defer func() { _ = client.Close() }()

	resp, err := client.Complete(context.Background(), []llm.Message{
		{Role: llm.RoleUser, Content: "Think about this"},
	}, nil)

	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestClient_BuildRequest_ThinkingNotSupportedModel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]any
		_ = json.NewDecoder(r.Body).Decode(&reqBody)

		// 验证不支持 thinking 的模型不会有 thinkingConfig
		_, hasThinking := reqBody["thinkingConfig"]
		assert.False(t, hasThinking, "gemini-1.5-flash should not have thinkingConfig")

		resp := map[string]any{
			"candidates": []any{
				map[string]any{
					"content":      map[string]any{"parts": []any{map[string]any{"text": "Response"}}},
					"finishReason": "STOP",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 使用不支持 thinking 的模型
	client, err := New(&Config{
		APIKey:         "test-key",
		BaseURL:        server.URL,
		Model:          "gemini-1.5-flash", // 不支持 thinking
		EnableThinking: true,               // 即使开启也不会生效
	})
	require.NoError(t, err)
	defer func() { _ = client.Close() }()

	resp, err := client.Complete(context.Background(), []llm.Message{
		{Role: llm.RoleUser, Content: "Hello"},
	}, nil)

	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestClient_BuildRequest_WithResponseFormat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]any
		_ = json.NewDecoder(r.Body).Decode(&reqBody)

		// 验证结构化输出配置
		genConfig, ok := reqBody["generationConfig"].(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, "application/json", genConfig["responseMimeType"])
		assert.NotNil(t, genConfig["responseSchema"])

		resp := map[string]any{
			"candidates": []any{
				map[string]any{
					"content":      map[string]any{"parts": []any{map[string]any{"text": `{"name":"test"}`}}},
					"finishReason": "STOP",
				},
			},
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
		ResponseFormat: &llm.ResponseFormat{
			Type: "json_schema",
			Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{"type": "string"},
				},
			},
		},
	}

	resp, err := client.Complete(context.Background(), []llm.Message{
		{Role: llm.RoleUser, Content: "Return JSON"},
	}, opts)

	require.NoError(t, err)
	require.NotNil(t, resp)
}

// ═══════════════════════════════════════════════════════════════════════════
// 辅助函数测试
// ═══════════════════════════════════════════════════════════════════════════

func TestSupportsThinking(t *testing.T) {
	testCases := []struct {
		model    string
		expected bool
	}{
		{"gemini-2.5-pro", true},
		{"gemini-2.5-flash", true},
		{"gemini-2.5-flash-lite", false},
		{"gemini-2.0-flash", false},
		{"gemini-1.5-pro", false},
		{"gemini-1.5-flash", false},
	}

	for _, tc := range testCases {
		t.Run(tc.model, func(t *testing.T) {
			assert.Equal(t, tc.expected, supportsThinking(tc.model))
		})
	}
}

func TestMapSchemaType(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"string", "STRING"},
		{"number", "NUMBER"},
		{"integer", "INTEGER"},
		{"boolean", "BOOLEAN"},
		{"array", "ARRAY"},
		{"object", "OBJECT"},
		{"unknown", "STRING"}, // 默认 STRING
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, mapSchemaType(tc.input))
		})
	}
}

func TestConvertToGeminiSchema(t *testing.T) {
	// 测试 nil schema
	result := convertToGeminiSchema(nil)
	assert.Equal(t, "OBJECT", result["type"])

	// 测试完整 schema 转换
	schema := map[string]any{
		"type":        "object",
		"description": "Test schema",
		"properties": map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "Name field",
			},
			"count": map[string]any{
				"type": "integer",
			},
		},
		"required": []any{"name"},
	}

	result = convertToGeminiSchema(schema)

	assert.Equal(t, "OBJECT", result["type"])
	assert.Equal(t, "Test schema", result["description"])
	assert.Equal(t, []any{"name"}, result["required"])

	props, ok := result["properties"].(map[string]any)
	require.True(t, ok)
	nameField, ok := props["name"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "STRING", nameField["type"])
	assert.Equal(t, "Name field", nameField["description"])
}

func TestConvertToGeminiSchema_ArrayItems(t *testing.T) {
	schema := map[string]any{
		"type": "array",
		"items": map[string]any{
			"type": "string",
		},
	}

	result := convertToGeminiSchema(schema)

	assert.Equal(t, "ARRAY", result["type"])
	items, ok := result["items"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "STRING", items["type"])
}

func TestConvertToGeminiSchema_Enum(t *testing.T) {
	schema := map[string]any{
		"type": "string",
		"enum": []any{"small", "medium", "large"},
	}

	result := convertToGeminiSchema(schema)

	assert.Equal(t, "STRING", result["type"])
	assert.Equal(t, []any{"small", "medium", "large"}, result["enum"])
}

// ═══════════════════════════════════════════════════════════════════════════
// 接口实现验证
// ═══════════════════════════════════════════════════════════════════════════

func TestClient_ImplementsProvider(t *testing.T) {
	var _ llm.Provider = (*Client)(nil)
}
