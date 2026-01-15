package llm

import (
	"errors"
	"fmt"
	"net/http"
)

// ═══════════════════════════════════════════════════════════════════════════
// 错误类型
// ═══════════════════════════════════════════════════════════════════════════

// ErrorType 错误类型
type ErrorType string

const (
	// ErrTypeConfig 配置错误
	ErrTypeConfig ErrorType = "config_error"

	// ErrTypeRequest 请求错误（序列化、构建等）
	ErrTypeRequest ErrorType = "request_error"

	// ErrTypeHTTP HTTP 层错误（网络、超时等）
	ErrTypeHTTP ErrorType = "http_error"

	// ErrTypeAPI API 业务错误（4xx, 5xx）
	ErrTypeAPI ErrorType = "api_error"

	// ErrTypeResponse 响应解析错误
	ErrTypeResponse ErrorType = "response_error"

	// ErrTypeStream 流式错误
	ErrTypeStream ErrorType = "stream_error"
)

// ═══════════════════════════════════════════════════════════════════════════
// 基础错误
// ═══════════════════════════════════════════════════════════════════════════

// BaseError 基础错误实现
type BaseError struct {
	Type    ErrorType
	Message string
	Err     error
}

func (e *BaseError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *BaseError) Unwrap() error {
	return e.Err
}

// ═══════════════════════════════════════════════════════════════════════════
// 配置错误
// ═══════════════════════════════════════════════════════════════════════════

// ConfigError 配置错误
type ConfigError struct {
	*BaseError
}

// NewConfigError 创建配置错误
func NewConfigError(message string, err error) *ConfigError {
	return &ConfigError{
		BaseError: &BaseError{
			Type:    ErrTypeConfig,
			Message: message,
			Err:     err,
		},
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// 请求错误
// ═══════════════════════════════════════════════════════════════════════════

// RequestError 请求错误
type RequestError struct {
	*BaseError

	Stage string // "marshal", "build", etc.
}

// NewRequestError 创建请求错误
func NewRequestError(stage string, err error) *RequestError {
	return &RequestError{
		BaseError: &BaseError{
			Type:    ErrTypeRequest,
			Message: fmt.Sprintf("failed to %s request", stage),
			Err:     err,
		},
		Stage: stage,
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// HTTP 错误
// ═══════════════════════════════════════════════════════════════════════════

// HTTPError HTTP 层错误
type HTTPError struct {
	*BaseError
}

// NewHTTPError 创建 HTTP 错误
func NewHTTPError(message string, err error) *HTTPError {
	return &HTTPError{
		BaseError: &BaseError{
			Type:    ErrTypeHTTP,
			Message: message,
			Err:     err,
		},
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// API 错误
// ═══════════════════════════════════════════════════════════════════════════

// APIError API 业务错误
type APIError struct {
	*BaseError

	StatusCode int
	Response   string
	Provider   string
	RequestID  string
	ErrorCode  string // Provider 特定的错误代码
}

// NewAPIError 创建 API 错误
func NewAPIError(statusCode int, response string) *APIError {
	return &APIError{
		BaseError: &BaseError{
			Type:    ErrTypeAPI,
			Message: fmt.Sprintf("API returned error status %d", statusCode),
		},
		StatusCode: statusCode,
		Response:   response,
	}
}

// WithProvider 设置 Provider 名称
func (e *APIError) WithProvider(provider string) *APIError {
	e.Provider = provider
	return e
}

// WithRequestID 设置请求 ID
func (e *APIError) WithRequestID(requestID string) *APIError {
	e.RequestID = requestID
	return e
}

// WithErrorCode 设置错误代码
func (e *APIError) WithErrorCode(code string) *APIError {
	e.ErrorCode = code
	return e
}

func (e *APIError) Error() string {
	base := e.BaseError.Error()
	if e.RequestID != "" {
		return fmt.Sprintf("%s (request_id: %s)", base, e.RequestID)
	}
	return base
}

// IsRetryable 检查错误是否可重试
func (e *APIError) IsRetryable() bool {
	// 429 (Rate Limit), 500, 502, 503, 504 可重试
	return e.StatusCode == http.StatusTooManyRequests ||
		e.StatusCode >= 500 && e.StatusCode <= 504
}

// ═══════════════════════════════════════════════════════════════════════════
// 响应解析错误
// ═══════════════════════════════════════════════════════════════════════════

// ResponseError 响应解析错误
type ResponseError struct {
	*BaseError

	Field string // 出错的字段
}

// NewResponseError 创建响应错误
func NewResponseError(field string, err error) *ResponseError {
	return &ResponseError{
		BaseError: &BaseError{
			Type:    ErrTypeResponse,
			Message: fmt.Sprintf("failed to parse response field '%s'", field),
			Err:     err,
		},
		Field: field,
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// 流式错误
// ═══════════════════════════════════════════════════════════════════════════

// StreamError 流式错误
type StreamError struct {
	*BaseError
}

// NewStreamError 创建流式错误
func NewStreamError(message string, err error) *StreamError {
	return &StreamError{
		BaseError: &BaseError{
			Type:    ErrTypeStream,
			Message: message,
			Err:     err,
		},
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// 错误匹配函数（支持 errors.Is/As）
// ═══════════════════════════════════════════════════════════════════════════

// IsConfigError 检查是否为配置错误
func IsConfigError(err error) bool {
	var e *ConfigError
	return errors.As(err, &e)
}

// IsRequestError 检查是否为请求错误
func IsRequestError(err error) bool {
	var e *RequestError
	return errors.As(err, &e)
}

// IsHTTPError 检查是否为 HTTP 错误
func IsHTTPError(err error) bool {
	var e *HTTPError
	return errors.As(err, &e)
}

// IsAPIError 检查是否为 API 错误
func IsAPIError(err error) bool {
	var e *APIError
	return errors.As(err, &e)
}

// IsResponseError 检查是否为响应解析错误
func IsResponseError(err error) bool {
	var e *ResponseError
	return errors.As(err, &e)
}

// IsStreamError 检查是否为流式错误
func IsStreamError(err error) bool {
	var e *StreamError
	return errors.As(err, &e)
}

// IsRetryableError 检查错误是否可重试
func IsRetryableError(err error) bool {
	var e *APIError
	if errors.As(err, &e) {
		return e.IsRetryable()
	}
	return false
}

// GetAPIError 提取 APIError（如果存在）
func GetAPIError(err error) (*APIError, bool) {
	var e *APIError
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}

// GetStatusCode 提取 HTTP 状态码（如果是 API 错误）
func GetStatusCode(err error) int {
	if e, ok := GetAPIError(err); ok {
		return e.StatusCode
	}
	return 0
}
