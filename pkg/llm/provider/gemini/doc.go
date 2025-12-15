// Package gemini 实现 Google Gemini LLM Provider
//
// 支持 Gemini API 和 Vertex AI 两种后端。
//
// # 基础使用
//
//	provider, err := gemini.New(&gemini.Config{
//	    APIKey: "your-api-key",
//	    Model:  "gemini-1.5-flash",
//	})
//
//	resp, err := provider.Complete(ctx, messages, opts)
//
// # Vertex AI 后端
//
//	provider, err := gemini.New(&gemini.Config{
//	    Model:          "gemini-1.5-flash",
//	    VertexProject:  "your-project",
//	    VertexLocation: "us-central1",
//	    VertexCredFile: "/path/to/credentials.json",
//	})
//
// # Thinking 模式
//
// Gemini 2.5 系列支持 thinking 能力：
//
//	provider, err := gemini.New(&gemini.Config{
//	    Model:          "gemini-2.5-flash",
//	    EnableThinking: true,
//	    ThinkingBudget: 24576,  // 最大 24K tokens
//	})
//
// # 支持的模型
//
//   - gemini-2.5-pro: 最强模型，32K thinking tokens
//   - gemini-2.5-flash: 快速模型，24K thinking tokens
//   - gemini-2.5-flash-lite: 轻量模型，不支持 thinking
//   - gemini-2.0-flash: 旧版快速模型
//   - gemini-1.5-pro/flash: 旧版模型
package gemini
