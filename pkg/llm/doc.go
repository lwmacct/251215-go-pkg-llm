// Package llm 提供 LLM（大语言模型）的统一抽象层
//
// 本包定义了与 LLM 服务交互所需的核心类型和接口，包括：
//   - [Provider]: 统一的 LLM 调用抽象
//   - [Message]: 对话消息结构
//   - [Event]: 流式事件定义
//   - [ProviderType]: Provider 类型与元数据
//
// 完整使用示例请参考 example_test.go。
//
// # 快速开始
//
// 零配置使用（从环境变量读取 API Key）：
//
//	p, err := provider.Default()                        // 默认 OpenRouter
//	p, err := provider.Default(llm.ProviderTypeOpenAI)  // 指定类型
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
// [ProviderType] 枚举支持的 Provider 类型，并提供元数据查询：
//   - DefaultBaseURL(): 默认 API 地址
//   - DefaultModel(): 默认模型
//   - GetEnvAPIKey(): 从环境变量获取 API Key
//   - IsOpenAICompatible(): 是否兼容 OpenAI 协议
//
// 支持的 Provider：
//   - OpenAI、Anthropic、Gemini（原生协议）
//   - OpenRouter、DeepSeek、Ollama、Azure、GLM、Doubao、Moonshot、Groq、Mistral（OpenAI 兼容）
//
// # 协议实现
//
// 具体的 Provider 实现位于子包：
//   - [pkg/llm/provider/openai]: OpenAI 协议实现
//   - [pkg/llm/provider/anthropic]: Anthropic 协议实现
//   - [pkg/llm/provider/gemini]: Gemini 协议实现
//   - [pkg/llm/provider/localmock]: 本地 Mock 实现（用于测试）
//
// # 包文件组织
//
//   - types.go: Provider 接口、Options、Response
//   - message.go: Message、ContentBlock、ToolCall
//   - event.go: Event、EventType
//   - provider_type.go: ProviderType 枚举与元数据
//   - config.go: Config 配置与 DefaultConfig
package llm
