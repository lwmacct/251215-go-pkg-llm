package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ═══════════════════════════════════════════════════════════════════════════
// GetContent 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestMessage_GetContent_FromContent(t *testing.T) {
	msg := Message{
		Role:    RoleUser,
		Content: "Hello, World!",
	}

	result := msg.GetContent()

	assert.Equal(t, "Hello, World!", result)
}

func TestMessage_GetContent_FromContentBlocks(t *testing.T) {
	msg := Message{
		Role: RoleAssistant,
		ContentBlocks: []ContentBlock{
			&TextBlock{Text: "First text block"},
			&TextBlock{Text: "Second text block"},
		},
	}

	result := msg.GetContent()

	// 应该返回第一个 TextBlock 的内容
	assert.Equal(t, "First text block", result)
}

func TestMessage_GetContent_ContentPriority(t *testing.T) {
	// 当同时存在 Content 和 ContentBlocks 时，优先返回 Content
	msg := Message{
		Role:    RoleAssistant,
		Content: "Direct content",
		ContentBlocks: []ContentBlock{
			&TextBlock{Text: "Block content"},
		},
	}

	result := msg.GetContent()

	assert.Equal(t, "Direct content", result)
}

func TestMessage_GetContent_Empty(t *testing.T) {
	msg := Message{
		Role: RoleAssistant,
	}

	result := msg.GetContent()

	assert.Empty(t, result)
}

func TestMessage_GetContent_MixedBlocks(t *testing.T) {
	// ContentBlocks 中没有 TextBlock 时
	msg := Message{
		Role: RoleAssistant,
		ContentBlocks: []ContentBlock{
			&ToolCall{ID: "call_1", Name: "func1"},
		},
	}

	result := msg.GetContent()

	assert.Empty(t, result, "Should return empty when no TextBlock")
}

// ═══════════════════════════════════════════════════════════════════════════
// GetToolCalls 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestMessage_GetToolCalls_HasToolCalls(t *testing.T) {
	msg := Message{
		Role: RoleAssistant,
		ContentBlocks: []ContentBlock{
			&TextBlock{Text: "Let me check"},
			&ToolCall{
				ID:    "call_1",
				Name:  "get_weather",
				Input: map[string]any{"city": "Tokyo"},
			},
			&ToolCall{
				ID:    "call_2",
				Name:  "get_time",
				Input: map[string]any{},
			},
		},
	}

	calls := msg.GetToolCalls()

	require.Len(t, calls, 2)
	assert.Equal(t, "call_1", calls[0].ID)
	assert.Equal(t, "get_weather", calls[0].Name)
	assert.Equal(t, "call_2", calls[1].ID)
	assert.Equal(t, "get_time", calls[1].Name)
}

func TestMessage_GetToolCalls_NoToolCalls(t *testing.T) {
	msg := Message{
		Role:    RoleAssistant,
		Content: "Just text",
	}

	calls := msg.GetToolCalls()

	assert.Empty(t, calls)
}

func TestMessage_GetToolCalls_MixedBlocks(t *testing.T) {
	msg := Message{
		Role: RoleAssistant,
		ContentBlocks: []ContentBlock{
			&TextBlock{Text: "Text"},
			&ToolCall{ID: "call_1", Name: "func1"},
			&ToolResultBlock{ToolUseID: "call_1", Content: "result"},
			&ThinkingBlock{Thinking: "Thinking..."},
		},
	}

	calls := msg.GetToolCalls()

	require.Len(t, calls, 1)
	assert.Equal(t, "call_1", calls[0].ID)
}

// ═══════════════════════════════════════════════════════════════════════════
// GetToolResults 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestMessage_GetToolResults(t *testing.T) {
	msg := Message{
		Role: RoleUser,
		ContentBlocks: []ContentBlock{
			&ToolResultBlock{
				ToolUseID: "call_1",
				Content:   "Result 1",
			},
			&ToolResultBlock{
				ToolUseID: "call_2",
				Content:   "Result 2",
				IsError:   true,
			},
		},
	}

	results := msg.GetToolResults()

	require.Len(t, results, 2)
	assert.Equal(t, "call_1", results[0].ToolUseID)
	assert.Equal(t, "Result 1", results[0].Content)
	assert.False(t, results[0].IsError)
	assert.Equal(t, "call_2", results[1].ToolUseID)
	assert.True(t, results[1].IsError)
}

func TestMessage_GetToolResults_NoResults(t *testing.T) {
	msg := Message{
		Role:    RoleUser,
		Content: "Just text",
	}

	results := msg.GetToolResults()

	assert.Empty(t, results)
}

// ═══════════════════════════════════════════════════════════════════════════
// HasToolCalls 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestMessage_HasToolCalls_True(t *testing.T) {
	msg := Message{
		Role: RoleAssistant,
		ContentBlocks: []ContentBlock{
			&ToolCall{ID: "call_1", Name: "func1"},
		},
	}

	assert.True(t, msg.HasToolCalls())
}

func TestMessage_HasToolCalls_False(t *testing.T) {
	msg := Message{
		Role:    RoleAssistant,
		Content: "No tool calls",
	}

	assert.False(t, msg.HasToolCalls())
}

func TestMessage_HasToolCalls_MixedBlocks(t *testing.T) {
	msg := Message{
		Role: RoleAssistant,
		ContentBlocks: []ContentBlock{
			&TextBlock{Text: "Text"},
			&ThinkingBlock{Thinking: "Thinking"},
			&ToolCall{ID: "call_1", Name: "func1"},
		},
	}

	assert.True(t, msg.HasToolCalls())
}

// ═══════════════════════════════════════════════════════════════════════════
// HasToolResults 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestMessage_HasToolResults_True(t *testing.T) {
	msg := Message{
		Role: RoleUser,
		ContentBlocks: []ContentBlock{
			&ToolResultBlock{ToolUseID: "call_1", Content: "result"},
		},
	}

	assert.True(t, msg.HasToolResults())
}

func TestMessage_HasToolResults_False(t *testing.T) {
	msg := Message{
		Role:    RoleUser,
		Content: "No tool results",
	}

	assert.False(t, msg.HasToolResults())
}

// ═══════════════════════════════════════════════════════════════════════════
// BlockType 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestTextBlock_BlockType(t *testing.T) {
	block := &TextBlock{Text: "test"}
	assert.Equal(t, "text", block.BlockType())
}

func TestToolResultBlock_BlockType(t *testing.T) {
	block := &ToolResultBlock{ToolUseID: "id", Content: "content"}
	assert.Equal(t, "tool_result", block.BlockType())
}

func TestToolCall_BlockType(t *testing.T) {
	block := &ToolCall{ID: "id", Name: "name"}
	assert.Equal(t, "tool_use", block.BlockType())
}

func TestThinkingBlock_BlockType(t *testing.T) {
	block := &ThinkingBlock{Thinking: "thinking"}
	assert.Equal(t, "thinking", block.BlockType())
}
