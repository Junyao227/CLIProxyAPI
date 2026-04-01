// Package openai 提供 OpenAI API 端点的 HTTP 处理器测试
package openai

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/usage"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

// TestStreamUsageIntegration 测试流式响应中 usage 字段的集成
func TestStreamUsageIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建测试请求
	requestBody := `{
		"model": "gpt-4",
		"messages": [
			{"role": "user", "content": "Hello, how are you?"}
		],
		"stream": true
	}`

	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewBufferString(requestBody))
	req.Header.Set("Content-Type", "application/json")

	// 创建响应记录器
	w := httptest.NewRecorder()

	// 创建 Gin 上下文
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// 模拟流式响应处理
	flusher, ok := w.(http.Flusher)
	assert.True(t, ok, "ResponseRecorder should implement http.Flusher")

	// 设置 SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	// 提取 messages
	messages := []usage.Message{
		{Role: "user", Content: "Hello, how are you?"},
	}

	// 创建 usage 累积器
	accumulator := usage.NewStreamUsageAccumulator("gpt-4", messages)

	// 模拟流式数据块
	chunks := []string{
		`{"choices":[{"delta":{"role":"assistant","content":"I'm"},"finish_reason":null}]}`,
		`{"choices":[{"delta":{"content":" doing"},"finish_reason":null}]}`,
		`{"choices":[{"delta":{"content":" well"},"finish_reason":null}]}`,
		`{"choices":[{"delta":{"content":", thank"},"finish_reason":null}]}`,
		`{"choices":[{"delta":{"content":" you"},"finish_reason":null}]}`,
		`{"choices":[{"delta":{"content":"!"},"finish_reason":null}]}`,
	}

	// 发送所有内容块
	for _, chunk := range chunks {
		accumulator.AccumulateChunk([]byte(chunk))
		c.Writer.Write([]byte("data: " + chunk + "\n\n"))
		flusher.Flush()
	}

	// 生成并发送 usage 数据块
	usageChunk, err := accumulator.GenerateUsageChunk()
	assert.NoError(t, err, "应该成功生成 usage 数据块")
	assert.NotNil(t, usageChunk, "usage 数据块不应为空")

	c.Writer.Write([]byte("data: " + string(usageChunk) + "\n\n"))
	flusher.Flush()

	// 发送 [DONE]
	c.Writer.Write([]byte("data: [DONE]\n\n"))
	flusher.Flush()

	// 验证响应
	response := w.Body.String()

	// 解析 SSE 响应
	scanner := bufio.NewScanner(strings.NewReader(response))
	var dataLines []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			dataLines = append(dataLines, strings.TrimPrefix(line, "data: "))
		}
	}

	// 验证至少有内容块 + usage 块 + [DONE]
	assert.GreaterOrEqual(t, len(dataLines), 8, "应该有至少 8 个数据行（6个内容块 + 1个usage块 + [DONE]）")

	// 验证最后一个数据块是 [DONE]
	assert.Equal(t, "[DONE]", dataLines[len(dataLines)-1], "最后一个数据块应该是 [DONE]")

	// 验证倒数第二个数据块包含 usage 字段
	usageLine := dataLines[len(dataLines)-2]
	assert.NotEqual(t, "[DONE]", usageLine, "倒数第二个数据块不应该是 [DONE]")

	// 解析 usage 数据块
	usageData := gjson.Parse(usageLine)
	assert.True(t, usageData.Get("usage").Exists(), "应该包含 usage 字段")
	assert.True(t, usageData.Get("usage.prompt_tokens").Exists(), "应该包含 prompt_tokens")
	assert.True(t, usageData.Get("usage.completion_tokens").Exists(), "应该包含 completion_tokens")
	assert.True(t, usageData.Get("usage.total_tokens").Exists(), "应该包含 total_tokens")

	// 验证 token 数量合理
	promptTokens := usageData.Get("usage.prompt_tokens").Int()
	completionTokens := usageData.Get("usage.completion_tokens").Int()
	totalTokens := usageData.Get("usage.total_tokens").Int()

	assert.Greater(t, promptTokens, int64(0), "prompt_tokens 应该大于 0")
	assert.Greater(t, completionTokens, int64(0), "completion_tokens 应该大于 0")
	assert.Equal(t, promptTokens+completionTokens, totalTokens, "total_tokens 应该等于 prompt_tokens + completion_tokens")

	t.Logf("Usage: prompt=%d, completion=%d, total=%d", promptTokens, completionTokens, totalTokens)
}

