package core

import (
	"bufio"
	"encoding/json"
	"io"
	"strings"

	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
)

// ═══════════════════════════════════════════════════════════════════════════
// SSE 事件处理器接口
// ═══════════════════════════════════════════════════════════════════════════

// EventHandler SSE 事件处理器接口
//
// 每个 Provider 实现此接口来处理协议特有的 SSE 事件格式。
//
// 设计原则：
//   - 协议差异显式化：不同的事件格式通过接口方法体现
//   - 单一职责：只负责事件解析和转换
//   - 零假设：不假设事件格式，由实现者定义
//
// 协议差异示例：
//   - OpenAI: 无显式事件类型，总是 "data:" 前缀，[DONE] 终止
//   - Anthropic: 有显式事件类型（event:），多种终止方式
type EventHandler interface {
	// HandleEvent 处理单个 SSE 事件
	//
	// 职责：
	//   - 解析事件数据
	//   - 提取流式增量（文本、工具调用、推理内容）
	//   - 判断是否完成
	//
	// 参数：
	//   - eventType: 事件类型（OpenAI 为空，Anthropic 有值）
	//   - data: 已解析的事件数据 map
	//
	// 返回：
	//   - events: 转换后的 Event 列表（一个事件可能产生多个 event）
	//   - stop: 是否应该停止解析（用于处理完成信号）
	//
	// 实现要点：
	//   - OpenAI: eventType 总是空字符串，通过 data 结构判断类型
	//   - Anthropic: eventType 驱动不同的处理逻辑
	HandleEvent(eventType string, data map[string]any) (events []*llm.Event, stop bool)

	// ShouldStopOnData 检查是否应在特定数据时停止
	//
	// 职责：
	//   - 处理协议特有的终止信号
	//
	// 参数：
	//   - data: 原始数据行内容（未解析的字符串）
	//
	// 返回：
	//   - true: 应该停止解析（OpenAI [DONE]）
	//   - false: 继续解析（Anthropic 不使用此机制）
	//
	// 实现要点：
	//   - OpenAI: 检查 data == "[DONE]"
	//   - Anthropic: 总是返回 false（使用事件类型判断完成）
	ShouldStopOnData(data string) bool
}

// ═══════════════════════════════════════════════════════════════════════════
// SSE 解析器
// ═══════════════════════════════════════════════════════════════════════════

// SSEParser SSE (Server-Sent Events) 解析器
//
// 职责：
//   - 解析 SSE 流格式（event:/data: 行）
//   - 处理协议差异（OpenAI [DONE] vs Anthropic event types）
//   - 委托 EventHandler 处理具体事件
//
// SSE 格式规范：
//
//	event: event_type
//	data: {"key": "value"}
//
//	data: {"key": "value"}
//
// 设计原则：
//   - 通用扫描逻辑：统一的行读取和前缀匹配
//   - 协议差异委托：具体事件处理交给 EventHandler
//   - 容错处理：JSON 解析失败静默忽略
//
// 使用示例：
//
//	handler := openai.NewEventHandler()
//	parser := core.NewSSEParser(handler)
//
//	events := make(chan *llm.Event, 10)
//	go parser.Parse(resp.RawBody(), events)
//
//	for event := range events {
//	    fmt.Print(event.TextDelta)
//	}
type SSEParser struct {
	handler EventHandler
}

// NewSSEParser 创建 SSE 解析器
//
// 参数：
//   - handler: 协议特定的事件处理器
//
// 返回：
//   - SSE 解析器实例
func NewSSEParser(handler EventHandler) *SSEParser {
	return &SSEParser{handler: handler}
}

// Parse 解析 SSE 流
//
// 通用流程：
//  1. 逐行扫描流
//  2. 解析 "event:" 行（Anthropic）
//  3. 解析 "data:" 行
//  4. 检查终止信号（OpenAI [DONE]）
//  5. JSON 解析数据
//  6. 委托 handler 处理事件
//  7. 发送 events 到 channel
//
// 参数：
//   - body: HTTP 响应体（io.ReadCloser）
//   - events: Event 输出 channel
//
// 行为：
//   - 自动关闭 body
//   - 自动关闭 events channel
//   - JSON 解析失败静默忽略（继续处理下一行）
//   - 遇到终止信号或 handler 返回 stop 时退出
//
// 注意：
//   - 此方法应在 goroutine 中调用
//   - channel 缓冲区建议 10
//
// 示例：
//
//	events := make(chan *llm.Event, 10)
//	go parser.Parse(resp.RawBody(), events)
//
//	for event := range events {
//	    if event.Type == llm.EventTypeText {
//	        fmt.Print(event.TextDelta)
//	    }
//	}
func (p *SSEParser) Parse(body io.ReadCloser, events chan<- *llm.Event) {
	defer func() { _ = body.Close() }()
	defer close(events)

	scanner := bufio.NewScanner(body)
	var currentEvent string

	for scanner.Scan() {
		line := scanner.Text()

		// 解析事件类型（Anthropic 使用）
		// 格式: event: message_start
		if after, ok := strings.CutPrefix(line, "event: "); ok {
			currentEvent = after
			continue
		}

		// 解析数据行
		// 格式: data: {"key": "value"}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// 检查终止信号（OpenAI [DONE]）
		if p.handler.ShouldStopOnData(data) {
			events <- &llm.Event{Type: llm.EventTypeDone, FinishReason: "stop"}
			return
		}

		// 解析 JSON 数据
		var payload map[string]any
		if err := json.Unmarshal([]byte(data), &payload); err != nil {
			// JSON 解析失败，静默忽略
			continue
		}

		// 委托 handler 处理事件
		parsedEvents, shouldStop := p.handler.HandleEvent(currentEvent, payload)
		for _, event := range parsedEvents {
			events <- event
		}

		// 检查是否应该停止
		if shouldStop {
			return
		}
	}
}
