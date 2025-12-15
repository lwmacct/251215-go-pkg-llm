// Package gemini 实现 Google Gemini API 的协议适配器
//
// Gemini API 使用独特的 Content/Parts 格式，与 OpenAI 和 Anthropic 都不同。
//
// # 协议特点
//
//   - 内容格式：Content{Role, Parts[]} 结构
//   - 角色映射：user→user, assistant→model
//   - 系统消息：使用独立的 systemInstruction 字段
//   - 工具格式：functionDeclarations 数组
//   - 认证方式：API Key 作为查询参数 ?key=XXX
//
// # 请求格式示例
//
//	{
//	  "systemInstruction": {"parts": [{"text": "..."}]},
//	  "contents": [
//	    {"role": "user", "parts": [{"text": "..."}]},
//	    {"role": "model", "parts": [{"text": "..."}]}
//	  ],
//	  "tools": [{"functionDeclarations": [...]}],
//	  "generationConfig": {...}
//	}
//
// # Thinking 支持
//
// Gemini 2.5 系列模型支持 thinking/thoughts 能力：
//   - gemini-2.5-pro: 最大 32K thinking tokens
//   - gemini-2.5-flash: 最大 24K thinking tokens
//   - gemini-2.5-flash-lite: 不支持 thinking
//
// 通过 thinkingConfig 启用：
//
//	{
//	  "thinkingConfig": {
//	    "includeThoughts": true,
//	    "thinkingBudget": 32768
//	  }
//	}
package gemini
