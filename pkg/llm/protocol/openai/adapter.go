package openai

import (
	"encoding/json"

	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm/core"
)

// ═══════════════════════════════════════════════════════════════════════════
// OpenAI 协议适配器
// ═══════════════════════════════════════════════════════════════════════════

// Adapter OpenAI 协议适配器
//
// 实现 core.ProtocolAdapter 接口，处理 OpenAI API 特有的协议格式。
//
// 关键协议差异：
//  1. 工具参数：必须序列化为 JSON 字符串
//  2. 工具结果：必须展开为独立的 tool 角色消息
//  3. 系统消息：内联在消息数组中
//  4. Token 字段名：prompt_tokens, completion_tokens
type Adapter struct{}

// NewAdapter 创建 OpenAI 协议适配器
func NewAdapter() *Adapter {
	return &Adapter{}
}

// ═══════════════════════════════════════════════════════════════════════════
// ConvertToAPI - 消息转换为 OpenAI 格式
// ═══════════════════════════════════════════════════════════════════════════

// ConvertToAPI 实现 OpenAI 特有的消息转换逻辑
//
// OpenAI 协议要求：
//   - ToolResult 必须展开为独立的 tool 角色消息
//   - 工具调用参数必须序列化为 JSON 字符串
//   - 包含工具调用的消息必须有 content 字段（即使为空）
func (a *Adapter) ConvertToAPI(messages []llm.Message) []map[string]any {
	result := make([]map[string]any, 0, len(messages))

	for _, msg := range messages {
		// 跳过系统消息（由 Transformer 统一处理）
		if msg.Role == llm.RoleSystem {
			continue
		}

		// ⚠️ OpenAI 特殊处理：ToolResult 展开为独立消息
		if hasToolResults(msg.ContentBlocks) {
			for _, block := range msg.ContentBlocks {
				if tr, ok := block.(*llm.ToolResultBlock); ok {
					result = append(result, map[string]any{
						"role":         "tool",
						"tool_call_id": tr.ToolUseID,
						"content":      tr.Content,
					})
				}
			}
			// 处理完所有 ToolResult 后跳过这条消息
			continue
		}

		// 构建普通消息
		m := map[string]any{"role": string(msg.Role)}

		// 提取文本内容
		if content := extractTextContent(msg); content != "" {
			m["content"] = content
		}

		// 处理工具调用（仅 assistant 角色）
		if msg.Role == llm.RoleAssistant {
			if toolCalls := extractToolCalls(msg.ContentBlocks); len(toolCalls) > 0 {
				m["tool_calls"] = toolCalls
				// OpenAI 要求有 content 字段（即使为空）
				if m["content"] == nil {
					m["content"] = ""
				}
			}
		}

		result = append(result, m)
	}

	return result
}

// extractToolCalls 提取工具调用（OpenAI 格式）
//
// OpenAI 要求：
//   - 工具参数必须序列化为 JSON 字符串
//   - 结构：{"id": "...", "type": "function", "function": {"name": "...", "arguments": "..."}}
func extractToolCalls(blocks []llm.ContentBlock) []map[string]any {
	var result []map[string]any

	for _, block := range blocks {
		if tu, ok := block.(*llm.ToolCall); ok {
			// ⚠️ 关键差异：参数序列化为 JSON 字符串
			args, _ := json.Marshal(tu.Input)
			result = append(result, map[string]any{
				"id":   tu.ID,
				"type": "function",
				"function": map[string]any{
					"name":      tu.Name,
					"arguments": string(args), // ← JSON 字符串
				},
			})
		}
	}

	return result
}

// ═══════════════════════════════════════════════════════════════════════════
// ConvertFromAPI - 解析 OpenAI 响应
// ═══════════════════════════════════════════════════════════════════════════

