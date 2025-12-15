package llm

import "context"

// ═══════════════════════════════════════════════════════════════════════════
// Provider 接口
// ═══════════════════════════════════════════════════════════════════════════

// Provider LLM 提供者接口
type Provider interface {
	// Complete 同步完成
	Complete(ctx context.Context, messages []Message, opts *Options) (*Response, error)

	// Stream 流式完成
	Stream(ctx context.Context, messages []Message, opts *Options) (<-chan *Event, error)

	// Close 关闭连接
	Close() error
}

// ═══════════════════════════════════════════════════════════════════════════
// Provider 选项与响应
// ═══════════════════════════════════════════════════════════════════════════

// Options Provider 选项
type Options struct {
	// 基础配置
	System      string  `json:"system,omitempty"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`

	// 采样参数
	TopP             float64  `json:"top_p,omitempty"`
	FrequencyPenalty float64  `json:"frequency_penalty,omitempty"`
	PresencePenalty  float64  `json:"presence_penalty,omitempty"`
	StopSequences    []string `json:"stop_sequences,omitempty"`

	// Reasoning 模型参数 (o1/o3, DeepSeek R1 等)
	Reasoning       string `json:"reasoning,omitempty"`        // 推理力度: "minimal", "low", "medium", "high"
	EnableReasoning bool   `json:"enable_reasoning,omitempty"` // 启用原生推理 tokens
	ReasoningBudget int    `json:"reasoning_budget,omitempty"` // 推理 token 预算 (Anthropic 最小 1024)

	// 结构化输出
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`

	// 工具
	Tools []ToolSchema `json:"tools,omitempty"`

	// 扩展
	Metadata map[string]any `json:"metadata,omitempty"`
}

// ResponseFormat 响应格式配置 (Structured Output)
type ResponseFormat struct {
	Type   string         `json:"type"`             // "json_schema", "json_object", "text"
	Name   string         `json:"name,omitempty"`   // Schema 名称
	Schema map[string]any `json:"schema,omitempty"` // JSON Schema 定义
}

// ToolSchema 工具 Schema
type ToolSchema struct {
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	InputSchema   map[string]any `json:"input_schema"`
	InputExamples []any          `json:"input_examples,omitempty"` // Anthropic input_examples (beta)
}

// Response Provider 响应
type Response struct {
	Message      Message        `json:"message"`
	FinishReason string         `json:"finish_reason"`
	Model        string         `json:"model,omitempty"` // 实际使用的模型
	Usage        *TokenUsage    `json:"usage,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

// TokenUsage Token 使用量
type TokenUsage struct {
	InputTokens     int64 `json:"input_tokens"`
	OutputTokens    int64 `json:"output_tokens"`
	TotalTokens     int64 `json:"total_tokens"`
	ReasoningTokens int64 `json:"reasoning_tokens,omitempty"` // 推理 tokens (DeepSeek R1, o1/o3 等)
	CachedTokens    int64 `json:"cached_tokens,omitempty"`    // Prompt Caching tokens
}
