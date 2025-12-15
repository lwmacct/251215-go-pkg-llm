package gemini

import (
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm/core"
)

// ═══════════════════════════════════════════════════════════════════════════
// Gemini 协议适配器
// ═══════════════════════════════════════════════════════════════════════════

// Adapter Gemini 协议适配器
//
// 实现 core.ProtocolAdapter 接口，处理 Gemini API 特有的协议格式。
//
// 关键协议差异：
//  1. 内容格式：Content{Role, Parts[]} 而非 message{role, content}
//  2. 角色映射：assistant → model
//  3. 工具参数：直接对象（类似 Anthropic）
//  4. 工具结果：作为 functionResponse Part
//  5. 系统消息：独立的 systemInstruction 字段
//  6. Token 字段名：promptTokenCount, candidatesTokenCount
type Adapter struct{}

// NewAdapter 创建 Gemini 协议适配器
func NewAdapter() *Adapter {
	return &Adapter{}
}

// ═══════════════════════════════════════════════════════════════════════════
// ConvertToAPI - 消息转换为 Gemini 格式
// ═══════════════════════════════════════════════════════════════════════════

// ConvertToAPI 实现 Gemini 特有的消息转换逻辑
//
// Gemini 协议要求：
//   - 消息格式为 Content{Role, Parts[]}
//   - 角色映射：user→user, assistant→model, tool→function
//   - ToolResult 作为 functionResponse Part
//   - 工具调用参数直接是对象（不序列化为 JSON 字符串）
func (a *Adapter) ConvertToAPI(messages []llm.Message) []map[string]any {
	result := make([]map[string]any, 0, len(messages))

	for _, msg := range messages {
		// 跳过系统消息（由 Transformer 统一处理，传递到 systemInstruction）
		if msg.Role == llm.RoleSystem {
			continue
		}

		// 构建 Gemini Content 结构
		content := map[string]any{
			"role": mapRole(msg.Role),
		}

		// 构建 Parts 数组
		parts := buildParts(msg)
		if len(parts) > 0 {
			content["parts"] = parts
		}

		result = append(result, content)
	}

	return result
}

// mapRole 将统一角色映射到 Gemini 角色
func mapRole(role llm.Role) string {
	switch role {
	case llm.RoleUser:
		return "user"
	case llm.RoleAssistant:
		return "model" // ⚠️ Gemini 使用 "model" 而非 "assistant"
	case llm.RoleTool:
		return "function" // 工具结果使用 function 角色
	default:
		return string(role)
	}
}

// buildParts 构建 Gemini Parts 数组
func buildParts(msg llm.Message) []map[string]any {
	var parts []map[string]any

	// 如果有 ContentBlocks，优先使用
	if len(msg.ContentBlocks) > 0 {
		for _, block := range msg.ContentBlocks {
			switch b := block.(type) {
			case *llm.TextBlock:
				parts = append(parts, map[string]any{
					"text": b.Text,
				})

			case *llm.ToolCall:
				// Gemini 使用 functionCall 格式
				parts = append(parts, map[string]any{
					"functionCall": map[string]any{
						"name": b.Name,
						"args": b.Input, // ⚠️ 直接对象，不序列化
					},
				})

			case *llm.ToolResultBlock:
				// Gemini 使用 functionResponse 格式
				parts = append(parts, map[string]any{
					"functionResponse": map[string]any{
						"name": b.ToolUseID, // 使用 ToolUseID 作为函数名
						"response": map[string]any{
							"content": b.Content,
							"error":   b.IsError,
						},
					},
				})

			case *llm.ThinkingBlock:
				// Gemini 的 thinking 内容标记为 thought: true
				parts = append(parts, map[string]any{
					"text":    b.Thinking,
					"thought": true,
				})
			}
		}
	}

	// 如果没有 ContentBlocks 但有 Content，添加文本 Part
	if len(parts) == 0 && msg.Content != "" {
		parts = append(parts, map[string]any{
			"text": msg.Content,
		})
	}

	return parts
}

// ═══════════════════════════════════════════════════════════════════════════
// ConvertFromAPI - 解析 Gemini 响应
// ═══════════════════════════════════════════════════════════════════════════

