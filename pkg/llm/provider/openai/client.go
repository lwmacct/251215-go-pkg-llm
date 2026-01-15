package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm/core"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm/protocol/openai"
)

// ═══════════════════════════════════════════════════════════════════════════
// 配置和客户端
// ═══════════════════════════════════════════════════════════════════════════

// Config 客户端配置
type Config struct {
	// APIKey API 密钥（必需）
	APIKey string

	// BaseURL API 基础地址，默认 https://api.openai.com/v1
	BaseURL string

	// Model 默认模型名称
	Model string

	// Timeout 请求超时时间，默认 120 秒
	Timeout time.Duration

	// Headers 额外的请求头
	Headers map[string]string
}

// Client OpenAI 兼容的 LLM 客户端
//
// 实现 [llm.Provider] 接口，支持同步和流式完成。
//
// 架构设计：
//   - 嵌入 core.BaseClient 复用通用逻辑
//   - 保留 transformer 用于 buildRequest
//   - 协议差异由 protocol/openai 适配器封装
type Client struct {
	*core.BaseClient

	config      *Config
	transformer *core.Transformer
}

// New 创建新的 OpenAI 客户端
//
// 参数 config 必须包含 APIKey。如果 BaseURL 为空，默认使用 OpenAI 官方地址。
func New(config *Config) (*Client, error) {
	// 创建 BaseClient
	baseClient, err := core.NewBaseClient(
		config,
		openai.NewAdapter(),
		openai.NewEventHandler(),
	)
	if err != nil {
		return nil, err
	}

	// 创建 transformer 用于 buildRequest
	transformer := core.NewTransformer(openai.NewAdapter())

	return &Client{
		BaseClient:  baseClient,
		config:      config,
		transformer: transformer,
	}, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// Provider 接口实现
// ═══════════════════════════════════════════════════════════════════════════

// Complete 同步完成
//
// 实现 [llm.Provider] 接口。发送消息到 LLM 并等待完整响应。
func (c *Client) Complete(ctx context.Context, messages []llm.Message, opts *llm.Options) (*llm.Response, error) {
	return c.BaseClient.Complete(ctx, messages, opts, c)
}

// Stream 流式完成
//
// 实现 [llm.Provider] 接口。返回一个 channel，逐块接收 LLM 响应。
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
		baseURL = "https://api.openai.com/v1"
	}

	model := c.Model
	if model == "" {
		model = "gpt-4o"
	}

	timeout := c.Timeout
	if timeout == 0 {
		timeout = 120 * time.Second
	}

	return baseURL, model, timeout
}

// BuildHeaders 构建请求头
func (c *Config) BuildHeaders() map[string]string {
	headers := map[string]string{
		"Authorization": "Bearer " + c.APIKey,
		"Content-Type":  "application/json",
	}
	maps.Copy(headers, c.Headers)
	return headers
}

// ProviderName 返回 Provider 名称
func (c *Config) ProviderName() string {
	return "openai"
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
	if model == "" {
		model = "gpt-4o"
	}

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
		"model":    model,
		"messages": apiMessages,
		"stream":   stream,
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
	if opts.FrequencyPenalty != 0 {
		req["frequency_penalty"] = opts.FrequencyPenalty
	}
	if opts.PresencePenalty != 0 {
		req["presence_penalty"] = opts.PresencePenalty
	}
	if len(opts.StopSequences) > 0 {
		req["stop"] = opts.StopSequences
	}

	// 工具定义
	if len(opts.Tools) > 0 {
		tools := make([]map[string]any, 0, len(opts.Tools))
		for _, tool := range opts.Tools {
			description := tool.Description

			// OpenAI 不支持 input_examples，将其格式化到 description 中
			if len(tool.InputExamples) > 0 {
				description += "\n\nExamples:"
				var descriptionSb255 strings.Builder
				for i, ex := range tool.InputExamples {
					exJSON, _ := json.Marshal(ex) //nolint:errchkjson // best effort
					descriptionSb255.WriteString(fmt.Sprintf("\n%d. %s", i+1, string(exJSON)))
				}
				description += descriptionSb255.String()
			}

			tools = append(tools, map[string]any{
				"type": "function",
				"function": map[string]any{
					"name":        tool.Name,
					"description": description,
					"parameters":  tool.InputSchema,
				},
			})
		}
		req["tools"] = tools
	}

	// Reasoning 力度 (Reasoning 模型)
	if opts.Reasoning != "" {
		req["reasoning_effort"] = opts.Reasoning
	}

	// 结构化输出
	if opts.ResponseFormat != nil {
		switch opts.ResponseFormat.Type {
		case "json_schema":
			req["response_format"] = map[string]any{
				"type": "json_schema",
				"json_schema": map[string]any{
					"name":   opts.ResponseFormat.Name,
					"schema": opts.ResponseFormat.Schema,
				},
			}
		case "json_object":
			req["response_format"] = map[string]any{"type": "json_object"}
		}
	}

	return req
}
