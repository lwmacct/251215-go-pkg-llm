package core

import (
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
)

// ═══════════════════════════════════════════════════════════════════════════
// 协议适配器接口
// ═══════════════════════════════════════════════════════════════════════════

// ProtocolAdapter 协议适配器接口
//
// 每个 LLM Provider 实现此接口来定义协议特有的转换逻辑。
//
// 设计原则：
//   - 显式差异：协议差异通过接口方法明确体现
//   - 零魔法：所有转换逻辑可追踪，无隐式行为
//   - 单一职责：只负责协议格式转换，不涉及业务逻辑
//
// 职责边界：
//   - ✅ 负责：消息格式转换、参数序列化策略
//   - ❌ 不负责：HTTP 通信、配置管理、错误处理
type ProtocolAdapter interface {
	// ConvertToAPI 将统一的 Message 转换为 API 请求格式
	//
	// 职责：
	//   - 处理角色映射 (system/user/assistant/tool)
	//   - 处理内容格式差异 (string vs content array)
	//   - 处理工具调用格式差异
	//   - 处理工具结果格式差异
	//
	// 硬约束示例：
	//   - OpenAI: 工具参数必须序列化为 JSON 字符串
	//   - Anthropic: 工具参数必须保持为对象
	//   - OpenAI: ToolResult 必须独立消息 (role=tool)
	//   - Anthropic: ToolResult 必须内联 (content array)
	//
	// 参数：
	//   - messages: 统一的内部消息格式
	//
	// 返回：
	//   - API 特定格式的消息数组
	ConvertToAPI(messages []llm.Message) []map[string]any

	// ConvertFromAPI 将 API 响应转换为统一的 Message
	//
	// 职责：
	//   - 解析响应结构
	//   - 提取文本内容
	//   - 提取工具调用（反序列化 JSON 字符串 → 对象）
	//   - 映射完成原因
	//
	// 参数：
	//   - apiResp: API 返回的原始响应 map
	//
	// 返回：
	//   - msg: 统一格式的 Message
	//   - finishReason: 标准化的完成原因
	ConvertFromAPI(apiResp map[string]any) (msg llm.Message, finishReason string)

	// ConvertUsage 解析 Token 使用量
	//
	// 职责：
	//   - 映射字段名差异 (prompt_tokens vs input_tokens)
	//   - 计算总量
	//   - 提取推理 tokens / 缓存 tokens
	//
	// 参数：
	//   - apiResp: API 返回的原始响应 map
	//
	// 返回：
	//   - Token 使用量，如果无 usage 字段则返回 nil
	ConvertUsage(apiResp map[string]any) *llm.TokenUsage

	// GetSystemMessageHandling 返回系统消息处理策略
	//
	// 返回：
	//   - SystemInline: 系统消息作为普通消息 (OpenAI)
	//   - SystemSeparate: 系统消息作为独立参数 (Anthropic)
	GetSystemMessageHandling() SystemMessageStrategy
}

// ═══════════════════════════════════════════════════════════════════════════
// 系统消息策略
// ═══════════════════════════════════════════════════════════════════════════

// SystemMessageStrategy 系统消息处理策略
//
// 定义系统提示（system prompt）如何传递给 API：
//   - SystemInline: 系统消息作为第一条普通消息（role=system）
//   - SystemSeparate: 系统消息作为独立的请求参数
type SystemMessageStrategy string

const (
	// SystemInline 系统消息内联在消息数组中
	//
	// 使用场景：OpenAI API
	// 格式：[{"role": "system", "content": "..."}, ...]
	SystemInline SystemMessageStrategy = "inline"

	// SystemSeparate 系统消息作为独立参数
	//
	// 使用场景：Anthropic API
	// 格式：{"system": "...", "messages": [...]}
	SystemSeparate SystemMessageStrategy = "separate"
)

// ═══════════════════════════════════════════════════════════════════════════
// 系统消息处理策略
// ═══════════════════════════════════════════════════════════════════════════
