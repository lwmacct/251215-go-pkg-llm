package mock

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
	"gopkg.in/yaml.v3"
)

//go:embed examples/unified.yaml
var exampleConfigYAML []byte

// Config 配置文件结构
type Config struct {
	// DefaultResponse 默认响应（当没有指定场景时使用）
	DefaultResponse string `yaml:"default_response" json:"default_response"`

	// Scenarios 场景列表（通过 name 标识，直接指定使用）
	Scenarios []Scenario `yaml:"scenarios" json:"scenarios"`

	// Delay 响应延迟（如 "100ms", "1s"）
	Delay string `yaml:"delay" json:"delay"`

	// SimulateError 模拟错误消息
	SimulateError string `yaml:"simulate_error" json:"simulate_error"`
}

// Scenario 场景（通过 name 标识，支持多轮对话）
type Scenario struct {
	// Name 场景名称（必需，用于指定场景）
	Name string `yaml:"name" json:"name"`

	// Turns 对话轮次列表
	Turns []Turn `yaml:"turns" json:"turns"`
}

// Turn 单轮对话
type Turn struct {
	// User 用户消息（可选，用于文档说明）
	User string `yaml:"user,omitempty" json:"user,omitempty"`

	// Assistant 助手响应（支持模板语法）
	Assistant string `yaml:"assistant,omitempty" json:"assistant,omitempty"`

	// Tools 工具调用列表（可选）
	Tools []ToolCall `yaml:"tools,omitempty" json:"tools,omitempty"`
}

// ToolCall 工具调用
type ToolCall struct {
	// Name 工具名称
	Name string `yaml:"name" json:"name"`

	// Input 工具输入参数（支持模板语法）
	Input map[string]any `yaml:"input,omitempty" json:"input,omitempty"`
}

// LoadConfigFile 从文件加载配置
func LoadConfigFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(path))
	return LoadConfigFromBytes(data, ext)
}

// LoadConfigFromBytes 从字节数据加载配置
func LoadConfigFromBytes(data []byte, format string) (*Config, error) {
	cfg := &Config{}

	// 规范化格式字符串（支持 ".yaml" 或 "yaml"）
	format = strings.TrimPrefix(strings.ToLower(format), ".")

	switch format {
	case "yaml", "yml":
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parse YAML: %w", err)
		}
	case "json":
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parse JSON: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported format: %s (expected yaml, yml, or json)", format)
	}

	return cfg, nil
}

// LoadExampleConfig 加载内嵌的示例配置
func LoadExampleConfig() (*Config, error) {
	return LoadConfigFromBytes(exampleConfigYAML, "yaml")
}

// WithConfigFile 从配置文件加载设置
func WithConfigFile(path string) Option {
	return func(c *Client) {
		cfg, err := LoadConfigFile(path)
		if err != nil {
			// 将错误存储到客户端，在首次调用时返回
			c.err = fmt.Errorf("load config file: %w", err)
			return
		}

		// 应用配置
		applyConfig(c, cfg)
	}
}

// WithConfig 从配置对象加载设置
func WithConfig(cfg *Config) Option {
	return func(c *Client) {
		if cfg == nil {
			return
		}
		applyConfig(c, cfg)
	}
}

