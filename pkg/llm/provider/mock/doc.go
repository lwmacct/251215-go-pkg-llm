// Package mock 提供本地 Mock LLM Provider 实现
//
// 本包实现了 [ts_provider.Provider] 接口，用于测试和开发场景，
// 无需真实的 LLM API 即可验证业务逻辑。
//
// # 概述
//
// [Client] 是核心类型，提供可预测的响应行为：
//
//   - 支持通过场景名称直接指定响应（推荐方式）
//   - 支持多轮对话场景
//   - 支持工具调用模拟
//   - 支持模板语法（环境变量注入）
//   - 记录所有调用详情，便于测试验证
//
// # 快速开始
//
//	// 创建 client（无参数时使用默认配置 testdata/unified.yaml）
//	client := mock.New()
//	defer client.Close()
//
//	// 指定使用某个场景
//	client.UseScenario("greeting")
//
//	// 调用
//	resp, err := client.Complete(ctx, messages, nil)
//
// # 场景指定模式（推荐）
//
// 通过 [Client.UseScenario] 指定使用哪个场景：
//
//	cfg := &mock.Config{
//	    Scenarios: []mock.Scenario{
//	        {
//	            Name: "booking",
//	            Turns: []mock.Turn{
//	                {User: "订餐", Assistant: "几位？"},
//	                {User: "3位", Assistant: "什么时间？"},
//	                {User: "7点", Assistant: "预订完成！"},
//	            },
//	        },
//	    },
//	}
//
//	client := mock.New(mock.WithConfig(cfg))
//	client.UseScenario("booking")
//
//	// 每次调用自动推进到下一轮
//	resp1, _ := client.Complete(ctx, nil, nil) // "几位？"
//	resp2, _ := client.Complete(ctx, nil, nil) // "什么时间？"
//	resp3, _ := client.Complete(ctx, nil, nil) // "预订完成！"
//
// # 工具调用模拟
//
// 场景支持模拟工具调用：
//
//	Scenario{
//	    Name: "weather",
//	    Turns: []Turn{
//	        {
//	            User:      "查天气",
//	            Assistant: "查询中...",
//	            Tools: []ToolCall{
//	                {
//	                    Name: "get_weather",
//	                    Input: map[string]any{
//	                        "city": "{{.CITY | default `Tokyo`}}",
//	                    },
//	                },
//	            },
//	        },
//	    },
//	}
//
// # 模板语法
//
// 响应文本和工具参数支持 Go 模板语法：
//
//   - {{.VAR}}: 直接访问环境变量
//   - {{.VAR | default "fallback"}}: 带默认值
//   - {{coalesce .VAR1 .VAR2 "default"}}: 多级回退
//   - {{env "VAR"}}: 显式获取环境变量
//
// # 调试辅助
//
// 提供便捷方法用于调试：
//
//	client.GetLastInput()            // 获取最后一次用户输入
//	client.GetAllInputs()            // 获取所有用户输入
//	client.GetScenarioTurnIndex(name) // 获取场景当前轮次
//	client.GetScenarioNames()        // 获取所有场景名称
//
// # 配置选项
//
// 使用选项函数配置 Client 行为：
//
//   - [WithResponse]: 设置预设响应文本
//   - [WithResponses]: 设置响应队列（多次调用依次返回）
//   - [WithResponseFunc]: 设置动态响应函数
//   - [WithMessageFunc]: 设置完整消息响应函数（支持工具调用）
//   - [WithDelay]: 设置响应延迟
//   - [WithError]: 设置返回错误
//   - [WithConfigFile]: 从 YAML/JSON 文件加载配置
//   - [WithConfig]: 从配置对象加载设置
//
// # 线程安全
//
// [Client] 是线程安全的，可以并发调用 Complete 和 Stream 方法。
package mock
