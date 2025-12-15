package gemini

import (
	"encoding/json"

	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm/core"
)

// ═══════════════════════════════════════════════════════════════════════════
// Gemini SSE 事件处理器
// ═══════════════════════════════════════════════════════════════════════════

// EventHandler Gemini SSE 事件处理器
//
// 实现 core.EventHandler 接口，处理 Gemini 流式响应的特有格式。
//
// Gemini 流式格式：
//   - 使用 JSON 行格式（每行一个完整 JSON 对象）
//   - 数据结构：candidates[0].content.parts[]
//   - 无显式 [DONE] 终止信号，通过 finishReason 判断
//
// 流式响应块：
//
//	{
//	  "candidates": [{
//	    "content": {
//	      "role": "model",
//	      "parts": [
//	        {"text": "..."},
//	        {"text": "...", "thought": true},
//	        {"functionCall": {"name": "...", "args": {...}}}
//	      ]
//	    },
//	    "finishReason": "STOP"
//	  }],
//	  "usageMetadata": {...}
//	}
type EventHandler struct{}

// NewEventHandler 创建 Gemini 事件处理器
func NewEventHandler() *EventHandler {
	return &EventHandler{}
}

// ═══════════════════════════════════════════════════════════════════════════
// HandleEvent - 处理流式事件
// ═══════════════════════════════════════════════════════════════════════════

// HandleEvent 处理 Gemini 流式事件
//
// Gemini 特点：
//   - 响应结构为 candidates[0].content.parts[]
//   - parts 数组可能包含多个元素（文本、工具调用、thinking）
//   - thought: true 标记 thinking 内容
//   - functionCall 格式与 OpenAI 不同
func (h *EventHandler) HandleEvent(eventType string, data map[string]any) ([]*llm.Event, bool) {
	var result []*llm.Event

	// 提取 candidates[0]
	candidates, _ := data["candidates"].([]any)
	if len(candidates) == 0 {
		return result, false
	}

	candidate := candidates[0].(map[string]any)

	// 检查完成原因
	if fr, ok := candidate["finishReason"].(string); ok && fr != "" {
		// 映射 Gemini 完成原因到标准格式
		finishReason := mapFinishReasonForEvent(fr)
		result = append(result, &llm.Event{
			Type:         llm.EventTypeDone,
			FinishReason: finishReason,
		})
		return result, true // 停止处理
	}

	// 提取 content
	content, ok := candidate["content"].(map[string]any)
	if !ok {
		return result, false
	}

	// 解析 parts
	parts, ok := content["parts"].([]any)
	if !ok || len(parts) == 0 {
		return result, false
	}

	// 处理每个 part
	for i, part := range parts {
		partMap, ok := part.(map[string]any)
		if !ok {
			continue
		}

		// 检查是否为 thinking 内容
		isThought, _ := partMap["thought"].(bool)

		// 文本内容
		if text, ok := partMap["text"].(string); ok && text != "" {
			if isThought {
				// Thinking 内容
				result = append(result, &llm.Event{
					Type: llm.EventTypeThinking,
					Reasoning: &llm.ReasoningDelta{
						ThoughtDelta: text,
					},
				})
			} else {
				// 普通文本
				result = append(result, &llm.Event{
					Type:      llm.EventTypeText,
					TextDelta: text,
				})
			}
		}

		// 函数调用
		if fc, ok := partMap["functionCall"].(map[string]any); ok {
			name := core.GetString(fc["name"])
			args, _ := fc["args"].(map[string]any)

			// 序列化 args 为 JSON 字符串以符合 ToolCallDelta 接口
			var argsDelta string
			if args != nil {
				argsBytes, _ := json.Marshal(args)
				argsDelta = string(argsBytes)
			}

			result = append(result, &llm.Event{
				Type: llm.EventTypeToolCall,
				ToolCall: &llm.ToolCallDelta{
					Index:          i,
					ID:             generateStreamToolCallID(),
					Name:           name,
					ArgumentsDelta: argsDelta,
				},
			})
		}
	}

	return result, false
}

// mapFinishReasonForEvent 将 Gemini 完成原因映射到标准格式（用于事件处理）
func mapFinishReasonForEvent(reason string) string {
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

// 流式工具调用 ID 计数器
var streamToolCallCounter int

func generateStreamToolCallID() string {
	streamToolCallCounter++
	return "gemini_call_" + string(rune('a'+streamToolCallCounter%26))
}

// ═══════════════════════════════════════════════════════════════════════════
// ShouldStopOnData - 检查终止信号
// ═══════════════════════════════════════════════════════════════════════════

// ShouldStopOnData 检查 Gemini 的终止信号
//
// Gemini 不使用 "[DONE]" 终止信号，而是通过 finishReason 字段判断。
// 因此这个方法总是返回 false，依赖 HandleEvent 中的 finishReason 检查。
func (h *EventHandler) ShouldStopOnData(data string) bool {
	// Gemini 不使用显式终止信号
	// 注意：Gemini 的流式响应可能以空行结束
	return false
}

// 确保 EventHandler 实现了 core.EventHandler 接口
var _ core.EventHandler = (*EventHandler)(nil)
