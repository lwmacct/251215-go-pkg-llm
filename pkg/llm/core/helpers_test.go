package core

import (
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════
// GetInt64 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestGetInt64(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want int64
	}{
		{
			name: "float64 转换",
			val:  float64(100.5),
			want: 100,
		},
		{
			name: "int 转换",
			val:  int(42),
			want: 42,
		},
		{
			name: "int64 直接返回",
			val:  int64(99),
			want: 99,
		},
		{
			name: "nil 返回 0",
			val:  nil,
			want: 0,
		},
		{
			name: "string 返回 0",
			val:  "123",
			want: 0,
		},
		{
			name: "bool 返回 0",
			val:  true,
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetInt64(tt.val)
			if got != tt.want {
				t.Errorf("GetInt64() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// GetFloat64 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestGetFloat64(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want float64
	}{
		{
			name: "float64 直接返回",
			val:  float64(3.14),
			want: 3.14,
		},
		{
			name: "int 转换",
			val:  int(42),
			want: 42.0,
		},
		{
			name: "int64 转换",
			val:  int64(100),
			want: 100.0,
		},
		{
			name: "nil 返回 0.0",
			val:  nil,
			want: 0.0,
		},
		{
			name: "string 返回 0.0",
			val:  "3.14",
			want: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetFloat64(tt.val)
			if got != tt.want {
				t.Errorf("GetFloat64() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// GetString 测试
// ═══════════════════════════════════════════════════════════════════════════

func TestGetString(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want string
	}{
		{
			name: "string 直接返回",
			val:  "hello",
			want: "hello",
		},
		{
			name: "空字符串",
			val:  "",
			want: "",
		},
		{
			name: "nil 返回空字符串",
			val:  nil,
			want: "",
		},
		{
			name: "int 返回空字符串",
			val:  42,
			want: "",
		},
		{
			name: "float64 返回空字符串",
			val:  3.14,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetString(tt.val)
			if got != tt.want {
				t.Errorf("GetString() = %v, want %v", got, tt.want)
			}
		})
	}
}
