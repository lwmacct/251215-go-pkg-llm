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
// 零配置（推荐）：
//
//	p, err := provider.Default()                        // 默认 OpenRouter
//	p, err := provider.Default(llm.ProviderTypeOpenAI)  // 指定类型
//
// 自定义配置：
//
//	cfg := llm.DefaultConfig(llm.ProviderTypeOpenAI)
//	cfg.Model = "gpt-4o"
//	p, err := provider.New(cfg)
//
// 完全手动配置：
//
//	p, err := provider.New(&llm.Config{
//	    Type:   llm.ProviderTypeOpenAI,
//	    APIKey: "sk-xxx",
//	    Model:  "gpt-4o",
//	})

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
func DefaultConfig(types ...ProviderType) *Config {
	t := ProviderTypeOpenRouter
	if len(types) > 0 {
		t = types[0]
	}
	return &Config{
		Type:       t,
		APIKey:     t.GetEnvAPIKey(),
		BaseURL:    t.GetEnvBaseURL(),
		Model:      t.GetEnvModel(),
		Timeout:    120 * time.Second,
		MaxRetries: 3,
	}
}
