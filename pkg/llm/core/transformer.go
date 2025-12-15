package core

import (
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
)

// ═══════════════════════════════════════════════════════════════════════════
// 消息转换器
// ═══════════════════════════════════════════════════════════════════════════

// Transformer 消息转换器
//
// 封装通用的消息转换逻辑，使用 ProtocolAdapter 处理协议差异。
//
// 设计原则：
//   - 模板方法模式：定义转换流程骨架，具体差异委托给 adapter
//   - 单一职责：只负责消息转换流程编排
//   - 依赖倒置：依赖抽象的 ProtocolAdapter 接口
//
// 使用示例：
//
//	adapter := openai.NewAdapter()
//	transformer := core.NewTransformer(adapter)
//
//	// 构建 API 请求消息
//	apiMsgs := transformer.BuildAPIMessages(messages, systemPrompt)
//
//	// 解析 API 响应
//	msg, reason, usage := transformer.ParseAPIResponse(apiResp)
type Transformer struct {
	adapter ProtocolAdapter
}

// NewTransformer 创建消息转换器
//
// 参数：
//   - adapter: 协议特定的适配器实现
//
// 返回：
//   - 转换器实例
func NewTransformer(adapter ProtocolAdapter) *Transformer {
	return &Transformer{adapter: adapter}
}

// BuildAPIMessages 构建 API 请求消息数组
//
// 通用流程：
//  1. 检查消息有效性
//  2. 过滤系统消息（根据协议策略处理）
//  3. 委托 adapter 转换每条消息
//  4. 根据协议策略处理系统提示
//
// 参数：
//   - messages: 统一格式的内部消息
//   - systemPrompt: 系统提示内容（可选）
//
// 返回：
//   - API 特定格式的消息数组
//
// 注意：
//   - 系统消息的处理方式由 adapter.GetSystemMessageHandling() 决定
//   - SystemInline: 系统提示插入消息数组开头
//   - SystemSeparate: 系统提示不处理（由调用方作为独立参数传递）
func (t *Transformer) BuildAPIMessages(
	messages []llm.Message,
	systemPrompt string,
) []map[string]any {
	// 预处理：过滤系统消息（系统消息由独立参数处理）
	var userMessages []llm.Message
	for _, msg := range messages {
		if msg.Role != llm.RoleSystem {
			userMessages = append(userMessages, msg)
		}
	}

	// 委托 adapter 转换消息
	apiMsgs := t.adapter.ConvertToAPI(userMessages)

	// 处理系统提示（根据协议策略）
	if systemPrompt != "" {
		switch t.adapter.GetSystemMessageHandling() {
		case SystemInline:
			// OpenAI 策略：插入到数组开头
			systemMsg := map[string]any{
				"role":    "system",
				"content": systemPrompt,
			}
			apiMsgs = append([]map[string]any{systemMsg}, apiMsgs...)

		case SystemSeparate:
			// Anthropic 策略：不处理（由调用方作为独立参数传递）
			// 调用方应该将 systemPrompt 放在请求的 "system" 字段
		}
	}

	return apiMsgs
}

// ParseAPIResponse 解析 API 响应
//
// 通用流程：
//  1. 委托 adapter 转换响应为统一 Message
//  2. 委托 adapter 解析 Token 使用量
//  3. 返回统一格式的结果
//
// 参数：
//   - apiResp: API 返回的原始响应 map
//
// 返回：
//   - msg: 统一格式的 Message
//   - finishReason: 标准化的完成原因
//   - usage: Token 使用量统计（可能为 nil）
//
// 示例：
//
//	var apiResp map[string]any
//	// ... HTTP 请求获取响应 ...
//
//	msg, reason, usage := transformer.ParseAPIResponse(apiResp)
//	fmt.Println("完成原因:", reason)
//	fmt.Println("使用 tokens:", usage.TotalTokens)
func (t *Transformer) ParseAPIResponse(apiResp map[string]any) (
	msg llm.Message,
	finishReason string,
	usage *llm.TokenUsage,
) {
	// 委托 adapter 转换消息
	msg, finishReason = t.adapter.ConvertFromAPI(apiResp)

	// 委托 adapter 解析使用量
	usage = t.adapter.ConvertUsage(apiResp)

	return
}
