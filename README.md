# go-pkg-llm

<!--TOC-->

- [特性](#特性) `:15+8`
- [安装](#安装) `:23+6`
- [快速开始](#快速开始) `:29+28`
- [Provider 类型](#provider-类型) `:57+17`
- [文档](#文档) `:74+3`

<!--TOC-->

Go 语言 LLM 统一抽象层，提供多提供商的统一调用接口。

## 特性

- **统一接口** - 同步调用与流式响应
- **多提供商** - OpenAI、Anthropic、Gemini 等 12 种 Provider
- **工具调用** - 完整的 Function Calling 支持
- **推理模式** - 支持 DeepSeek R1、Claude Extended Thinking
- **环境变量** - 自动探测配置

## 安装

```bash
go get github.com/lwmacct/251215-go-pkg-llm
```

## 快速开始

```go
package main

import (
    "context"
    "fmt"

    "github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
    "github.com/lwmacct/251215-go-pkg-llm/pkg/llm/provider"
)

func main() {
    p, _ := provider.New(&provider.Config{
        Type:   llm.ProviderTypeOpenAI,
        APIKey: "sk-xxx",
    })
    defer p.Close()

    resp, _ := p.Complete(context.Background(), []llm.Message{
        {Role: llm.RoleUser, Content: "Hello!"},
    }, nil)

    fmt.Println(resp.Message.Content)
}
```

## Provider 类型

| 类型         | 说明                |
| ------------ | ------------------- |
| `openai`     | OpenAI 及兼容服务   |
| `anthropic`  | Anthropic Claude    |
| `gemini`     | Google Gemini       |
| `openrouter` | OpenRouter 聚合服务 |
| `deepseek`   | DeepSeek            |
| `ollama`     | Ollama 本地模型     |
| `azure`      | Azure OpenAI        |
| `glm`        | 智谱 GLM            |
| `doubao`     | 字节豆包            |
| `moonshot`   | Kimi                |
| `groq`       | Groq                |
| `mistral`    | Mistral             |

## 文档

详细 API 文档请参考 [pkg/llm/doc.go](pkg/llm/doc.go)，使用示例参考 [pkg/llm/example_test.go](pkg/llm/example_test.go)。
