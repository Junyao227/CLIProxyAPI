// Package usage 提供 CLI Proxy API 服务器的使用量跟踪和日志记录功能。
package usage

import (
	"fmt"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// StreamUsageAccumulator 用于累积流式响应的 token 使用量
// 在流式响应过程中收集所有数据块，并在最后生成包含 usage 的数据块
type StreamUsageAccumulator struct {
	model            string    // 模型名称
	messages         []Message // 请求的消息列表
	completionChunks []string  // 累积的 completion 内容块
	estimator        *TokenEstimator
	promptTokens     int  // 缓存的 prompt tokens
	hasCalculated    bool // 是否已计算 prompt tokens
}

// NewStreamUsageAccumulator 创建一个新的流式 usage 累积器
//
// 参数:
//   - model: 模型名称
//   - messages: 请求的消息列表
//
// 返回值:
//   - *StreamUsageAccumulator: 新的累积器实例
func NewStreamUsageAccumulator(model string, messages []Message) *StreamUsageAccumulator {
	return &StreamUsageAccumulator{
		model:            model,
		messages:         messages,
		completionChunks: make([]string, 0),
		estimator:        NewTokenEstimator(),
		hasCalculated:    false,
	}
}

// AccumulateChunk 累积一个流式响应数据块
// 从数据块中提取 content 并累积，用于最终的 token 估算
//
// 参数:
//   - chunkJSON: 流式响应数据块的 JSON 字节数组
func (a *StreamUsageAccumulator) AccumulateChunk(chunkJSON []byte) {
	// 从流式响应块中提取 content
	// 流式响应格式: {"choices":[{"delta":{"content":"..."}}]}
	content := gjson.GetBytes(chunkJSON, "choices.0.delta.content")
	if content.Exists() && content.String() != "" {
		a.completionChunks = append(a.completionChunks, content.String())
	}
}

// GenerateUsageChunk 生成包含 usage 信息的最终数据块
// 这个数据块应该在所有内容块之后、[DONE] 之前发送
//
// 返回值:
//   - []byte: 包含 usage 信息的 JSON 数据块
//   - error: 如果生成失败则返回错误
func (a *StreamUsageAccumulator) GenerateUsageChunk() ([]byte, error) {
	// 计算 prompt tokens（只计算一次）
	if !a.hasCalculated {
		var err error
		a.promptTokens, err = a.estimator.EstimatePromptTokens(a.model, a.messages)
		if err != nil {
			return nil, fmt.Errorf("failed to estimate prompt tokens: %w", err)
		}
		a.hasCalculated = true
	}

	// 合并所有 completion 内容块
	var fullCompletion string
	for _, chunk := range a.completionChunks {
		fullCompletion += chunk
	}

	// 估算 completion tokens
	completionTokens, err := a.estimator.EstimateCompletionTokens(a.model, fullCompletion)
	if err != nil {
		return nil, fmt.Errorf("failed to estimate completion tokens: %w", err)
	}

	// 计算总 tokens
	totalTokens := a.promptTokens + completionTokens

	// 创建包含 usage 的流式响应块
	// 格式: {"choices":[{"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":100,"completion_tokens":50,"total_tokens":150}}
	result := []byte("{}")

	// 设置 choices 数组
	result, err = sjson.SetBytes(result, "choices", []map[string]interface{}{
		{
			"index":         0,
			"delta":         map[string]interface{}{},
			"finish_reason": "stop",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to set choices: %w", err)
	}

	// 设置 usage 字段
	result, err = sjson.SetBytes(result, "usage.prompt_tokens", a.promptTokens)
	if err != nil {
		return nil, fmt.Errorf("failed to set prompt_tokens: %w", err)
	}

	result, err = sjson.SetBytes(result, "usage.completion_tokens", completionTokens)
	if err != nil {
		return nil, fmt.Errorf("failed to set completion_tokens: %w", err)
	}

	result, err = sjson.SetBytes(result, "usage.total_tokens", totalTokens)
	if err != nil {
		return nil, fmt.Errorf("failed to set total_tokens: %w", err)
	}

	return result, nil
}

// EnsureStreamUsageField 确保流式响应数据块包含 usage 字段
// 如果数据块已包含 usage 字段，则直接返回
// 如果数据块是最后一个块（包含 finish_reason），则添加 usage 字段
//
// 参数:
//   - chunkJSON: 流式响应数据块的 JSON 字节数组
//   - model: 模型名称
//   - messages: 请求的消息列表
//   - accumulatedContent: 累积的所有 completion 内容
//
// 返回值:
//   - []byte: 确保包含 usage 字段的数据块 JSON
//   - error: 如果处理失败则返回错误
func EnsureStreamUsageField(chunkJSON []byte, model string, messages []Message, accumulatedContent string) ([]byte, error) {
	// 检查是否已包含 usage 字段
	if gjson.GetBytes(chunkJSON, "usage").Exists() {
		usage := gjson.GetBytes(chunkJSON, "usage")

		// 验证 usage 字段是否完整
		hasPromptTokens := usage.Get("prompt_tokens").Exists()
		hasCompletionTokens := usage.Get("completion_tokens").Exists()
		hasTotalTokens := usage.Get("total_tokens").Exists()

		if hasPromptTokens && hasCompletionTokens && hasTotalTokens {
			// usage 字段完整，直接返回
			return chunkJSON, nil
		}
	}

	// 检查是否是最后一个块（包含 finish_reason）
	finishReason := gjson.GetBytes(chunkJSON, "choices.0.finish_reason")
	if !finishReason.Exists() || finishReason.String() == "" {
		// 不是最后一个块，直接返回原数据
		return chunkJSON, nil
	}

	// 是最后一个块，需要添加 usage 字段
	estimator := NewTokenEstimator()

	// 估算 prompt tokens
	promptTokens, err := estimator.EstimatePromptTokens(model, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to estimate prompt tokens: %w", err)
	}

	// 估算 completion tokens
	completionTokens, err := estimator.EstimateCompletionTokens(model, accumulatedContent)
	if err != nil {
		return nil, fmt.Errorf("failed to estimate completion tokens: %w", err)
	}

	// 计算总 tokens
	totalTokens := promptTokens + completionTokens

	// 将 usage 字段添加到数据块中
	result := chunkJSON
	result, err = sjson.SetBytes(result, "usage.prompt_tokens", promptTokens)
	if err != nil {
		return nil, fmt.Errorf("failed to set prompt_tokens: %w", err)
	}

	result, err = sjson.SetBytes(result, "usage.completion_tokens", completionTokens)
	if err != nil {
		return nil, fmt.Errorf("failed to set completion_tokens: %w", err)
	}

	result, err = sjson.SetBytes(result, "usage.total_tokens", totalTokens)
	if err != nil {
		return nil, fmt.Errorf("failed to set total_tokens: %w", err)
	}

	return result, nil
}
