package llm

import "time"

// ═══════════════════════════════════════════════════════════════════════════
// 事件类型 - 统一的流式事件系统
// ═══════════════════════════════════════════════════════════════════════════

// EventType 事件类型
type EventType string

const (
	EventTypeText       EventType = "text"        // 文本增量
	EventTypeToolCall   EventType = "tool_call"   // 工具调用
	EventTypeToolResult EventType = "tool_result" // 工具执行结果 (Agent 层填充)
	EventTypeReasoning  EventType = "reasoning"   // 推理过程 (DeepSeek R1 等)
	EventTypeThinking   EventType = "thinking"    // 思考过程 (Anthropic extended thinking)
	EventTypeDone       EventType = "done"        // 完成
	EventTypeError      EventType = "error"       // 错误
)

// Event 统一事件结构
//
// 这是 LLM 层和 Agent 层共用的事件类型。
// 支持流式响应的所有事件类型，包括文本增量、工具调用、推理过程等。
//
// 使用示例：
//
//	for event := range provider.Stream(ctx, messages, opts) {
//	    switch event.Type {
//	    case llm.EventTypeText:
//	        fmt.Print(event.TextDelta)
//	    case llm.EventTypeToolCall:
//	        fmt.Printf("[Tool: %s]\n", event.ToolCall.Name)
//	    case llm.EventTypeDone:
//	        fmt.Printf("\nDone! Reason: %s\n", event.FinishReason)
//	    }
//	}
type Event struct {
	Type  EventType `json:"type"`
	Index int       `json:"index,omitempty"`

	// Text event - 文本增量
	TextDelta string `json:"text_delta,omitempty"`

	// ToolCall event - 工具调用增量
	ToolCall *ToolCallDelta `json:"tool_call,omitempty"`

	// ToolResult event - 工具执行结果 (Agent 层填充)
	ToolResult *ToolResult `json:"tool_result,omitempty"`

	// Reasoning/Thinking event - 推理过程增量
	Reasoning *ReasoningDelta `json:"reasoning,omitempty"`

	// Done event - 完成原因
	FinishReason string `json:"finish_reason,omitempty"`

	// Error event - 错误信息
	Error        error  `json:"-"`               // 错误对象 (不序列化)
	ErrorMessage string `json:"error,omitempty"` // 错误消息 (序列化用)

	// Metadata - 元数据
	Delta     any       `json:"delta,omitempty"`    // 通用增量数据
	Timestamp time.Time `json:"timestamp,omitzero"` // 时间戳
}

// Text 获取文本内容（兼容方法）
func (e *Event) Text() string {
	return e.TextDelta
}

// ═══════════════════════════════════════════════════════════════════════════
// 事件相关类型
// ═══════════════════════════════════════════════════════════════════════════

// ToolResult 工具执行结果
//
// Agent 执行工具后，通过 EventTypeToolResult 事件返回结果。
type ToolResult struct {
	ToolID  string `json:"tool_id"`
	Name    string `json:"name"`
	Content string `json:"content"`
	IsError bool   `json:"is_error,omitempty"`
}

// ToolCallDelta 工具调用增量
type ToolCallDelta struct {
	Index          int    `json:"index"`
	ID             string `json:"id,omitempty"`
	Name           string `json:"name,omitempty"`
	ArgumentsDelta string `json:"arguments_delta,omitempty"`
}

// ReasoningDelta 推理内容增量
type ReasoningDelta struct {
	ThoughtDelta string `json:"thought_delta,omitempty"`
}
