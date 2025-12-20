package core_test

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm/core"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm/protocol/anthropic"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm/protocol/openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ═══════════════════════════════════════════════════════════════════════════
// Mock EventHandler - 用于隔离测试 SSEParser
// ═══════════════════════════════════════════════════════════════════════════

// mockEventHandler 用于测试 SSEParser 的通用逻辑
type mockEventHandler struct {
	// 调用记录
	calls []mockEventCall

	// 配置响应
	eventsToReturn []*llm.Event
	stopToReturn   bool
	stopOnData     string // 返回 true 的数据字符串
}

type mockEventCall struct {
	eventType string
	data      map[string]any
}

func newMockEventHandler() *mockEventHandler {
	return &mockEventHandler{}
}

func (m *mockEventHandler) WithEvents(events ...*llm.Event) *mockEventHandler {
	m.eventsToReturn = events
	return m
}

func (m *mockEventHandler) WithStop(stop bool) *mockEventHandler {
	m.stopToReturn = stop
	return m
}

func (m *mockEventHandler) WithStopOnData(data string) *mockEventHandler {
	m.stopOnData = data
	return m
}

func (m *mockEventHandler) HandleEvent(eventType string, data map[string]any) ([]*llm.Event, bool) {
	m.calls = append(m.calls, mockEventCall{eventType: eventType, data: data})
	return m.eventsToReturn, m.stopToReturn
}

func (m *mockEventHandler) ShouldStopOnData(data string) bool {
	return m.stopOnData != "" && data == m.stopOnData
}

// ═══════════════════════════════════════════════════════════════════════════
// SSEParser 单元测试 - 使用 Mock Handler
// ═══════════════════════════════════════════════════════════════════════════

func TestSSEParser_Parse_BasicDataLine(t *testing.T) {
	handler := newMockEventHandler().WithEvents(&llm.Event{
		Type:      llm.EventTypeText,
		TextDelta: "Hello",
	})
	parser := core.NewSSEParser(handler)

	sseData := `data: {"message": "test"}
`
	reader := io.NopCloser(strings.NewReader(sseData))
	events := make(chan *llm.Event, 10)

	go parser.Parse(reader, events)

	// 收集事件
	var collected []*llm.Event //nolint:prealloc // channel 收集数量未知
	for e := range events {
		collected = append(collected, e)
	}

	// 验证 handler 被调用
	require.Len(t, handler.calls, 1, "Handler should be called once")
	assert.Empty(t, handler.calls[0].eventType, "eventType should be empty for data-only lines")
	assert.Equal(t, "test", handler.calls[0].data["message"])

	// 验证事件被转发
	require.Len(t, collected, 1)
	assert.Equal(t, llm.EventTypeText, collected[0].Type)
	assert.Equal(t, "Hello", collected[0].TextDelta)
}

func TestSSEParser_Parse_EventTypeLine(t *testing.T) {
	handler := newMockEventHandler().WithEvents(&llm.Event{
		Type:      llm.EventTypeText,
		TextDelta: "World",
	})
	parser := core.NewSSEParser(handler)

	// Anthropic 风格：event: 行 + data: 行
	sseData := `event: content_block_delta
data: {"delta": {"type": "text_delta", "text": "World"}}
`
	reader := io.NopCloser(strings.NewReader(sseData))
	events := make(chan *llm.Event, 10)

	go parser.Parse(reader, events)

	// 消费事件（此测试只验证 eventType 传递）
	for range events {
	}

	// 验证 eventType 被正确传递
	require.Len(t, handler.calls, 1)
	assert.Equal(t, "content_block_delta", handler.calls[0].eventType)
}

func TestSSEParser_Parse_InvalidJSON(t *testing.T) {
	handler := newMockEventHandler()
	parser := core.NewSSEParser(handler)

	// 无效 JSON 应该被静默忽略
	sseData := `data: {invalid json}
data: {"valid": "json"}
`
	reader := io.NopCloser(strings.NewReader(sseData))
	events := make(chan *llm.Event, 10)

	go parser.Parse(reader, events)

	// 等待解析完成
	for range events {
	}

	// 只有有效 JSON 会触发 handler
	require.Len(t, handler.calls, 1, "Only valid JSON should trigger handler")
	assert.Equal(t, "json", handler.calls[0].data["valid"])
}

