package mock

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Complete(t *testing.T) {
	t.Run("default response with config", func(t *testing.T) {
		// New() 无参数时使用默认配置文件
		client := New()
		defer func() { _ = client.Close() }()

		resp, err := client.Complete(context.Background(), nil, nil)
		require.NoError(t, err)
		// 默认配置文件的 default_response
		assert.Equal(t, "抱歉，我不理解您的问题。请指定具体的场景。", resp.Message.Content)
		assert.Equal(t, llm.RoleAssistant, resp.Message.Role)
		assert.Equal(t, "stop", resp.FinishReason)
	})

	t.Run("default response with option", func(t *testing.T) {
		// 使用 Option 时不加载默认配置
		client := New(WithResponse("This is a mock response."))
		defer func() { _ = client.Close() }()

		resp, err := client.Complete(context.Background(), nil, nil)
		require.NoError(t, err)
		assert.Equal(t, "This is a mock response.", resp.Message.Content)
	})

	t.Run("custom response", func(t *testing.T) {
		client := New(WithResponse("Hello, World!"))
		defer func() { _ = client.Close() }()

		resp, err := client.Complete(context.Background(), nil, nil)
		require.NoError(t, err)
		assert.Equal(t, "Hello, World!", resp.Message.Content)
	})

	t.Run("with delay", func(t *testing.T) {
		client := New(WithDelay(50 * time.Millisecond))
		defer func() { _ = client.Close() }()

		start := time.Now()
		resp, err := client.Complete(context.Background(), nil, nil)
		elapsed := time.Since(start)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.GreaterOrEqual(t, elapsed, 50*time.Millisecond)
	})

	t.Run("with error", func(t *testing.T) {
		expectedErr := errors.New("mock error")
		client := New(WithError(expectedErr))
		defer func() { _ = client.Close() }()

		resp, err := client.Complete(context.Background(), nil, nil)
		require.ErrorIs(t, err, expectedErr)
		assert.Nil(t, resp)
	})

	t.Run("context cancellation", func(t *testing.T) {
		client := New(WithDelay(1 * time.Second))
		defer func() { _ = client.Close() }()

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		resp, err := client.Complete(ctx, nil, nil)
		require.ErrorIs(t, err, context.DeadlineExceeded)
		assert.Nil(t, resp)
	})

	t.Run("token usage calculation", func(t *testing.T) {
		client := New(WithResponse("Test"))
		defer func() { _ = client.Close() }()

		messages := []llm.Message{
			{Role: llm.RoleUser, Content: "Hello"},
			{Role: llm.RoleAssistant, Content: "Hi"},
		}

		resp, err := client.Complete(context.Background(), messages, nil)
		require.NoError(t, err)
		assert.NotNil(t, resp.Usage)
		assert.Equal(t, int64(20), resp.Usage.InputTokens) // 2 messages * 10
		assert.Equal(t, int64(1), resp.Usage.OutputTokens) // len("Test")/4
	})
}

func TestClient_WithResponses(t *testing.T) {
	t.Run("multiple responses cycle", func(t *testing.T) {
		client := New(WithResponses("First", "Second", "Third"))
		defer func() { _ = client.Close() }()

		resp1, _ := client.Complete(context.Background(), nil, nil)
		resp2, _ := client.Complete(context.Background(), nil, nil)
		resp3, _ := client.Complete(context.Background(), nil, nil)
		resp4, _ := client.Complete(context.Background(), nil, nil) // cycles back

		assert.Equal(t, "First", resp1.Message.Content)
		assert.Equal(t, "Second", resp2.Message.Content)
		assert.Equal(t, "Third", resp3.Message.Content)
		assert.Equal(t, "First", resp4.Message.Content)
	})
}

func TestClient_WithResponseFunc(t *testing.T) {
	t.Run("dynamic response", func(t *testing.T) {
		client := New(WithResponseFunc(func(msgs []llm.Message, count int) string {
			if len(msgs) == 0 {
				return fmt.Sprintf("Empty input, call #%d", count)
			}
			return fmt.Sprintf("Got: %s, call #%d", msgs[len(msgs)-1].Content, count)
		}))
		defer func() { _ = client.Close() }()

		messages := []llm.Message{{Role: llm.RoleUser, Content: "Hello"}}

		resp1, _ := client.Complete(context.Background(), messages, nil)
		resp2, _ := client.Complete(context.Background(), messages, nil)

		assert.Equal(t, "Got: Hello, call #1", resp1.Message.Content)
		assert.Equal(t, "Got: Hello, call #2", resp2.Message.Content)
	})
}

