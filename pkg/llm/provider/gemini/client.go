package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"

	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm/core"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm/protocol/gemini"
)

// ═══════════════════════════════════════════════════════════════════════════
// 常量定义
// ═══════════════════════════════════════════════════════════════════════════

const (
	// DefaultBaseURL Gemini API 默认地址
	DefaultBaseURL = "https://generativelanguage.googleapis.com/v1beta"

	// DefaultModel 默认模型
	DefaultModel = "gemini-1.5-flash"

	// DefaultTimeout 默认超时时间
	DefaultTimeout = 120 * time.Second

	// DefaultMaxTokens 默认最大输出 tokens
	DefaultMaxTokens = 8192
)

// 模型常量
const (
	ModelGemini25Pro       = "gemini-2.5-pro"
	ModelGemini25Flash     = "gemini-2.5-flash"
	ModelGemini25FlashLite = "gemini-2.5-flash-lite"
	ModelGemini20Flash     = "gemini-2.0-flash"
	ModelGemini15Pro       = "gemini-1.5-pro"
	ModelGemini15Flash     = "gemini-1.5-flash"
)

// ═══════════════════════════════════════════════════════════════════════════
// 配置和客户端
// ═══════════════════════════════════════════════════════════════════════════

// Config 客户端配置
type Config struct {
	// APIKey Gemini API 密钥（Gemini API 后端必需）
	APIKey string

	// BaseURL API 基础地址，默认 https://generativelanguage.googleapis.com/v1beta
	BaseURL string

	// Model 默认模型名称
	Model string

	// Timeout 请求超时时间，默认 120 秒
	Timeout time.Duration

	// Headers 额外的请求头
	Headers map[string]string

	// Thinking 配置（Gemini 2.5 系列）
	EnableThinking bool  // 启用 thinking 模式
	ThinkingBudget int32 // thinking tokens 预算，0 表示动态

	// Vertex AI 配置
	VertexProject  string // GCP 项目 ID
	VertexLocation string // GCP 区域，默认 us-central1
	VertexCredFile string // 服务账户凭证文件路径
}

// Client Gemini LLM 客户端
//
// 实现 [llm.Provider] 接口，支持同步和流式完成。
//
// 架构设计：
//   - 使用 core.Transformer 处理消息转换
//   - 使用 core.SSEParser 处理流式响应
//   - 协议差异由 protocol/gemini 适配器封装
type Client struct {
	config      *Config
	resty       *resty.Client
	transformer *core.Transformer
	sseParser   *core.SSEParser

	// 内部状态
	useVertexAI bool
}