// TestExtractMessages 测试从请求 JSON 中提取 messages
func TestExtractMessages(t *testing.T) {
	tests := []struct {
		name     string
		rawJSON  string
		expected []usage.Message
	}{
		{
			name: "单条消息",
			rawJSON: `{
				"messages": [
					{"role": "user", "content": "Hello"}
				]
			}`,
			expected: []usage.Message{
				{Role: "user", Content: "Hello"},
			},
		},
		{
			name: "多条消息",
			rawJSON: `{
				"messages": [
					{"role": "system", "content": "You are a helpful assistant."},
					{"role": "user", "content": "Hello"},
					{"role": "assistant", "content": "Hi there!"},
					{"role": "user", "content": "How are you?"}
				]
			}`,
			expected: []usage.Message{
				{Role: "system", Content: "You are a helpful assistant."},
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi there!"},
				{Role: "user", Content: "How are you?"},
			},
		},
		{
			name:     "缺少 messages 字段",
			rawJSON:  `{"model": "gpt-4"}`,
			expected: []usage.Message{},
		},
		{
			name: "空 messages 数组",
			rawJSON: `{
				"messages": []
			}`,
			expected: []usage.Message{},
		},
		{
			name: "包含空内容的消息（应被跳过）",
			rawJSON: `{
				"messages": [
					{"role": "user", "content": "Hello"},
					{"role": "assistant", "content": ""},
					{"role": "user", "content": "Are you there?"}
				]
			}`,
			expected: []usage.Message{
				{Role: "user", Content: "Hello"},
				{Role: "user", Content: "Are you there?"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractMessages([]byte(tt.rawJSON))
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestStreamUsageWithCompletionsFormat 测试 completions 格式的流式响应 usage 处理
func TestStreamUsageWithCompletionsFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建测试请求
	requestBody := `{
		"model": "gpt-4",
		"prompt": "Once upon a time",
		"stream": true
	}`

	req := httptest.NewRequest("POST", "/v1/completions", bytes.NewBufferString(requestBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	flusher, ok := w.(http.Flusher)
	assert.True(t, ok)

	// 设置 SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	// 对于 completions 格式，messages 从 prompt 转换而来
	messages := []usage.Message{
		{Role: "user", Content: "Once upon a time"},
	}

	accumulator := usage.NewStreamUsageAccumulator("gpt-4", messages)

	// 模拟 chat completions 格式的数据块（内部格式）
	chunks := []string{
		`{"choices":[{"delta":{"content":" there"},"finish_reason":null}]}`,
		`{"choices":[{"delta":{"content":" was"},"finish_reason":null}]}`,
		`{"choices":[{"delta":{"content":" a"},"finish_reason":null}]}`,
		`{"choices":[{"delta":{"content":" princess"},"finish_reason":null}]}`,
	}

	// 发送内容块（转换为 completions 格式）
	for _, chunk := range chunks {
		accumulator.AccumulateChunk([]byte(chunk))
		
		// 转换为 completions 格式
		converted := convertChatCompletionsStreamChunkToCompletions([]byte(chunk))
		if converted != nil {
			c.Writer.Write([]byte("data: " + string(converted) + "\n\n"))
			flusher.Flush()
		}
	}

	// 生成 usage 数据块
	usageChunk, err := accumulator.GenerateUsageChunk()
	assert.NoError(t, err)

	// 转换 usage 数据块为 completions 格式
	convertedUsage := convertChatCompletionsStreamChunkToCompletions(usageChunk)
	assert.NotNil(t, convertedUsage, "转换后的 usage 数据块不应为空")

	c.Writer.Write([]byte("data: " + string(convertedUsage) + "\n\n"))
	flusher.Flush()

	// 发送 [DONE]
	c.Writer.Write([]byte("data: [DONE]\n\n"))
	flusher.Flush()

	// 验证响应
	response := w.Body.String()
	scanner := bufio.NewScanner(strings.NewReader(response))
	var dataLines []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			dataLines = append(dataLines, strings.TrimPrefix(line, "data: "))
		}
	}

	// 验证最后一个是 [DONE]
	assert.Equal(t, "[DONE]", dataLines[len(dataLines)-1])

	// 验证倒数第二个包含 usage
	usageLine := dataLines[len(dataLines)-2]
	usageData := gjson.Parse(usageLine)
	
	// completions 格式应该有 usage 字段
	assert.True(t, usageData.Get("usage").Exists(), "completions 格式应该包含 usage 字段")
	assert.True(t, usageData.Get("usage.prompt_tokens").Exists())
	assert.True(t, usageData.Get("usage.completion_tokens").Exists())
	assert.True(t, usageData.Get("usage.total_tokens").Exists())

	// 验证 object 类型
	assert.Equal(t, "text_completion", usageData.Get("object").String(), "object 应该是 text_completion")
}

// BenchmarkStreamUsageAccumulation 基准测试流式 usage 累积性能
func BenchmarkStreamUsageAccumulation(b *testing.B) {
	messages := []usage.Message{
		{Role: "user", Content: "Hello, how are you?"},
	}

	chunk := []byte(`{"choices":[{"delta":{"content":"Hello"},"finish_reason":null}]}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		accumulator := usage.NewStreamUsageAccumulator("gpt-4", messages)
		for j := 0; j < 100; j++ {
			accumulator.AccumulateChunk(chunk)
		}
		_, _ = accumulator.GenerateUsageChunk()
	}
}

// BenchmarkExtractMessages 基准测试消息提取性能
func BenchmarkExtractMessages(b *testing.B) {
	rawJSON := []byte(`{
		"messages": [
			{"role": "system", "content": "You are a helpful assistant."},
			{"role": "user", "content": "Hello"},
			{"role": "assistant", "content": "Hi there!"},
			{"role": "user", "content": "How are you?"}
		]
	}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = extractMessages(rawJSON)
	}
}

// TestStreamUsageErrorHandling 测试流式 usage 处理的错误情况
func TestStreamUsageErrorHandling(t *testing.T) {
	t.Run("空消息列表", func(t *testing.T) {
		accumulator := usage.NewStreamUsageAccumulator("gpt-4", []usage.Message{})
		
		chunk := []byte(`{"choices":[{"delta":{"content":"Hello"},"finish_reason":null}]}`)
		accumulator.AccumulateChunk(chunk)
		
		usageChunk, err := accumulator.GenerateUsageChunk()
		// 即使消息为空，也应该能生成 usage（prompt_tokens 为 0）
		assert.NoError(t, err)
		assert.NotNil(t, usageChunk)
		
		usageData := gjson.ParseBytes(usageChunk)
		assert.Equal(t, int64(0), usageData.Get("usage.prompt_tokens").Int())
		assert.Greater(t, usageData.Get("usage.completion_tokens").Int(), int64(0))
	})

	t.Run("无内容块", func(t *testing.T) {
		messages := []usage.Message{
			{Role: "user", Content: "Hello"},
		}
		accumulator := usage.NewStreamUsageAccumulator("gpt-4", messages)
		
		// 不累积任何内容块
		usageChunk, err := accumulator.GenerateUsageChunk()
		assert.NoError(t, err)
		assert.NotNil(t, usageChunk)
		
		usageData := gjson.ParseBytes(usageChunk)
		assert.Greater(t, usageData.Get("usage.prompt_tokens").Int(), int64(0))
		assert.Equal(t, int64(0), usageData.Get("usage.completion_tokens").Int())
	})

	t.Run("无效的 JSON 块", func(t *testing.T) {
		messages := []usage.Message{
			{Role: "user", Content: "Hello"},
		}
		accumulator := usage.NewStreamUsageAccumulator("gpt-4", messages)
		
		// 累积无效的 JSON
		invalidChunk := []byte(`{invalid json}`)
		accumulator.AccumulateChunk(invalidChunk)
		
		// 应该能够处理无效 JSON（跳过该块）
		usageChunk, err := accumulator.GenerateUsageChunk()
		assert.NoError(t, err)
		assert.NotNil(t, usageChunk)
	})
}
