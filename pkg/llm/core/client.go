package core

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-resty/resty/v2"

	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
)

// ═══════════════════════════════════════════════════════════════════════════
// 接口定义
// ═══════════════════════════════════════════════════════════════════════════

// ProviderConfig Provider 配置接口
//
// 每个 Provider 实现此接口来定义其特有的配置和默认值。
type ProviderConfig interface {
	// Validate 验证配置
	// 返回错误如果配置无效
	Validate() error

	// GetDefaults 获取默认值
	// 返回 baseURL, model, timeout
	GetDefaults() (baseURL, model string, timeout time.Duration)

	// BuildHeaders 构建请求头
	// 返回认证头和其他必要的 HTTP 头
	BuildHeaders() map[string]string

	// ProviderName 返回 Provider 名称
	// 用于错误日志和追踪
	ProviderName() string
}

// EndpointBuilder 端点构建器接口
//
// 某些 Provider（如 Gemini）需要动态构建端点。
type EndpointBuilder interface {
	// BuildCompleteEndpoint 构建 Complete 端点
	BuildCompleteEndpoint() string

	// BuildStreamEndpoint 构建 Stream 端点
	BuildStreamEndpoint() string
}

// RequestBuilder 请求构建器接口
//
// 每个 Provider 实现此接口来定义协议特定的请求体构建逻辑。
type RequestBuilder interface {
	// BuildRequest 构建请求体
	// 返回 API 特定格式的请求体 map
	BuildRequest(messages []llm.Message, opts *llm.Options, stream bool) (map[string]any, error)
}

// ═══════════════════════════════════════════════════════════════════════════
// BaseClient 基础客户端
// ═══════════════════════════════════════════════════════════════════════════

// BaseClient 基础客户端
//
// 封装了 HTTP 通信、协议适配、错误处理等通用逻辑。
// 所有 Provider 可以嵌入 BaseClient 来复用这些功能。
//
// 架构设计：
//   - 模板方法模式：定义请求流程骨架，具体差异委托给接口
//   - 依赖倒置：依赖抽象的接口（ProviderConfig、RequestBuilder）
//   - 单一职责：只负责 HTTP 通信和通用流程编排
//
// 使用示例：
//
//	config := &openai.Config{APIKey: "sk-xxx"}
//	baseClient, _ := core.NewBaseClient(config, openai.NewAdapter(), openai.NewEventHandler())
//
//	client := &openai.Client{BaseClient: baseClient, config: config}
type BaseClient struct {
	config          ProviderConfig
	resty           *resty.Client
	transformer     *Transformer
	sseParser       *SSEParser
	endpointBuilder EndpointBuilder // 可选，用于 Gemini 等动态端点的 Provider
}

// NewBaseClient 创建基础客户端
//
// 参数：
//   - config: Provider 特定配置，实现 ProviderConfig 接口
//   - adapter: 协议适配器，处理消息格式转换
//   - eventHandler: SSE 事件处理器，处理流式响应
//
// 返回：
//   - BaseClient 实例
//   - 错误（如果配置验证失败）
func NewBaseClient(
	config ProviderConfig,
	adapter ProtocolAdapter,
	eventHandler EventHandler,
) (*BaseClient, error) {
	// 1. 验证配置
	if err := config.Validate(); err != nil {
		return nil, llm.NewConfigError("config validation failed", err)
	}

	// 2. 获取默认值
	baseURL, _, timeout := config.GetDefaults()

	// 3. 构建请求头
	headers := config.BuildHeaders()

	// 4. 创建 resty 客户端
	r := resty.New()
	r.SetBaseURL(baseURL)
	r.SetTimeout(timeout)
	for k, v := range headers {
		r.SetHeader(k, v)
	}

	// 5. 创建协议适配器和转换器
	transformer := NewTransformer(adapter)
	sseParser := NewSSEParser(eventHandler)

	return &BaseClient{
		config:      config,
		resty:       r,
		transformer: transformer,
		sseParser:   sseParser,
	}, nil
}

// SetEndpointBuilder 设置端点构建器
//
// 某些 Provider（如 Gemini）需要动态构建端点。
func (c *BaseClient) SetEndpointBuilder(builder EndpointBuilder) {
	c.endpointBuilder = builder
}