// New 创建新的 Gemini 客户端
//
// 参数 config 必须包含 APIKey（Gemini API）或 VertexProject（Vertex AI）。
func New(config *Config) (*Client, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	// 确定后端类型
	useVertexAI := config.VertexProject != ""

	// 验证配置
	if !useVertexAI && config.APIKey == "" {
		return nil, fmt.Errorf("API key is required for Gemini API backend")
	}

	// 应用默认值
	baseURL := config.BaseURL
	if baseURL == "" {
		if useVertexAI {
			location := config.VertexLocation
			if location == "" {
				location = "us-central1"
			}
			baseURL = fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1", location)
		} else {
			baseURL = DefaultBaseURL
		}
	}

	model := config.Model
	if model == "" {
		model = DefaultModel
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	// 构建请求头
	headers := map[string]string{
		"Content-Type": "application/json",
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

	// 创建协议适配器和转换器
	adapter := gemini.NewAdapter()
	eventHandler := gemini.NewEventHandler()

	return &Client{
		config:      &Config{APIKey: config.APIKey, BaseURL: baseURL, Model: model, Timeout: timeout, Headers: headers, EnableThinking: config.EnableThinking, ThinkingBudget: config.ThinkingBudget, VertexProject: config.VertexProject, VertexLocation: config.VertexLocation, VertexCredFile: config.VertexCredFile},
		resty:       r,
		transformer: core.NewTransformer(adapter),
		sseParser:   core.NewSSEParser(eventHandler),
		useVertexAI: useVertexAI,
	}, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// Provider 接口实现
// ═══════════════════════════════════════════════════════════════════════════

// Complete 同步完成
//
// 实现 [llm.Provider] 接口。发送消息到 Gemini 并等待完整响应。
func (c *Client) Complete(ctx context.Context, messages []llm.Message, opts *llm.Options) (*llm.Response, error) {
	body := c.buildRequest(messages, opts, false)
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	endpoint := c.buildEndpoint(false)

	var apiResp map[string]any
	resp, err := c.resty.R().
		SetContext(ctx).
		SetBody(bodyBytes).
		SetResult(&apiResp).
		Post(endpoint)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode() >= 400 {
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode(), resp.String())
	}

	// 使用 Transformer 解析响应
	msg, finishReason, usage := c.transformer.ParseAPIResponse(apiResp)

	return &llm.Response{
		Message:      msg,
		FinishReason: finishReason,
		Model:        c.config.Model,
		Usage:        usage,
	}, nil
}

// Stream 流式完成
//
// 实现 [llm.Provider] 接口。返回一个 channel，逐块接收 Gemini 响应。
func (c *Client) Stream(ctx context.Context, messages []llm.Message, opts *llm.Options) (<-chan *llm.Event, error) {
	body := c.buildRequest(messages, opts, true)
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	endpoint := c.buildEndpoint(true)

	resp, err := c.resty.R().
		SetContext(ctx).
		SetBody(bodyBytes).
		SetDoNotParseResponse(true).
		Post(endpoint)
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

// buildEndpoint 构建 API 端点
func (c *Client) buildEndpoint(stream bool) string {
	model := c.config.Model

	if c.useVertexAI {
		// Vertex AI 端点格式
		// /projects/{project}/locations/{location}/publishers/google/models/{model}:generateContent
		location := c.config.VertexLocation
		if location == "" {
			location = "us-central1"
		}
		action := "generateContent"
		if stream {
			action = "streamGenerateContent"
		}
		return fmt.Sprintf("/projects/%s/locations/%s/publishers/google/models/%s:%s",
			c.config.VertexProject, location, model, action)
	}

	// Gemini API 端点格式
	// /models/{model}:generateContent?key={apiKey}
	action := "generateContent"
	if stream {
		action = "streamGenerateContent"
	}
	return fmt.Sprintf("/models/%s:%s?key=%s", model, action, c.config.APIKey)
}

// buildRequest 构建 API 请求体
func (c *Client) buildRequest(messages []llm.Message, opts *llm.Options, stream bool) map[string]any {
	// 合并选项
	if opts == nil {
		opts = &llm.Options{}
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
		"contents": apiMessages,
	}

	// 系统指令（如果有）
	if systemPrompt != "" {
		req["systemInstruction"] = map[string]any{
			"parts": []map[string]any{
				{"text": systemPrompt},
			},
		}
	}

	// 生成配置
	genConfig := map[string]any{}

	if opts.MaxTokens > 0 {
		genConfig["maxOutputTokens"] = opts.MaxTokens
	} else {
		genConfig["maxOutputTokens"] = DefaultMaxTokens
	}

	if opts.Temperature >= 0 {
		genConfig["temperature"] = opts.Temperature
	}
	if opts.TopP > 0 {
		genConfig["topP"] = opts.TopP
	}
	if len(opts.StopSequences) > 0 {
		genConfig["stopSequences"] = opts.StopSequences
	}

	// 结构化输出
	if opts.ResponseFormat != nil && opts.ResponseFormat.Type == "json_schema" {
		genConfig["responseMimeType"] = "application/json"
		if opts.ResponseFormat.Schema != nil {
			genConfig["responseSchema"] = opts.ResponseFormat.Schema
		}
	}

	if len(genConfig) > 0 {
		req["generationConfig"] = genConfig
	}

	// Thinking 配置（Gemini 2.5 系列）
	if c.config.EnableThinking && supportsThinking(c.config.Model) {
		thinkingConfig := map[string]any{
			"includeThoughts": true,
		}
		if c.config.ThinkingBudget > 0 {
			thinkingConfig["thinkingBudget"] = c.config.ThinkingBudget
		}
		req["thinkingConfig"] = thinkingConfig
	}

	// 工具定义
	if len(opts.Tools) > 0 {
		functionDeclarations := make([]map[string]any, 0, len(opts.Tools))
		for _, tool := range opts.Tools {
			functionDeclarations = append(functionDeclarations, map[string]any{
				"name":        tool.Name,
				"description": tool.Description,
				"parameters":  convertToGeminiSchema(tool.InputSchema),
			})
		}
		req["tools"] = []map[string]any{
			{"functionDeclarations": functionDeclarations},
		}
	}

	return req
}

// ═══════════════════════════════════════════════════════════════════════════
// 辅助函数
// ═══════════════════════════════════════════════════════════════════════════

// supportsThinking 检查模型是否支持 thinking 能力
func supportsThinking(model string) bool {
	switch model {
	case ModelGemini25Pro, ModelGemini25Flash:
		return true
	default:
		return false
	}
}

// convertToGeminiSchema 将标准 JSON Schema 转换为 Gemini 格式
//
// Gemini 使用 genai.Schema 格式，与标准 JSON Schema 略有不同。
func convertToGeminiSchema(schema map[string]any) map[string]any {
	if schema == nil {
		return map[string]any{
			"type": "OBJECT",
		}
	}

	result := make(map[string]any)

	// 类型映射
	if t, ok := schema["type"].(string); ok {
		result["type"] = mapSchemaType(t)
	}

	// 描述
	if desc, ok := schema["description"].(string); ok {
		result["description"] = desc
	}

	// 属性
	if props, ok := schema["properties"].(map[string]any); ok {
		convertedProps := make(map[string]any)
		for k, v := range props {
			if propMap, ok := v.(map[string]any); ok {
				convertedProps[k] = convertToGeminiSchema(propMap)
			}
		}
		result["properties"] = convertedProps
	}

	// 必需字段
	if required, ok := schema["required"].([]any); ok {
		result["required"] = required
	}

	// 数组项
	if items, ok := schema["items"].(map[string]any); ok {
		result["items"] = convertToGeminiSchema(items)
	}

	// 枚举
	if enum, ok := schema["enum"].([]any); ok {
		result["enum"] = enum
	}

	return result
}

// mapSchemaType 将 JSON Schema 类型映射到 Gemini 类型
func mapSchemaType(t string) string {
	switch t {
	case "string":
		return "STRING"
	case "number":
		return "NUMBER"
	case "integer":
		return "INTEGER"
	case "boolean":
		return "BOOLEAN"
	case "array":
		return "ARRAY"
	case "object":
		return "OBJECT"
	default:
		return "STRING"
	}
}

// 确保 Client 实现了 Provider 接口
var _ llm.Provider = (*Client)(nil)
