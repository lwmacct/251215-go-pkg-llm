package anthropic

import (
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm/core"
)

// ═══════════════════════════════════════════════════════════════════════════
// Anthropic 协议适配器
// ═══════════════════════════════════════════════════════════════════════════

// Adapter Anthropic 协议适配器
//
// 实现 core.ProtocolAdapter 接口，处理 Anthropic API 特有的协议格式。
//
// 关键协议差异：
//  1. 内容数组：使用 content 数组承载所有内容块
//  2. 工具参数：直接传递对象（无需序列化为 JSON 字符串）
//  3. 工具结果：内联在 content 数组中
//  4. 系统消息：独立的 system 参数（SystemSeparate）
//  5. Token 字段名：input_tokens, output_tokens（无 total_tokens）
type Adapter struct{}

// NewAdapter 创建 Anthropic 协议适配器
func NewAdapter() *Adapter {
	return &Adapter{}
}

// ═══════════════════════════════════════════════════════════════════════════
// ConvertToAPI - 消息转换为 Anthropic 格式
// ═══════════════════════════════════════════════════════════════════════════

// ConvertToAPI 实现 Anthropic 特有的消息转换逻辑
//
// Anthropic 协议要求：
//   - 使用 content 数组承载所有内容块
//   - 工具参数直接传递对象（无需序列化为 JSON 字符串）
//   - ToolResult 内联在 content 数组中（不展开为独立消息）
//   - content 数组必须非空
func (a *Adapter) ConvertToAPI(messages []llm.Message) []map[string]any {
	result := make([]map[string]any, 0, len(messages))

	for _, msg := range messages {
		// 跳过系统消息（由 Transformer 统一处理）
		if msg.Role == llm.RoleSystem {
			continue
		}

		m := map[string]any{"role": string(msg.Role)}

		// Anthropic 使用 content 数组
		var content []map[string]any

		// 优先处理 ContentBlocks
		if len(msg.ContentBlocks) > 0 {
			for _, block := range msg.ContentBlocks {
				switch b := block.(type) {
				case *llm.TextBlock:
					content = append(content, map[string]any{
						"type": "text",
						"text": b.Text,
					})

				case *llm.ToolCall:
					// ⚠️ 关键差异：参数直接是对象，不是 JSON 字符串
					content = append(content, map[string]any{
						"type":  "tool_use",
						"id":    b.ID,
						"name":  b.Name,
						"input": b.Input, // ← 直接对象
					})

				case *llm.ToolResultBlock:
					// ⚠️ 关键差异：ToolResult 内联在 content 数组中
					content = append(content, map[string]any{
						"type":        "tool_result",
						"tool_use_id": b.ToolUseID,
						"content":     b.Content,
					})
				}
			}
		} else if msg.Content != "" {
			// 降级到 Content 字段
			content = append(content, map[string]any{
				"type": "text",
				"text": msg.Content,
			})
		}

		// Anthropic 要求 content 必须非空
		if len(content) > 0 {
			m["content"] = content
			result = append(result, m)
		}
	}

	return result
}

// ═══════════════════════════════════════════════════════════════════════════
// ConvertFromAPI - 解析 Anthropic 响应
// ═══════════════════════════════════════════════════════════════════════════

// ConvertFromAPI 解析 Anthropic 响应为统一 Message
//
// Anthropic 响应格式：
//
//	{
//	  "content": [
//	    {"type": "text", "text": "..."},
//	    {"type": "tool_use", "id": "...", "name": "...", "input": {...}}
//	  ],
//	  "stop_reason": "end_turn"
//	}
func (a *Adapter) ConvertFromAPI(resp map[string]any) (llm.Message, string) {
	msg := llm.Message{Role: llm.RoleAssistant}

	// 提取 content 数组
	contentArray, _ := resp["content"].([]any)
	var blocks []llm.ContentBlock
	var textContent string

	for _, item := range contentArray {
		block, ok := item.(map[string]any)
		if !ok {
			continue
		}

		blockType, _ := block["type"].(string)

		switch blockType {
		case "text":
			text, _ := block["text"].(string)
			textContent = text
			blocks = append(blocks, &llm.TextBlock{Text: text})

		case "tool_use":
			id, _ := block["id"].(string)
			name, _ := block["name"].(string)
			// ⚠️ 关键差异：参数直接是对象（无需反序列化）
			input, _ := block["input"].(map[string]any)
			blocks = append(blocks, &llm.ToolCall{
				ID:    id,
				Name:  name,
				Input: input, // ← 直接对象
			})
		}
	}

	// 设置 ContentBlocks 或 Content
	if len(blocks) > 0 {
		msg.ContentBlocks = blocks
		// 如果只有单个文本块，同时设置 Content
		if len(blocks) == 1 && textContent != "" {
			msg.Content = textContent
		} else {
			msg.Content = "" // 清空，使用 ContentBlocks
		}
	}

	// 转换 stop_reason -> finish_reason
	stopReason, _ := resp["stop_reason"].(string)
	finishReason := convertStopReason(stopReason)

	return msg, finishReason
}

// ═══════════════════════════════════════════════════════════════════════════
// ConvertUsage - 解析 Token 使用量
// ═══════════════════════════════════════════════════════════════════════════

// ConvertUsage 解析 Anthropic 的 Token 使用量
//
// Anthropic 字段名：
//   - input_tokens, output_tokens（无 total_tokens）
//   - cache_read_input_tokens（Prompt Caching）
func (a *Adapter) ConvertUsage(resp map[string]any) *llm.TokenUsage {
	usage, ok := resp["usage"].(map[string]any)
	if !ok {
		return nil
	}

	result := &llm.TokenUsage{
		InputTokens:  core.GetInt64(usage["input_tokens"]),
		OutputTokens: core.GetInt64(usage["output_tokens"]),
	}

	// 手动计算 total_tokens（Anthropic 不返回此字段）
	result.TotalTokens = result.InputTokens + result.OutputTokens

	// Anthropic Prompt Caching
	if cacheRead := core.GetInt64(usage["cache_read_input_tokens"]); cacheRead > 0 {
		result.CachedTokens = cacheRead
	}

	return result
}

// ═══════════════════════════════════════════════════════════════════════════
// GetSystemMessageHandling - 系统消息策略
// ═══════════════════════════════════════════════════════════════════════════

// GetSystemMessageHandling 返回 Anthropic 的系统消息处理策略
//
// Anthropic 使用 SystemSeparate：系统消息作为独立的 "system" 参数传递。
func (a *Adapter) GetSystemMessageHandling() core.SystemMessageStrategy {
	return core.SystemSeparate
}

// ═══════════════════════════════════════════════════════════════════════════
// 辅助函数
// ═══════════════════════════════════════════════════════════════════════════

// convertStopReason 转换 Anthropic stop_reason 为标准 finish_reason
//
// Anthropic 映射：
//   - end_turn       -> stop
//   - max_tokens     -> length
//   - tool_use       -> tool_calls
//   - stop_sequence  -> stop
func convertStopReason(stopReason string) string {
	switch stopReason {
	case "end_turn":
		return "stop"
	case "max_tokens":
		return "length"
	case "tool_use":
		return "tool_calls"
	case "stop_sequence":
		return "stop"
	default:
		return stopReason
	}
}

// 确保 Adapter 实现了 ProtocolAdapter 接口
var _ core.ProtocolAdapter = (*Adapter)(nil)
