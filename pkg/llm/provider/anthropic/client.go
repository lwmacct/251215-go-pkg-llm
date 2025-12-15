package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"

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
//   - 使用 core.Transformer 处理消息转换
//   - 使用 core.SSEParser 处理流式响应
//   - 协议差异由 protocol/anthropic 适配器封装
type Client struct {
	config      *Config
	resty       *resty.Client
	transformer *core.Transformer
	sseParser   *core.SSEParser
}

// New 创建新的 Anthropic 客户端
//
// 参数 config 必须包含 APIKey。
func New(config *Config) (*Client, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}
	if config.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	// 应用默认值
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}

	model := config.Model
	if model == "" {
		model = "claude-3-5-haiku-latest"
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 120 * time.Second
	}

	anthropicVersion := config.AnthropicVersion
	if anthropicVersion == "" {
		anthropicVersion = "2023-06-01"
	}

	// 构建请求头（Anthropic 使用 X-Api-Key）
	headers := map[string]string{
		"X-Api-Key":         config.APIKey,
		"anthropic-version": anthropicVersion,
		"Content-Type":      "application/json",
	}
	for k, v := range config.Headers {
		headers[k] = v
	}

	// 创建 resty 客户端
	r := resty.New()
	r.SetBaseURL(baseURL)
	r.SetTimeout(timeout)
	for k, v := range headers {
		r.SetHeader(k, v)
	}

	// 保存处理后的配置
	finalConfig := *config
	finalConfig.Model = model
	finalConfig.BaseURL = baseURL

	// 创建协议适配器和转换器
	adapter := anthropic.NewAdapter()
	eventHandler := anthropic.NewEventHandler()

	return &Client{
		config:      &finalConfig,
		resty:       r,
		transformer: core.NewTransformer(adapter),
		sseParser:   core.NewSSEParser(eventHandler),
	}, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// Provider 接口实现
// ═══════════════════════════════════════════════════════════════════════════

// Complete 同步完成
//
// 实现 [llm.Provider] 接口。发送消息到 Claude 并等待完整响应。
func (c *Client) Complete(ctx context.Context, messages []llm.Message, opts *llm.Options) (*llm.Response, error) {
	body := c.buildRequest(messages, opts, false)
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	var apiResp map[string]any
	resp, err := c.resty.R().
		SetContext(ctx).
		SetBody(bodyBytes).
		SetResult(&apiResp).
		Post("/messages")
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode() >= 400 {
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode(), resp.String())
	}

	// 使用 Transformer 解析响应
	msg, finishReason, usage := c.transformer.ParseAPIResponse(apiResp)

	// 提取实际使用的模型
	model := c.config.Model
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

// Stream 流式完成
//
// 实现 [llm.Provider] 接口。返回一个 channel，逐块接收 Claude 响应。
func (c *Client) Stream(ctx context.Context, messages []llm.Message, opts *llm.Options) (<-chan *llm.Event, error) {
	body := c.buildRequest(messages, opts, true)
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.resty.R().
		SetContext(ctx).
		SetBody(bodyBytes).
		SetDoNotParseResponse(true).
		Post("/messages")
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode() >= 400 {
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode(), resp.String())
	}

	chunks := make(chan *llm.Event, 10)
	// 使用 SSEParser 解析流式响应
	go c.sseParser.Parse(resp.RawBody(), chunks)
	return chunks, nil
}

// Close 关闭客户端
//
// 实现 [llm.Provider] 接口。当前实现为空操作。
func (c *Client) Close() error {
	return nil
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
