package provider

import (
	"testing"

	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ═══════════════════════════════════════════════════════════════════════════
// New 函数测试
// ═══════════════════════════════════════════════════════════════════════════

func TestNew_NilConfig(t *testing.T) {
	p, err := New(nil)

	assert.Nil(t, p)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "config is required")
}

func TestNew_MissingAPIKey(t *testing.T) {
	cfg := &llm.Config{
		Type: llm.ProviderTypeOpenAI,
		// 没有 APIKey
	}

	p, err := New(cfg)

	assert.Nil(t, p)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API key is required")
}

func TestNew_OllamaNoAPIKeyOK(t *testing.T) {
	// 注意：虽然 provider.go 跳过了 API key 检查，
	// 但 openai.New() 内部也会检查 API key。
	// 这是当前的实现行为，Ollama 实际上需要提供一个 API key（可以是任意值）
	cfg := &llm.Config{
		Type:    llm.ProviderTypeOllama,
		BaseURL: "http://localhost:11434/v1",
		APIKey:  "ollama", // Ollama 可以使用任意 API key
	}

	p, err := New(cfg)

	require.NoError(t, err)
	require.NotNil(t, p)
	defer func() { _ = p.Close() }()
}

func TestNew_DefaultProviderType(t *testing.T) {
	// 不指定 Type 时默认使用 OpenRouter
	cfg := &llm.Config{
		APIKey: "test-key",
	}

	p, err := New(cfg)

	require.NoError(t, err)
	require.NotNil(t, p)
	defer func() { _ = p.Close() }()
}

func TestNew_OpenAI(t *testing.T) {
	cfg := &llm.Config{
		Type:   llm.ProviderTypeOpenAI,
		APIKey: "test-key",
	}

	p, err := New(cfg)

	require.NoError(t, err)
	require.NotNil(t, p)
	defer func() { _ = p.Close() }()
}

func TestNew_Anthropic(t *testing.T) {
	cfg := &llm.Config{
		Type:   llm.ProviderTypeAnthropic,
		APIKey: "test-key",
	}

	p, err := New(cfg)

	require.NoError(t, err)
	require.NotNil(t, p)
	defer func() { _ = p.Close() }()
}

func TestNew_Gemini(t *testing.T) {
	cfg := &llm.Config{
		Type:   llm.ProviderTypeGemini,
		APIKey: "test-key",
	}

	p, err := New(cfg)

	require.NoError(t, err)
	require.NotNil(t, p)
	defer func() { _ = p.Close() }()
}

func TestNew_UnsupportedType(t *testing.T) {
	cfg := &llm.Config{
		Type:   "unsupported_provider",
		APIKey: "test-key",
	}

	p, err := New(cfg)

	assert.Nil(t, p)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}

func TestNew_OpenAICompatibleProviders(t *testing.T) {
	// 所有 OpenAI 兼容的 Provider 类型
	compatibleTypes := []llm.ProviderType{
		llm.ProviderTypeOpenAI,
		llm.ProviderTypeOpenRouter,
		llm.ProviderTypeDeepSeek,
		llm.ProviderTypeAzure,
		llm.ProviderTypeGLM,
		llm.ProviderTypeDoubao,
		llm.ProviderTypeMoonshot,
		llm.ProviderTypeGroq,
		llm.ProviderTypeMistral,
	}

	for _, ptype := range compatibleTypes {
		t.Run(string(ptype), func(t *testing.T) {
			cfg := &llm.Config{
				Type:   ptype,
				APIKey: "test-key",
			}

			p, err := New(cfg)

			require.NoError(t, err, "Provider type %s should be supported", ptype)
			require.NotNil(t, p)
			defer func() { _ = p.Close() }()
		})
	}
}

func TestNew_WithBaseURL(t *testing.T) {
	cfg := &llm.Config{
		Type:    llm.ProviderTypeOpenAI,
		APIKey:  "test-key",
		BaseURL: "https://custom.api.example.com/v1",
	}

	p, err := New(cfg)

	require.NoError(t, err)
	require.NotNil(t, p)
	defer func() { _ = p.Close() }()
}

func TestNew_WithModel(t *testing.T) {
	cfg := &llm.Config{
		Type:   llm.ProviderTypeOpenAI,
		APIKey: "test-key",
		Model:  "gpt-4-turbo",
	}

	p, err := New(cfg)

	require.NoError(t, err)
	require.NotNil(t, p)
	defer func() { _ = p.Close() }()
}

// ═══════════════════════════════════════════════════════════════════════════
// Mock 函数测试
// ═══════════════════════════════════════════════════════════════════════════

func TestMock(t *testing.T) {
	p := Mock()

	require.NotNil(t, p)
	defer func() { _ = p.Close() }()
}

// ═══════════════════════════════════════════════════════════════════════════
// Must 函数测试
// ═══════════════════════════════════════════════════════════════════════════

func TestMust_Success(t *testing.T) {
	cfg := &llm.Config{
		Type:   llm.ProviderTypeOpenAI,
		APIKey: "test-key",
	}

	p := Must(cfg)

	require.NotNil(t, p)
	defer func() { _ = p.Close() }()
}

func TestMust_Panic(t *testing.T) {
	defer func() {
		r := recover()
		assert.NotNil(t, r, "Must should panic on invalid config")
	}()

	// nil config 应该导致 panic
	Must(nil)
}

// ═══════════════════════════════════════════════════════════════════════════
// extractHeaders 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestNew_WithHeaders(t *testing.T) {
	cfg := &llm.Config{
		Type:   llm.ProviderTypeOpenAI,
		APIKey: "test-key",
		Extra: map[string]any{
			"headers": map[string]string{
				"X-Custom-Header": "custom-value",
			},
		},
	}

	p, err := New(cfg)

	require.NoError(t, err)
	require.NotNil(t, p)
	defer func() { _ = p.Close() }()
}

func TestNew_ExtraWithoutHeaders(t *testing.T) {
	cfg := &llm.Config{
		Type:   llm.ProviderTypeOpenAI,
		APIKey: "test-key",
		Extra: map[string]any{
			"other": "value",
		},
	}

	p, err := New(cfg)

	require.NoError(t, err)
	require.NotNil(t, p)
	defer func() { _ = p.Close() }()
}