func TestSSEParser_Parse_EmptyStream(t *testing.T) {
	handler := newMockEventHandler()
	parser := core.NewSSEParser(handler)

	reader := io.NopCloser(strings.NewReader(""))
	events := make(chan *llm.Event, 10)

	go parser.Parse(reader, events)

	var collected []*llm.Event //nolint:prealloc // channel 收集数量未知
	for e := range events {
		collected = append(collected, e)
	}

	// 空流应该正常关闭，无事件
	assert.Empty(t, collected, "Empty stream should produce no events")
	assert.Empty(t, handler.calls, "Handler should not be called")
}

func TestSSEParser_Parse_StopOnData(t *testing.T) {
	// 模拟 OpenAI 的 [DONE] 终止信号
	handler := newMockEventHandler().WithStopOnData("[DONE]")
	parser := core.NewSSEParser(handler)

	sseData := `data: {"choices": [{"delta": {"content": "Hi"}}]}
data: [DONE]
data: {"choices": [{"delta": {"content": "This should not be processed"}}]}
`
	reader := io.NopCloser(strings.NewReader(sseData))
	events := make(chan *llm.Event, 10)

	go parser.Parse(reader, events)

	var collected []*llm.Event //nolint:prealloc // channel 收集数量未知
	for e := range events {
		collected = append(collected, e)
	}

	// [DONE] 后应该有一个 done 事件，后续数据不处理
	// handler 只被调用一次（[DONE] 之前的那条）
	require.Len(t, handler.calls, 1, "Handler should only be called once before [DONE]")

	// 应该有 done 事件
	require.NotEmpty(t, collected)
	lastEvent := collected[len(collected)-1]
	assert.Equal(t, llm.EventTypeDone, lastEvent.Type)
}

func TestSSEParser_Parse_HandlerStopSignal(t *testing.T) {
	// handler 返回 stop=true 时提前退出
	handler := newMockEventHandler().WithStop(true)
	parser := core.NewSSEParser(handler)

	sseData := `data: {"first": true}
data: {"second": true}
data: {"third": true}
`
	reader := io.NopCloser(strings.NewReader(sseData))
	events := make(chan *llm.Event, 10)

	go parser.Parse(reader, events)

	for range events {
	}

	// 第一次调用后应该停止
	require.Len(t, handler.calls, 1, "Should stop after first handler call")
	assert.Equal(t, true, handler.calls[0].data["first"])
}

func TestSSEParser_Parse_MultipleEventsFromHandler(t *testing.T) {
	// handler 返回多个事件
	handler := newMockEventHandler().WithEvents(
		&llm.Event{Type: llm.EventTypeText, TextDelta: "Part1"},
		&llm.Event{Type: llm.EventTypeText, TextDelta: "Part2"},
	)
	parser := core.NewSSEParser(handler)

	sseData := `data: {"content": "test"}
`
	reader := io.NopCloser(strings.NewReader(sseData))
	events := make(chan *llm.Event, 10)

	go parser.Parse(reader, events)

	var collected []*llm.Event //nolint:prealloc // channel 收集数量未知
	for e := range events {
		collected = append(collected, e)
	}

	// 应该收到两个事件
	require.Len(t, collected, 2)
	assert.Equal(t, "Part1", collected[0].TextDelta)
	assert.Equal(t, "Part2", collected[1].TextDelta)
}

func TestSSEParser_Parse_IgnoreNonDataLines(t *testing.T) {
	handler := newMockEventHandler()
	parser := core.NewSSEParser(handler)

	// 包含注释、空行、其他前缀的行应该被忽略
	sseData := `: this is a comment
id: 123

retry: 3000
data: {"valid": true}
`
	reader := io.NopCloser(strings.NewReader(sseData))
	events := make(chan *llm.Event, 10)

	go parser.Parse(reader, events)

	for range events {
	}

	// 只有 data: 行触发 handler
	require.Len(t, handler.calls, 1)
	assert.Equal(t, true, handler.calls[0].data["valid"])
}

// ═══════════════════════════════════════════════════════════════════════════
// 联合测试 - SSEParser + 真实 EventHandler
// ═══════════════════════════════════════════════════════════════════════════

