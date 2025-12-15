package localmock

import (
	"context"
	"sync"
	"time"

	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
)

// CallRecord 记录一次调用的详情
type CallRecord struct {
	Messages []llm.Message
	Options  *llm.Options
	Time     time.Time
}

// Client Mock LLM Provider
type Client struct {
	mu              sync.RWMutex
	configPath      string                    // 配置文件路径
	response        string                    // 默认响应
	responses       []string                  // 响应队列（依次返回）
	respIdx         int                       // 当前响应索引
	respFunc        ResponseFunc              // 动态响应函数
	msgFunc         MessageResponseFunc       // 完整消息响应函数（支持工具调用）
	delay           time.Duration             // 响应延迟
	err             error                     // 返回错误
	calls           []CallRecord              // 调用记录
	counter         int                       // 调用计数
	scenarios       map[string]*scenarioState // 场景状态（通过 name 索引）
	currentScenario string                    // 当前使用的场景名称
}

// ResponseFunc 动态响应函数类型
// 接收消息列表和调用次数，返回响应文本
type ResponseFunc func(messages []llm.Message, callCount int) string

// MessageResponseFunc 完整消息响应函数类型
// 接收消息列表和调用次数，返回完整的 Message（可包含 ToolCalls）
type MessageResponseFunc func(messages []llm.Message, callCount int) llm.Message

// New 创建 Mock Client
//
// 可选参数:
//   - 无参数: 使用内嵌的示例配置
//   - 一个字符串参数: 使用指定的配置文件路径
//   - Option 类型参数: 使用 Option 函数配置
//
// 使用示例:
//
//	client := localmock.New()                           // 使用内嵌示例配置
//	client := localmock.New("custom/config.yaml")       // 使用指定配置文件
//	client := localmock.New(localmock.WithDelay(100ms)) // 使用 Option
func New(args ...any) *Client {
	c := &Client{
		response: "This is a mock response.",
		calls:    make([]CallRecord, 0),
	}

	// 解析参数
	var configPath string
	var opts []Option

	for _, arg := range args {
		switch v := arg.(type) {
		case string:
			configPath = v
		case Option:
			opts = append(opts, v)
		}
	}

	// 如果提供了配置路径，使用它；否则如果没有 Option，使用默认配置
	if configPath != "" {
		c.configPath = configPath
		cfg, err := LoadConfigFile(configPath)
		if err != nil {
			c.err = err
		} else {
			applyConfig(c, cfg)
		}
	} else if len(opts) == 0 {
		// 无任何参数时使用内嵌示例配置
		cfg, err := LoadExampleConfig()
		if err != nil {
			// 如果加载失败，使用默认响应（不应该发生）
			c.err = err
		} else {
			applyConfig(c, cfg)
		}
	}

	// 应用 Option
	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Option 配置选项函数
type Option func(*Client)

// WithResponse 设置预设响应文本
func WithResponse(text string) Option {
	return func(c *Client) {
		c.response = text
	}
}

// WithResponses 设置响应队列（依次返回，用完后循环）
func WithResponses(texts ...string) Option {
	return func(c *Client) {
		c.responses = texts
	}
}

// WithResponseFunc 设置动态响应函数
func WithResponseFunc(fn ResponseFunc) Option {
	return func(c *Client) {
		c.respFunc = fn
	}
}

// WithMessageFunc 设置完整消息响应函数（支持工具调用）
func WithMessageFunc(fn MessageResponseFunc) Option {
	return func(c *Client) {
		c.msgFunc = fn
	}
}

// WithDelay 设置响应延迟
func WithDelay(d time.Duration) Option {
	return func(c *Client) {
		c.delay = d
	}
}

// WithError 设置返回错误
func WithError(err error) Option {
	return func(c *Client) {
		c.err = err
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// 场景管理方法
// ═══════════════════════════════════════════════════════════════════════════

// UseScenario 设置当前使用的场景（通过名称）
//
// 设置后，Complete 方法会使用该场景的配置返回响应
// 每次调用 Complete 会自动推进到下一轮
func (c *Client) UseScenario(name string) *Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.currentScenario = name
	return c
}

// ResetScenario 重置指定场景的轮次到起始位置
func (c *Client) ResetScenario(name string) *Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	if s, ok := c.scenarios[name]; ok {
		s.turnIdx = 0
	}
	return c
}

// ResetAllScenarios 重置所有场景的轮次
func (c *Client) ResetAllScenarios() *Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, s := range c.scenarios {
		s.turnIdx = 0
	}
	return c
}

