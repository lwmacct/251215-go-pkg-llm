package anthropic

import (
	"context"
	"maps"
	"time"

	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm/core"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm/protocol/anthropic"
)

// ═══════════════════════════════════════════════════════════════════════════
// 配置和客户端
// ═══════════════════════════════════════════════════════════════════════════

// Config 客户端配置
type Config struct {
	// APIKey API 密钥（必需）
	APIKey string

	// BaseURL API 基础地址，默认 https://api.anthropic.com/v1
	BaseURL string

	// Model 默认模型名称，默认 claude-3-5-haiku-latest
	Model string

	// Timeout 请求超时时间，默认 120 秒
	Timeout time.Duration

	// Headers 额外的请求头
	Headers map[string]string

	// AnthropicVersion API 版本，默认 2023-06-01
	AnthropicVersion string
}

// Client Anthropic Claude API 客户端
//
// 实现 [llm.Provider] 接口，支持同步和流式完成。
//
// 架构设计：
//   - 嵌入 core.BaseClient 复用通用逻辑
//   - 保留 transformer 用于 buildRequest
//   - 协议差异由 protocol/anthropic 适配器封装
type Client struct {
	*core.BaseClient

	config      *Config
	transformer *core.Transformer
}

// New 创建新的 Anthropic 客户端
//
// 参数 config 必须包含 APIKey。
func New(config *Config) (*Client, error) {
	// 创建 BaseClient
	baseClient, err := core.NewBaseClient(
		config,
		anthropic.NewAdapter(),
		anthropic.NewEventHandler(),
	)
	if err != nil {
		return nil, err
	}

	// 创建 transformer 用于 buildRequest
	transformer := core.NewTransformer(anthropic.NewAdapter())

	// 保存处理后的配置
	finalConfig := *config
	if finalConfig.Model == "" {
		finalConfig.Model = "claude-3-5-haiku-latest"
	}
	if finalConfig.BaseURL == "" {
		finalConfig.BaseURL = "https://api.anthropic.com/v1"
	}
	if finalConfig.Timeout == 0 {
		finalConfig.Timeout = 120 * time.Second
	}
	if finalConfig.AnthropicVersion == "" {
		finalConfig.AnthropicVersion = "2023-06-01"
	}

	client := &Client{
		BaseClient:  baseClient,
		config:      &finalConfig,
		transformer: transformer,
	}

	// 设置端点构建器（Anthropic 使用固定端点）
	baseClient.SetEndpointBuilder(&anthropicEndpointBuilder{})

	return client, nil
}

// anthropicEndpointBuilder Anthropic 端点构建器
type anthropicEndpointBuilder struct{}

func (b *anthropicEndpointBuilder) BuildCompleteEndpoint() string {
	return "/messages"
}

func (b *anthropicEndpointBuilder) BuildStreamEndpoint() string {
	return "/messages"
}

// ═══════════════════════════════════════════════════════════════════════════
// Provider 接口实现
// ═══════════════════════════════════════════════════════════════════════════

// Complete 同步完成
//
// 实现 [llm.Provider] 接口。发送消息到 Claude 并等待完整响应。
func (c *Client) Complete(ctx context.Context, messages []llm.Message, opts *llm.Options) (*llm.Response, error) {
	return c.BaseClient.Complete(ctx, messages, opts, c)
}

// Stream 流式完成
//
// 实现 [llm.Provider] 接口。返回一个 channel，逐块接收 Claude 响应。
func (c *Client) Stream(ctx context.Context, messages []llm.Message, opts *llm.Options) (<-chan *llm.Event, error) {
	return c.BaseClient.Stream(ctx, messages, opts, c)
}

