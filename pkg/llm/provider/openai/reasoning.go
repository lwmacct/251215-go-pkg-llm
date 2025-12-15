package openai

import "strings"

// ═══════════════════════════════════════════════════════════════════════════
// Reasoning 模型适配
// ═══════════════════════════════════════════════════════════════════════════

// reasoningModelPrefixes Reasoning 模型前缀列表
// 这些模型有特殊要求：temperature 必须为 1，不支持 top_p
var reasoningModelPrefixes = []string{
	// OpenAI Reasoning 模型
	"o1-",
	"o1-mini",
	"o1-preview",
	"o3-",
	"o3-mini",
	"o4-",
	"o4-mini",
	"gpt-5",
	"gpt-5-mini",
	"gpt-5-nano",
	// DeepSeek Reasoning 模型
	"deepseek-reasoner",
	"deepseek-r1",
}

// IsReasoningModel 判断是否为 Reasoning 模型
//
// Reasoning 模型 (如 o1, o3, DeepSeek R1) 有特殊的 API 限制：
//   - temperature 必须为 1
//   - 不支持 top_p 参数
//   - 支持 reasoning_effort 参数
func IsReasoningModel(model string) bool {
	modelLower := strings.ToLower(model)
	for _, prefix := range reasoningModelPrefixes {
		if strings.HasPrefix(modelLower, prefix) {
			return true
		}
	}
	return false
}

// AdaptTemperatureForModel 根据模型类型适配温度参数
//
// Reasoning 模型强制返回 1.0，其他模型返回原值
func AdaptTemperatureForModel(model string, requestedTemp float64) float64 {
	if IsReasoningModel(model) {
		return 1.0
	}
	return requestedTemp
}

// ReasoningEffort Reasoning 力度级别
type ReasoningEffort string

const (
	ReasoningEffortMinimal ReasoningEffort = "minimal"
	ReasoningEffortLow     ReasoningEffort = "low"
	ReasoningEffortMedium  ReasoningEffort = "medium"
	ReasoningEffortHigh    ReasoningEffort = "high"
)

// IsValidReasoningEffort 验证 Reasoning 力度是否有效
func IsValidReasoningEffort(effort string) bool {
	switch ReasoningEffort(effort) {
	case ReasoningEffortMinimal, ReasoningEffortLow, ReasoningEffortMedium, ReasoningEffortHigh:
		return true
	default:
		return effort == ""
	}
}
