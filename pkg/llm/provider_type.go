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

// providerMeta Provider 元数据
type providerMeta struct {
	openAICompatible bool
	baseURL          string
	model            string
	envVarName       string
}

// providerRegistry 集中管理所有 Provider 配置
var providerRegistry = map[ProviderType]providerMeta{
	ProviderTypeOpenAI:     {true, "https://api.openai.com/v1", "gpt-4o-mini", "OPENAI_API_KEY"},
	ProviderTypeOpenRouter: {true, "https://openrouter.ai/api/v1", "anthropic/claude-haiku-4.5", "OPENROUTER_API_KEY"},
	ProviderTypeAnthropic:  {false, "https://api.anthropic.com/v1", "claude-3-5-haiku-latest", "ANTHROPIC_API_KEY"},
	ProviderTypeDeepSeek:   {true, "https://api.deepseek.com/v1", "deepseek-chat", "DEEPSEEK_API_KEY"},
	ProviderTypeOllama:     {true, "http://localhost:11434/v1", "llama3.2", ""},
	ProviderTypeAzure:      {true, "", "", "AZURE_API_KEY"},
	ProviderTypeGemini:     {false, "https://generativelanguage.googleapis.com/v1beta", "gemini-1.5-flash", "GOOGLE_API_KEY"},
	ProviderTypeLocalMock:  {false, "", "", ""},
	ProviderTypeGLM:        {true, "https://open.bigmodel.cn/api/paas/v4", "glm-4-flash", "BIGMODEL_API_KEY"},
	ProviderTypeDoubao:     {true, "https://ark.cn-beijing.volces.com/api/v3", "", "DOUBAO_API_KEY"},
	ProviderTypeMoonshot:   {true, "https://api.moonshot.cn/v1", "moonshot-v1-128k", "MOONSHOT_API_KEY"},
	ProviderTypeGroq:       {true, "https://api.groq.com/openai/v1", "llama-3.3-70b-versatile", "GROQ_API_KEY"},
	ProviderTypeMistral:    {true, "https://api.mistral.ai/v1", "mistral-large-latest", "MISTRAL_API_KEY"},
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
	if name := providerRegistry[t].envVarName; name != "" {
		return os.Getenv(name)
	}
	return ""
}
