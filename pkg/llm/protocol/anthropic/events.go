package anthropic

import (
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm/core"
)

// ═══════════════════════════════════════════════════════════════════════════
// Anthropic SSE 事件处理器
// ═══════════════════════════════════════════════════════════════════════════

// EventHandler Anthropic SSE 事件处理器
//
// 实现 core.EventHandler 接口，处理 Anthropic 流式响应的特有格式。
//
// Anthropic 流式格式：
//   - 有显式事件类型（event: message_start, content_block_delta 等）
//   - 根据事件类型处理不同的数据结构
//   - 无终止信号字符串（使用 message_stop 事件）
//
// 事件类型：
//   - message_start:        消息开始
//   - content_block_start:  内容块开始（包含工具调用初始化）
//   - content_block_delta:  内容块增量（文本、工具参数、推理）
//   - content_block_stop:   内容块结束
//   - message_delta:        消息元数据增量（包含 stop_reason）
//   - message_stop:         消息结束
//   - ping:                 心跳
type EventHandler struct{}

// NewEventHandler 创建 Anthropic 事件处理器
func NewEventHandler() *EventHandler {
	return &EventHandler{}
}

// ═══════════════════════════════════════════════════════════════════════════
// HandleEvent - 处理流式事件
// ═══════════════════════════════════════════════════════════════════════════

// HandleEvent 处理 Anthropic 流式事件
//
// Anthropic 特点：
//   - eventType 驱动不同的处理逻辑
//   - content_block_delta 包含多种 delta 类型（text_delta, input_json_delta, thinking_delta）
//   - 使用 index 字段关联工具调用
func (h *EventHandler) HandleEvent(eventType string, data map[string]any) ([]*llm.Event, bool) {
	var result []*llm.Event

	switch eventType {
	case "content_block_start":
		// 工具调用开始
		if block, ok := data["content_block"].(map[string]any); ok {
			if blockType, _ := block["type"].(string); blockType == "tool_use" {
				result = append(result, &llm.Event{
					Type: "tool_call",
					ToolCall: &llm.ToolCallDelta{
						Index: int(core.GetFloat64(data["index"])),
						ID:    core.GetString(block["id"]),
						Name:  core.GetString(block["name"]),
					},
				})
			}
		}

	case "content_block_delta":
		// 内容增量
		delta, ok := data["delta"].(map[string]any)
		if !ok {
			return result, false
		}

		deltaType, _ := delta["type"].(string)

		switch deltaType {
		case "text_delta":
			// 文本增量
			text, _ := delta["text"].(string)
			if text != "" {
				result = append(result, &llm.Event{
					Type:      "text",
					TextDelta: text,
				})
			}

		case "input_json_delta":
			// 工具参数增量
			partialJSON, _ := delta["partial_json"].(string)
			if partialJSON != "" {
				result = append(result, &llm.Event{
					Type: "tool_call",
					ToolCall: &llm.ToolCallDelta{
						Index:          int(core.GetFloat64(data["index"])),
						ArgumentsDelta: partialJSON,
					},
				})
			}

		case "thinking_delta":
			// 推理内容（Claude 3.5+ 支持）
			thinking, _ := delta["thinking"].(string)
			if thinking != "" {
				result = append(result, &llm.Event{
					Type: "reasoning",
					Reasoning: &llm.ReasoningDelta{
						ThoughtDelta: thinking,
					},
				})
			}
		}

	case "message_delta":
		// 消息完成（包含 stop_reason）
		if delta, ok := data["delta"].(map[string]any); ok {
			if stopReason, ok := delta["stop_reason"].(string); ok && stopReason != "" {
				result = append(result, &llm.Event{
					Type:         "done",
					FinishReason: convertStopReason(stopReason),
				})
			}
		}

	case "message_stop":
		// 确保发送完成信号
		result = append(result, &llm.Event{
			Type:         "done",
			FinishReason: "stop",
		})

	case "message_start", "content_block_stop", "ping":
		// 这些事件不需要处理
		// message_start: 消息开始（无需输出）
		// content_block_stop: 内容块结束（无需输出）
		// ping: 心跳（无需输出）

	default:
		// 未知事件类型，静默忽略
	}

	return result, false
}

// ═══════════════════════════════════════════════════════════════════════════
// ShouldStopOnData - 检查终止信号
// ═══════════════════════════════════════════════════════════════════════════

// ShouldStopOnData 检查是否应在特定数据时停止
//
// Anthropic 不使用数据字符串终止信号（使用事件类型），所以总是返回 false。
func (h *EventHandler) ShouldStopOnData(data string) bool {
	return false
}

// 确保 EventHandler 实现了 core.EventHandler 接口
var _ core.EventHandler = (*EventHandler)(nil)
