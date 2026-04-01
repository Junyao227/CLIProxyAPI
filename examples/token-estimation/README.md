# Token 估算器示例

这个示例演示了如何使用 CLIProxyAPI 的 token 估算器来计算不同场景下的 token 使用量。

## 运行示例

```bash
cd CLIProxyAPI
go run ./examples/token-estimation/main.go
```

## 示例场景

### 1. 简单对话
演示单条用户消息的 token 估算。

### 2. 多轮对话
演示包含 system、user、assistant 多轮对话的 token 估算。

### 3. 不同模型对比
对比 GPT、Claude、Gemini 模型的 token 估算结果。

### 4. 长文本估算
演示长文本内容的 token 估算能力。

### 5. 代码内容估算
演示包含代码的内容的 token 估算。

### 6. 多语言内容
演示中文、英文、法文混合内容的 token 估算。

## 预期输出

```
=== Token 估算器示例 ===

示例 1: 简单对话
  模型: gpt-4
  Prompt tokens: 14
  Completion tokens: 8
  Total tokens: 22

示例 2: 多轮对话
  模型: gpt-4
  Prompt tokens: 47
  Completion tokens: 7
  Total tokens: 54

...
```

## 相关文档

- [Token 估算器使用指南](../../docs/token-estimator.md)
- [设计文档](../../../.kiro/specs/cliproxyapi-newapi-integration/design.md)
