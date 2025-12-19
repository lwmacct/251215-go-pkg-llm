package openai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"

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
//   - 使用 core.Transformer 处理消息转换
//   - 使用 core.SSEParser 处理流式响应
//   - 协议差异由 protocol/openai 适配器封装
type Client struct {
	config      *Config
	resty       *resty.Client
	transformer *core.Transformer
	sseParser   *core.SSEParser
}

// New 创建新的 OpenAI 客户端
//
// 参数 config 必须包含 APIKey。如果 BaseURL 为空，默认使用 OpenAI 官方地址。
func New(config *Config) (*Client, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}
	if config.APIKey == "" {
		return nil, errors.New("API key is required")
	}

	// 应用默认值
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 120 * time.Second
	}

	// 构建请求头
	headers := map[string]string{
		"Authorization": "Bearer " + config.APIKey,
		"Content-Type":  "application/json",
	}
	maps.Copy(headers, config.Headers)

	// 创建 resty 客户端
	r := resty.New()
	r.SetBaseURL(baseURL)
	r.SetTimeout(timeout)
	for k, v := range headers {
		r.SetHeader(k, v)
	}

	// 创建协议适配器和转换器
	adapter := openai.NewAdapter()
	eventHandler := openai.NewEventHandler()

	return &Client{
		config:      config,
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
// 实现 [llm.Provider] 接口。发送消息到 LLM 并等待完整响应。
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
		Post("/chat/completions")
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
// 实现 [llm.Provider] 接口。返回一个 channel，逐块接收 LLM 响应。
// 使用 [ParseStream] 或 [StreamParser] 聚合完整消息。
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
		Post("/chat/completions")
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
// 实现 [llm.Provider] 接口。当前实现为空操作，HTTP 客户端无需显式关闭。
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
