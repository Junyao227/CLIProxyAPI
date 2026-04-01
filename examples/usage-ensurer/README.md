# Usage Field Ensurer 示例

本示例演示如何使用 `usage.EnsureUsageField` 函数确保 API 响应包含完整的 `usage` 字段。

## 功能说明

`EnsureUsageField` 函数会：

1. **检查现有 usage 字段**: 如果响应已包含完整的 `usage` 字段（包含 `prompt_tokens`、`completion_tokens`、`total_tokens`），则直接返回原响应
2. **自动估算**: 如果 usage 字段缺失或不完整，使用 token 估算器自动估算并添加
3. **保持兼容性**: 确保所有响应都符合 OpenAI API 规范

## 运行示例

```bash
cd CLIProxyAPI
go run ./examples/usage-ensurer/main.go
```

## 示例场景

### 示例 1: 响应已包含完整的 usage 字段

当上游 API 返回完整的 usage 信息时，函数会直接返回原响应，不做任何修改。

**输入**:
```json
{
  "id": "chatcmpl-123",
  "model": "gpt-4",
  "choices": [{
    "message": {
      "content": "Hello! How can I help you?"
    }
  }],
  "usage": {
    "prompt_tokens": 10,
    "completion_tokens": 8,
    "total_tokens": 18
  }
}
```

**输出**: 与输入相同（usage 字段未被修改）

### 示例 2: 响应缺失 usage 字段

当上游 API 不返回 usage 信息时，函数会自动估算并添加。

**输入**:
```json
{
  "id": "chatcmpl-456",
  "model": "claude-sonnet-4",
  "choices": [{
    "message": {
      "content": "I can help you with that. What would you like to know?"
    }
  }]
}
```

**输出**: 添加了估算的 usage 字段
```json
{
  "id": "chatcmpl-456",
  "model": "claude-sonnet-4",
  "choices": [{
    "message": {
      "content": "I can help you with that. What would you like to know?"
    }
  }],
  "usage": {
    "prompt_tokens": 13,
    "completion_tokens": 14,
    "total_tokens": 27
  }
}
```

### 示例 3: 响应包含不完整的 usage 字段

当 usage 字段存在但不完整时，函数会重新估算并补全所有字段。

**输入**:
```json
{
  "id": "chatcmpl-789",
  "model": "gemini-2.5-pro",
  "choices": [{
    "message": {
      "content": "Gemini is ready to assist you."
    }
  }],
  "usage": {
    "prompt_tokens": 15
  }
}
```

**输出**: 补全了缺失的字段
```json
{
  "id": "chatcmpl-789",
  "model": "gemini-2.5-pro",
  "choices": [{
    "message": {
      "content": "Gemini is ready to assist you."
    }
  }],
  "usage": {
    "prompt_tokens": 12,
    "completion_tokens": 8,
    "total_tokens": 20
  }
}
```

### 示例 4: 多条消息的处理

函数支持包含多条消息的对话历史，会正确估算所有消息的 token 总数。

**输入消息**:
```go
messages := []usage.Message{
    {Role: "system", Content: "You are a helpful programming assistant."},
    {Role: "user", Content: "What programming language should I use for my backend?"},
    {Role: "assistant", Content: "Could you tell me more about your project requirements?"},
    {Role: "user", Content: "I need high performance and good concurrency support."},
}
```

**输出**: 正确估算了多条消息的 token 总数
```json
{
  "usage": {
    "prompt_tokens": 59,
    "completion_tokens": 23,
    "total_tokens": 82
  }
}
```

## 在 HTTP 处理器中使用

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/router-for-me/CLIProxyAPI/v6/internal/usage"
    "github.com/tidwall/gjson"
)

func handleChatCompletion(c *gin.Context) {
    // 读取请求
    requestJSON, _ := c.GetRawData()
    
    // 解析消息列表
    messages, err := usage.ParseMessagesFromRequest(requestJSON)
    if err != nil {
        c.JSON(400, gin.H{"error": "Invalid request"})
        return
    }
    
    // 获取模型名称
    model := gjson.GetBytes(requestJSON, "model").String()
    
    // 调用上游 API（这里省略具体实现）
    responseJSON := callUpstreamAPI(requestJSON)
    
    // 确保响应包含 usage 字段
    responseWithUsage, err := usage.EnsureUsageField(responseJSON, model, messages)
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to process response"})
        return
    }
    
    // 返回响应
    c.Data(200, "application/json", responseWithUsage)
}
```

## 支持的模型

函数支持以下模型系列的 token 估算：

- **GPT 系列**: gpt-3.5-turbo, gpt-4, gpt-4-turbo, gpt-5, o1, o3 等
- **Claude 系列**: claude-3-opus, claude-3-sonnet, claude-sonnet-4, claude-opus-4-5 等
- **Gemini 系列**: gemini-pro, gemini-2.5-pro, gemini-ultra 等

## 估算准确性

根据设计文档要求，token 估算的误差应控制在实际值的 ±10% 范围内。函数使用 `tiktoken-go` 库进行估算，该库与 OpenAI 官方的 tiktoken 库兼容。

## 性能优化

1. **零拷贝**: 当 usage 字段已存在且完整时，直接返回原响应
2. **编码器缓存**: TokenEstimator 内部缓存编码器，避免重复创建
3. **高效 JSON 操作**: 使用 gjson/sjson 进行 JSON 操作，避免完整的序列化/反序列化

## 相关文档

- [任务 2.2 实现总结](../../docs/task-2.2-summary.md)
- [Token 估算器文档](../../docs/token-estimator.md)
- [设计文档](../../.kiro/specs/cliproxyapi-newapi-integration/design.md)
