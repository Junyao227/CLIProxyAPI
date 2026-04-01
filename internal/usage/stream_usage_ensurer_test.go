package usage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// TestStreamUsageAccumulator_AccumulateChunk 测试累积流式响应块
func TestStreamUsageAccumulator_AccumulateChunk(t *testing.T) {
	messages := []Message{
		{Role: "user", Content: "Hello"},
	}

	accumulator := NewStreamUsageAccumulator("gpt-4", messages)

	// 累积多个数据块
	chunk1 := []byte(`{"choices":[{"delta":{"content":"Hello"}}]}`)
	chunk2 := []byte(`{"choices":[{"delta":{"content":" world"}}]}`)
	chunk3 := []byte(`{"choices":[{"delta":{"content":"!"}}]}`)

	accumulator.AccumulateChunk(chunk1)
	accumulator.AccumulateChunk(chunk2)
	accumulator.AccumulateChunk(chunk3)

	// 验证累积的内容
	assert.Equal(t, 3, len(accumulator.completionChunks))
	assert.Equal(t, "Hello", accumulator.completionChunks[0])
	assert.Equal(t, " world", accumulator.completionChunks[1])
	assert.Equal(t, "!", accumulator.completionChunks[2])
}

// TestStreamUsageAccumulator_GenerateUsageChunk 测试生成 usage 数据块
func TestStreamUsageAccumulator_GenerateUsageChunk(t *testing.T) {
	messages := []Message{
		{Role: "user", Content: "Hello"},
	}

	accumulator := NewStreamUsageAccumulator("gpt-4", messages)

	// 累积一些内容
	chunk1 := []byte(`{"choices":[{"delta":{"content":"Hello"}}]}`)
	chunk2 := []byte(`{"choices":[{"delta":{"content":" world"}}]}`)
	accumulator.AccumulateChunk(chunk1)
	accumulator.AccumulateChunk(chunk2)

	// 生成 usage 数据块
	usageChunk, err := accumulator.GenerateUsageChunk()
	require.NoError(t, err)

	// 验证 usage 数据块格式
	usage := gjson.GetBytes(usageChunk, "usage")
	assert.True(t, usage.Exists())

	promptTokens := usage.Get("prompt_tokens").Int()
	completionTokens := usage.Get("completion_tokens").Int()
	totalTokens := usage.Get("total_tokens").Int()

	// 验证 tokens 数量合理
	assert.Greater(t, promptTokens, int64(0))
	assert.Greater(t, completionTokens, int64(0))
	assert.Equal(t, promptTokens+completionTokens, totalTokens)

	// 验证包含 finish_reason
	finishReason := gjson.GetBytes(usageChunk, "choices.0.finish_reason")
	assert.Equal(t, "stop", finishReason.String())
}

// TestStreamUsageAccumulator_EmptyContent 测试空内容的处理
func TestStreamUsageAccumulator_EmptyContent(t *testing.T) {
	messages := []Message{
		{Role: "user", Content: "Hello"},
	}

	accumulator := NewStreamUsageAccumulator("gpt-4", messages)

	// 不累积任何内容，直接生成 usage 块
	usageChunk, err := accumulator.GenerateUsageChunk()
	require.NoError(t, err)

	// 验证 usage 字段存在
	usage := gjson.GetBytes(usageChunk, "usage")
	assert.True(t, usage.Exists())

	// 空内容应该产生 0 completion tokens
	completionTokens := usage.Get("completion_tokens").Int()
	assert.Equal(t, int64(0), completionTokens)
}

// TestEnsureStreamUsageField_WithExistingUsage 测试已包含 usage 的数据块
func TestEnsureStreamUsageField_WithExistingUsage(t *testing.T) {
	chunkJSON := []byte(`{
		"choices": [{
			"index": 0,
			"delta": {},
			"finish_reason": "stop"
		}],
		"usage": {
			"prompt_tokens": 10,
			"completion_tokens": 5,
			"total_tokens": 15
		}
	}`)

	messages := []Message{
		{Role: "user", Content: "Hello"},
	}

	result, err := EnsureStreamUsageField(chunkJSON, "gpt-4", messages, "Hello")
	require.NoError(t, err)

	// 验证 usage 字段未被修改
	usage := gjson.GetBytes(result, "usage")
	assert.True(t, usage.Exists())
	assert.Equal(t, int64(10), usage.Get("prompt_tokens").Int())
	assert.Equal(t, int64(5), usage.Get("completion_tokens").Int())
	assert.Equal(t, int64(15), usage.Get("total_tokens").Int())
}

// TestEnsureStreamUsageField_WithoutFinishReason 测试非最后一个数据块
func TestEnsureStreamUsageField_WithoutFinishReason(t *testing.T) {
	chunkJSON := []byte(`{
		"choices": [{
			"index": 0,
			"delta": {
				"content": "Hello"
			}
		}]
	}`)

	messages := []Message{
		{Role: "user", Content: "Test"},
	}

	result, err := EnsureStreamUsageField(chunkJSON, "gpt-4", messages, "Hello")
	require.NoError(t, err)

	// 非最后一个块不应该添加 usage
	usage := gjson.GetBytes(result, "usage")
	assert.False(t, usage.Exists())
}

