package usage

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// TestEnsureUsageField_WithExistingUsage 测试当响应已包含完整 usage 字段时直接返回
func TestEnsureUsageField_WithExistingUsage(t *testing.T) {
	responseJSON := []byte(`{
		"id": "chatcmpl-123",
		"object": "chat.completion",
		"created": 1234567890,
		"model": "gpt-4",
		"choices": [{
			"index": 0,
			"message": {
				"role": "assistant",
				"content": "Hello! How can I help you?"
			},
			"finish_reason": "stop"
		}],
		"usage": {
			"prompt_tokens": 10,
			"completion_tokens": 8,
			"total_tokens": 18
		}
	}`)

	messages := []Message{
		{Role: "user", Content: "Hello"},
	}

	result, err := EnsureUsageField(responseJSON, "gpt-4", messages)
	require.NoError(t, err)

	// 验证 usage 字段未被修改
	usage := gjson.GetBytes(result, "usage")
	assert.True(t, usage.Exists())
	assert.Equal(t, int64(10), usage.Get("prompt_tokens").Int())
	assert.Equal(t, int64(8), usage.Get("completion_tokens").Int())
	assert.Equal(t, int64(18), usage.Get("total_tokens").Int())
}

// TestEnsureUsageField_WithoutUsage 测试当响应缺失 usage 字段时进行估算
func TestEnsureUsageField_WithoutUsage(t *testing.T) {
	responseJSON := []byte(`{
		"id": "chatcmpl-123",
		"object": "chat.completion",
		"created": 1234567890,
		"model": "gpt-4",
		"choices": [{
			"index": 0,
			"message": {
				"role": "assistant",
				"content": "Hello! How can I help you today?"
			},
			"finish_reason": "stop"
		}]
	}`)

	messages := []Message{
		{Role: "user", Content: "Hello"},
	}

	result, err := EnsureUsageField(responseJSON, "gpt-4", messages)
	require.NoError(t, err)

	// 验证 usage 字段已被添加
	usage := gjson.GetBytes(result, "usage")
	assert.True(t, usage.Exists())

	promptTokens := usage.Get("prompt_tokens").Int()
	completionTokens := usage.Get("completion_tokens").Int()
	totalTokens := usage.Get("total_tokens").Int()

	// 验证 tokens 数量合理（大于 0）
	assert.Greater(t, promptTokens, int64(0))
	assert.Greater(t, completionTokens, int64(0))
	assert.Equal(t, promptTokens+completionTokens, totalTokens)
}

// TestEnsureUsageField_WithIncompleteUsage 测试当 usage 字段不完整时重新估算
func TestEnsureUsageField_WithIncompleteUsage(t *testing.T) {
	responseJSON := []byte(`{
		"id": "chatcmpl-123",
		"object": "chat.completion",
		"created": 1234567890,
		"model": "claude-sonnet-4",
		"choices": [{
			"index": 0,
			"message": {
				"role": "assistant",
				"content": "I can help you with that."
			},
			"finish_reason": "stop"
		}],
		"usage": {
			"prompt_tokens": 10
		}
	}`)

	messages := []Message{
		{Role: "user", Content: "Can you help me?"},
	}

	result, err := EnsureUsageField(responseJSON, "claude-sonnet-4", messages)
	require.NoError(t, err)

	// 验证 usage 字段已被补全
	usage := gjson.GetBytes(result, "usage")
	assert.True(t, usage.Exists())
	assert.True(t, usage.Get("prompt_tokens").Exists())
	assert.True(t, usage.Get("completion_tokens").Exists())
	assert.True(t, usage.Get("total_tokens").Exists())

	// 验证 total_tokens 等于 prompt_tokens + completion_tokens
	promptTokens := usage.Get("prompt_tokens").Int()
	completionTokens := usage.Get("completion_tokens").Int()
	totalTokens := usage.Get("total_tokens").Int()
	assert.Equal(t, promptTokens+completionTokens, totalTokens)
}

// TestEnsureUsageField_MultipleMessages 测试多条消息的 token 估算
func TestEnsureUsageField_MultipleMessages(t *testing.T) {
	responseJSON := []byte(`{
		"id": "chatcmpl-123",
		"object": "chat.completion",
		"created": 1234567890,
		"model": "gpt-4",
		"choices": [{
			"index": 0,
			"message": {
				"role": "assistant",
				"content": "Based on your requirements, I recommend using Go for backend development."
			},
			"finish_reason": "stop"
		}]
	}`)

	messages := []Message{
		{Role: "system", Content: "You are a helpful programming assistant."},
		{Role: "user", Content: "What programming language should I use for my backend?"},
		{Role: "assistant", Content: "Could you tell me more about your project requirements?"},
		{Role: "user", Content: "I need high performance and good concurrency support."},
	}

	result, err := EnsureUsageField(responseJSON, "gpt-4", messages)
	require.NoError(t, err)

	// 验证 usage 字段存在且合理
	usage := gjson.GetBytes(result, "usage")
	assert.True(t, usage.Exists())

	promptTokens := usage.Get("prompt_tokens").Int()
	completionTokens := usage.Get("completion_tokens").Int()
	totalTokens := usage.Get("total_tokens").Int()

	// 多条消息应该产生更多的 prompt tokens
	assert.Greater(t, promptTokens, int64(20))
	assert.Greater(t, completionTokens, int64(10))
	assert.Equal(t, promptTokens+completionTokens, totalTokens)
}

