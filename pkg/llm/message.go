package llm

// ═══════════════════════════════════════════════════════════════════════════
// 角色定义
// ═══════════════════════════════════════════════════════════════════════════

// Role 消息角色
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// ═══════════════════════════════════════════════════════════════════════════
// 消息结构
// ═══════════════════════════════════════════════════════════════════════════

// Message 对话消息
type Message struct {
	Role          Role           `json:"role"`
	Content       string         `json:"content,omitempty"`
	ContentBlocks []ContentBlock `json:"content_blocks,omitempty"`
}

// GetContent 获取消息文本内容
func (m *Message) GetContent() string {
	if m.Content != "" {
		return m.Content
	}
	for _, block := range m.ContentBlocks {
		if tb, ok := block.(*TextBlock); ok {
			return tb.Text
		}
	}
	return ""
}

// GetToolCalls 获取消息中的工具调用
func (m *Message) GetToolCalls() []*ToolCall {
	var calls []*ToolCall
	for _, block := range m.ContentBlocks {
		if tu, ok := block.(*ToolCall); ok {
			calls = append(calls, tu)
		}
	}
	return calls
}

// GetToolResults 获取消息中的工具结果
func (m *Message) GetToolResults() []*ToolResultBlock {
	var results []*ToolResultBlock
	for _, block := range m.ContentBlocks {
		if tr, ok := block.(*ToolResultBlock); ok {
			results = append(results, tr)
		}
	}
	return results
}

// HasToolCalls 检查消息是否包含工具调用
func (m *Message) HasToolCalls() bool {
	for _, block := range m.ContentBlocks {
		if _, ok := block.(*ToolCall); ok {
			return true
		}
	}
	return false
}

// HasToolResults 检查消息是否包含工具结果
func (m *Message) HasToolResults() bool {
	for _, block := range m.ContentBlocks {
		if _, ok := block.(*ToolResultBlock); ok {
			return true
		}
	}
	return false
}

// ═══════════════════════════════════════════════════════════════════════════
// 内容块类型
// ═══════════════════════════════════════════════════════════════════════════

// ContentBlock 内容块接口
type ContentBlock interface {
	BlockType() string
}

// TextBlock 文本块
type TextBlock struct {
	Text string `json:"text"`
}

// BlockType 实现 ContentBlock 接口
func (b *TextBlock) BlockType() string { return "text" }

// ToolResultBlock 工具结果块
type ToolResultBlock struct {
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error,omitempty"`
}

// BlockType 实现 ContentBlock 接口
func (b *ToolResultBlock) BlockType() string { return "tool_result" }

// ═══════════════════════════════════════════════════════════════════════════
// 工具调用
// ═══════════════════════════════════════════════════════════════════════════

// ToolCall 工具调用（实现 ContentBlock 接口）
type ToolCall struct {
	ID    string         `json:"id"`
	Name  string         `json:"name"`
	Input map[string]any `json:"input"`
}

// BlockType 实现 ContentBlock 接口
func (tc *ToolCall) BlockType() string { return "tool_use" }

// ThinkingBlock 思考/推理内容块
//
// 用于存储模型的思考过程，支持：
//   - Gemini 2.5 系列的 thinking
//   - Anthropic Claude 的 extended thinking
//   - DeepSeek R1 的 reasoning
type ThinkingBlock struct {
	Thinking string `json:"thinking"`
}

// BlockType 实现 ContentBlock 接口
func (b *ThinkingBlock) BlockType() string { return "thinking" }
