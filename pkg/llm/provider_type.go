package llm

import "os"

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

	// ProviderTypeMock 本地 Mock（测试用）
	ProviderTypeMock ProviderType = "mock"

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

// providerMeta Provider 元数据
type providerMeta struct {
	openAICompatible bool
	baseURL          string
	model            string
	apiKeyEnvVar     string
	modelEnvVar      string
	baseURLEnvVar    string
}

// providerRegistry 集中管理所有 Provider 配置
var providerRegistry = map[ProviderType]providerMeta{
	ProviderTypeOpenAI:     {true, "https://api.openai.com/v1", "gpt-4o-mini", "OPENAI_API_KEY", "OPENAI_MODEL", "OPENAI_BASE_URL"},
	ProviderTypeOpenRouter: {true, "https://openrouter.ai/api/v1", "anthropic/claude-haiku-4.5", "OPENROUTER_API_KEY", "OPENROUTER_MODEL", "OPENROUTER_BASE_URL"},
	ProviderTypeAnthropic:  {false, "https://api.anthropic.com/v1", "claude-3-5-haiku-latest", "ANTHROPIC_API_KEY", "ANTHROPIC_MODEL", "ANTHROPIC_BASE_URL"},
	ProviderTypeDeepSeek:   {true, "https://api.deepseek.com/v1", "deepseek-chat", "DEEPSEEK_API_KEY", "DEEPSEEK_MODEL", "DEEPSEEK_BASE_URL"},
	ProviderTypeOllama:     {true, "http://localhost:11434/v1", "llama3.2", "", "OLLAMA_MODEL", "OLLAMA_BASE_URL"},
	ProviderTypeAzure:      {true, "", "", "AZURE_API_KEY", "AZURE_MODEL", "AZURE_BASE_URL"},
	ProviderTypeGemini:     {false, "https://generativelanguage.googleapis.com/v1beta", "gemini-1.5-flash", "GOOGLE_API_KEY", "GOOGLE_MODEL", "GOOGLE_BASE_URL"},
	ProviderTypeMock:       {false, "", "", "", "", ""},
	ProviderTypeGLM:        {true, "https://open.bigmodel.cn/api/paas/v4", "glm-4-flash", "BIGMODEL_API_KEY", "BIGMODEL_MODEL", "BIGMODEL_BASE_URL"},
	ProviderTypeDoubao:     {true, "https://ark.cn-beijing.volces.com/api/v3", "", "DOUBAO_API_KEY", "DOUBAO_MODEL", "DOUBAO_BASE_URL"},
	ProviderTypeMoonshot:   {true, "https://api.moonshot.cn/v1", "moonshot-v1-128k", "MOONSHOT_API_KEY", "MOONSHOT_MODEL", "MOONSHOT_BASE_URL"},
	ProviderTypeGroq:       {true, "https://api.groq.com/openai/v1", "llama-3.3-70b-versatile", "GROQ_API_KEY", "GROQ_MODEL", "GROQ_BASE_URL"},
	ProviderTypeMistral:    {true, "https://api.mistral.ai/v1", "mistral-large-latest", "MISTRAL_API_KEY", "MISTRAL_MODEL", "MISTRAL_BASE_URL"},
}

// String 返回字符串表示
func (t ProviderType) String() string {
	return string(t)
}

// IsOpenAICompatible 判断是否为 OpenAI 兼容协议
func (t ProviderType) IsOpenAICompatible() bool {
	return providerRegistry[t].openAICompatible
}

// DefaultBaseURL 返回默认 Base URL
func (t ProviderType) DefaultBaseURL() string {
	return providerRegistry[t].baseURL
}

// DefaultModel 返回默认模型
func (t ProviderType) DefaultModel() string {
	return providerRegistry[t].model
}

// GetEnvAPIKey 获取对应环境变量的 API Key 值
// 优先使用自定义环境变量名，回退到默认环境变量名
func (t ProviderType) GetEnvAPIKey(customEnvNames ...string) string {
	for _, name := range customEnvNames {
		if val := os.Getenv(name); val != "" {
			return val
		}
	}
	if name := providerRegistry[t].apiKeyEnvVar; name != "" {
		return os.Getenv(name)
	}
	return ""
}

// GetEnvModel 获取模型名称（环境变量优先）
// 优先使用自定义环境变量名，回退到默认环境变量名，最后回退到默认模型
func (t ProviderType) GetEnvModel(customEnvNames ...string) string {
	for _, name := range customEnvNames {
		if val := os.Getenv(name); val != "" {
			return val
		}
	}
	if name := providerRegistry[t].modelEnvVar; name != "" {
		if val := os.Getenv(name); val != "" {
			return val
		}
	}
	return providerRegistry[t].model
}

// GetEnvBaseURL 获取 Base URL（环境变量优先）
// 优先使用自定义环境变量名，回退到默认环境变量名，最后回退到默认 URL
func (t ProviderType) GetEnvBaseURL(customEnvNames ...string) string {
	for _, name := range customEnvNames {
		if val := os.Getenv(name); val != "" {
			return val
		}
	}
	if name := providerRegistry[t].baseURLEnvVar; name != "" {
		if val := os.Getenv(name); val != "" {
			return val
		}
	}
	return providerRegistry[t].baseURL
}