// ConvertFromAPI 解析 Gemini 响应为统一 Message
//
// Gemini 响应格式：
//
//	{
//	  "candidates": [{
//	    "content": {
//	      "role": "model",
//	      "parts": [
//	        {"text": "..."},
//	        {"functionCall": {"name": "...", "args": {...}}},
//	        {"text": "...", "thought": true}
//	      ]
//	    },
//	    "finishReason": "STOP"
//	  }],
//	  "usageMetadata": {...}
//	}
func (a *Adapter) ConvertFromAPI(resp map[string]any) (llm.Message, string) {
	msg := llm.Message{Role: llm.RoleAssistant}

	// 提取 candidates[0]
	candidates, _ := resp["candidates"].([]any)
	if len(candidates) == 0 {
		return msg, ""
	}

	candidate := candidates[0].(map[string]any)
	content, _ := candidate["content"].(map[string]any)
	finishReason := mapFinishReason(core.GetString(candidate["finishReason"]))

	// 解析 parts
	parts, _ := content["parts"].([]any)
	if len(parts) == 0 {
		return msg, finishReason
	}

	var blocks []llm.ContentBlock
	var textContent string

	for _, part := range parts {
		partMap := part.(map[string]any)

		// 检查是否为 thinking 内容
		isThought, _ := partMap["thought"].(bool)

		// 文本内容
		if text, ok := partMap["text"].(string); ok {
			if isThought {
				// Thinking 内容
				blocks = append(blocks, &llm.ThinkingBlock{
					Thinking: text,
				})
			} else {
				// 普通文本
				if len(blocks) == 0 {
					textContent = text
				}
				blocks = append(blocks, &llm.TextBlock{Text: text})
			}
		}

		// 函数调用
		if fc, ok := partMap["functionCall"].(map[string]any); ok {
			args, _ := fc["args"].(map[string]any)
			blocks = append(blocks, &llm.ToolCall{
				ID:    generateToolCallID(), // Gemini 不返回 ID，需要生成
				Name:  core.GetString(fc["name"]),
				Input: args,
			})
		}
	}

	// 设置消息内容
	if len(blocks) > 0 {
		msg.ContentBlocks = blocks
		// 如果只有一个文本块，也设置 Content 字段以保持兼容
		if len(blocks) == 1 {
			if tb, ok := blocks[0].(*llm.TextBlock); ok {
				msg.Content = tb.Text
			}
		}
	} else if textContent != "" {
		msg.Content = textContent
	}

	return msg, finishReason
}

// mapFinishReason 将 Gemini 完成原因映射到标准格式
func mapFinishReason(reason string) string {
	switch reason {
	case "STOP":
		return "stop"
	case "MAX_TOKENS":
		return "length"
	case "SAFETY":
		return "content_filter"
	case "RECITATION":
		return "content_filter"
	case "OTHER":
		return "stop"
	default:
		return reason
	}
}

// generateToolCallID 生成工具调用 ID
//
// Gemini API 不返回工具调用 ID，需要自行生成。
// 使用简单的计数器格式，因为 Gemini 的工具调用是顺序的。
var toolCallCounter int

func generateToolCallID() string {
	toolCallCounter++
	return "call_" + string(rune('0'+toolCallCounter%10))
}

// ═══════════════════════════════════════════════════════════════════════════
// ConvertUsage - 解析 Token 使用量
// ═══════════════════════════════════════════════════════════════════════════

// ConvertUsage 解析 Gemini 的 Token 使用量
//
// Gemini 字段名：
//   - promptTokenCount, candidatesTokenCount, totalTokenCount
//   - thoughtsTokenCount (thinking 模式)
//   - cachedContentTokenCount (prompt caching)
func (a *Adapter) ConvertUsage(resp map[string]any) *llm.TokenUsage {
	usage, ok := resp["usageMetadata"].(map[string]any)
	if !ok {
		return nil
	}

	result := &llm.TokenUsage{
		InputTokens:  core.GetInt64(usage["promptTokenCount"]),
		OutputTokens: core.GetInt64(usage["candidatesTokenCount"]),
		TotalTokens:  core.GetInt64(usage["totalTokenCount"]),
	}

	// Thinking tokens (Gemini 2.5)
	if thoughtsTokens := core.GetInt64(usage["thoughtsTokenCount"]); thoughtsTokens > 0 {
		result.ReasoningTokens = thoughtsTokens
	}

	// Prompt Caching tokens
	if cachedTokens := core.GetInt64(usage["cachedContentTokenCount"]); cachedTokens > 0 {
		result.CachedTokens = cachedTokens
	}

	return result
}

// ═══════════════════════════════════════════════════════════════════════════
// GetSystemMessageHandling - 系统消息策略
// ═══════════════════════════════════════════════════════════════════════════

// GetSystemMessageHandling 返回 Gemini 的系统消息处理策略
//
// Gemini 使用 SystemSeparate：系统消息作为独立的 systemInstruction 参数。
//
// 格式示例：
//
//	{
//	  "systemInstruction": {"parts": [{"text": "..."}]},
//	  "contents": [...]
//	}
func (a *Adapter) GetSystemMessageHandling() core.SystemMessageStrategy {
	return core.SystemSeparate
}

// 确保 Adapter 实现了 ProtocolAdapter 接口
var _ core.ProtocolAdapter = (*Adapter)(nil)
