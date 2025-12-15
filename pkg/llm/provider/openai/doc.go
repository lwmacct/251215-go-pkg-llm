// Package openai 提供 OpenAI 兼容格式的 LLM Provider 实现
//
// 本包实现了 [agent.Provider] 接口，支持所有 OpenAI 兼容的 API 服务，
// 包括 OpenAI 官方 API、OpenRouter、Azure OpenAI、本地 Ollama 等。
//
// # 概述
//
// [Client] 是核心类型，封装了与 OpenAI 兼容 API 的通信逻辑：
//
//   - 支持同步完成 (Complete) 和流式完成 (Stream)
//   - 支持工具调用 (Tool Calling / Function Calling)
//   - 自动处理消息格式转换
//   - 内置 SSE 流式响应解析
//
// # 快速开始
//
//	client, err := openai.New(&openai.Config{
//	    APIKey:  "sk-xxx",
//	    BaseURL: "https://api.openai.com/v1",
//	    Model:   "gpt-4",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
//	// 同步完成
//	resp, err := client.Complete(ctx, messages, nil)
//
//	// 流式完成
//	stream, err := client.Stream(ctx, messages, nil)
//
// # 支持的服务
//
// 本包支持所有遵循 OpenAI Chat Completions API 格式的服务：
//
//   - OpenAI: https://api.openai.com/v1
//   - OpenRouter: https://openrouter.ai/api/v1
//   - Azure OpenAI: https://{resource}.openai.azure.com/openai/deployments/{deployment}
//   - Ollama: http://localhost:11434/v1
//   - 其他兼容服务
//
// # 消息格式
//
// 使用 [agent.Message] 作为输入，自动转换为 OpenAI API 格式：
//
//   - system: 系统提示词
//   - user: 用户消息
//   - assistant: 助手响应（可包含 tool_calls）
//   - tool: 工具执行结果
//
// # 流式响应
//
// 使用 [StreamParser] 解析 SSE 流式响应，自动聚合文本和工具调用：
//
//	stream, _ := client.Stream(ctx, messages, nil)
//	result := openai.ParseStream(stream)
//	fmt.Println(result.Message.GetContent())
//
// # 错误处理
//
// API 错误会包装为标准 error，包含 HTTP 状态码和响应内容。
// 建议使用重试中间件处理临时性错误。
//
// # 线程安全
//
// [Client] 是线程安全的，可以并发调用 Complete 和 Stream 方法。
// 但不应在多个 goroutine 中同时修改配置。
package openai
