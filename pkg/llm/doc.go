// Package llm 提供 LLM（大语言模型）的统一抽象层
//
// 本包定义了与 LLM 服务交互所需的核心类型和接口，包括：
//   - [Provider]: 统一的 LLM 调用抽象
//   - [Message]: 对话消息结构
//   - [Event]: 流式事件定义
//   - 环境变量探测：自动配置
//
// 完整使用示例请参考 example_test.go。
//
// # 核心类型
//
// [Provider] 接口定义了 LLM 服务的调用契约，支持同步和流式两种模式。
//
// [Message] 表示对话中的单条消息，支持多种内容块（文本、工具调用、工具结果）。
//
// [Event] 用于流式响应，包含文本增量、工具调用、完成、错误等事件类型。
//
// # 工具调用
//
// [ToolCall] 表示模型发起的工具调用请求。
//
// [ToolResult] 表示工具执行结果。
//
// # Provider 类型
//
// [ProviderType] 枚举支持的 Provider 类型：
//   - ProviderTypeOpenAI: OpenAI 及兼容服务
//   - ProviderTypeAnthropic: Anthropic Claude
//   - ProviderTypeOpenRouter: OpenRouter 聚合服务
//
// # 环境变量
//
// 本包支持从环境变量自动探测配置：
//
// API Key（按优先级）:
//   - OPENAI_API_KEY
//   - ANTHROPIC_API_KEY
//   - OPENROUTER_API_KEY
//   - LLM_API_KEY
//
// Base URL:
//   - OPENAI_BASE_URL
//   - OPENAI_API_BASE
//   - LLM_BASE_URL
//
// Model:
//   - OPENAI_MODEL
//   - LLM_MODEL
//   - MODEL
//
// # 协议实现
//
// 具体的 Provider 实现位于子包：
//   - [pkg/llm/provider/openai]: OpenAI 协议实现
//   - [pkg/llm/provider/anthropic]: Anthropic 协议实现
//   - [pkg/llm/provider/localmock]: 本地 Mock 实现（用于测试）
//
// # 与 Agent 包的关系
//
// 本包是底层协议抽象，[pkg/agent] 包在此基础上构建 Agent 功能。
//
// # 包文件组织
//
//   - types.go: Provider 接口、Options、Response
//   - message.go: Message、ContentBlock、ToolCall
//   - event.go: Event、EventType
//   - provider_type.go: ProviderType 枚举
//   - env.go: 环境变量探测
package llm
