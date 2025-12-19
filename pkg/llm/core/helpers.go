package core

// ═══════════════════════════════════════════════════════════════════════════
// 类型转换辅助函数
// ═══════════════════════════════════════════════════════════════════════════

// GetInt64 将 any 类型安全转换为 int64
//
// 支持的输入类型：
//   - float64: JSON 数字的默认类型
//   - int: Go 原生整数
//   - int64: Go 64位整数
//
// 其他类型返回 0（零值）。
//
// 使用场景：
//   - 解析 API 响应中的 token 数量
//   - 解析流式响应中的索引
//
// 示例：
//
//	usage := apiResp["usage"].(map[string]any)
//	inputTokens := GetInt64(usage["input_tokens"])  // 处理 float64
func GetInt64(val any) int64 {
	switch v := val.(type) {
	case float64:
		return int64(v)
	case int:
		return int64(v)
	case int64:
		return v
	default:
		return 0
	}
}

// GetFloat64 将 any 类型安全转换为 float64
//
// 支持的输入类型：
//   - float64: JSON 数字的默认类型
//   - int: Go 原生整数
//   - int64: Go 64位整数
//
// 其他类型返回 0.0（零值）。
//
// 使用场景：
//   - 解析流式响应中的索引（Anthropic 使用 float64）
//   - 解析温度等浮点参数
//
// 示例：
//
//	index := GetFloat64(data["index"])  // Anthropic 返回 float64
func GetFloat64(val any) float64 {
	switch v := val.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return 0
	}
}

// GetString 将 any 类型安全转换为 string
//
// 支持的输入类型：
//   - string: JSON 字符串
//
// 其他类型返回 ""（空字符串）。
//
// 使用场景：
//   - 解析 API 响应中的字符串字段
//   - 解析流式响应中的 ID、名称等
//
// 示例：
//
//	id := GetString(block["id"])
//	name := GetString(block["name"])
func GetString(val any) string {
	if s, ok := val.(string); ok {
		return s
	}
	return ""
}