// applyConfig 应用配置到客户端
func applyConfig(c *Client, cfg *Config) {
	// 设置默认响应
	if cfg.DefaultResponse != "" {
		c.response = cfg.DefaultResponse
	}

	// 加载场景（通过 name 索引）
	if len(cfg.Scenarios) > 0 {
		c.scenarios = make(map[string]*scenarioState)
		for _, s := range cfg.Scenarios {
			if s.Name != "" {
				c.scenarios[s.Name] = &scenarioState{
					scenario: s,
					turnIdx:  0,
				}
			}
		}
	}

	// 设置延迟
	if cfg.Delay != "" {
		if d, err := time.ParseDuration(cfg.Delay); err == nil {
			c.delay = d
		}
	}

	// 设置错误
	if cfg.SimulateError != "" {
		c.err = fmt.Errorf("%s", cfg.SimulateError)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// 场景状态管理
// ═══════════════════════════════════════════════════════════════════════════

// scenarioState 场景状态
type scenarioState struct {
	scenario Scenario
	turnIdx  int // 当前轮次索引
}

// buildTurnResponse 构建当前轮次的响应消息
func (s *scenarioState) buildTurnResponse(messages []llm.Message, data map[string]string) llm.Message {
	if s.turnIdx >= len(s.scenario.Turns) {
		return llm.Message{
			Role:    llm.RoleAssistant,
			Content: "[场景已结束]",
		}
	}

	turn := s.scenario.Turns[s.turnIdx]
	msg := llm.Message{Role: llm.RoleAssistant}

	// 处理文本响应（支持模板）
	if turn.Assistant != "" {
		rendered, err := renderTemplateWithData(turn.Assistant, data)
		if err != nil {
			rendered = turn.Assistant
		}
		msg.Content = rendered
	}

	// 处理工具调用
	if len(turn.Tools) > 0 {
		var blocks []llm.ContentBlock
		if msg.Content != "" {
			blocks = append(blocks, &llm.TextBlock{Text: msg.Content})
		}
		for _, tool := range turn.Tools {
			renderedInput := renderToolInput(tool.Input, messages)
			blocks = append(blocks, &llm.ToolCall{
				ID:    generateToolID(tool.Name),
				Name:  tool.Name,
				Input: renderedInput,
			})
		}
		msg.ContentBlocks = blocks
		msg.Content = ""
	}

	return msg
}

// ═══════════════════════════════════════════════════════════════════════════
// 模板渲染 (对齐 agent/internal/config/template.go 设计)
// ═══════════════════════════════════════════════════════════════════════════

// templateFuncs 模板函数映射
var templateFuncs = template.FuncMap{
	"env":      envFunc,
	"default":  defaultFunc,
	"coalesce": coalesceFunc,
}

// envFunc 获取环境变量
func envFunc(key string, defaultVal ...string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	if len(defaultVal) > 0 {
		return defaultVal[0]
	}
	return ""
}

// defaultFunc 提供默认值
func defaultFunc(defaultVal, value any) any {
	if value == nil {
		return defaultVal
	}
	if str, ok := value.(string); ok && str == "" {
		return defaultVal
	}
	return value
}

// coalesceFunc 返回第一个非空值
func coalesceFunc(values ...any) any {
	for _, v := range values {
		if v == nil {
			continue
		}
		if str, ok := v.(string); ok && str == "" {
			continue
		}
		return v
	}
	return nil
}

// renderToolInput 渲染工具输入参数
func renderToolInput(input map[string]any, messages []llm.Message) map[string]any {
	result := make(map[string]any)
	data := createTemplateData(messages)

	for key, val := range input {
		if strVal, ok := val.(string); ok {
			if rendered, err := renderTemplateWithData(strVal, data); err == nil {
				result[key] = rendered
			} else {
				result[key] = strVal
			}
		} else {
			result[key] = val
		}
	}

	return result
}

// renderTemplateWithData 使用指定数据渲染模板
func renderTemplateWithData(text string, data map[string]string) (string, error) {
	tmpl, err := template.New("param").Funcs(templateFuncs).Parse(text)
	if err != nil {
		return text, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return text, err
	}

	return buf.String(), nil
}

// createTemplateData 创建模板数据
func createTemplateData(messages []llm.Message) map[string]string {
	vars := make(map[string]string)
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			vars[parts[0]] = parts[1]
		}
	}

	if len(messages) > 0 {
		lastMsg := messages[len(messages)-1]
		vars["LAST_USER_MESSAGE"] = getMessageContent(lastMsg)
	}

	return vars
}

// generateToolID 生成工具调用 ID
func generateToolID(toolName string) string {
	return fmt.Sprintf("call_%s_%d", toolName, time.Now().UnixNano())
}

// ═══════════════════════════════════════════════════════════════════════════
// 辅助函数
// ═══════════════════════════════════════════════════════════════════════════

// getMessageContent 提取消息内容
func getMessageContent(msg llm.Message) string {
	if msg.Content != "" {
		return msg.Content
	}

	// 优先提取文本块
	for _, block := range msg.ContentBlocks {
		if tb, ok := block.(*llm.TextBlock); ok {
			return tb.Text
		}
	}

	// 如果没有文本块，尝试提取工具结果块（用于工具调用场景）
	for _, block := range msg.ContentBlocks {
		if trb, ok := block.(*llm.ToolResultBlock); ok {
			return trb.Content
		}
	}

	return ""
}
