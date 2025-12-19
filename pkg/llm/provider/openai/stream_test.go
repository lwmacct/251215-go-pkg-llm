package openai

import (
	"testing"

	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStreamParser(t *testing.T) {
	parser := NewStreamParser()
	require.NotNil(t, parser)
	assert.NotNil(t, parser.toolBufs)
	assert.Empty(t, parser.textBuf)
}

func TestStreamParser_Parse_TextOnly(t *testing.T) {
	chunks := make(chan *llm.Event, 5)
	go func() {
		defer close(chunks)
		chunks <- &llm.Event{Type: "text", TextDelta: "Hello"}
		chunks <- &llm.Event{Type: "text", TextDelta: ", "}
		chunks <- &llm.Event{Type: "text", TextDelta: "World!"}
		chunks <- &llm.Event{Type: "done", FinishReason: "stop"}
	}()

	result := NewStreamParser().Parse(chunks)

	assert.Equal(t, "Hello, World!", result.Message.GetContent())
	assert.Equal(t, "stop", result.FinishReason)
	assert.Equal(t, llm.RoleAssistant, result.Message.Role)
}

func TestStreamParser_Parse_ToolCalls(t *testing.T) {
	chunks := make(chan *llm.Event, 10)
	go func() {
		defer close(chunks)
		// First tool call
		chunks <- &llm.Event{
			Type: "tool_call",
			ToolCall: &llm.ToolCallDelta{
				Index: 0,
				ID:    "call_1",
				Name:  "search",
			},
		}
		chunks <- &llm.Event{
			Type: "tool_call",
			ToolCall: &llm.ToolCallDelta{
				Index:          0,
				ArgumentsDelta: `{"query":`,
			},
		}
		chunks <- &llm.Event{
			Type: "tool_call",
			ToolCall: &llm.ToolCallDelta{
				Index:          0,
				ArgumentsDelta: `"test"}`,
			},
		}

		// Second tool call
		chunks <- &llm.Event{
			Type: "tool_call",
			ToolCall: &llm.ToolCallDelta{
				Index: 1,
				ID:    "call_2",
				Name:  "calculate",
			},
		}
		chunks <- &llm.Event{
			Type: "tool_call",
			ToolCall: &llm.ToolCallDelta{
				Index:          1,
				ArgumentsDelta: `{"expr":"1+1"}`,
			},
		}

		chunks <- &llm.Event{Type: "done", FinishReason: "tool_calls"}
	}()

	result := NewStreamParser().Parse(chunks)

	assert.Equal(t, "tool_calls", result.FinishReason)
	require.Len(t, result.Message.ContentBlocks, 2)

	// First tool
	tool1, ok := result.Message.ContentBlocks[0].(*llm.ToolCall)
	require.True(t, ok)
	assert.Equal(t, "call_1", tool1.ID)
	assert.Equal(t, "search", tool1.Name)
	assert.Equal(t, "test", tool1.Input["query"])

	// Second tool
	tool2, ok := result.Message.ContentBlocks[1].(*llm.ToolCall)
	require.True(t, ok)
	assert.Equal(t, "call_2", tool2.ID)
	assert.Equal(t, "calculate", tool2.Name)
	assert.Equal(t, "1+1", tool2.Input["expr"])
}

func TestStreamParser_Parse_MixedContent(t *testing.T) {
	chunks := make(chan *llm.Event, 10)
	go func() {
		defer close(chunks)
		chunks <- &llm.Event{Type: "text", TextDelta: "Let me search for that."}
		chunks <- &llm.Event{
			Type: "tool_call",
			ToolCall: &llm.ToolCallDelta{
				Index: 0,
				ID:    "call_abc",
				Name:  "web_search",
			},
		}
		chunks <- &llm.Event{
			Type: "tool_call",
			ToolCall: &llm.ToolCallDelta{
				Index:          0,
				ArgumentsDelta: `{"q":"news"}`,
			},
		}
		chunks <- &llm.Event{Type: "done", FinishReason: "tool_calls"}
	}()

	result := NewStreamParser().Parse(chunks)

	assert.Equal(t, "tool_calls", result.FinishReason)
	require.Len(t, result.Message.ContentBlocks, 2)

	// Text block first
	textBlock, ok := result.Message.ContentBlocks[0].(*llm.TextBlock)
	require.True(t, ok)
	assert.Equal(t, "Let me search for that.", textBlock.Text)

	// Tool block second
	toolBlock, ok := result.Message.ContentBlocks[1].(*llm.ToolCall)
	require.True(t, ok)
	assert.Equal(t, "call_abc", toolBlock.ID)
}

func TestStreamParser_Parse_EmptyStream(t *testing.T) {
	chunks := make(chan *llm.Event, 1)
	close(chunks)

	result := NewStreamParser().Parse(chunks)

	assert.Empty(t, result.Message.ContentBlocks)
	assert.Empty(t, result.FinishReason)
}

func TestStreamParser_Feed(t *testing.T) {
	parser := NewStreamParser()

	parser.Feed(llm.Event{Type: "text", TextDelta: "Part 1"})
	assert.Equal(t, "Part 1", parser.CurrentText())

	parser.Feed(llm.Event{Type: "text", TextDelta: " Part 2"})
	assert.Equal(t, "Part 1 Part 2", parser.CurrentText())
}

func TestStreamParser_CurrentText(t *testing.T) {
	parser := NewStreamParser()
	assert.Empty(t, parser.CurrentText())

	parser.Feed(llm.Event{Type: "text", TextDelta: "Hello"})
	assert.Equal(t, "Hello", parser.CurrentText())
}

func TestStreamParser_Build(t *testing.T) {
	parser := NewStreamParser()

	parser.Feed(llm.Event{Type: "text", TextDelta: "Response"})
	parser.Feed(llm.Event{
		Type: "tool_call",
		ToolCall: &llm.ToolCallDelta{
			Index: 0,
			ID:    "call_1",
			Name:  "test",
		},
	})
	parser.Feed(llm.Event{
		Type: "tool_call",
		ToolCall: &llm.ToolCallDelta{
			Index:          0,
			ArgumentsDelta: `{}`,
		},
	})

	msg := parser.Build()

	assert.Equal(t, llm.RoleAssistant, msg.Role)
	require.Len(t, msg.ContentBlocks, 2)
}

func TestStreamParser_handleToolCall_NilDelta(t *testing.T) {
	parser := NewStreamParser()

	// Should not panic
	parser.handleToolCall(nil)

	assert.Empty(t, parser.toolBufs)
}

func TestStreamParser_handleToolCall_IncrementalUpdates(t *testing.T) {
	parser := NewStreamParser()

	// Initial tool call with ID and name
	parser.handleToolCall(&llm.ToolCallDelta{
		Index: 0,
		ID:    "call_1",
		Name:  "search",
	})

	// Incremental arguments
	parser.handleToolCall(&llm.ToolCallDelta{
		Index:          0,
		ArgumentsDelta: `{"key":`,
	})
	parser.handleToolCall(&llm.ToolCallDelta{
		Index:          0,
		ArgumentsDelta: `"value"}`,
	})

	assert.Len(t, parser.toolBufs, 1)
	buf := parser.toolBufs[0]
	assert.Equal(t, "call_1", buf.id)
	assert.Equal(t, "search", buf.name)
	assert.JSONEq(t, `{"key":"value"}`, buf.argsBuf)
}

func TestStreamParser_buildMessage_SkipsEmptyToolIDs(t *testing.T) {
	parser := NewStreamParser()

	// Tool with empty ID should be skipped
	parser.toolBufs[0] = &toolBuffer{
		id:      "",
		name:    "test",
		argsBuf: "{}",
	}
	parser.maxIndex = 0

	msg := parser.buildMessage()

	assert.Empty(t, msg.ContentBlocks)
}

func TestStreamParser_buildMessage_InvalidJSON(t *testing.T) {
	parser := NewStreamParser()

	parser.toolBufs[0] = &toolBuffer{
		id:      "call_1",
		name:    "test",
		argsBuf: "invalid json",
	}
	parser.maxIndex = 0

	msg := parser.buildMessage()

	require.Len(t, msg.ContentBlocks, 1)
	tool, ok := msg.ContentBlocks[0].(*llm.ToolCall)
	require.True(t, ok)
	assert.Nil(t, tool.Input) // Invalid JSON results in nil
}

func TestParseStream(t *testing.T) {
	chunks := make(chan *llm.Event, 3)
	go func() {
		defer close(chunks)
		chunks <- &llm.Event{Type: "text", TextDelta: "Test"}
		chunks <- &llm.Event{Type: "done", FinishReason: "stop"}
	}()

	result := ParseStream(chunks)

	assert.Equal(t, "Test", result.Message.GetContent())
	assert.Equal(t, "stop", result.FinishReason)
}

func TestStreamParser_MultipleToolsOutOfOrder(t *testing.T) {
	parser := NewStreamParser()

	// Tool at index 2 first
	parser.handleToolCall(&llm.ToolCallDelta{
		Index: 2,
		ID:    "call_3",
		Name:  "tool3",
	})

	// Tool at index 0
	parser.handleToolCall(&llm.ToolCallDelta{
		Index: 0,
		ID:    "call_1",
		Name:  "tool1",
	})

	// Tool at index 1
	parser.handleToolCall(&llm.ToolCallDelta{
		Index: 1,
		ID:    "call_2",
		Name:  "tool2",
	})

	// Add arguments
	parser.handleToolCall(&llm.ToolCallDelta{Index: 0, ArgumentsDelta: "{}"})
	parser.handleToolCall(&llm.ToolCallDelta{Index: 1, ArgumentsDelta: "{}"})
	parser.handleToolCall(&llm.ToolCallDelta{Index: 2, ArgumentsDelta: "{}"})

	msg := parser.buildMessage()

	// Should be in order 0, 1, 2
	require.Len(t, msg.ContentBlocks, 3)

	tool0, ok := msg.ContentBlocks[0].(*llm.ToolCall)
	require.True(t, ok)
	tool1, ok := msg.ContentBlocks[1].(*llm.ToolCall)
	require.True(t, ok)
	tool2, ok := msg.ContentBlocks[2].(*llm.ToolCall)
	require.True(t, ok)

	assert.Equal(t, "call_1", tool0.ID)
	assert.Equal(t, "call_2", tool1.ID)
	assert.Equal(t, "call_3", tool2.ID)
}

// ═══════════════════════════════════════════════════════════════════════════
// 推理内容测试 (DeepSeek R1, Kimi thinking 等)
// ═══════════════════════════════════════════════════════════════════════════

func TestStreamParser_Parse_ReasoningOnly(t *testing.T) {
	chunks := make(chan *llm.Event, 5)
	go func() {
		defer close(chunks)
		chunks <- &llm.Event{
			Type:      "reasoning",
			Reasoning: &llm.ReasoningDelta{ThoughtDelta: "Let me think..."},
		}
		chunks <- &llm.Event{
			Type:      "reasoning",
			Reasoning: &llm.ReasoningDelta{ThoughtDelta: " I need to analyze this."},
		}
		chunks <- &llm.Event{Type: "text", TextDelta: "Here is my answer."}
		chunks <- &llm.Event{Type: "done", FinishReason: "stop"}
	}()

	result := NewStreamParser().Parse(chunks)

	assert.Equal(t, "Here is my answer.", result.Message.GetContent())
	assert.Equal(t, "Let me think... I need to analyze this.", result.Reasoning)
	assert.Equal(t, "stop", result.FinishReason)
}

func TestStreamParser_Parse_ReasoningWithToolCalls(t *testing.T) {
	chunks := make(chan *llm.Event, 10)
	go func() {
		defer close(chunks)
		// Reasoning phase
		chunks <- &llm.Event{
			Type:      "reasoning",
			Reasoning: &llm.ReasoningDelta{ThoughtDelta: "I should search for this."},
		}
		// Text output
		chunks <- &llm.Event{Type: "text", TextDelta: "Let me search."}
		// Tool call
		chunks <- &llm.Event{
			Type: "tool_call",
			ToolCall: &llm.ToolCallDelta{
				Index: 0,
				ID:    "call_1",
				Name:  "search",
			},
		}
		chunks <- &llm.Event{
			Type: "tool_call",
			ToolCall: &llm.ToolCallDelta{
				Index:          0,
				ArgumentsDelta: `{"q":"test"}`,
			},
		}
		chunks <- &llm.Event{Type: "done", FinishReason: "tool_calls"}
	}()

	result := NewStreamParser().Parse(chunks)

	assert.Equal(t, "I should search for this.", result.Reasoning)
	assert.Equal(t, "tool_calls", result.FinishReason)
	require.Len(t, result.Message.ContentBlocks, 2) // Text + Tool

	textBlock, ok := result.Message.ContentBlocks[0].(*llm.TextBlock)
	require.True(t, ok)
	assert.Equal(t, "Let me search.", textBlock.Text)

	toolBlock, ok := result.Message.ContentBlocks[1].(*llm.ToolCall)
	require.True(t, ok)
	assert.Equal(t, "search", toolBlock.Name)
}

func TestStreamParser_Feed_Reasoning(t *testing.T) {
	parser := NewStreamParser()

	parser.Feed(llm.Event{
		Type:      "reasoning",
		Reasoning: &llm.ReasoningDelta{ThoughtDelta: "Step 1: "},
	})
	assert.Equal(t, "Step 1: ", parser.CurrentReasoning())

	parser.Feed(llm.Event{
		Type:      "reasoning",
		Reasoning: &llm.ReasoningDelta{ThoughtDelta: "analyze the problem"},
	})
	assert.Equal(t, "Step 1: analyze the problem", parser.CurrentReasoning())
}

func TestStreamParser_Feed_ReasoningNilDelta(t *testing.T) {
	parser := NewStreamParser()

	// Should not panic when Reasoning is nil
	parser.Feed(llm.Event{Type: "reasoning", Reasoning: nil})

	assert.Empty(t, parser.CurrentReasoning())
}

func TestStreamParser_CurrentReasoning(t *testing.T) {
	parser := NewStreamParser()
	assert.Empty(t, parser.CurrentReasoning())

	parser.Feed(llm.Event{
		Type:      "reasoning",
		Reasoning: &llm.ReasoningDelta{ThoughtDelta: "Thinking..."},
	})
	assert.Equal(t, "Thinking...", parser.CurrentReasoning())
}