func TestClient_CallRecording(t *testing.T) {
	t.Run("records calls", func(t *testing.T) {
		client := New(WithResponse("OK"))
		defer func() { _ = client.Close() }()

		messages := []llm.Message{{Role: llm.RoleUser, Content: "Test"}}

		_, _ = client.Complete(context.Background(), messages, nil)
		_, _ = client.Complete(context.Background(), nil, nil)

		assert.Equal(t, 2, client.CallCount())

		calls := client.Calls()
		assert.Len(t, calls, 2)
		assert.Len(t, calls[0].Messages, 1)
		assert.Empty(t, calls[1].Messages)

		lastCall := client.LastCall()
		assert.NotNil(t, lastCall)
		assert.Empty(t, lastCall.Messages)
	})

	t.Run("reset clears records", func(t *testing.T) {
		client := New(WithResponse("OK"))
		defer func() { _ = client.Close() }()

		_, _ = client.Complete(context.Background(), nil, nil)
		_, _ = client.Complete(context.Background(), nil, nil)

		assert.Equal(t, 2, client.CallCount())

		client.Reset()

		assert.Equal(t, 0, client.CallCount())
		assert.Empty(t, client.Calls())
		assert.Nil(t, client.LastCall())
	})
}

func TestClient_Stream(t *testing.T) {
	t.Run("stream response", func(t *testing.T) {
		client := New(WithResponse("Hi"))
		defer func() { _ = client.Close() }()

		stream, err := client.Stream(context.Background(), nil, nil)
		require.NoError(t, err)

		var text string
		var done bool
		var textSb186 strings.Builder
		for chunk := range stream {
			if chunk.Type == "text" {
				textSb186.WriteString(chunk.TextDelta)
			}
			if chunk.Type == "done" {
				done = true
				assert.Equal(t, "stop", chunk.FinishReason)
			}
		}
		text += textSb186.String()

		assert.Equal(t, "Hi", text)
		assert.True(t, done)
	})

	t.Run("stream with error", func(t *testing.T) {
		expectedErr := errors.New("stream error")
		client := New(WithError(expectedErr))
		defer func() { _ = client.Close() }()

		stream, err := client.Stream(context.Background(), nil, nil)
		require.ErrorIs(t, err, expectedErr)
		assert.Nil(t, stream)
	})

	t.Run("stream context cancellation", func(t *testing.T) {
		client := New(WithResponse("Long response text"), WithDelay(1*time.Second))
		defer func() { _ = client.Close() }()

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		stream, err := client.Stream(ctx, nil, nil)
		require.NoError(t, err)

		var count int
		for range stream {
			count++
		}
		// 由于延迟，应该没有收到任何内容
		assert.Equal(t, 0, count)
	})

	t.Run("stream records call", func(t *testing.T) {
		client := New(WithResponse("OK"))
		defer func() { _ = client.Close() }()

		messages := []llm.Message{{Role: llm.RoleUser, Content: "Test"}}
		stream, _ := client.Stream(context.Background(), messages, nil)

		// Drain the stream
		for range stream {
		}

		assert.Equal(t, 1, client.CallCount())
		lastCall := client.LastCall()
		assert.NotNil(t, lastCall)
		assert.Len(t, lastCall.Messages, 1)
	})
}

func TestClient_SetResponse(t *testing.T) {
	client := New(WithResponse("Initial"))
	defer func() { _ = client.Close() }()

	resp, err := client.Complete(context.Background(), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "Initial", resp.Message.Content)

	client.SetResponse("Updated")

	resp, err = client.Complete(context.Background(), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "Updated", resp.Message.Content)
}

func TestClient_SetError(t *testing.T) {
	client := New()
	defer func() { _ = client.Close() }()

	// 初始无错误
	resp, err := client.Complete(context.Background(), nil, nil)
	require.NoError(t, err)
	assert.NotNil(t, resp)

	// 设置错误
	expectedErr := errors.New("dynamic error")
	client.SetError(expectedErr)

	resp, err = client.Complete(context.Background(), nil, nil)
	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, resp)

	// 清除错误
	client.SetError(nil)

	resp, err = client.Complete(context.Background(), nil, nil)
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestClient_Concurrent(t *testing.T) {
	client := New(WithResponse("Concurrent"))
	defer func() { _ = client.Close() }()

	const goroutines = 10
	done := make(chan bool, goroutines)

	for range goroutines {
		go func() {
			resp, err := client.Complete(context.Background(), nil, nil)
			assert.NoError(t, err)
			assert.Equal(t, "Concurrent", resp.Message.Content)
			done <- true
		}()
	}

	for range goroutines {
		<-done
	}

	// All calls should be recorded
	assert.Equal(t, goroutines, client.CallCount())
}
