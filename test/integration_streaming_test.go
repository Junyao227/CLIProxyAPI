// Package test 提供集成测试
package test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// TestStreaming_SSEFormat 测试流式响应的 SSE 格式
// 验证返回的是正确的 Server-Sent Events 格式
func TestStreaming_SSEFormat(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试（使用 -short 标志）")
	}

	newAPIURL := getEnvOrDefault("NEW_API_URL", "http://localhost:3000")
	newAPIToken := os.Getenv("NEW_API_TOKEN")
	if newAPIToken == "" {
		t.Skip("未设置 NEW_API_TOKEN 环境变量，跳过集成测试")
	}

	// 构建流式请求
	requestBody := map[string]interface{}{
		"model": "gpt-4",
		"messages": []map[string]string{
			{"role": "user", "content": "Count from 1 to 5, one number per line."},
		},
		"stream":     true,
		"max_tokens": 50,
	}

	requestJSON, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", newAPIURL+"/v1/chat/completions", bytes.NewReader(requestJSON))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+newAPIToken)
	req.Header.Set("Accept", "text/event-stream")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// 验证响应状态码
	assert.Equal(t, http.StatusOK, resp.StatusCode, "应该返回 200 OK")

	// 验证 Content-Type
	contentType := resp.Header.Get("Content-Type")
	assert.Contains(t, contentType, "text/event-stream", "Content-Type 应该是 text/event-stream")

	// 读取流式响应
	reader := bufio.NewReader(resp.Body)
	chunkCount := 0
	hasUsageChunk := false

	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		// 跳过空行
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 验证 SSE 格式
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			// 检查是否是结束标记
			if data == "[DONE]" {
				t.Log("收到流式响应结束标记")
				break
			}

			// 验证是有效的 JSON
			assert.True(t, gjson.Valid(data), "SSE 数据应该是有效的 JSON")

			result := gjson.Parse(data)

			// 验证基本字段
			assert.True(t, result.Get("id").Exists(), "每个数据块应包含 id")
			assert.True(t, result.Get("object").Exists(), "每个数据块应包含 object")
			assert.Equal(t, "chat.completion.chunk", result.Get("object").String())

			// 检查是否包含 usage 字段
			if result.Get("usage").Exists() {
				hasUsageChunk = true
				t.Log("收到包含 usage 字段的数据块")

				usage := result.Get("usage")
				assert.True(t, usage.Get("prompt_tokens").Exists(), "usage 应包含 prompt_tokens")
				assert.True(t, usage.Get("completion_tokens").Exists(), "usage 应包含 completion_tokens")
				assert.True(t, usage.Get("total_tokens").Exists(), "usage 应包含 total_tokens")

				promptTokens := usage.Get("prompt_tokens").Int()
				completionTokens := usage.Get("completion_tokens").Int()
				totalTokens := usage.Get("total_tokens").Int()

				t.Logf("Usage: prompt=%d, completion=%d, total=%d",
					promptTokens, completionTokens, totalTokens)

				assert.Greater(t, promptTokens, int64(0), "prompt_tokens 应大于 0")
				assert.Greater(t, completionTokens, int64(0), "completion_tokens 应大于 0")
				assert.Equal(t, promptTokens+completionTokens, totalTokens)
			}

			chunkCount++
		}
	}

	t.Logf("收到 %d 个数据块", chunkCount)

	// 验证收到了数据块
	assert.Greater(t, chunkCount, 0, "应该收到至少一个数据块")

	// 验证最后一个数据块包含 usage 信息
	assert.True(t, hasUsageChunk, "流式响应应包含 usage 信息")
}

// TestStreaming_ContentAccumulation 测试流式响应内容累积
// 验证可以正确累积流式响应的内容
func TestStreaming_ContentAccumulation(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试（使用 -short 标志）")
	}

	newAPIURL := getEnvOrDefault("NEW_API_URL", "http://localhost:3000")
	newAPIToken := os.Getenv("NEW_API_TOKEN")
	if newAPIToken == "" {
		t.Skip("未设置 NEW_API_TOKEN 环境变量，跳过集成测试")
	}

	// 构建流式请求
	requestBody := map[string]interface{}{
		"model": "gpt-4",
		"messages": []map[string]string{
			{"role": "user", "content": "Say 'Hello, streaming test!' and nothing else."},
		},
		"stream":     true,
		"max_tokens": 20,
	}

	requestJSON, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", newAPIURL+"/v1/chat/completions", bytes.NewReader(requestJSON))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+newAPIToken)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 累积内容
	var accumulatedContent strings.Builder
	reader := bufio.NewReader(resp.Body)

	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		result := gjson.Parse(data)
		choices := result.Get("choices").Array()

		if len(choices) > 0 {
			delta := choices[0].Get("delta")
			if content := delta.Get("content"); content.Exists() {
				accumulatedContent.WriteString(content.String())
			}
		}
	}

	finalContent := accumulatedContent.String()
	t.Logf("累积的内容: %s", finalContent)

	// 验证累积的内容不为空
	assert.NotEmpty(t, finalContent, "累积的内容不应为空")
}