// GetScenarioNames 获取所有可用的场景名称
func (c *Client) GetScenarioNames() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	names := make([]string, 0, len(c.scenarios))
	for name := range c.scenarios {
		names = append(names, name)
	}
	return names
}

// GetCurrentScenario 获取当前场景名称
func (c *Client) GetCurrentScenario() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.currentScenario
}

// GetScenarioTurnIndex 获取指定场景的当前轮次索引
func (c *Client) GetScenarioTurnIndex(name string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if s, ok := c.scenarios[name]; ok {
		return s.turnIdx
	}
	return -1
}

// GetScenarioUserInputs 获取指定场景定义的所有用户输入
// 返回场景中每个轮次的 User 字段值，便于编写测试
func (c *Client) GetScenarioUserInputs(name string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	s, ok := c.scenarios[name]
	if !ok {
		return nil
	}
	inputs := make([]string, 0, len(s.scenario.Turns))
	for _, turn := range s.scenario.Turns {
		if turn.User != "" {
			inputs = append(inputs, turn.User)
		}
	}
	return inputs
}

// getScenarioResponse 获取场景响应（内部方法，需要在锁内调用）
func (c *Client) getScenarioResponse(messages []llm.Message) *llm.Message {
	if c.currentScenario == "" {
		return nil
	}

	s, ok := c.scenarios[c.currentScenario]
	if !ok {
		return nil
	}

	// 构建响应
	data := createTemplateData(messages)
	msg := s.buildTurnResponse(messages, data)

	// 推进轮次
	s.turnIdx++

	return &msg
}

// getResponse 获取当前响应（内部方法，需要在锁内调用）
func (c *Client) getResponse(messages []llm.Message) string {
	// 优先使用动态响应函数
	if c.respFunc != nil {
		return c.respFunc(messages, c.counter)
	}

	// 其次使用响应队列
	if len(c.responses) > 0 {
		resp := c.responses[c.respIdx%len(c.responses)]
		c.respIdx++
		return resp
	}

	// 最后使用默认响应
	return c.response
}

// getMessage 获取完整消息响应（内部方法，需要在锁内调用）
// 如果设置了 msgFunc 则返回完整消息，否则返回 nil
func (c *Client) getMessage(messages []llm.Message) *llm.Message {
	if c.msgFunc != nil {
		msg := c.msgFunc(messages, c.counter)
		return &msg
	}
	return nil
}

// Complete 同步完成
func (c *Client) Complete(ctx context.Context, messages []llm.Message, opts *llm.Options) (*llm.Response, error) {
	c.mu.Lock()
	c.counter++
	delay := c.delay
	err := c.err

	// 记录调用
	c.calls = append(c.calls, CallRecord{
		Messages: messages,
		Options:  opts,
		Time:     time.Now(),
	})

	// 优先使用场景响应
	var msgResp *llm.Message
	if c.currentScenario != "" {
		msgResp = c.getScenarioResponse(messages)
	}

	// 其次使用完整消息响应函数
	if msgResp == nil {
		msgResp = c.getMessage(messages)
	}

	// 最后使用简单响应
	var response string
	if msgResp == nil {
		response = c.getResponse(messages)
	}
	c.mu.Unlock()

	// 模拟延迟
	if delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// 模拟错误
	if err != nil {
		return nil, err
	}

	// 如果有完整消息响应，使用它
	if msgResp != nil {
		msgResp.Role = llm.RoleAssistant
		finishReason := "stop"
		// 检查是否包含工具调用
		for _, block := range msgResp.ContentBlocks {
			if _, ok := block.(*llm.ToolCall); ok {
				finishReason = "tool_calls"
				break
			}
		}
		return &llm.Response{
			Message:      *msgResp,
			FinishReason: finishReason,
			Usage: &llm.TokenUsage{
				InputTokens:  int64(len(messages) * 10),
				OutputTokens: 20,
				TotalTokens:  int64(len(messages)*10 + 20),
			},
		}, nil
	}

	// 返回预设响应
	return &llm.Response{
		Message: llm.Message{
			Role:    llm.RoleAssistant,
			Content: response,
		},
		FinishReason: "stop",
		Usage: &llm.TokenUsage{
			InputTokens:  int64(len(messages) * 10),
			OutputTokens: int64(len(response) / 4),
			TotalTokens:  int64(len(messages)*10 + len(response)/4),
		},
	}, nil
}