// TestEnsureStreamUsageField_WithFinishReason 测试最后一个数据块添加 usage
func TestEnsureStreamUsageField_WithFinishReason(t *testing.T) {
	chunkJSON := []byte(`{
		"choices": [{
			"index": 0,
			"delta": {},
			"finish_reason": "stop"
		}]
	}`)

	messages := []Message{
		{Role: "user", Content: "Hello"},
	}

	accumulatedContent := "Hello world!"

	result, err := EnsureStreamUsageField(chunkJSON, "gpt-4", messages, accumulatedContent)
	require.NoError(t, err)

	// 验证 usage 字段已被添加
	usage := gjson.GetBytes(result, "usage")
	assert.True(t, usage.Exists())

	promptTokens := usage.Get("prompt_tokens").Int()
	completionTokens := usage.Get("completion_tokens").Int()
	totalTokens := usage.Get("total_tokens").Int()

	// 验证 tokens 数量合理
	assert.Greater(t, promptTokens, int64(0))
	assert.Greater(t, completionTokens, int64(0))
	assert.Equal(t, promptTokens+completionTokens, totalTokens)
}

// TestEnsureStreamUsageField_DifferentModels 测试不同模型的流式 usage
func TestEnsureStreamUsageField_DifferentModels(t *testing.T) {
	chunkJSON := []byte(`{
		"choices": [{
			"index": 0,
			"delta": {},
			"finish_reason": "stop"
		}]
	}`)

	messages := []Message{
		{Role: "user", Content: "Tell me about AI."},
	}

	accumulatedContent := "AI is artificial intelligence."

	models := []string{"gpt-4", "claude-sonnet-4", "gemini-2.5-pro"}

	for _, model := range models {
		result, err := EnsureStreamUsageField(chunkJSON, model, messages, accumulatedContent)
		require.NoError(t, err, "Failed for model: %s", model)

		// 验证每个模型都能正确估算
		usage := gjson.GetBytes(result, "usage")
		assert.True(t, usage.Exists(), "Usage missing for model: %s", model)
		assert.Greater(t, usage.Get("prompt_tokens").Int(), int64(0), "Invalid prompt_tokens for model: %s", model)
		assert.Greater(t, usage.Get("completion_tokens").Int(), int64(0), "Invalid completion_tokens for model: %s", model)
	}
}

// TestStreamUsageAccumulator_MultipleChunks 测试多个数据块的累积
func TestStreamUsageAccumulator_MultipleChunks(t *testing.T) {
	messages := []Message{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: "Write a short poem."},
	}

	accumulator := NewStreamUsageAccumulator("gpt-4", messages)

	// 模拟流式响应的多个数据块
	chunks := []string{
		`{"choices":[{"delta":{"role":"assistant","content":"Roses"}}]}`,
		`{"choices":[{"delta":{"content":" are"}}]}`,
		`{"choices":[{"delta":{"content":" red"}}]}`,
		`{"choices":[{"delta":{"content":","}}]}`,
		`{"choices":[{"delta":{"content":" violets"}}]}`,
		`{"choices":[{"delta":{"content":" are"}}]}`,
		`{"choices":[{"delta":{"content":" blue"}}]}`,
	}

	for _, chunk := range chunks {
		accumulator.AccumulateChunk([]byte(chunk))
	}

	// 生成 usage 数据块
	usageChunk, err := accumulator.GenerateUsageChunk()
	require.NoError(t, err)

	// 验证 usage 信息
	usage := gjson.GetBytes(usageChunk, "usage")
	assert.True(t, usage.Exists())

	promptTokens := usage.Get("prompt_tokens").Int()
	completionTokens := usage.Get("completion_tokens").Int()
	totalTokens := usage.Get("total_tokens").Int()

	// 多条消息应该产生更多的 prompt tokens
	assert.Greater(t, promptTokens, int64(10))
	// 累积的内容应该产生合理的 completion tokens
	assert.Greater(t, completionTokens, int64(5))
	assert.Equal(t, promptTokens+completionTokens, totalTokens)
}

// TestEnsureStreamUsageField_IncompleteUsage 测试不完整的 usage 字段
func TestEnsureStreamUsageField_IncompleteUsage(t *testing.T) {
	chunkJSON := []byte(`{
		"choices": [{
			"index": 0,
			"delta": {},
			"finish_reason": "stop"
		}],
		"usage": {
			"prompt_tokens": 10
		}
	}`)

	messages := []Message{
		{Role: "user", Content: "Hello"},
	}

	result, err := EnsureStreamUsageField(chunkJSON, "gpt-4", messages, "Hello world")
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

// TestStreamUsageAccumulator_ChunksWithoutContent 测试不包含 content 的数据块
func TestStreamUsageAccumulator_ChunksWithoutContent(t *testing.T) {
	messages := []Message{
		{Role: "user", Content: "Hello"},
	}

	accumulator := NewStreamUsageAccumulator("gpt-4", messages)

	// 累积一些不包含 content 的数据块
	chunk1 := []byte(`{"choices":[{"delta":{"role":"assistant"}}]}`)
	chunk2 := []byte(`{"choices":[{"delta":{"content":"Hello"}}]}`)
	chunk3 := []byte(`{"choices":[{"delta":{}}]}`)

	accumulator.AccumulateChunk(chunk1)
	accumulator.AccumulateChunk(chunk2)
	accumulator.AccumulateChunk(chunk3)

	// 只有 chunk2 应该被累积
	assert.Equal(t, 1, len(accumulator.completionChunks))
	assert.Equal(t, "Hello", accumulator.completionChunks[0])
}