// TestStreaming_QuotaDeduction 测试流式响应后的配额扣减
// 验证配额在流式响应结束后被正确扣减
func TestStreaming_QuotaDeduction(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试（使用 -short 标志）")
	}

	newAPIURL := getEnvOrDefault("NEW_API_URL", "http://localhost:3000")
	newAPIToken := os.Getenv("NEW_API_TOKEN")
	if newAPIToken == "" {
		t.Skip("未设置 NEW_API_TOKEN 环境变量，跳过集成测试")
	}

	// 查询初始配额
	initialQuota, err := getQuota(newAPIURL, newAPIToken)
	if err != nil {
		t.Skipf("无法查询配额: %v", err)
	}

	t.Logf("初始配额: %d", initialQuota)

	// 发送流式请求
	requestBody := map[string]interface{}{
		"model": "gpt-4",
		"messages": []map[string]string{
			{"role": "user", "content": "Hello"},
		},
		"stream":     true,
		"max_tokens": 10,
	}

	requestJSON, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", newAPIURL+"/v1/chat/completions", bytes.NewReader(requestJSON))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+newAPIToken)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 读取完整的流式响应
	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}
		}
	}

	// 等待配额更新
	time.Sleep(3 * time.Second)

	// 查询更新后的配额
	finalQuota, err := getQuota(newAPIURL, newAPIToken)
	if err != nil {
		t.Skipf("无法查询配额: %v", err)
	}

	t.Logf("最终配额: %d", finalQuota)

	// 验证配额被扣减
	assert.Less(t, finalQuota, initialQuota, "流式响应结束后配额应该被扣减")
}

// TestStreaming_ErrorHandling 测试流式响应的错误处理
// 验证在流式响应过程中发生错误时的处理
func TestStreaming_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试（使用 -short 标志）")
	}

	newAPIURL := getEnvOrDefault("NEW_API_URL", "http://localhost:3000")
	newAPIToken := os.Getenv("NEW_API_TOKEN")
	if newAPIToken == "" {
		t.Skip("未设置 NEW_API_TOKEN 环境变量，跳过集成测试")
	}

	// 发送一个可能导致错误的请求（例如无效的模型）
	requestBody := map[string]interface{}{
		"model": "invalid-model-name",
		"messages": []map[string]string{
			{"role": "user", "content": "Test"},
		},
		"stream": true,
	}

	requestJSON, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", newAPIURL+"/v1/chat/completions", bytes.NewReader(requestJSON))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+newAPIToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// 验证错误响应
	// 可能是 400 Bad Request 或其他错误状态码
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Logf("错误响应: %s", string(body))

		// 验证错误响应格式
		result := gjson.ParseBytes(body)
		assert.True(t, result.Get("error").Exists(), "错误响应应包含 error 字段")
	}
}

// TestStreaming_LongResponse 测试长流式响应
// 验证可以正确处理较长的流式响应
func TestStreaming_LongResponse(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试（使用 -short 标志）")
	}

	newAPIURL := getEnvOrDefault("NEW_API_URL", "http://localhost:3000")
	newAPIToken := os.Getenv("NEW_API_TOKEN")
	if newAPIToken == "" {
		t.Skip("未设置 NEW_API_TOKEN 环境变量，跳过集成测试")
	}

	// 构建请求以生成较长的响应
	requestBody := map[string]interface{}{
		"model": "gpt-4",
		"messages": []map[string]string{
			{"role": "user", "content": "Write a short paragraph about artificial intelligence."},
		},
		"stream":     true,
		"max_tokens": 200,
	}

	requestJSON, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", newAPIURL+"/v1/chat/completions", bytes.NewReader(requestJSON))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+newAPIToken)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 读取流式响应并统计
	reader := bufio.NewReader(resp.Body)
	chunkCount := 0
	var totalCompletionTokens int64

	startTime := time.Now()

	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		result := gjson.Parse(data)

		// 检查 usage 字段
		if usage := result.Get("usage"); usage.Exists() {
			totalCompletionTokens = usage.Get("completion_tokens").Int()
		}

		chunkCount++
	}

	elapsed := time.Since(startTime)

	t.Logf("收到 %d 个数据块，耗时: %v", chunkCount, elapsed)
	t.Logf("总 completion tokens: %d", totalCompletionTokens)

	// 验证收到了足够的数据块
	assert.Greater(t, chunkCount, 10, "长响应应该有多个数据块")
	assert.Greater(t, totalCompletionTokens, int64(50), "长响应应该有较多的 tokens")
}

// BenchmarkStreaming 流式响应性能基准测试
func BenchmarkStreaming(b *testing.B) {
	newAPIURL := getEnvOrDefault("NEW_API_URL", "http://localhost:3000")
	newAPIToken := os.Getenv("NEW_API_TOKEN")
	if newAPIToken == "" {
		b.Skip("未设置 NEW_API_TOKEN 环境变量，跳过基准测试")
	}

	requestBody := map[string]interface{}{
		"model": "gpt-4",
		"messages": []map[string]string{
			{"role": "user", "content": "Say hello"},
		},
		"stream":     true,
		"max_tokens": 10,
	}

	requestJSON, _ := json.Marshal(requestBody)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", newAPIURL+"/v1/chat/completions", bytes.NewReader(requestJSON))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+newAPIToken)

		client := &http.Client{Timeout: 60 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			b.Logf("请求失败: %v", err)
			continue
		}

		// 读取完整的流式响应
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}
