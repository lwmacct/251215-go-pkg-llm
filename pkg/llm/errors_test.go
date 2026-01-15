package llm

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ═══════════════════════════════════════════════════════════════════════════
// ConfigError 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestConfigError(t *testing.T) {
	t.Run("创建配置错误（无底层错误）", func(t *testing.T) {
		err := NewConfigError("API key is required", nil)

		require.NotNil(t, err)
		assert.True(t, IsConfigError(err))
		assert.False(t, IsRequestError(err))
		assert.Contains(t, err.Error(), "config_error")
		assert.Contains(t, err.Error(), "API key is required")
	})

	t.Run("创建配置错误（带底层错误）", func(t *testing.T) {
		underlying := errors.New("underlying error")
		err := NewConfigError("invalid config", underlying)

		require.NotNil(t, err)
		assert.True(t, IsConfigError(err))
		assert.Contains(t, err.Error(), "config_error")
		assert.Contains(t, err.Error(), "invalid config")
		assert.Contains(t, err.Error(), "underlying error")
	})

	t.Run("错误链支持", func(t *testing.T) {
		underlying := errors.New("underlying error")
		err := NewConfigError("config failed", underlying)

		require.ErrorIs(t, err, underlying)
		assert.Equal(t, underlying, errors.Unwrap(err))
	})
}

// ═══════════════════════════════════════════════════════════════════════════
// RequestError 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestRequestError(t *testing.T) {
	t.Run("创建请求错误", func(t *testing.T) {
		err := NewRequestError("marshal", errors.New("JSON error"))

		require.NotNil(t, err)
		assert.True(t, IsRequestError(err))
		assert.False(t, IsConfigError(err))
		assert.Equal(t, "marshal", err.Stage)
		assert.Contains(t, err.Error(), "request_error")
		assert.Contains(t, err.Error(), "marshal")
	})

	t.Run("不同阶段的错误", func(t *testing.T) {
		stages := []string{"marshal", "build", "validate"}
		for _, stage := range stages {
			err := NewRequestError(stage, errors.New(stage+" error"))
			assert.Equal(t, stage, err.Stage)
			assert.Contains(t, err.Error(), "failed to "+stage)
		}
	})
}

// ═══════════════════════════════════════════════════════════════════════════
// HTTPError 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestHTTPError(t *testing.T) {
	t.Run("创建 HTTP 错误", func(t *testing.T) {
		err := NewHTTPError("connection failed", errors.New("timeout"))

		require.NotNil(t, err)
		assert.True(t, IsHTTPError(err))
		assert.False(t, IsAPIError(err))
		assert.Contains(t, err.Error(), "http_error")
		assert.Contains(t, err.Error(), "connection failed")
	})
}

// ═══════════════════════════════════════════════════════════════════════════
// APIError 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestAPIError(t *testing.T) {
	t.Run("创建 API 错误", func(t *testing.T) {
		err := NewAPIError(401, "Unauthorized")

		require.NotNil(t, err)
		assert.True(t, IsAPIError(err))
		assert.False(t, IsConfigError(err))
		assert.Equal(t, 401, err.StatusCode)
		assert.Equal(t, "Unauthorized", err.Response)
		assert.Contains(t, err.Error(), "api_error")
		assert.Contains(t, err.Error(), "401")
	})

	t.Run("API 错误链式设置", func(t *testing.T) {
		err := NewAPIError(429, "Rate limit exceeded").
			WithProvider("openai").
			WithRequestID("req-123").
			WithErrorCode("rate_limit_exceeded")

		assert.Equal(t, "openai", err.Provider)
		assert.Equal(t, "req-123", err.RequestID)
		assert.Equal(t, "rate_limit_exceeded", err.ErrorCode)
		assert.Contains(t, err.Error(), "req-123")
	})

	t.Run("IsRetryable 判断", func(t *testing.T) {
		tests := []struct {
			name       string
			statusCode int
			retryable  bool
		}{
			{"200 OK", 200, false},
			{"400 Bad Request", 400, false},
			{"401 Unauthorized", 401, false},
			{"404 Not Found", 404, false},
			{"429 Rate Limit", 429, true},
			{"500 Internal Server Error", 500, true},
			{"502 Bad Gateway", 502, true},
			{"503 Service Unavailable", 503, true},
			{"504 Gateway Timeout", 504, true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := NewAPIError(tt.statusCode, "error")
				assert.Equal(t, tt.retryable, err.IsRetryable())
			})
		}
	})

	t.Run("GetAPIError 提取", func(t *testing.T) {
		apiErr := NewAPIError(500, "Server Error")

		// 成功提取
		extracted, ok := GetAPIError(apiErr)
		assert.True(t, ok)
		assert.Equal(t, 500, extracted.StatusCode)

		// 非错误类型无法提取
		_, ok = GetAPIError(errors.New("other error"))
		assert.False(t, ok)
	})

	t.Run("GetStatusCode 提取", func(t *testing.T) {
		err := NewAPIError(403, "Forbidden")
		assert.Equal(t, 403, GetStatusCode(err))

		// 非 API 错误返回 0
		assert.Equal(t, 0, GetStatusCode(errors.New("other error")))
	})
}

