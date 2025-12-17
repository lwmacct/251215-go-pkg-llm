// Package provider 提供 LLM Provider 的统一工厂和实现
//
// 使用方式：
//
//	p, err := provider.New(&llm.Config{
//	    Type:   llm.ProviderTypeOpenAI,
//	    APIKey: "sk-xxx",
//	    Model:  "gpt-4",
//	})
//
//	// 本地 Mock（无需配置）
//	p := provider.LocalMock()
package provider

import (
	"fmt"

	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm/provider/anthropic"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm/provider/gemini"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm/provider/localmock"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm/provider/openai"
)

// ═══════════════════════════════════════════════════════════════════════════
// 工厂函数
// ═══════════════════════════════════════════════════════════════════════════

// New 创建 Provider
func New(cfg *llm.Config) (llm.Provider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	apiKey := cfg.APIKey
	providerType := cfg.Type

	// 确定 Provider 类型（默认 OpenRouter）
	if providerType == "" {
		providerType = llm.ProviderTypeOpenRouter
	}

	// Ollama 不需要 API Key
	if providerType != llm.ProviderTypeOllama && apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	// 根据类型创建对应的 Provider
	switch providerType {
	case llm.ProviderTypeOpenAI, llm.ProviderTypeOpenRouter,
		llm.ProviderTypeDeepSeek, llm.ProviderTypeOllama, llm.ProviderTypeAzure,
		llm.ProviderTypeGLM, llm.ProviderTypeDoubao, llm.ProviderTypeMoonshot,
		llm.ProviderTypeGroq, llm.ProviderTypeMistral:
		return newOpenAI(cfg, apiKey, providerType)

	case llm.ProviderTypeAnthropic:
		return newAnthropic(cfg, apiKey)

	case llm.ProviderTypeGemini:
		return newGemini(cfg, apiKey)

	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}
}

// extractHeaders 从 Extra 中提取 headers
func extractHeaders(cfg *llm.Config) map[string]string {
	if cfg.Extra == nil {
		return nil
	}
	if h, ok := cfg.Extra["headers"].(map[string]string); ok {
		return h
	}
	return nil
}

// newOpenAI 创建 OpenAI 兼容 Provider
func newOpenAI(cfg *llm.Config, apiKey string, ptype llm.ProviderType) (llm.Provider, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = ptype.DefaultBaseURL()
	}

	model := cfg.Model
	if model == "" {
		model = ptype.DefaultModel()
	}

	return openai.New(&openai.Config{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   model,
		Timeout: cfg.Timeout,
		Headers: extractHeaders(cfg),
	})
}

// newAnthropic 创建 Anthropic Provider
func newAnthropic(cfg *llm.Config, apiKey string) (llm.Provider, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = llm.ProviderTypeAnthropic.DefaultBaseURL()
	}

	model := cfg.Model
	if model == "" {
		model = llm.ProviderTypeAnthropic.DefaultModel()
	}

	return anthropic.New(&anthropic.Config{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   model,
		Timeout: cfg.Timeout,
		Headers: extractHeaders(cfg),
	})
}

// newGemini 创建 Gemini Provider
func newGemini(cfg *llm.Config, apiKey string) (llm.Provider, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = llm.ProviderTypeGemini.DefaultBaseURL()
	}

	model := cfg.Model
	if model == "" {
		model = llm.ProviderTypeGemini.DefaultModel()
	}

	return gemini.New(&gemini.Config{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   model,
		Timeout: cfg.Timeout,
		Headers: extractHeaders(cfg),
	})
}

// ═══════════════════════════════════════════════════════════════════════════
// 便捷函数
// ═══════════════════════════════════════════════════════════════════════════

// LocalMock 创建 LocalMock Provider（用于测试）
func LocalMock() llm.Provider {
	return localmock.New()
}

// Must 创建 Provider，失败时 panic
func Must(cfg *llm.Config) llm.Provider {
	p, err := New(cfg)
	if err != nil {
		panic(err)
	}
	return p
}

// Default 使用默认配置创建 Provider
// 不指定类型时默认使用 OpenRouter，从对应环境变量读取 APIKey
func Default(types ...llm.ProviderType) (llm.Provider, error) {
	return New(llm.DefaultConfig(types...))
}

// MustDefault 使用默认配置创建 Provider，失败时 panic
func MustDefault(types ...llm.ProviderType) llm.Provider {
	p, err := Default(types...)
	if err != nil {
		panic(err)
	}
	return p
}
