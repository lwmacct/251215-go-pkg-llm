package openai

import (
	"encoding/json"

	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
)

// StreamResult 流式解析结果
type StreamResult struct {
	Message      llm.Message // 聚合后的完整消息
	FinishReason string      // 完成原因
	Reasoning    string      // 推理内容 (DeepSeek R1, Kimi thinking 等)
}

// StreamParser 流式响应解析器
//
// 将流式响应块聚合为完整消息。支持文本内容、推理内容和多个工具调用的并行聚合。
type StreamParser struct {
	textBuf      string
	reasoningBuf string // 推理内容缓冲区
	toolBufs     map[int]*toolBuffer
	maxIndex     int
}

type toolBuffer struct {
	id      string
	name    string
	argsBuf string
}

// NewStreamParser 创建新的流解析器
func NewStreamParser() *StreamParser {
	return &StreamParser{
		toolBufs: make(map[int]*toolBuffer),
	}
}

// Parse 解析流式响应并返回完整消息
//
// 从 channel 读取所有 Event，聚合文本内容和工具调用，
// 返回完整的 Message 和完成原因。
//
// 示例：
//
//	stream, _ := client.Stream(ctx, messages, nil)
//	result := openai.NewStreamParser().Parse(stream)
//	fmt.Println(result.Message.GetContent())
func (p *StreamParser) Parse(stream <-chan *llm.Event) StreamResult {
	var finishReason string

	for chunk := range stream {
		switch chunk.Type {
		case llm.EventTypeText:
			p.textBuf += chunk.TextDelta
		case llm.EventTypeReasoning:
			if chunk.Reasoning != nil {
				p.reasoningBuf += chunk.Reasoning.ThoughtDelta
			}
		case llm.EventTypeToolCall:
			p.handleToolCall(chunk.ToolCall)
		case llm.EventTypeDone:
			finishReason = chunk.FinishReason
		default:
			// 忽略其他事件类型
		}
	}

	return StreamResult{
		Message:      p.buildMessage(),
		FinishReason: finishReason,
		Reasoning:    p.reasoningBuf,
	}
}

// Feed 增量喂入单个响应块
//
// 用于需要实时处理每个块的场景，而非等待全部完成。
func (p *StreamParser) Feed(chunk llm.Event) {
	switch chunk.Type {
	case llm.EventTypeText:
		p.textBuf += chunk.TextDelta
	case llm.EventTypeReasoning:
		if chunk.Reasoning != nil {
			p.reasoningBuf += chunk.Reasoning.ThoughtDelta
		}
	case llm.EventTypeToolCall:
		p.handleToolCall(chunk.ToolCall)
	default:
		// 忽略其他事件类型
	}
}

// CurrentText 获取当前累积的文本内容
func (p *StreamParser) CurrentText() string {
	return p.textBuf
}

// CurrentReasoning 获取当前累积的推理内容
func (p *StreamParser) CurrentReasoning() string {
	return p.reasoningBuf
}

// Build 构建当前状态的消息
//
// 可以在流式传输过程中调用，获取当前累积的消息状态。
func (p *StreamParser) Build() llm.Message {
	return p.buildMessage()
}

func (p *StreamParser) handleToolCall(tc *llm.ToolCallDelta) {
	if tc == nil {
		return
	}

	buf, exists := p.toolBufs[tc.Index]
	if !exists {
		buf = &toolBuffer{}
		p.toolBufs[tc.Index] = buf
	}

	if tc.ID != "" {
		buf.id = tc.ID
	}
	if tc.Name != "" {
		buf.name = tc.Name
	}
	if tc.ArgumentsDelta != "" {
		buf.argsBuf += tc.ArgumentsDelta
	}

	if tc.Index > p.maxIndex {
		p.maxIndex = tc.Index
	}
}

func (p *StreamParser) buildMessage() llm.Message {
	var blocks []llm.ContentBlock

	if p.textBuf != "" {
		blocks = append(blocks, &llm.TextBlock{Text: p.textBuf})
	}

	// 按索引顺序添加工具调用
	for i := 0; i <= p.maxIndex; i++ {
		buf, ok := p.toolBufs[i]
		if !ok || buf.id == "" {
			continue
		}

		var args map[string]any
		_ = json.Unmarshal([]byte(buf.argsBuf), &args)

		blocks = append(blocks, &llm.ToolCall{
			ID:    buf.id,
			Name:  buf.name,
			Input: args,
		})
	}

	return llm.Message{
		Role:          llm.RoleAssistant,
		ContentBlocks: blocks,
	}
}

// ParseStream 便捷函数：解析流式响应
//
// 等价于 NewStreamParser().Parse(stream)
func ParseStream(stream <-chan *llm.Event) StreamResult {
	return NewStreamParser().Parse(stream)
}