// ConvertFromAPI 解析 OpenAI 响应为统一 Message
//
// OpenAI 响应格式：
//
//	{
//	  "choices": [{
//	    "message": {
//	      "content": "...",
//	      "tool_calls": [{"function": {"arguments": "{...}"}}]
//	    },
//	    "finish_reason": "stop"
//	  }]
//	}
func (a *Adapter) ConvertFromAPI(resp map[string]any) (llm.Message, string) {
	msg := llm.Message{Role: llm.RoleAssistant}

	// 提取 choices[0]
	choices, _ := resp["choices"].([]any)
	if len(choices) == 0 {
		return msg, ""
	}

	choice := choices[0].(map[string]any)
	messageData, _ := choice["message"].(map[string]any)
	finishReason, _ := choice["finish_reason"].(string)

	// 提取文本内容
	if content, ok := messageData["content"].(string); ok {
		msg.Content = content
	}

	// 提取工具调用
	if toolCalls, ok := messageData["tool_calls"].([]any); ok {
		var blocks []llm.ContentBlock

		// 如果有文本内容，添加 TextBlock
		if msg.Content != "" {
			blocks = append(blocks, &llm.TextBlock{Text: msg.Content})
		}

		// 添加 ToolUseBlock
		for _, tc := range toolCalls {
			tcMap := tc.(map[string]any)
			fn := tcMap["function"].(map[string]any)

			// ⚠️ 关键差异：反序列化 JSON 字符串
			var args map[string]any
			if argsStr, ok := fn["arguments"].(string); ok {
				_ = json.Unmarshal([]byte(argsStr), &args) // ← 从字符串解析
			}

			blocks = append(blocks, &llm.ToolCall{
				ID:    core.GetString(tcMap["id"]),
				Name:  core.GetString(fn["name"]),
				Input: args,
			})
		}

		// 设置 ContentBlocks
		msg.ContentBlocks = blocks
		msg.Content = "" // 清空，使用 ContentBlocks
	}

	return msg, finishReason
}

// ═══════════════════════════════════════════════════════════════════════════
// ConvertUsage - 解析 Token 使用量
// ═══════════════════════════════════════════════════════════════════════════

// ConvertUsage 解析 OpenAI 的 Token 使用量
//
// OpenAI 字段名：
//   - prompt_tokens, completion_tokens, total_tokens
//   - completion_tokens_details.reasoning_tokens
//   - prompt_tokens_details.cached_tokens
func (a *Adapter) ConvertUsage(resp map[string]any) *llm.TokenUsage {
	usage, ok := resp["usage"].(map[string]any)
	if !ok {
		return nil
	}

	result := &llm.TokenUsage{
		InputTokens:  core.GetInt64(usage["prompt_tokens"]),
		OutputTokens: core.GetInt64(usage["completion_tokens"]),
		TotalTokens:  core.GetInt64(usage["total_tokens"]),
	}

	// 推理 tokens (o1/o3, DeepSeek R1)
	if details, ok := usage["completion_tokens_details"].(map[string]any); ok {
		result.ReasoningTokens = core.GetInt64(details["reasoning_tokens"])
	}

	// Prompt Caching tokens
	if details, ok := usage["prompt_tokens_details"].(map[string]any); ok {
		result.CachedTokens = core.GetInt64(details["cached_tokens"])
	}

	return result
}

// ═══════════════════════════════════════════════════════════════════════════
// GetSystemMessageHandling - 系统消息策略
// ═══════════════════════════════════════════════════════════════════════════

// GetSystemMessageHandling 返回 OpenAI 的系统消息处理策略
//
// OpenAI 使用 SystemInline：系统消息作为第一条普通消息。
func (a *Adapter) GetSystemMessageHandling() core.SystemMessageStrategy {
	return core.SystemInline
}

// ═══════════════════════════════════════════════════════════════════════════
// 辅助函数
// ═══════════════════════════════════════════════════════════════════════════

// hasToolResults 检查消息是否包含 ToolResult
func hasToolResults(blocks []llm.ContentBlock) bool {
	for _, b := range blocks {
		if _, ok := b.(*llm.ToolResultBlock); ok {
			return true
		}
	}
	return false
}

// extractTextContent 提取文本内容（优先 ContentBlocks，次优 Content）
func extractTextContent(msg llm.Message) string {
	// 优先从 ContentBlocks 提取
	for _, b := range msg.ContentBlocks {
		if tb, ok := b.(*llm.TextBlock); ok {
			return tb.Text
		}
	}
	// 降级到 Content 字段
	return msg.Content
}

// 确保 Adapter 实现了 ProtocolAdapter 接口
var _ core.ProtocolAdapter = (*Adapter)(nil)
