// Package usage 提供 CLI Proxy API 服务器的使用量跟踪和日志记录功能。
package usage

import (
	"encoding/json"
	"fmt"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// UsageField 表示 OpenAI 兼容的 usage 字段结构
type UsageField struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// EnsureUsageField 确保响应包含 usage 字段
// 如果响应已包含 usage 字段，则直接返回原响应
// 如果缺失，则使用 token 估算器估算并添加 usage 字段
//
// 参数:
//   - responseJSON: 原始响应的 JSON 字节数组
//   - model: 模型名称（用于 token 估算）
//   - messages: 请求的消息列表（用于估算 prompt tokens）
//
// 返回值:
//   - []byte: 确保包含 usage 字段的响应 JSON
//   - error: 如果处理失败则返回错误
func EnsureUsageField(responseJSON []byte, model string, messages []Message) ([]byte, error) {
	// 检查响应是否已包含 usage 字段
	if gjson.GetBytes(responseJSON, "usage").Exists() {
		usage := gjson.GetBytes(responseJSON, "usage")
		
		// 验证 usage 字段是否完整（包含必需的三个字段）
		hasPromptTokens := usage.Get("prompt_tokens").Exists()
		hasCompletionTokens := usage.Get("completion_tokens").Exists()
		hasTotalTokens := usage.Get("total_tokens").Exists()
		
		if hasPromptTokens && hasCompletionTokens && hasTotalTokens {
			// usage 字段完整，直接返回
			return responseJSON, nil
		}
	}

	// usage 字段缺失或不完整，需要估算
	estimator := NewTokenEstimator()

	// 估算 prompt tokens
	promptTokens, err := estimator.EstimatePromptTokens(model, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to estimate prompt tokens: %w", err)
	}

	// 提取响应内容用于估算 completion tokens
	content := extractCompletionContent(responseJSON)
	completionTokens, err := estimator.EstimateCompletionTokens(model, content)
	if err != nil {
		return nil, fmt.Errorf("failed to estimate completion tokens: %w", err)
	}

	// 计算总 tokens
	totalTokens := promptTokens + completionTokens

	// 将 usage 字段添加到响应中
	result := responseJSON
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

// extractCompletionContent 从响应中提取 completion 内容
// 支持标准的 OpenAI 响应格式
func extractCompletionContent(responseJSON []byte) string {
	// 尝试从 choices[0].message.content 提取
	content := gjson.GetBytes(responseJSON, "choices.0.message.content")
	if content.Exists() {
		return content.String()
	}

	// 尝试从 choices[0].text 提取（completions API 格式）
	text := gjson.GetBytes(responseJSON, "choices.0.text")
	if text.Exists() {
		return text.String()
	}

	// 如果都不存在，返回空字符串
	return ""
}

// ParseMessagesFromRequest 从请求 JSON 中解析消息列表
// 用于从原始请求中提取 messages 字段
//
// 参数:
//   - requestJSON: 请求的 JSON 字节数组
//
// 返回值:
//   - []Message: 解析出的消息列表
//   - error: 如果解析失败则返回错误
func ParseMessagesFromRequest(requestJSON []byte) ([]Message, error) {
	messagesResult := gjson.GetBytes(requestJSON, "messages")
	if !messagesResult.Exists() {
		return nil, fmt.Errorf("messages field not found in request")
	}

	var messages []Message
	err := json.Unmarshal([]byte(messagesResult.Raw), &messages)
	if err != nil {
		return nil, fmt.Errorf("failed to parse messages: %w", err)
	}

	return messages, nil
}