// Close 关闭客户端
//
// 实现 [llm.Provider] 接口。当前实现为空操作。
func (c *Client) Close() error {
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════
// core.ProviderConfig 接口实现
// ═══════════════════════════════════════════════════════════════════════════

// Validate 验证配置
func (c *Config) Validate() error {
	if c == nil {
		return llm.NewConfigError("config is required", nil)
	}
	if c.APIKey == "" {
		return llm.NewConfigError("API key is required", nil)
	}
	return nil
}

// GetDefaults 获取默认值
func (c *Config) GetDefaults() (string, string, time.Duration) {
	baseURL := c.BaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}

	model := c.Model
	if model == "" {
		model = "claude-3-5-haiku-latest"
	}

	timeout := c.Timeout
	if timeout == 0 {
		timeout = 120 * time.Second
	}

	return baseURL, model, timeout
}

// BuildHeaders 构建请求头
// Anthropic 使用 X-Api-Key 而不是 Authorization
func (c *Config) BuildHeaders() map[string]string {
	version := c.AnthropicVersion
	if version == "" {
		version = "2023-06-01"
	}

	headers := map[string]string{
		"X-Api-Key":         c.APIKey,
		"anthropic-version": version,
		"Content-Type":      "application/json",
	}
	maps.Copy(headers, c.Headers)
	return headers
}

// ProviderName 返回 Provider 名称
func (c *Config) ProviderName() string {
	return "anthropic"
}

// GetModel 返回模型名称（辅助方法）
func (c *Config) GetModel() string {
	return c.Model
}

// ═══════════════════════════════════════════════════════════════════════════
// core.RequestBuilder 接口实现
// ═══════════════════════════════════════════════════════════════════════════

// BuildRequest 实现 core.RequestBuilder 接口
func (c *Client) BuildRequest(messages []llm.Message, opts *llm.Options, stream bool) (map[string]any, error) {
	return c.buildRequest(messages, opts, stream), nil
}

// ═══════════════════════════════════════════════════════════════════════════
// 请求构建
// ═══════════════════════════════════════════════════════════════════════════

// buildRequest 构建 API 请求体
func (c *Client) buildRequest(messages []llm.Message, opts *llm.Options, stream bool) map[string]any {
	// 合并选项
	if opts == nil {
		opts = &llm.Options{}
	}

	// 确定模型
	model := c.config.Model

	// 提取系统提示
	var systemPrompt string
	if opts.System != "" {
		systemPrompt = opts.System
	} else {
		for _, msg := range messages {
			if msg.Role == llm.RoleSystem {
				systemPrompt = msg.Content
				break
			}
		}
	}

	// 使用 Transformer 转换消息
	apiMessages := c.transformer.BuildAPIMessages(messages, systemPrompt)

	// 构建请求
	req := map[string]any{
		"model":      model,
		"messages":   apiMessages,
		"max_tokens": 8192, // Anthropic 要求必须提供
		"stream":     stream,
	}

	// Anthropic 使用独立的 system 参数
	if systemPrompt != "" {
		req["system"] = systemPrompt
	}

	// 应用选项
	if opts.MaxTokens > 0 {
		req["max_tokens"] = opts.MaxTokens
	}
	if opts.Temperature >= 0 {
		req["temperature"] = opts.Temperature
	}
	if opts.TopP > 0 {
		req["top_p"] = opts.TopP
	}
	if len(opts.StopSequences) > 0 {
		req["stop_sequences"] = opts.StopSequences
	}

	// 工具定义
	if len(opts.Tools) > 0 {
		tools := make([]map[string]any, 0, len(opts.Tools))
		hasExamples := false
		for _, tool := range opts.Tools {
			toolDef := map[string]any{
				"name":         tool.Name,
				"description":  tool.Description,
				"input_schema": tool.InputSchema,
			}
			// 添加 input_examples（如果有）
			if len(tool.InputExamples) > 0 {
				toolDef["input_examples"] = tool.InputExamples
				hasExamples = true
			}
			tools = append(tools, toolDef)
		}
		req["tools"] = tools

		// 如果有 examples，添加 beta header
		if hasExamples {
			req["betas"] = []string{"advanced-tool-use-2025-11-20"}
		}
	}

	// Thinking 模式 (Claude 3.5+ Extended Thinking)
	if opts.EnableReasoning {
		req["thinking"] = map[string]any{
			"type":   "enabled",
			"budget": opts.ReasoningBudget,
		}
	}

	return req
}
