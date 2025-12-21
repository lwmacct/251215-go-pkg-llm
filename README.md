# Go 语言 LLM 统一抽象层，提供多提供商的统一调用接口。

[![License](https://img.shields.io/github/license/lwmacct/251215-go-pkg-llm)](LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/lwmacct/251215-go-pkg-llm.svg)](https://pkg.go.dev/github.com/lwmacct/251215-go-pkg-llm)
[![Go CI](https://github.com/lwmacct/251215-go-pkg-llm/actions/workflows/go-ci.yml/badge.svg)](https://github.com/lwmacct/251215-go-pkg-llm/actions/workflows/go-ci.yml)
[![codecov](https://codecov.io/gh/lwmacct/251215-go-pkg-llm/branch/main/graph/badge.svg)](https://codecov.io/gh/lwmacct/251215-go-pkg-llm)
[![Go Report Card](https://goreportcard.com/badge/github.com/lwmacct/251215-go-pkg-llm)](https://goreportcard.com/report/github.com/lwmacct/251215-go-pkg-llm)
[![GitHub Tag](https://img.shields.io/github/v/tag/lwmacct/251215-go-pkg-llm?sort=semver)](https://github.com/lwmacct/251215-go-pkg-llm/tags)

<!--TOC-->

## Table of Contents

- [特性](#特性) `:22+8`
- [安装](#安装) `:30+6`
- [快速开始](#快速开始) `:36+35`
- [Provider 类型](#provider-类型) `:71+17`
- [文档](#文档) `:88+3`

<!--TOC-->

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
    // 方式一：零配置（从环境变量 OPENROUTER_API_KEY 读取）
    p, _ := provider.Default()
    defer p.Close()

    // 方式二：指定 Provider 类型（从环境变量 OPENAI_API_KEY 读取）
    // p, _ := provider.Default(llm.ProviderTypeOpenAI)

    // 方式三：完全自定义配置
    // p, _ := provider.New(&llm.Config{
    //     Type:   llm.ProviderTypeOpenAI,
    //     APIKey: "sk-xxx",
    // })

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