// TestEnsureUsageField_EmptyMessages 测试空消息列表的处理
func TestEnsureUsageField_EmptyMessages(t *testing.T) {
	responseJSON := []byte(`{
		"id": "chatcmpl-123",
		"object": "chat.completion",
		"created": 1234567890,
		"model": "gpt-4",
		"choices": [{
			"index": 0,
			"message": {
				"role": "assistant",
				"content": "Hello"
			},
			"finish_reason": "stop"
		}]
	}`)

	messages := []Message{}

	result, err := EnsureUsageField(responseJSON, "gpt-4", messages)
	require.NoError(t, err)

	// 验证 usage 字段存在
	usage := gjson.GetBytes(result, "usage")
	assert.True(t, usage.Exists())

	// 空消息列表应该产生 0 prompt tokens
	promptTokens := usage.Get("prompt_tokens").Int()
	assert.Equal(t, int64(0), promptTokens)
}

// TestEnsureUsageField_DifferentModels 测试不同模型的 token 估算
func TestEnsureUsageField_DifferentModels(t *testing.T) {
	responseJSON := []byte(`{
		"id": "chatcmpl-123",
		"object": "chat.completion",
		"created": 1234567890,
		"model": "gemini-2.5-pro",
		"choices": [{
			"index": 0,
			"message": {
				"role": "assistant",
				"content": "Gemini is a powerful language model."
			},
			"finish_reason": "stop"
		}]
	}`)

	messages := []Message{
		{Role: "user", Content: "Tell me about Gemini."},
	}

	models := []string{"gpt-4", "claude-sonnet-4", "gemini-2.5-pro"}

	for _, model := range models {
		result, err := EnsureUsageField(responseJSON, model, messages)
		require.NoError(t, err, "Failed for model: %s", model)

		// 验证每个模型都能正确估算
		usage := gjson.GetBytes(result, "usage")
		assert.True(t, usage.Exists(), "Usage missing for model: %s", model)
		assert.Greater(t, usage.Get("prompt_tokens").Int(), int64(0), "Invalid prompt_tokens for model: %s", model)
		assert.Greater(t, usage.Get("completion_tokens").Int(), int64(0), "Invalid completion_tokens for model: %s", model)
	}
}

// TestExtractCompletionContent 测试从响应中提取 completion 内容
func TestExtractCompletionContent(t *testing.T) {
	tests := []struct {
		name     string
		response string
		expected string
	}{
		{
			name: "标准 chat completions 格式",
			response: `{
				"choices": [{
					"message": {
						"content": "Hello, world!"
					}
				}]
			}`,
			expected: "Hello, world!",
		},
		{
			name: "completions API 格式",
			response: `{
				"choices": [{
					"text": "This is a completion."
				}]
			}`,
			expected: "This is a completion.",
		},
		{
			name:     "无内容",
			response: `{"choices": [{}]}`,
			expected: "",
		},
		{
			name:     "空响应",
			response: `{}`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := extractCompletionContent([]byte(tt.response))
			assert.Equal(t, tt.expected, content)
		})
	}
}

// TestParseMessagesFromRequest 测试从请求中解析消息列表
func TestParseMessagesFromRequest(t *testing.T) {
	tests := []struct {
		name        string
		request     string
		expectError bool
		expected    []Message
	}{
		{
			name: "标准请求格式",
			request: `{
				"model": "gpt-4",
				"messages": [
					{"role": "user", "content": "Hello"},
					{"role": "assistant", "content": "Hi there!"}
				]
			}`,
			expectError: false,
			expected: []Message{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi there!"},
			},
		},
		{
			name: "单条消息",
			request: `{
				"model": "gpt-4",
				"messages": [
					{"role": "user", "content": "Test"}
				]
			}`,
			expectError: false,
			expected: []Message{
				{Role: "user", Content: "Test"},
			},
		},
		{
			name:        "缺失 messages 字段",
			request:     `{"model": "gpt-4"}`,
			expectError: true,
		},
		{
			name:        "无效的 JSON",
			request:     `{invalid json}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages, err := ParseMessagesFromRequest([]byte(tt.request))

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, len(tt.expected), len(messages))
				for i, expected := range tt.expected {
					assert.Equal(t, expected.Role, messages[i].Role)
					assert.Equal(t, expected.Content, messages[i].Content)
				}
			}
		})
	}
}

// TestEnsureUsageField_JSONStructure 测试返回的 JSON 结构完整性
func TestEnsureUsageField_JSONStructure(t *testing.T) {
	responseJSON := []byte(`{
		"id": "chatcmpl-123",
		"object": "chat.completion",
		"created": 1234567890,
		"model": "gpt-4",
		"choices": [{
			"index": 0,
			"message": {
				"role": "assistant",
				"content": "Test response"
			},
			"finish_reason": "stop"
		}]
	}`)

	messages := []Message{
		{Role: "user", Content: "Test"},
	}

	result, err := EnsureUsageField(responseJSON, "gpt-4", messages)
	require.NoError(t, err)

	// 验证返回的是有效的 JSON
	var parsed map[string]interface{}
	err = json.Unmarshal(result, &parsed)
	require.NoError(t, err)

	// 验证原有字段未被破坏
	assert.Equal(t, "chatcmpl-123", parsed["id"])
	assert.Equal(t, "chat.completion", parsed["object"])
	assert.Equal(t, float64(1234567890), parsed["created"])
	assert.Equal(t, "gpt-4", parsed["model"])

	// 验证 usage 字段存在且结构正确
	usage, ok := parsed["usage"].(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, usage, "prompt_tokens")
	assert.Contains(t, usage, "completion_tokens")
	assert.Contains(t, usage, "total_tokens")
}
