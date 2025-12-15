package localmock_test

import (
	"context"
	"fmt"

	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm/provider/localmock"
)

func Example_basic() {
	// 使用 Option 创建 localmock client
	client := localmock.New(localmock.WithResponse("Hello, I am a mock assistant."))
	defer func() { _ = client.Close() }()

	// 构造消息
	messages := []llm.Message{
		{Role: llm.RoleUser, Content: "Hello!"},
	}

	// 同步调用
	resp, err := client.Complete(context.Background(), messages, nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println(resp.Message.Content)
	// Output: Hello, I am a mock assistant.
}

func Example_clientStream() {
	client := localmock.New(localmock.WithResponse("Hi!"))
	defer func() { _ = client.Close() }()

	stream, err := client.Stream(context.Background(), nil, nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// 收集流式响应
	var text string
	for chunk := range stream {
		if chunk.Type == "text" {
			text += chunk.TextDelta
		}
	}

	fmt.Println(text)
	// Output: Hi!
}

func Example_withResponse() {
	client := localmock.New(localmock.WithResponse("Custom response"))
	defer func() { _ = client.Close() }()

	resp, _ := client.Complete(context.Background(), nil, nil)
	fmt.Println(resp.Message.Content)
	// Output: Custom response
}

func Example_clientUseScenario() {
	// 使用配置对象创建客户端
	cfg := &localmock.Config{
		DefaultResponse: "Default answer",
		Scenarios: []localmock.Scenario{
			{
				Name: "greeting",
				Turns: []localmock.Turn{
					{User: "hello", Assistant: "Hi there!"},
				},
			},
			{
				Name: "booking",
				Turns: []localmock.Turn{
					{User: "book", Assistant: "几位？"},
					{User: "3位", Assistant: "什么时间？"},
					{User: "7点", Assistant: "预订完成！"},
				},
			},
		},
	}

	client := localmock.New(localmock.WithConfig(cfg))
	defer func() { _ = client.Close() }()

	ctx := context.Background()

	// 指定使用 greeting 场景
	client.UseScenario("greeting")
	resp, _ := client.Complete(ctx, []llm.Message{
		{Role: llm.RoleUser, Content: "hello world"},
	}, nil)
	fmt.Println(resp.Message.Content)

	// 切换到 booking 场景（多轮对话）
	client.UseScenario("booking")
	resp1, _ := client.Complete(ctx, nil, nil)
	fmt.Println(resp1.Message.Content)

	resp2, _ := client.Complete(ctx, nil, nil)
	fmt.Println(resp2.Message.Content)

	// Output:
	// Hi there!
	// 几位？
	// 什么时间？
}

func Example_withConfigFile() {
	// 从 YAML 文件加载配置
	client := localmock.New(localmock.WithConfigFile("examples/unified.yaml"))
	defer func() { _ = client.Close() }()

	// 使用 greeting 场景
	client.UseScenario("greeting")

	resp, _ := client.Complete(context.Background(), []llm.Message{
		{Role: llm.RoleUser, Content: "你好"},
	}, nil)

	fmt.Println(resp.Message.Content)
	// Output: 你好！很高兴见到你！
}

func Example_loadExampleConfig() {
	// 加载内嵌的示例配置
	cfg, _ := localmock.LoadExampleConfig()
	client := localmock.New(localmock.WithConfig(cfg))
	defer func() { _ = client.Close() }()

	// 使用 weather 场景
	client.UseScenario("weather")

	resp, _ := client.Complete(context.Background(), []llm.Message{
		{Role: llm.RoleUser, Content: "天气怎么样"},
	}, nil)

	fmt.Println(resp.Message.Content)
	// Output: 今天天气晴朗，温度适宜。
}

func Example_withResponses() {
	// 设置响应队列
	client := localmock.New(localmock.WithResponses(
		"First response",
		"Second response",
		"Third response",
	))
	defer func() { _ = client.Close() }()

	ctx := context.Background()

	// 依次返回不同响应
	resp1, _ := client.Complete(ctx, nil, nil)
	fmt.Println(resp1.Message.Content)

	resp2, _ := client.Complete(ctx, nil, nil)
	fmt.Println(resp2.Message.Content)

	resp3, _ := client.Complete(ctx, nil, nil)
	fmt.Println(resp3.Message.Content)

	// 循环回到第一个
	resp4, _ := client.Complete(ctx, nil, nil)
	fmt.Println(resp4.Message.Content)

	// Output:
	// First response
	// Second response
	// Third response
	// First response
}

func Example_clientGetLastInput() {
	client := localmock.New(localmock.WithResponse("OK"))
	defer func() { _ = client.Close() }()

	ctx := context.Background()

	// 调用 Complete
	_, _ = client.Complete(ctx, []llm.Message{
		{Role: llm.RoleUser, Content: "Hello"},
	}, nil)

	// 获取最后一次输入
	fmt.Println(client.GetLastInput())
	// Output: Hello
}
