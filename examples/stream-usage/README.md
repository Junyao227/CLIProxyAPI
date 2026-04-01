# 流式响应 Usage 字段示例

本示例演示如何使用 `StreamUsageAccumulator` 在流式响应中添加 usage 字段。

## 功能

- 模拟 OpenAI Chat Completions API 的流式响应
- 使用 `StreamUsageAccumulator` 累积内容并生成 usage 数据块
- 在最后一个数据块（[DONE] 之前）添加完整的 usage 信息

## 运行示例

### 1. 启动服务器

```bash
cd CLIProxyAPI/examples/stream-usage
go run main.go
```

服务器将在 `http://localhost:8080` 启动。

### 2. 发送流式请求

使用 curl 发送请求：

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "messages": [
      {"role": "user", "content": "Hello"}
    ],
    "stream": true
  }'
```

### 3. 预期输出

```
data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":"!"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":" How"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":" can"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":" I"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":" help"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":" you"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":" today"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":"?"},"finish_reason":null}]}

data: {"choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":8,"completion_tokens":9,"total_tokens":17}}

data: [DONE]
```

**注意最后一个数据块**包含了完整的 usage 信息：
```json
{
  "choices": [{
    "index": 0,
    "delta": {},
    "finish_reason": "stop"
  }],
  "usage": {
    "prompt_tokens": 8,
    "completion_tokens": 9,
    "total_tokens": 17
  }
}
```

## 代码说明

### 1. 创建 Usage 累积器

```go
accumulator := usage.NewStreamUsageAccumulator(model, messages)
```

### 2. 累积每个数据块

```go
for _, chunk := range chunks {
    // 累积内容用于 token 估算
    accumulator.AccumulateChunk([]byte(chunk))
    
    // 发送数据块到客户端
    fmt.Fprintf(c.Writer, "data: %s\n\n", chunk)
    flusher.Flush()
}
```

### 3. 生成并发送 Usage 数据块

```go
// 生成包含 usage 的最后数据块
usageChunk, err := accumulator.GenerateUsageChunk()
if err != nil {
    fmt.Printf("Error generating usage chunk: %v\n", err)
} else {
    fmt.Fprintf(c.Writer, "data: %s\n\n", string(usageChunk))
    flusher.Flush()
}

// 发送 [DONE] 标记
fmt.Fprint(c.Writer, "data: [DONE]\n\n")
flusher.Flush()
```

## 关键点

1. **SSE 格式**: 每行以 `data: ` 开头，以 `\n\n` 结尾
2. **Usage 位置**: usage 数据块在所有内容块之后、[DONE] 之前
3. **Finish Reason**: 包含 usage 的数据块必须有 `finish_reason: "stop"`
4. **Token 估算**: 自动估算 prompt_tokens 和 completion_tokens

## 相关文档

- [流式响应 Usage 字段集成指南](../../docs/stream-usage-integration.md)
- [任务 2.3 总结](../../docs/task-2.3-summary.md)