// ═══════════════════════════════════════════════════════════════════════════
// ResponseError 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestResponseError(t *testing.T) {
	t.Run("创建响应解析错误", func(t *testing.T) {
		err := NewResponseError("content", errors.New("parse error"))

		require.NotNil(t, err)
		assert.True(t, IsResponseError(err))
		assert.Equal(t, "content", err.Field)
		assert.Contains(t, err.Error(), "response_error")
		assert.Contains(t, err.Error(), "content")
	})
}

// ═══════════════════════════════════════════════════════════════════════════
// StreamError 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestStreamError(t *testing.T) {
	t.Run("创建流式错误", func(t *testing.T) {
		err := NewStreamError("SSE parse failed", errors.New("invalid format"))

		require.NotNil(t, err)
		assert.True(t, IsStreamError(err))
		assert.Contains(t, err.Error(), "stream_error")
		assert.Contains(t, err.Error(), "SSE parse failed")
	})
}

// ═══════════════════════════════════════════════════════════════════════════
// 错误匹配函数测试
// ═══════════════════════════════════════════════════════════════════════════

func TestErrorMatching(t *testing.T) {
	t.Run("IsRetryableError 函数", func(t *testing.T) {
		// 可重试的错误
		assert.True(t, IsRetryableError(NewAPIError(429, "")))
		assert.True(t, IsRetryableError(NewAPIError(500, "")))

		// 不可重试的错误
		assert.False(t, IsRetryableError(NewAPIError(400, "")))
		assert.False(t, IsRetryableError(NewAPIError(401, "")))
		assert.False(t, IsRetryableError(NewConfigError("", nil)))
	})

	t.Run("多种错误类型匹配", func(t *testing.T) {
		errors := []struct {
			err error
			fn  func(error) bool
		}{
			{NewConfigError("", nil), IsConfigError},
			{NewRequestError("", nil), IsRequestError},
			{NewHTTPError("", nil), IsHTTPError},
			{NewAPIError(500, ""), IsAPIError},
			{NewResponseError("", nil), IsResponseError},
			{NewStreamError("", nil), IsStreamError},
		}

		for _, tt := range errors {
			assert.True(t, tt.fn(tt.err), "Error type check failed")
		}
	})
}

// ═══════════════════════════════════════════════════════════════════════════
// 错误链测试
// ═══════════════════════════════════════════════════════════════════════════

func TestErrorChaining(t *testing.T) {
	t.Run("嵌套错误链", func(t *testing.T) {
		underlying := errors.New("root cause")
		requestErr := NewRequestError("marshal", underlying)
		configErr := NewConfigError("config invalid", requestErr)

		// 验证错误链
		require.ErrorIs(t, configErr, underlying)
		require.ErrorIs(t, configErr, requestErr)
		assert.True(t, IsConfigError(configErr))

		// 验证 Unwrap
		unwrapped := errors.Unwrap(configErr)
		assert.Equal(t, requestErr, unwrapped)
	})

	t.Run("多次 Unwrap", func(t *testing.T) {
		root := errors.New("root")
		l1 := NewRequestError("stage1", root)
		l2 := NewRequestError("stage2", l1)
		l3 := NewRequestError("stage3", l2)

		// 连续 Unwrap
		u1 := errors.Unwrap(l3)
		assert.Equal(t, l2, u1)

		u2 := errors.Unwrap(u1)
		assert.Equal(t, l1, u2)

		u3 := errors.Unwrap(u2)
		assert.Equal(t, root, u3)
	})
}

// ═══════════════════════════════════════════════════════════════════════════
// 集成测试场景
// ═══════════════════════════════════════════════════════════════════════════

func TestErrorScenarios(t *testing.T) {
	t.Run("API 错误完整场景", func(t *testing.T) {
		// 模拟 API 返回 401
		err := NewAPIError(401, `{"error": {"message": "Invalid API key"}}`).
			WithProvider("openai").
			WithRequestID("req-abc123").
			WithErrorCode("invalid_api_key")

		// 验证所有信息
		assert.True(t, IsAPIError(err))
		assert.Equal(t, 401, err.StatusCode)
		assert.Equal(t, "openai", err.Provider)
		assert.Equal(t, "req-abc123", err.RequestID)
		assert.Equal(t, "invalid_api_key", err.ErrorCode)
		assert.Contains(t, err.Error(), "req-abc123")

		// 验证不可重试
		assert.False(t, err.IsRetryable())
	})

	t.Run("可重试的 429 错误", func(t *testing.T) {
		err := NewAPIError(http.StatusTooManyRequests, "Rate limit").
			WithProvider("anthropic").
			WithRequestID("req-xyz").
			WithErrorCode("rate_limit")

		assert.True(t, err.IsRetryable())
		assert.True(t, IsRetryableError(err))
	})

	t.Run("配置错误导致请求失败", func(t *testing.T) {
		configErr := NewConfigError("missing API key", nil)
		requestErr := NewRequestError("build", configErr)

		// 既是请求错误，底层也是配置错误
		assert.True(t, IsRequestError(requestErr))
		require.ErrorIs(t, requestErr, configErr)
		assert.True(t, IsConfigError(errors.Unwrap(requestErr)))
	})
}