// Complete 同步完成（通用实现）
//
// 实现了 llm.Provider 接口的 Complete 方法。
//
// 通用流程：
//  1. 构建 API 请求体（委托给 RequestBuilder）
//  2. 序列化请求体
//  3. 发送 HTTP POST 请求
//  4. 检查 HTTP 状态码
//  5. 解析响应（使用 Transformer）
//  6. 返回统一格式的 Response
//
// 参数：
//   - ctx: 上下文，支持取消和超时
//   - messages: 对话消息列表
//   - opts: 请求选项（temperature、max_tokens 等）
//   - requestBuilder: 请求构建器，实现协议特定的请求体构建
//
// 返回：
//   - Response: 统一格式的响应
//   - 错误：配置错误、请求错误、API 错误等
func (c *BaseClient) Complete(
	ctx context.Context,
	messages []llm.Message,
	opts *llm.Options,
	requestBuilder RequestBuilder,
) (*llm.Response, error) {
	// 1. 构建请求体
	body, err := requestBuilder.BuildRequest(messages, opts, false)
	if err != nil {
		return nil, llm.NewRequestError("build request", err)
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, llm.NewRequestError("marshal request", err)
	}

	// 2. 确定端点
	endpoint := c.getCompleteEndpoint()

	// 3. 发送请求
	var apiResp map[string]any
	resp, err := c.resty.R().
		SetContext(ctx).
		SetBody(bodyBytes).
		SetResult(&apiResp).
		Post(endpoint)
	if err != nil {
		return nil, llm.NewHTTPError("request failed", err)
	}

	// 4. 检查 HTTP 错误
	if resp.StatusCode() >= 400 {
		apiErr := llm.NewAPIError(resp.StatusCode(), resp.String())

		// 尝试提取请求 ID（从响应头）
		if requestID := resp.Header().Get("X-Request-ID"); requestID != "" {
			apiErr = apiErr.WithRequestID(requestID)
		}

		// 设置 Provider 类型
		apiErr = apiErr.WithProvider(c.config.ProviderName())

		return nil, apiErr
	}

	// 5. 解析响应
	msg, finishReason, usage := c.transformer.ParseAPIResponse(apiResp)

	// 6. 提取模型（如果响应中有）
	model := c.getModelFromConfig()
	if respModel, ok := apiResp["model"].(string); ok && respModel != "" {
		model = respModel
	}

	return &llm.Response{
		Message:      msg,
		FinishReason: finishReason,
		Model:        model,
		Usage:        usage,
	}, nil
}

// Stream 流式完成（通用实现）
//
// 实现了 llm.Provider 接口的 Stream 方法。
//
// 通用流程：
//  1. 构建 API 请求体（委托给 RequestBuilder）
//  2. 序列化请求体
//  3. 发送 HTTP POST 请求（不解析响应）
//  4. 检查 HTTP 状态码
//  5. 启动 SSE 解析（在 goroutine 中）
//  6. 返回事件 channel
//
// 参数：
//   - ctx: 上下文，支持取消和超时
//   - messages: 对话消息列表
//   - opts: 请求选项
//   - requestBuilder: 请求构建器
//
// 返回：
//   - Event channel: 逐块接收 LLM 响应
//   - 错误：配置错误、请求错误、API 错误等
//
// 注意：
//   - 返回的 channel 缓冲区大小为 10
//   - SSE 解析在 goroutine 中进行
//   - 完成或出错后 channel 会自动关闭
func (c *BaseClient) Stream(
	ctx context.Context,
	messages []llm.Message,
	opts *llm.Options,
	requestBuilder RequestBuilder,
) (<-chan *llm.Event, error) {
	// 1. 构建请求体
	body, err := requestBuilder.BuildRequest(messages, opts, true)
	if err != nil {
		return nil, llm.NewRequestError("build request", err)
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, llm.NewRequestError("marshal request", err)
	}

	// 2. 确定端点
	endpoint := c.getStreamEndpoint()

	// 3. 发送请求（不解析响应）
	resp, err := c.resty.R().
		SetContext(ctx).
		SetBody(bodyBytes).
		SetDoNotParseResponse(true).
		Post(endpoint)
	if err != nil {
		return nil, llm.NewHTTPError("request failed", err)
	}

	// 4. 检查 HTTP 错误
	if resp.StatusCode() >= 400 {
		apiErr := llm.NewAPIError(resp.StatusCode(), resp.String())

		// 尝试提取请求 ID
		if requestID := resp.Header().Get("X-Request-ID"); requestID != "" {
			apiErr = apiErr.WithRequestID(requestID)
		}

		// 设置 Provider 类型
		apiErr = apiErr.WithProvider(c.config.ProviderName())

		_ = resp.RawBody().Close()
		return nil, apiErr
	}

	// 5. 启动 SSE 解析
	chunks := make(chan *llm.Event, 10)
	go c.sseParser.Parse(resp.RawBody(), chunks)

	return chunks, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// 辅助方法
// ═══════════════════════════════════════════════════════════════════════════

// getCompleteEndpoint 获取 Complete 端点
func (c *BaseClient) getCompleteEndpoint() string {
	if c.endpointBuilder != nil {
		return c.endpointBuilder.BuildCompleteEndpoint()
	}
	return "/chat/completions" // 默认端点
}

// getStreamEndpoint 获取 Stream 端点
func (c *BaseClient) getStreamEndpoint() string {
	if c.endpointBuilder != nil {
		return c.endpointBuilder.BuildStreamEndpoint()
	}
	return "/chat/completions" // 默认端点
}

// getModelFromConfig 从配置获取模型名称
func (c *BaseClient) getModelFromConfig() string {
	// 通过类型断言获取具体配置的模型字段
	if cfg, ok := c.config.(interface{ GetModel() string }); ok {
		return cfg.GetModel()
	}
	return ""
}

// ═══════════════════════════════════════════════════════════════════════════
// 辅助函数
// ═══════════════════════════════════════════════════════════════════════════

// GetDefaultTimeout 获取默认超时时间的辅助函数
//
// 如果 timeout 为 0，返回默认的 120 秒。
func GetDefaultTimeout(timeout time.Duration) time.Duration {
	if timeout == 0 {
		return 120 * time.Second
	}
	return timeout
}

// NewInvalidConfigError 创建无效配置错误
func NewInvalidConfigError(field string) error {
	return llm.NewConfigError(field+" is required", nil)
}

// NewMissingAPIKeyError 创建缺少 API Key 错误
func NewMissingAPIKeyError() error {
	return llm.NewConfigError("API key is required", nil)
}