func TestSSEParser_Integration_OpenAI_TextStream(t *testing.T) {
	handler := openai.NewEventHandler()
	parser := core.NewSSEParser(handler)

	// 模拟真实的 OpenAI SSE 流
	sseData := `data: {"choices":[{"delta":{"content":"Hello"}}]}

data: {"choices":[{"delta":{"content":" World"}}]}

data: {"choices":[{"delta":{},"finish_reason":"stop"}]}

data: [DONE]
`
	reader := io.NopCloser(strings.NewReader(sseData))
	events := make(chan *llm.Event, 10)

	go parser.Parse(reader, events)

	var collected []*llm.Event
	timeout := time.After(1 * time.Second)

loop:
	for {
		select {
		case e, ok := <-events:
			if !ok {
				break loop
			}
			collected = append(collected, e)
		case <-timeout:
			t.Fatal("Test timed out")
		}
	}

	// 验证事件序列
	require.GreaterOrEqual(t, len(collected), 3, "Expected at least 3 events")

	// 文本事件
	textEvents := filterEventsByType(collected, llm.EventTypeText)
	assert.Len(t, textEvents, 2, "Expected 2 text events")
	assert.Equal(t, "Hello", textEvents[0].TextDelta)
	assert.Equal(t, " World", textEvents[1].TextDelta)

	// 完成事件
	doneEvents := filterEventsByType(collected, llm.EventTypeDone)
	assert.NotEmpty(t, doneEvents, "Expected done event")
}

func TestSSEParser_Integration_OpenAI_ToolCall(t *testing.T) {
	handler := openai.NewEventHandler()
	parser := core.NewSSEParser(handler)

	sseData := `data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_abc","function":{"name":"get_weather","arguments":""}}]}}]}

data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"city\":"}}]}}]}

data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"Tokyo\"}"}}]}}]}

data: {"choices":[{"delta":{},"finish_reason":"tool_calls"}]}

data: [DONE]
`
	reader := io.NopCloser(strings.NewReader(sseData))
	events := make(chan *llm.Event, 10)

	go parser.Parse(reader, events)

	var collected []*llm.Event //nolint:prealloc // channel 收集数量未知
	for e := range events {
		collected = append(collected, e)
	}

	// 验证工具调用事件
	toolEvents := filterEventsByType(collected, llm.EventTypeToolCall)
	assert.GreaterOrEqual(t, len(toolEvents), 1, "Expected tool call events")

	// 第一个工具调用应该有 ID 和 Name
	firstTool := toolEvents[0]
	assert.Equal(t, "call_abc", firstTool.ToolCall.ID)
	assert.Equal(t, "get_weather", firstTool.ToolCall.Name)
}

func TestSSEParser_Integration_Anthropic_TextStream(t *testing.T) {
	handler := anthropic.NewEventHandler()
	parser := core.NewSSEParser(handler)

	// 模拟真实的 Anthropic SSE 流
	sseData := `event: message_start
data: {"type":"message_start","message":{"id":"msg_123"}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" World"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn"}}

event: message_stop
data: {"type":"message_stop"}
`
	reader := io.NopCloser(strings.NewReader(sseData))
	events := make(chan *llm.Event, 10)

	go parser.Parse(reader, events)

	var collected []*llm.Event //nolint:prealloc // channel 收集数量未知
	for e := range events {
		collected = append(collected, e)
	}

	// 验证文本事件
	textEvents := filterEventsByType(collected, llm.EventTypeText)
	assert.Len(t, textEvents, 2, "Expected 2 text events")
	assert.Equal(t, "Hello", textEvents[0].TextDelta)
	assert.Equal(t, " World", textEvents[1].TextDelta)

	// 验证完成事件
	doneEvents := filterEventsByType(collected, llm.EventTypeDone)
	assert.NotEmpty(t, doneEvents, "Expected done events")
}

func TestSSEParser_Integration_Anthropic_ToolCall(t *testing.T) {
	handler := anthropic.NewEventHandler()
	parser := core.NewSSEParser(handler)

	sseData := `event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_123","name":"get_weather"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"city\":"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"\"London\"}"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"tool_use"}}

event: message_stop
data: {"type":"message_stop"}
`
	reader := io.NopCloser(strings.NewReader(sseData))
	events := make(chan *llm.Event, 10)

	go parser.Parse(reader, events)

	var collected []*llm.Event //nolint:prealloc // channel 收集数量未知
	for e := range events {
		collected = append(collected, e)
	}

	// 验证工具调用事件
	toolEvents := filterEventsByType(collected, llm.EventTypeToolCall)
	require.NotEmpty(t, toolEvents, "Expected tool call events")

	// 第一个应该是工具开始（有 ID 和 Name）
	assert.Equal(t, "toolu_123", toolEvents[0].ToolCall.ID)
	assert.Equal(t, "get_weather", toolEvents[0].ToolCall.Name)
}

// ═══════════════════════════════════════════════════════════════════════════
// Helper Functions
// ═══════════════════════════════════════════════════════════════════════════

func filterEventsByType(events []*llm.Event, eventType llm.EventType) []*llm.Event {
	var result []*llm.Event
	for _, e := range events {
		if e.Type == eventType {
			result = append(result, e)
		}
	}
	return result
}
