// Package anthropic 提供 Anthropic Claude API 原生实现
//
// 本包实现了 [llm.Provider] 接口，支持直接调用 Anthropic Claude API。
// 与 OpenAI 兼容包不同，本包使用 Anthropic 原生 API 格式。
//
// # 概述
//
// [Client] 是核心类型，提供以下功能：
//
//   - 同步完成 (Complete)
//   - 流式完成 (Stream)
//   - 工具调用 (Tool Use)
//   - Prompt Caching
//
// # 快速开始
//
//	client, err := anthropic.New(&anthropic.Config{
//	    APIKey: "sk-ant-...",
//	    Model:  "claude-3-5-haiku-latest",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
//	messages := []llm.Message{
//	    {Role: llm.RoleUser, Content: "Hello!"},
//	}
//
//	resp, err := client.Complete(ctx, messages, nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(resp.Message.Content)
//
// # 与 OpenAI 兼容包的区别
//
// 本包直接使用 Anthropic 原生 API，主要区别：
//
//   - 认证方式：使用 X-Api-Key 头部而非 Bearer Token
//   - 系统提示：作为独立参数而非消息
//   - 响应格式：content 数组而非 choices 数组
//   - 流式事件：使用 Anthropic 特有的事件类型
//
// # 线程安全
//
// [Client] 是线程安全的，可以并发调用 Complete 和 Stream 方法。
package anthropic
