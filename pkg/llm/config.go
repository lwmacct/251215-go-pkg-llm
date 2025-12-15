package llm

import "time"

// ═══════════════════════════════════════════════════════════════════════════
// Provider 配置
// ═══════════════════════════════════════════════════════════════════════════

// Config Provider 创建配置
//
// 用于通过统一工厂函数创建不同类型的 LLM Provider。
//
// 基本用法：
//
//	cfg := &llm.Config{
//	    Type:   llm.ProviderTypeOpenAI,
//	    APIKey: "sk-xxx",
//	    Model:  "gpt-4",
//	}
//
// 生产环境配置：
//
//	cfg := &llm.Config{
//	    Type:       llm.ProviderTypeOpenAI,
//	    APIKey:     "sk-xxx",
//	    Model:      "gpt-4o",
//	    Timeout:    2 * time.Minute,
//	    MaxRetries: 3,
//	}
//
// 云平台部署（Azure/Vertex AI）：
//
//	cfg := &llm.Config{
//	    Type:   llm.ProviderTypeAzure,
//	    APIKey: "xxx",
//	    Model:  "gpt-4o",
//	    Extra: map[string]any{
//	        "deployment":  "my-deployment",
//	        "api_version": "2025-01-15",
//	    },
//	}

type Config struct {
	// Provider 类型（默认 OpenRouter）
	Type ProviderType `json:"type,omitempty" yaml:"type,omitempty"`

	// APIKey（Ollama 除外，其他 Provider 必需）
	APIKey string `json:"api_key,omitempty" yaml:"api_key,omitempty"`

	// 可选字段（有默认值）
	Model   string `json:"model,omitempty" yaml:"model,omitempty"`
	BaseURL string `json:"base_url,omitempty" yaml:"base_url,omitempty"`

	// 网络配置
	Timeout    time.Duration `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	MaxRetries int           `json:"max_retries,omitempty" yaml:"max_retries,omitempty"`

	// 扩展配置
	Extra map[string]any `json:"extra,omitempty" yaml:"extra,omitempty"`
}

func DefaultConfig() *Config {
	return &Config{
		Type:       ProviderTypeOpenRouter,
		Timeout:    120 * time.Second,
		MaxRetries: 3,
	}
}
