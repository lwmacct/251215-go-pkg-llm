package openai

import (
	"testing"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════════
// 客户端创建测试
// ═══════════════════════════════════════════════════════════════════════════

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name:    "empty API key",
			config:  &Config{},
			wantErr: true,
		},
		{
			name: "valid config",
			config: &Config{
				APIKey: "test-key",
			},
			wantErr: false,
		},
		{
			name: "with custom baseURL",
			config: &Config{
				APIKey:  "test-key",
				BaseURL: "https://custom.api.com/v1",
			},
			wantErr: false,
		},
		{
			name: "with timeout",
			config: &Config{
				APIKey:  "test-key",
				Timeout: 60 * time.Second,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && client == nil {
				t.Error("Expected client to be non-nil")
			}
			if !tt.wantErr {
				// 验证 transformer 和 sseParser 已初始化
				if client.transformer == nil {
					t.Error("Expected transformer to be initialized")
				}
				if client.sseParser == nil {
					t.Error("Expected sseParser to be initialized")
				}
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// 接口实现测试
// ═══════════════════════════════════════════════════════════════════════════

func TestClient_Close(t *testing.T) {
	client, err := New(&Config{APIKey: "test-key"})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	err = client.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// buildRequest 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestClient_buildRequest(t *testing.T) {
	client, err := New(&Config{
		APIKey: "test-key",
		Model:  "gpt-4o",
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// 基础测试：验证 buildRequest 生成有效请求
	req := client.buildRequest(nil, nil, false)

	if req["model"] != "gpt-4o" {
		t.Errorf("Expected model 'gpt-4o', got %v", req["model"])
	}

	if req["stream"] != false {
		t.Errorf("Expected stream false, got %v", req["stream"])
	}

	if _, ok := req["messages"]; !ok {
		t.Error("Expected messages field in request")
	}
}
