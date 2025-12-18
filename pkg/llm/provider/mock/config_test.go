package mock

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ═══════════════════════════════════════════════════════════════════════════
// 基础配置加载测试
// ═══════════════════════════════════════════════════════════════════════════

func TestLoadConfigFile_YAML(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.yaml")
	content := `
default_response: "默认响应"
scenarios:
  - name: "test_scene"
    turns:
      - user: "test"
        assistant: "response"
delay: "100ms"
`
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)

	cfg, err := LoadConfigFile(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, "默认响应", cfg.DefaultResponse)
	assert.Len(t, cfg.Scenarios, 1)
	assert.Equal(t, "test_scene", cfg.Scenarios[0].Name)
	assert.Equal(t, "100ms", cfg.Delay)
}

func TestLoadConfigFile_JSON(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.json")
	content := `{
		"default_response": "default",
		"scenarios": [
			{
				"name": "hello_scene",
				"turns": [{"user": "hello", "assistant": "hi"}]
			}
		],
		"delay": "50ms"
	}`
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)

	cfg, err := LoadConfigFile(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, "default", cfg.DefaultResponse)
	assert.Len(t, cfg.Scenarios, 1)
}

func TestLoadConfigFile_InvalidFormat(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.txt")
	err := os.WriteFile(tmpFile, []byte("test"), 0644)
	require.NoError(t, err)

	_, err = LoadConfigFile(tmpFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported format")
}

func TestLoadConfigFile_NotFound(t *testing.T) {
	_, err := LoadConfigFile("/nonexistent/config.yaml")
	assert.Error(t, err)
}

// ═══════════════════════════════════════════════════════════════════════════
// 场景指定测试
// ═══════════════════════════════════════════════════════════════════════════

func TestScenario_UseScenario(t *testing.T) {
	cfg := &Config{
		DefaultResponse: "默认响应",
		Scenarios: []Scenario{
			{
				Name: "greeting",
				Turns: []Turn{
					{User: "hello", Assistant: "Hi there!"},
				},
			},
			{
				Name: "farewell",
				Turns: []Turn{
					{User: "bye", Assistant: "Goodbye!"},
				},
			},
		},
	}

	client := New(WithConfig(cfg))
	ctx := context.Background()

	// 未指定场景时使用默认响应
	resp, err := client.Complete(ctx, []llm.Message{
		{Role: llm.RoleUser, Content: "hello"},
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, "默认响应", resp.Message.Content)

	// 指定 greeting 场景
	client.UseScenario("greeting")
	resp, err = client.Complete(ctx, []llm.Message{
		{Role: llm.RoleUser, Content: "hello"},
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, "Hi there!", resp.Message.Content)

	// 切换到 farewell 场景
	client.UseScenario("farewell")
	resp, err = client.Complete(ctx, []llm.Message{
		{Role: llm.RoleUser, Content: "bye"},
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, "Goodbye!", resp.Message.Content)
}

func TestScenario_MultiTurn(t *testing.T) {
	cfg := &Config{
		Scenarios: []Scenario{
			{
				Name: "booking",
				Turns: []Turn{
					{User: "订餐", Assistant: "几位？"},
					{User: "3位", Assistant: "什么时间？"},
					{User: "7点", Assistant: "预订完成！"},
				},
			},
		},
	}

	client := New(WithConfig(cfg))
	ctx := context.Background()

	client.UseScenario("booking")

	// 第一轮
	resp1, err := client.Complete(ctx, []llm.Message{
		{Role: llm.RoleUser, Content: "我想订餐"},
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, "几位？", resp1.Message.Content)

	// 第二轮 - 自动推进到下一轮
	resp2, err := client.Complete(ctx, []llm.Message{
		{Role: llm.RoleUser, Content: "3位"},
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, "什么时间？", resp2.Message.Content)

	// 第三轮
	resp3, err := client.Complete(ctx, []llm.Message{
		{Role: llm.RoleUser, Content: "晚上7点"},
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, "预订完成！", resp3.Message.Content)

	// 超出轮次
	resp4, err := client.Complete(ctx, []llm.Message{
		{Role: llm.RoleUser, Content: "还有吗"},
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, "[场景已结束]", resp4.Message.Content)
}

func TestScenario_ResetScenario(t *testing.T) {
	cfg := &Config{
		Scenarios: []Scenario{
			{
				Name: "multi",
				Turns: []Turn{
					{User: "1", Assistant: "第一轮"},
					{User: "2", Assistant: "第二轮"},
				},
			},
		},
	}

	client := New(WithConfig(cfg))
	ctx := context.Background()

	client.UseScenario("multi")

	// 执行第一轮
	resp1, _ := client.Complete(ctx, nil, nil)
	assert.Equal(t, "第一轮", resp1.Message.Content)

	// 重置场景
	client.ResetScenario("multi")

	// 再次执行应该回到第一轮
	resp2, _ := client.Complete(ctx, nil, nil)
	assert.Equal(t, "第一轮", resp2.Message.Content)
}

// ═══════════════════════════════════════════════════════════════════════════
// 模板语法测试
// ═══════════════════════════════════════════════════════════════════════════

func TestScenario_Template(t *testing.T) {
	_ = os.Setenv("BOT_NAME", "测试机器人")
	_ = os.Setenv("VERSION", "v1.0")
	defer func() {
		_ = os.Unsetenv("BOT_NAME")
		_ = os.Unsetenv("VERSION")
	}()

	cfg := &Config{
		Scenarios: []Scenario{
			{
				Name: "info",
				Turns: []Turn{
					{
						User:      "info",
						Assistant: "我叫{{.BOT_NAME}}，版本{{.VERSION}}",
					},
				},
			},
		},
	}

	client := New(WithConfig(cfg))
	client.UseScenario("info")
	ctx := context.Background()

	resp, err := client.Complete(ctx, []llm.Message{
		{Role: llm.RoleUser, Content: "info"},
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, "我叫测试机器人，版本v1.0", resp.Message.Content)
}

func TestScenario_Template_DefaultValue(t *testing.T) {
	_ = os.Unsetenv("BOT_NAME")

	cfg := &Config{
		Scenarios: []Scenario{
			{
				Name: "name",
				Turns: []Turn{
					{
						User:      "name",
						Assistant: "{{.BOT_NAME | default \"默认机器人\"}}",
					},
				},
			},
		},
	}

	client := New(WithConfig(cfg))
	client.UseScenario("name")
	ctx := context.Background()

	resp, err := client.Complete(ctx, []llm.Message{
		{Role: llm.RoleUser, Content: "name"},
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, "默认机器人", resp.Message.Content)
}

func TestScenario_Template_Coalesce(t *testing.T) {
	_ = os.Unsetenv("VAR1")
	_ = os.Setenv("VAR2", "值2")
	defer func() { _ = os.Unsetenv("VAR2") }()

	cfg := &Config{
		Scenarios: []Scenario{
			{
				Name: "coalesce_test",
				Turns: []Turn{
					{
						User:      "test",
						Assistant: "{{coalesce .VAR1 .VAR2 \"默认\"}}",
					},
				},
			},
		},
	}

	client := New(WithConfig(cfg))
	client.UseScenario("coalesce_test")
	ctx := context.Background()

	resp, err := client.Complete(ctx, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "值2", resp.Message.Content)
}

// ═══════════════════════════════════════════════════════════════════════════
// 工具调用测试
// ═══════════════════════════════════════════════════════════════════════════

func TestScenario_ToolCalls(t *testing.T) {
	cfg := &Config{
		Scenarios: []Scenario{
			{
				Name: "weather",
				Turns: []Turn{
					{
						User:      "weather",
						Assistant: "查询中...",
						Tools: []ToolCall{
							{
								Name: "get_weather",
								Input: map[string]any{
									"city": "Tokyo",
									"unit": "celsius",
								},
							},
						},
					},
				},
			},
		},
	}

	client := New(WithConfig(cfg))
	client.UseScenario("weather")
	ctx := context.Background()

	resp, err := client.Complete(ctx, []llm.Message{
		{Role: llm.RoleUser, Content: "weather"},
	}, nil)
	require.NoError(t, err)

	// 验证响应结构
	require.Len(t, resp.Message.ContentBlocks, 2)

	// 验证文本块
	textBlock, ok := resp.Message.ContentBlocks[0].(*llm.TextBlock)
	require.True(t, ok)
	assert.Equal(t, "查询中...", textBlock.Text)

	// 验证工具调用块
	toolBlock, ok := resp.Message.ContentBlocks[1].(*llm.ToolCall)
	require.True(t, ok)
	assert.Equal(t, "get_weather", toolBlock.Name)
	assert.Equal(t, "Tokyo", toolBlock.Input["city"])
	assert.Equal(t, "celsius", toolBlock.Input["unit"])
	assert.NotEmpty(t, toolBlock.ID)

	// 验证 finish_reason
	assert.Equal(t, "tool_calls", resp.FinishReason)
}

func TestScenario_ToolCalls_WithTemplate(t *testing.T) {
	_ = os.Setenv("CITY", "Beijing")
	_ = os.Setenv("UNIT", "fahrenheit")
	defer func() {
		_ = os.Unsetenv("CITY")
		_ = os.Unsetenv("UNIT")
	}()

	cfg := &Config{
		Scenarios: []Scenario{
			{
				Name: "weather",
				Turns: []Turn{
					{
						User:      "weather",
						Assistant: "查询{{.CITY}}天气...",
						Tools: []ToolCall{
							{
								Name: "get_weather",
								Input: map[string]any{
									"city": "{{.CITY}}",
									"unit": "{{.UNIT | default \"celsius\"}}",
								},
							},
						},
					},
				},
			},
		},
	}

	client := New(WithConfig(cfg))
	client.UseScenario("weather")
	ctx := context.Background()

	resp, err := client.Complete(ctx, []llm.Message{
		{Role: llm.RoleUser, Content: "weather"},
	}, nil)
	require.NoError(t, err)

	// 验证文本模板渲染
	textBlock := resp.Message.ContentBlocks[0].(*llm.TextBlock)
	assert.Equal(t, "查询Beijing天气...", textBlock.Text)

	// 验证工具参数模板渲染
	toolBlock := resp.Message.ContentBlocks[1].(*llm.ToolCall)
	assert.Equal(t, "Beijing", toolBlock.Input["city"])
	assert.Equal(t, "fahrenheit", toolBlock.Input["unit"])
}

func TestScenario_ToolCalls_OnlyTools(t *testing.T) {
	cfg := &Config{
		Scenarios: []Scenario{
			{
				Name: "search",
				Turns: []Turn{
					{
						User: "search",
						Tools: []ToolCall{
							{
								Name: "web_search",
								Input: map[string]any{
									"query": "test",
								},
							},
						},
					},
				},
			},
		},
	}

	client := New(WithConfig(cfg))
	client.UseScenario("search")
	ctx := context.Background()

	resp, err := client.Complete(ctx, []llm.Message{
		{Role: llm.RoleUser, Content: "search"},
	}, nil)
	require.NoError(t, err)

	// 只有工具调用，没有文本
	require.Len(t, resp.Message.ContentBlocks, 1)
	toolBlock, ok := resp.Message.ContentBlocks[0].(*llm.ToolCall)
	require.True(t, ok)
	assert.Equal(t, "web_search", toolBlock.Name)
}

// ═══════════════════════════════════════════════════════════════════════════
// 配置选项测试
// ═══════════════════════════════════════════════════════════════════════════

func TestConfig_Delay(t *testing.T) {
	cfg := &Config{
		DefaultResponse: "响应",
		Delay:           "100ms",
	}

	client := New(WithConfig(cfg))
	ctx := context.Background()

	start := time.Now()
	_, err := client.Complete(ctx, []llm.Message{
		{Role: llm.RoleUser, Content: "test"},
	}, nil)
	duration := time.Since(start)

	require.NoError(t, err)
	assert.GreaterOrEqual(t, duration, 100*time.Millisecond)
}

func TestConfig_SimulateError(t *testing.T) {
	cfg := &Config{
		SimulateError: "模拟错误",
	}

	client := New(WithConfig(cfg))
	ctx := context.Background()

	_, err := client.Complete(ctx, []llm.Message{
		{Role: llm.RoleUser, Content: "test"},
	}, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "模拟错误")
}

func TestWithConfigFile(t *testing.T) {
	client := New(WithConfigFile("examples/unified.yaml"))
	require.NotNil(t, client)

	ctx := context.Background()

	// 使用 greeting 场景
	client.UseScenario("greeting")
	resp, err := client.Complete(ctx, []llm.Message{
		{Role: llm.RoleUser, Content: "你好"},
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, "你好！很高兴见到你！", resp.Message.Content)
}

func TestLoadExampleConfig(t *testing.T) {
	// 测试加载内嵌的示例配置
	cfg, err := LoadExampleConfig()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// 验证基础字段
	assert.Equal(t, "抱歉，我不理解您的问题。请指定具体的场景。", cfg.DefaultResponse)
	assert.NotEmpty(t, cfg.Scenarios)

	// 验证包含预期的场景
	scenarioNames := make(map[string]bool)
	for _, s := range cfg.Scenarios {
		scenarioNames[s.Name] = true
	}
	assert.True(t, scenarioNames["greeting"])
	assert.True(t, scenarioNames["weather"])
	assert.True(t, scenarioNames["agent_single"])
}

func TestLoadConfigFromBytes(t *testing.T) {
	yamlData := []byte(`
default_response: "测试响应"
scenarios:
  - name: "test"
    turns:
      - user: "hello"
        assistant: "world"
delay: "50ms"
`)

	cfg, err := LoadConfigFromBytes(yamlData, "yaml")
	require.NoError(t, err)
	assert.Equal(t, "测试响应", cfg.DefaultResponse)
	assert.Len(t, cfg.Scenarios, 1)
	assert.Equal(t, "test", cfg.Scenarios[0].Name)
	assert.Equal(t, "50ms", cfg.Delay)
}

func TestLoadConfigFromBytes_JSON(t *testing.T) {
	jsonData := []byte(`{
		"default_response": "JSON响应",
		"scenarios": [
			{"name": "json_test", "turns": [{"assistant": "ok"}]}
		]
	}`)

	cfg, err := LoadConfigFromBytes(jsonData, ".json")
	require.NoError(t, err)
	assert.Equal(t, "JSON响应", cfg.DefaultResponse)
	assert.Len(t, cfg.Scenarios, 1)
	assert.Equal(t, "json_test", cfg.Scenarios[0].Name)
}

func TestLoadConfigFromBytes_InvalidFormat(t *testing.T) {
	data := []byte("some data")
	_, err := LoadConfigFromBytes(data, "xml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported format")
}

func TestWithConfigFile_InvalidPath(t *testing.T) {
	client := New(WithConfigFile("/invalid/path.yaml"))
	require.NotNil(t, client)

	ctx := context.Background()
	_, err := client.Complete(ctx, []llm.Message{
		{Role: llm.RoleUser, Content: "test"},
	}, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "load config file")
}

func TestWithConfig_Nil(t *testing.T) {
	client := New(WithConfig(nil))
	require.NotNil(t, client)

	ctx := context.Background()
	resp, err := client.Complete(ctx, []llm.Message{
		{Role: llm.RoleUser, Content: "test"},
	}, nil)

	require.NoError(t, err)
	assert.Equal(t, "This is a mock response.", resp.Message.Content)
}

// ═══════════════════════════════════════════════════════════════════════════
// 场景管理测试
// ═══════════════════════════════════════════════════════════════════════════

func TestGetScenarioNames(t *testing.T) {
	cfg := &Config{
		Scenarios: []Scenario{
			{Name: "scene1", Turns: []Turn{{Assistant: "1"}}},
			{Name: "scene2", Turns: []Turn{{Assistant: "2"}}},
			{Name: "scene3", Turns: []Turn{{Assistant: "3"}}},
		},
	}

	client := New(WithConfig(cfg))
	names := client.GetScenarioNames()

	assert.Len(t, names, 3)
	assert.Contains(t, names, "scene1")
	assert.Contains(t, names, "scene2")
	assert.Contains(t, names, "scene3")
}

func TestGetScenarioTurnIndex(t *testing.T) {
	cfg := &Config{
		Scenarios: []Scenario{
			{
				Name: "multi",
				Turns: []Turn{
					{Assistant: "1"},
					{Assistant: "2"},
				},
			},
		},
	}

	client := New(WithConfig(cfg))
	client.UseScenario("multi")
	ctx := context.Background()

	assert.Equal(t, 0, client.GetScenarioTurnIndex("multi"))

	_, _ = client.Complete(ctx, nil, nil)
	assert.Equal(t, 1, client.GetScenarioTurnIndex("multi"))

	_, _ = client.Complete(ctx, nil, nil)
	assert.Equal(t, 2, client.GetScenarioTurnIndex("multi"))

	// 不存在的场景
	assert.Equal(t, -1, client.GetScenarioTurnIndex("nonexistent"))
}

func TestScenario_MultipleTools(t *testing.T) {
	cfg := &Config{
		Scenarios: []Scenario{
			{
				Name: "multi",
				Turns: []Turn{
					{
						User:      "multi",
						Assistant: "调用多个工具",
						Tools: []ToolCall{
							{Name: "tool1", Input: map[string]any{"p1": "v1"}},
							{Name: "tool2", Input: map[string]any{"p2": "v2"}},
						},
					},
				},
			},
		},
	}

	client := New(WithConfig(cfg))
	client.UseScenario("multi")
	ctx := context.Background()

	resp, err := client.Complete(ctx, []llm.Message{
		{Role: llm.RoleUser, Content: "multi"},
	}, nil)
	require.NoError(t, err)

	// 文本 + 2个工具
	require.Len(t, resp.Message.ContentBlocks, 3)
	assert.Equal(t, "tool1", resp.Message.ContentBlocks[1].(*llm.ToolCall).Name)
	assert.Equal(t, "tool2", resp.Message.ContentBlocks[2].(*llm.ToolCall).Name)
}

// ═══════════════════════════════════════════════════════════════════════════
// 调试辅助方法测试
// ═══════════════════════════════════════════════════════════════════════════

func TestGetLastInput(t *testing.T) {
	client := New(WithResponse("OK"))
	ctx := context.Background()

	// 无调用时返回空
	assert.Equal(t, "", client.GetLastInput())

	// 调用后返回最后的用户输入
	_, _ = client.Complete(ctx, []llm.Message{
		{Role: llm.RoleUser, Content: "Hello"},
	}, nil)
	assert.Equal(t, "Hello", client.GetLastInput())

	_, _ = client.Complete(ctx, []llm.Message{
		{Role: llm.RoleUser, Content: "World"},
	}, nil)
	assert.Equal(t, "World", client.GetLastInput())
}

func TestGetAllInputs(t *testing.T) {
	client := New(WithResponse("OK"))
	ctx := context.Background()

	_, _ = client.Complete(ctx, []llm.Message{
		{Role: llm.RoleUser, Content: "First"},
	}, nil)
	_, _ = client.Complete(ctx, []llm.Message{
		{Role: llm.RoleUser, Content: "Second"},
	}, nil)

	inputs := client.GetAllInputs()
	assert.Len(t, inputs, 2)
	assert.Equal(t, "First", inputs[0])
	assert.Equal(t, "Second", inputs[1])
}

// ═══════════════════════════════════════════════════════════════════════════
// 工具调用完整流程测试
// ═══════════════════════════════════════════════════════════════════════════

func TestScenario_ToolCallWorkflow(t *testing.T) {
	// 模拟完整的工具调用流程：
	// 1. 用户询问 → Assistant 返回工具调用
	// 2. 用户发送 ToolResult → Assistant 返回最终文本
	cfg := &Config{
		Scenarios: []Scenario{
			{
				Name: "weather_flow",
				Turns: []Turn{
					// Turn 1: 返回工具调用
					{
						User: "查天气",
						Tools: []ToolCall{
							{
								Name: "get_weather",
								Input: map[string]any{
									"city": "Beijing",
								},
							},
						},
					},
					// Turn 2: 收到 ToolResult 后返回最终结果
					{
						User:      "[ToolResult]",
						Assistant: "北京今天晴天，气温25°C。",
					},
				},
			},
		},
	}

	client := New(WithConfig(cfg))
	client.UseScenario("weather_flow")
	ctx := context.Background()

	// 第一次调用：用户询问，返回工具调用
	resp1, err := client.Complete(ctx, []llm.Message{
		{Role: llm.RoleUser, Content: "今天天气怎么样？"},
	}, nil)
	require.NoError(t, err)
	require.Len(t, resp1.Message.ContentBlocks, 1)
	toolBlock, ok := resp1.Message.ContentBlocks[0].(*llm.ToolCall)
	require.True(t, ok)
	assert.Equal(t, "get_weather", toolBlock.Name)
	assert.Equal(t, "tool_calls", resp1.FinishReason)

	// 第二次调用：发送 ToolResult，返回最终文本
	resp2, err := client.Complete(ctx, []llm.Message{
		{Role: llm.RoleUser, Content: "今天天气怎么样？"},
		{Role: llm.RoleAssistant, Content: ""}, // 工具调用
		{Role: llm.RoleUser, Content: "ToolResult: 晴天 25°C"},
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, "北京今天晴天，气温25°C。", resp2.Message.Content)
	assert.Equal(t, "stop", resp2.FinishReason)
}

func TestScenario_AgentLoop(t *testing.T) {
	// 模拟 Agent 循环：多次工具调用
	cfg := &Config{
		Scenarios: []Scenario{
			{
				Name: "agent",
				Turns: []Turn{
					// Step 1: 调用工具 A
					{
						User:      "分析文件",
						Assistant: "让我读取文件...",
						Tools: []ToolCall{
							{Name: "read_file", Input: map[string]any{"path": "main.go"}},
						},
					},
					// Step 2: 继续调用工具 B
					{
						User:      "[ToolResult A]",
						Assistant: "正在分析...",
						Tools: []ToolCall{
							{Name: "analyze", Input: map[string]any{"type": "code"}},
						},
					},
					// Step 3: 最终响应
					{
						User:      "[ToolResult B]",
						Assistant: "分析完成！代码质量良好。",
					},
				},
			},
		},
	}

	client := New(WithConfig(cfg))
	client.UseScenario("agent")
	ctx := context.Background()

	// Step 1
	resp1, _ := client.Complete(ctx, nil, nil)
	require.Len(t, resp1.Message.ContentBlocks, 2) // text + tool
	assert.Equal(t, "tool_calls", resp1.FinishReason)

	// Step 2
	resp2, _ := client.Complete(ctx, nil, nil)
	require.Len(t, resp2.Message.ContentBlocks, 2)
	assert.Equal(t, "tool_calls", resp2.FinishReason)

	// Step 3
	resp3, _ := client.Complete(ctx, nil, nil)
	assert.Equal(t, "分析完成！代码质量良好。", resp3.Message.Content)
	assert.Equal(t, "stop", resp3.FinishReason)
}