// Stream 流式完成
func (c *Client) Stream(ctx context.Context, messages []llm.Message, opts *llm.Options) (<-chan *llm.Event, error) {
	c.mu.Lock()
	c.counter++
	delay := c.delay
	err := c.err

	// 记录调用
	c.calls = append(c.calls, CallRecord{
		Messages: messages,
		Options:  opts,
		Time:     time.Now(),
	})

	// 获取响应
	response := c.getResponse(messages)
	c.mu.Unlock()

	// 立即返回错误
	if err != nil {
		return nil, err
	}

	chunks := make(chan *llm.Event, len(response)+1)

	go func() {
		defer close(chunks)

		// 模拟延迟（流式首包延迟）
		if delay > 0 {
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return
			}
		}

		// 逐字符流式返回
		for _, ch := range response {
			select {
			case <-ctx.Done():
				return
			case chunks <- &llm.Event{
				Type:      "text",
				TextDelta: string(ch),
			}:
			}
		}

		// 发送完成信号
		chunks <- &llm.Event{
			Type:         "done",
			FinishReason: "stop",
		}
	}()

	return chunks, nil
}

// Close 关闭连接
func (c *Client) Close() error {
	return nil
}

// SetResponse 动态修改响应（线程安全）
func (c *Client) SetResponse(text string) {
	c.mu.Lock()
	c.response = text
	c.mu.Unlock()
}

// SetError 动态修改错误（线程安全）
func (c *Client) SetError(err error) {
	c.mu.Lock()
	c.err = err
	c.mu.Unlock()
}

// Calls 返回所有调用记录
func (c *Client) Calls() []CallRecord {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]CallRecord, len(c.calls))
	copy(result, c.calls)
	return result
}

// CallCount 返回调用次数
func (c *Client) CallCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.counter
}

// LastCall 返回最后一次调用记录
func (c *Client) LastCall() *CallRecord {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.calls) == 0 {
		return nil
	}
	call := c.calls[len(c.calls)-1]
	return &call
}

// Reset 重置调用记录和计数器
func (c *Client) Reset() {
	c.mu.Lock()
	c.calls = make([]CallRecord, 0)
	c.counter = 0
	c.respIdx = 0
	c.mu.Unlock()
}

// ═══════════════════════════════════════════════════════════════════════════
// 调试辅助方法
// ═══════════════════════════════════════════════════════════════════════════

// GetLastInput 获取最后一次调用的用户输入消息
// 返回最后一条用户消息的内容，便于调试
func (c *Client) GetLastInput() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.calls) == 0 {
		return ""
	}

	lastCall := c.calls[len(c.calls)-1]
	for i := len(lastCall.Messages) - 1; i >= 0; i-- {
		if lastCall.Messages[i].Role == llm.RoleUser {
			return getMessageContent(lastCall.Messages[i])
		}
	}
	return ""
}

// GetLastOutput 获取最后一次调用的助手响应内容
// 需要先调用 Complete 方法，此方法返回该次调用的响应
// 如果需要获取响应，建议直接使用 Complete 的返回值
func (c *Client) GetLastOutput() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.calls) == 0 {
		return ""
	}

	// 返回预期的响应（根据当前状态模拟）
	lastCall := c.calls[len(c.calls)-1]

	// 优先使用 msgFunc
	if c.msgFunc != nil {
		msg := c.msgFunc(lastCall.Messages, len(c.calls))
		return getMessageContent(msg)
	}

	// 其次使用 respFunc
	if c.respFunc != nil {
		return c.respFunc(lastCall.Messages, len(c.calls))
	}

	// 使用响应队列（当前索引-1，因为已经递增过）
	if len(c.responses) > 0 {
		idx := (c.respIdx - 1) % len(c.responses)
		if idx < 0 {
			idx = 0
		}
		return c.responses[idx]
	}

	return c.response
}

// GetConfigPath 获取当前使用的配置文件路径
func (c *Client) GetConfigPath() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.configPath
}

// GetAllInputs 获取所有调用的用户输入
func (c *Client) GetAllInputs() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var inputs []string
	for _, call := range c.calls {
		for _, msg := range call.Messages {
			if msg.Role == llm.RoleUser {
				inputs = append(inputs, getMessageContent(msg))
			}
		}
	}
	return inputs
}

// 编译时接口检查
var _ llm.Provider = (*Client)(nil)
