package llm

import (
	"time"
)

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
	Type ProviderType `koanf:"type"`

	// APIKey（Ollama 除外，其他 Provider 必需）
	APIKey string `koanf:"api-key"`

	// 可选字段（有默认值）
	Model   string `koanf:"model"`
	BaseURL string `koanf:"base-url"`

	// 网络配置
	Timeout    time.Duration `koanf:"timeout"`
	MaxRetries int           `koanf:"max-retries"`

	// 扩展配置
	Extra map[string]any `koanf:"extra"`
}

// DefaultConfig 返回默认的 Provider 配置
// 不指定类型时默认使用 OpenRouter
func DefaultConfig(types ...ProviderType) Config {
	t := ProviderTypeOpenRouter
	if len(types) > 0 {
		t = types[0]
	}
	return Config{
		Type:       t,
		APIKey:     t.GetEnvAPIKey(),
		BaseURL:    t.DefaultBaseURL(),
		Model:      t.DefaultModel(),
		Timeout:    120 * time.Second,
		MaxRetries: 3,
	}
}
