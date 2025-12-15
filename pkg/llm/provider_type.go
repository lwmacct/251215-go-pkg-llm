package llm

// ProviderType LLM Provider 类型
type ProviderType string

const (
	// ProviderTypeOpenAI OpenAI 原生 API
	ProviderTypeOpenAI ProviderType = "openai"

	// ProviderTypeOpenRouter OpenRouter API（OpenAI 兼容）
	ProviderTypeOpenRouter ProviderType = "openrouter"

	// ProviderTypeAnthropic Anthropic 原生 API
	ProviderTypeAnthropic ProviderType = "anthropic"

	// ProviderTypeDeepSeek DeepSeek API（OpenAI 兼容）
	ProviderTypeDeepSeek ProviderType = "deepseek"

	// ProviderTypeOllama Ollama 本地模型（OpenAI 兼容）
	ProviderTypeOllama ProviderType = "ollama"

	// ProviderTypeAzure Azure OpenAI API
	ProviderTypeAzure ProviderType = "azure"

	// ProviderTypeGemini Google Gemini API
	ProviderTypeGemini ProviderType = "gemini"

	// ProviderTypeLocalMock 本地 Mock（测试用）
	ProviderTypeLocalMock ProviderType = "localmock"

	// ProviderTypeGLM 智谱 GLM API（OpenAI 兼容）
	ProviderTypeGLM ProviderType = "glm"

	// ProviderTypeDoubao 字节跳动豆包 API（OpenAI 兼容）
	ProviderTypeDoubao ProviderType = "doubao"

	// ProviderTypeMoonshot 月之暗面 Kimi API（OpenAI 兼容）
	ProviderTypeMoonshot ProviderType = "moonshot"

	// ProviderTypeGroq Groq 快速推理 API（OpenAI 兼容）
	ProviderTypeGroq ProviderType = "groq"

	// ProviderTypeMistral Mistral AI API（OpenAI 兼容）
	ProviderTypeMistral ProviderType = "mistral"
)

// String 返回字符串表示
func (t ProviderType) String() string {
	return string(t)
}

// IsOpenAICompatible 判断是否为 OpenAI 兼容协议
func (t ProviderType) IsOpenAICompatible() bool {
	switch t {
	case ProviderTypeOpenAI, ProviderTypeOpenRouter, ProviderTypeDeepSeek,
		ProviderTypeOllama, ProviderTypeAzure, ProviderTypeGLM, ProviderTypeDoubao,
		ProviderTypeMoonshot, ProviderTypeGroq, ProviderTypeMistral:
		return true
	default:
		return false
	}
}

// DefaultBaseURL 返回默认 Base URL
func (t ProviderType) DefaultBaseURL() string {
	switch t {
	case ProviderTypeOpenAI:
		return "https://api.openai.com/v1"
	case ProviderTypeOpenRouter:
		return "https://openrouter.ai/api/v1"
	case ProviderTypeAnthropic:
		return "https://api.anthropic.com/v1"
	case ProviderTypeDeepSeek:
		return "https://api.deepseek.com/v1"
	case ProviderTypeOllama:
		return "http://localhost:11434/v1"
	case ProviderTypeGemini:
		return "https://generativelanguage.googleapis.com/v1beta"
	case ProviderTypeGLM:
		return "https://open.bigmodel.cn/api/paas/v4"
	case ProviderTypeDoubao:
		return "https://ark.cn-beijing.volces.com/api/v3"
	case ProviderTypeMoonshot:
		return "https://api.moonshot.cn/v1"
	case ProviderTypeGroq:
		return "https://api.groq.com/openai/v1"
	case ProviderTypeMistral:
		return "https://api.mistral.ai/v1"
	default:
		return ""
	}
}

// DefaultModel 返回默认模型
func (t ProviderType) DefaultModel() string {
	switch t {
	case ProviderTypeOpenAI:
		return "gpt-4o-mini"
	case ProviderTypeOpenRouter:
		return "anthropic/claude-haiku-4.5"
	case ProviderTypeAnthropic:
		return "claude-3-5-haiku-latest"
	case ProviderTypeDeepSeek:
		return "deepseek-chat"
	case ProviderTypeOllama:
		return "llama3.2"
	case ProviderTypeGemini:
		return "gemini-1.5-flash"
	case ProviderTypeGLM:
		return "glm-4-flash"
	case ProviderTypeDoubao:
		return "" // 需要用户指定 endpoint_id
	case ProviderTypeMoonshot:
		return "moonshot-v1-128k"
	case ProviderTypeGroq:
		return "llama-3.3-70b-versatile"
	case ProviderTypeMistral:
		return "mistral-large-latest"
	default:
		return ""
	}
}
