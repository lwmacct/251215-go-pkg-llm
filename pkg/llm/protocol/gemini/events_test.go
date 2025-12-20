package gemini

import (
	"testing"

	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ═══════════════════════════════════════════════════════════════════════════
// HandleEvent 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestEventHandler_HandleEvent_TextDelta(t *testing.T) {
	handler := NewEventHandler()
	data := map[string]any{
		"candidates": []any{
			map[string]any{
				"content": map[string]any{
					"role": "model",
					"parts": []any{
						map[string]any{
							"text": "Hello, World!",
						},
					},
				},
			},
		},
	}

	events, stop := handler.HandleEvent("", data)

	assert.False(t, stop, "Should not stop on text delta")
	require.Len(t, events, 1)

	assert.Equal(t, llm.EventTypeText, events[0].Type)
	assert.Equal(t, "Hello, World!", events[0].TextDelta)
}

func TestEventHandler_HandleEvent_ThinkingDelta(t *testing.T) {
	handler := NewEventHandler()
	data := map[string]any{
		"candidates": []any{
			map[string]any{
				"content": map[string]any{
					"role": "model",
					"parts": []any{
						map[string]any{
							"text":    "Analyzing the problem...",
							"thought": true, // ⚠️ Gemini thinking 标记
						},
					},
				},
			},
		},
	}

	events, stop := handler.HandleEvent("", data)

	assert.False(t, stop)
	require.Len(t, events, 1)

	assert.Equal(t, llm.EventTypeThinking, events[0].Type)
	require.NotNil(t, events[0].Reasoning)
	assert.Equal(t, "Analyzing the problem...", events[0].Reasoning.ThoughtDelta)
}

func TestEventHandler_HandleEvent_FunctionCall(t *testing.T) {
	handler := NewEventHandler()
	data := map[string]any{
		"candidates": []any{
			map[string]any{
				"content": map[string]any{
					"role": "model",
					"parts": []any{
						map[string]any{
							"functionCall": map[string]any{
								"name": "get_weather",
								"args": map[string]any{
									"city": "Tokyo",
								},
							},
						},
					},
				},
			},
		},
	}

	events, stop := handler.HandleEvent("", data)

	assert.False(t, stop)
	require.Len(t, events, 1)

	assert.Equal(t, llm.EventTypeToolCall, events[0].Type)
	require.NotNil(t, events[0].ToolCall)
	assert.Equal(t, "get_weather", events[0].ToolCall.Name)
	// args 被序列化为 JSON 字符串
	assert.Contains(t, events[0].ToolCall.ArgumentsDelta, "Tokyo")
	// ID 是生成的
	assert.NotEmpty(t, events[0].ToolCall.ID)
}

func TestEventHandler_HandleEvent_FinishReason_Stop(t *testing.T) {
	handler := NewEventHandler()
	data := map[string]any{
		"candidates": []any{
			map[string]any{
				"finishReason": "STOP",
			},
		},
	}

	events, stop := handler.HandleEvent("", data)

	// ⚠️ 关键验证：finishReason 触发停止
	assert.True(t, stop, "Should stop on finishReason")
	require.Len(t, events, 1)

	assert.Equal(t, llm.EventTypeDone, events[0].Type)
	assert.Equal(t, "stop", events[0].FinishReason) // STOP -> stop
}

func TestEventHandler_HandleEvent_FinishReasonMapping(t *testing.T) {
	handler := NewEventHandler()

	testCases := []struct {
		geminiReason   string
		expectedReason string
	}{
		{"STOP", "stop"},
		{"MAX_TOKENS", "length"},
		{"SAFETY", "content_filter"},
		{"RECITATION", "content_filter"},
		{"OTHER", "stop"},
		{"UNKNOWN", "UNKNOWN"},
	}

	for _, tc := range testCases {
		data := map[string]any{
			"candidates": []any{
				map[string]any{
					"finishReason": tc.geminiReason,
				},
			},
		}

		events, _ := handler.HandleEvent("", data)
		require.Len(t, events, 1)
		assert.Equal(t, tc.expectedReason, events[0].FinishReason,
			"Gemini reason %q should map to %q", tc.geminiReason, tc.expectedReason)
	}
}

func TestEventHandler_HandleEvent_MultipleParts(t *testing.T) {
	handler := NewEventHandler()
	data := map[string]any{
		"candidates": []any{
			map[string]any{
				"content": map[string]any{
					"role": "model",
					"parts": []any{
						map[string]any{
							"text":    "Thinking...",
							"thought": true,
						},
						map[string]any{
							"text": "Here's my answer.",
						},
					},
				},
			},
		},
	}

	events, stop := handler.HandleEvent("", data)

	assert.False(t, stop)
	require.Len(t, events, 2, "Expected 2 events from 2 parts")

	// 第一个是 thinking
	assert.Equal(t, llm.EventTypeThinking, events[0].Type)
	assert.Equal(t, "Thinking...", events[0].Reasoning.ThoughtDelta)

	// 第二个是文本
	assert.Equal(t, llm.EventTypeText, events[1].Type)
	assert.Equal(t, "Here's my answer.", events[1].TextDelta)
}

func TestEventHandler_HandleEvent_EmptyCandidates(t *testing.T) {
	handler := NewEventHandler()
	data := map[string]any{
		"candidates": []any{},
	}

	events, stop := handler.HandleEvent("", data)

	assert.False(t, stop)
	assert.Empty(t, events, "Empty candidates should produce no events")
}

func TestEventHandler_HandleEvent_NoParts(t *testing.T) {
	handler := NewEventHandler()
	data := map[string]any{
		"candidates": []any{
			map[string]any{
				"content": map[string]any{
					"role": "model",
					// 没有 parts
				},
			},
		},
	}

	events, stop := handler.HandleEvent("", data)

	assert.False(t, stop)
	assert.Empty(t, events)
}

func TestEventHandler_HandleEvent_EmptyText(t *testing.T) {
	handler := NewEventHandler()
	data := map[string]any{
		"candidates": []any{
			map[string]any{
				"content": map[string]any{
					"role": "model",
					"parts": []any{
						map[string]any{
							"text": "", // 空文本
						},
					},
				},
			},
		},
	}

	events, stop := handler.HandleEvent("", data)

	assert.False(t, stop)
	assert.Empty(t, events, "Empty text should not produce event")
}

// ═══════════════════════════════════════════════════════════════════════════
// ShouldStopOnData 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestEventHandler_ShouldStopOnData(t *testing.T) {
	handler := NewEventHandler()

	// Gemini 不使用显式终止信号（如 [DONE]）
	// 依赖 finishReason 字段判断完成
	assert.False(t, handler.ShouldStopOnData("[DONE]"), "Gemini should not use [DONE]")
	assert.False(t, handler.ShouldStopOnData(""), "Empty string should not stop")
	assert.False(t, handler.ShouldStopOnData("any data"), "Should not stop on any data")
}

// ═══════════════════════════════════════════════════════════════════════════
// 接口实现验证
// ═══════════════════════════════════════════════════════════════════════════

func TestEventHandler_ImplementsEventHandler(t *testing.T) {
	var _ core.EventHandler = (*EventHandler)(nil)
}
