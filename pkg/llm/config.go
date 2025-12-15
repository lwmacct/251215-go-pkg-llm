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
	// Type Provider 类型（默认 OpenRouter）
	Type ProviderType

	// APIKey（Ollama 除外，其他 Provider 必需）
	APIKey string

	// 可选字段（有默认值）
	Model   string
	BaseURL string

	// 网络配置
	Timeout    time.Duration // HTTP 超时（默认 120s）
	MaxRetries int           // 重试次数（默认 3）

	// 扩展配置
	Extra map[string]any // Provider 特定配置（headers, deployment 等）
}

func DefaultConfig() *Config {
	return &Config{
		Type:       ProviderTypeOpenRouter,
		Timeout:    120 * time.Second,
		MaxRetries: 3,
	}
}
