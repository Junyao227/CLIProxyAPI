// Package usage 提供 CLI Proxy API 服务器的使用量跟踪和日志记录功能。
package usage

import (
	"fmt"
	"strings"

	"github.com/tiktoken-go/tokenizer"
)

// Message 表示聊天消息结构
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// TokenEstimator 提供 token 数量估算功能
// 支持 GPT、Claude、Gemini 等模型的 token 计数
type TokenEstimator struct {
	// encoders 缓存不同模型的编码器
	encoders map[string]tokenizer.Codec
}

// NewTokenEstimator 创建一个新的 token 估算器实例
//
// 返回值:
//   - *TokenEstimator: 新的 token 估算器实例
func NewTokenEstimator() *TokenEstimator {
	return &TokenEstimator{
		encoders: make(map[string]tokenizer.Codec),
	}
}

// EstimatePromptTokens 估算 prompt 的 token 数量
// 基于消息列表计算总的 prompt tokens
//
// 参数:
//   - model: 模型名称（如 "gpt-4", "claude-sonnet-4", "gemini-2.5-pro"）
//   - messages: 消息列表
//
// 返回值:
//   - int: 估算的 prompt token 数量
//   - error: 如果估算失败则返回错误
func (e *TokenEstimator) EstimatePromptTokens(model string, messages []Message) (int, error) {
	if len(messages) == 0 {
		return 0, nil
	}

	// 获取或创建编码器
	codec, err := e.getCodec(model)
	if err != nil {
		return 0, fmt.Errorf("failed to get codec for model %s: %w", model, err)
	}

	totalTokens := 0

	// 为每条消息计算 tokens
	// 包括消息格式的开销（role 标记等）
	for _, msg := range messages {
		// 计算 role 的 tokens（通常是 1-2 个 token）
		roleTokens := e.estimateTokens(codec, msg.Role)
		totalTokens += roleTokens

		// 计算 content 的 tokens
		contentTokens := e.estimateTokens(codec, msg.Content)
		totalTokens += contentTokens

		// 添加消息格式开销（每条消息约 3-4 个 token 用于格式化）
		totalTokens += 4
	}

	// 添加对话格式的基础开销
	totalTokens += 3

	return totalTokens, nil
}

// EstimateCompletionTokens 估算 completion 的 token 数量
// 基于响应内容计算 completion tokens
//
// 参数:
//   - model: 模型名称
//   - content: 响应内容文本
//
// 返回值:
//   - int: 估算的 completion token 数量
//   - error: 如果估算失败则返回错误
func (e *TokenEstimator) EstimateCompletionTokens(model string, content string) (int, error) {
	if content == "" {
		return 0, nil
	}

	// 获取或创建编码器
	codec, err := e.getCodec(model)
	if err != nil {
		return 0, fmt.Errorf("failed to get codec for model %s: %w", model, err)
	}

	tokens := e.estimateTokens(codec, content)
	return tokens, nil
}

// getCodec 获取或创建指定模型的编码器
// 使用缓存避免重复创建编码器
func (e *TokenEstimator) getCodec(model string) (tokenizer.Codec, error) {
	// 标准化模型名称
	normalizedModel := normalizeModelName(model)

	// 检查缓存
	if codec, exists := e.encoders[normalizedModel]; exists {
		return codec, nil
	}

	// 根据模型选择合适的编码器
	var codec tokenizer.Codec
	var err error

	switch {
	case isGPTModel(normalizedModel):
		// GPT 模型使用 cl100k_base 编码
		codec, err = tokenizer.Get(tokenizer.Cl100kBase)
	case isClaudeModel(normalizedModel):
		// Claude 模型使用 cl100k_base 编码（与 GPT-4 相同）
		codec, err = tokenizer.Get(tokenizer.Cl100kBase)
	case isGeminiModel(normalizedModel):
		// Gemini 模型使用 cl100k_base 编码
		codec, err = tokenizer.Get(tokenizer.Cl100kBase)
	default:
		// 默认使用 cl100k_base 编码
		codec, err = tokenizer.Get(tokenizer.Cl100kBase)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get tokenizer codec: %w", err)
	}

	// 缓存编码器
	e.encoders[normalizedModel] = codec

	return codec, nil
}

// estimateTokens 使用编码器估算文本的 token 数量
func (e *TokenEstimator) estimateTokens(codec tokenizer.Codec, text string) int {
	if text == "" {
		return 0
	}

	// 使用 tiktoken 编码文本
	ids, _, err := codec.Encode(text)
	if err != nil {
		// 如果编码失败，使用保守估算（字符数 / 4）
		return len(text) / 4
	}

	return len(ids)
}

// normalizeModelName 标准化模型名称，便于匹配
func normalizeModelName(model string) string {
	return strings.ToLower(strings.TrimSpace(model))
}

// isGPTModel 判断是否为 GPT 系列模型
func isGPTModel(model string) bool {
	return strings.Contains(model, "gpt") ||
		strings.Contains(model, "o1") ||
		strings.Contains(model, "o3")
}

// isClaudeModel 判断是否为 Claude 系列模型
func isClaudeModel(model string) bool {
	return strings.Contains(model, "claude")
}

// isGeminiModel 判断是否为 Gemini 系列模型
func isGeminiModel(model string) bool {
	return strings.Contains(model, "gemini")
}
