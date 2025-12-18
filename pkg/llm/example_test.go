package llm_test

import (
	"context"
	"fmt"

	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm/provider/mock"
)

// Example_basic 展示 LLM 包的基本用法
func Example_basic() {
	// 创建 Provider
	provider := mock.New(mock.WithResponse("Hello! I can help you."))
	defer func() { _ = provider.Close() }()

	// 构建消息
	messages := []llm.Message{
		{Role: llm.RoleUser, Content: "Hello!"},
	}

	// 同步调用
	resp, err := provider.Complete(context.Background(), messages, nil)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println(resp.Message.Content)
	// Output: Hello! I can help you.
}

// Example_contentBlocks 展示使用内容块构建消息
func Example_contentBlocks() {
	// 使用 ContentBlocks 构建消息
	msg := llm.Message{
		Role: llm.RoleAssistant,
		ContentBlocks: []llm.ContentBlock{
			&llm.TextBlock{Text: "Here is the result"},
		},
	}

	fmt.Println("Role:", msg.Role)
	fmt.Println("Content:", msg.GetContent())
	// Output:
	// Role: assistant
	// Content: Here is the result
}

// Example_toolCalls 展示工具调用消息
func Example_toolCalls() {
	// 构建包含工具调用的消息
	msg := llm.Message{
		Role: llm.RoleAssistant,
		ContentBlocks: []llm.ContentBlock{
			&llm.ToolCall{
				ID:    "call_123",
				Name:  "get_weather",
				Input: map[string]any{"city": "Beijing"},
			},
		},
	}

	fmt.Println("Has tool calls:", msg.HasToolCalls())
	toolCalls := msg.GetToolCalls()
	if len(toolCalls) > 0 {
		fmt.Println("Tool name:", toolCalls[0].Name)
	}
	// Output:
	// Has tool calls: true
	// Tool name: get_weather
}

// Example_stream 展示流式事件处理
func Example_stream() {
	provider := mock.New(mock.WithResponse("Streaming response"))
	defer func() { _ = provider.Close() }()

	messages := []llm.Message{
		{Role: llm.RoleUser, Content: "Hello"},
	}

	stream, err := provider.Stream(context.Background(), messages, nil)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// 收集流式响应
	var text string
	for event := range stream {
		switch event.Type {
		case llm.EventTypeText:
			text += event.TextDelta
		case llm.EventTypeDone:
			// 流结束
		case llm.EventTypeError:
			fmt.Println("Error:", event.Error)
		}
	}

	fmt.Println(text)
	// Output: Streaming response
}

// Example_optionsReasoning 展示 Reasoning 统一参数配置
func Example_optionsReasoning() {
	// 使用统一参数（推荐）
	// 支持: OpenAI o1/o3, Claude, Gemini 2.5
	opts := &llm.Options{
		Reasoning: "high", // "low"/"medium"/"high"
		MaxTokens: 8192,
	}

	fmt.Println("Reasoning:", opts.Reasoning)
	fmt.Println("MaxTokens:", opts.MaxTokens)
	// Output:
	// Reasoning: high
	// MaxTokens: 8192
}

// Example_optionsThinkingBudget 展示精确控制 Thinking Budget
func Example_optionsThinkingBudget() {
	// 精确控制 Thinking Token 预算
	// 适用于需要精细调节推理深度的场景
	opts := &llm.Options{
		EnableReasoning: true,
		ReasoningBudget: 4096, // tokens
		MaxTokens:       16000,
	}

	fmt.Println("EnableReasoning:", opts.EnableReasoning)
	fmt.Println("ReasoningBudget:", opts.ReasoningBudget)
	// Output:
	// EnableReasoning: true
	// ReasoningBudget: 4096
}
