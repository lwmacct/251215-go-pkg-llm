package openai

import (
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm/core"
)

// ═══════════════════════════════════════════════════════════════════════════
// OpenAI SSE 事件处理器
// ═══════════════════════════════════════════════════════════════════════════

// EventHandler OpenAI SSE 事件处理器
//
// 实现 core.EventHandler 接口，处理 OpenAI 流式响应的特有格式。
//
// OpenAI 流式格式：
//   - 无显式事件类型（eventType 总是空字符串）
//   - 数据结构：choices[0].delta
//   - 终止信号：data: [DONE]
//
// delta 结构：
//
//	{
//	  "choices": [{
//	    "delta": {
//	      "content": "...",                    // 文本增量
//	      "reasoning_content": "...",          // 推理内容 (DeepSeek R1)
//	      "tool_calls": [{"index": 0, ...}]   // 工具调用增量
//	    },
//	    "finish_reason": "stop"
//	  }]
//	}
type EventHandler struct{}

// NewEventHandler 创建 OpenAI 事件处理器
func NewEventHandler() *EventHandler {
	return &EventHandler{}
}

// ═══════════════════════════════════════════════════════════════════════════
// HandleEvent - 处理流式事件
// ═══════════════════════════════════════════════════════════════════════════

// HandleEvent 处理 OpenAI 流式事件
//
// OpenAI 特点：
//   - eventType 参数未使用（总是空字符串）
//   - 所有信息都在 data["choices"][0] 中
//   - delta 结构包含增量内容
func (h *EventHandler) HandleEvent(eventType string, data map[string]any) ([]*llm.Event, bool) {
	var result []*llm.Event

	// 提取 choices[0]
	choices, _ := data["choices"].([]any)
	if len(choices) == 0 {
		return result, false
	}

	choice := choices[0].(map[string]any)

	// 检查完成原因
	if fr, ok := choice["finish_reason"].(string); ok && fr != "" {
		result = append(result, &llm.Event{
			Type:         llm.EventTypeDone,
			FinishReason: fr,
		})
		return result, false
	}

	// 提取 delta
	delta, ok := choice["delta"].(map[string]any)
	if !ok {
		return result, false
	}

	// 处理文本内容
	if content, ok := delta["content"].(string); ok && content != "" {
		result = append(result, &llm.Event{
			Type:      llm.EventTypeText,
			TextDelta: content,
		})
	}

	// 处理推理内容 (DeepSeek R1, Kimi thinking)
	if reasoningContent, ok := delta["reasoning_content"].(string); ok && reasoningContent != "" {
		result = append(result, &llm.Event{
			Type: llm.EventTypeReasoning,
			Reasoning: &llm.ReasoningDelta{
				ThoughtDelta: reasoningContent,
			},
		})
	}

	// 处理工具调用
	if toolCalls, ok := delta["tool_calls"].([]any); ok {
		for _, tc := range toolCalls {
			tcMap := tc.(map[string]any)
			idxFloat, _ := tcMap["index"].(float64)

			d := &llm.ToolCallDelta{
				Index: int(idxFloat),
			}

			// 提取 ID
			if id, ok := tcMap["id"].(string); ok {
				d.ID = id
			}

			// 提取 function 信息
			if fn, ok := tcMap["function"].(map[string]any); ok {
				if name, ok := fn["name"].(string); ok {
					d.Name = name
				}
				if args, ok := fn["arguments"].(string); ok {
					d.ArgumentsDelta = args // JSON 增量字符串
				}
			}

			result = append(result, &llm.Event{
				Type:     llm.EventTypeToolCall,
				ToolCall: d,
			})
		}
	}

	return result, false
}

// ═══════════════════════════════════════════════════════════════════════════
// ShouldStopOnData - 检查终止信号
// ═══════════════════════════════════════════════════════════════════════════

// ShouldStopOnData 检查 OpenAI 的 [DONE] 终止信号
//
// OpenAI 使用特殊字符串 "[DONE]" 表示流结束：
//
//	data: [DONE]
func (h *EventHandler) ShouldStopOnData(data string) bool {
	return data == "[DONE]"
}

// 确保 EventHandler 实现了 core.EventHandler 接口
var _ core.EventHandler = (*EventHandler)(nil)
