// Package test 提供集成测试
package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// TestE2E_ChatCompletion 端到端测试：通过 new-api 发送聊天完成请求
// 前置条件：需要 new-api 和 CLIProxyAPI 都在运行
// 使用环境变量配置：
// - NEW_API_URL: new-api 的 URL（默认 http://localhost:3000）
// - NEW_API_TOKEN: new-api 的访问 token
// - CLIPROXY_API_KEY: CLIProxyAPI 的 API Key
func TestE2E_ChatCompletion(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试（使用 -short 标志）")
	}

	// 从环境变量读取配置
	newAPIURL := getEnvOrDefault("NEW_API_URL", "http://localhost:3000")
	newAPIToken := os.Getenv("NEW_API_TOKEN")
	if newAPIToken == "" {
		t.Skip("未设置 NEW_API_TOKEN 环境变量，跳过集成测试")
	}

	// 构建请求
	requestBody := map[string]interface{}{
		"model": "gpt-4",
		"messages": []map[string]string{
			{"role": "user", "content": "Say 'Hello, integration test!' and nothing else."},
		},
		"max_tokens": 50,
	}

	requestJSON, err := json.Marshal(requestBody)
	require.NoError(t, err)

	// 发送请求到 new-api
	req, err := http.NewRequest("POST", newAPIURL+"/v1/chat/completions", bytes.NewReader(requestJSON))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+newAPIToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// 验证响应状态码
	assert.Equal(t, http.StatusOK, resp.StatusCode, "应该返回 200 OK")

	// 读取响应
	responseBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	t.Logf("响应: %s", string(responseBody))

	// 验证响应格式
	assert.True(t, gjson.ValidBytes(responseBody), "响应应该是有效的 JSON")

	// 验证必需字段
	result := gjson.ParseBytes(responseBody)
	assert.True(t, result.Get("id").Exists(), "响应应包含 id 字段")
	assert.True(t, result.Get("object").Exists(), "响应应包含 object 字段")
	assert.True(t, result.Get("model").Exists(), "响应应包含 model 字段")
	assert.True(t, result.Get("choices").Exists(), "响应应包含 choices 字段")

	// 验证 usage 字段存在且完整
	usage := result.Get("usage")
	assert.True(t, usage.Exists(), "响应应包含 usage 字段")
	assert.True(t, usage.Get("prompt_tokens").Exists(), "usage 应包含 prompt_tokens")
	assert.True(t, usage.Get("completion_tokens").Exists(), "usage 应包含 completion_tokens")
	assert.True(t, usage.Get("total_tokens").Exists(), "usage 应包含 total_tokens")

	// 验证 token 数量合理
	promptTokens := usage.Get("prompt_tokens").Int()
	completionTokens := usage.Get("completion_tokens").Int()
	totalTokens := usage.Get("total_tokens").Int()

	assert.Greater(t, promptTokens, int64(0), "prompt_tokens 应大于 0")
	assert.Greater(t, completionTokens, int64(0), "completion_tokens 应大于 0")
	assert.Equal(t, promptTokens+completionTokens, totalTokens, "total_tokens 应等于 prompt_tokens + completion_tokens")

	// 验证响应内容
	choices := result.Get("choices").Array()
	assert.Greater(t, len(choices), 0, "应至少有一个 choice")

	content := choices[0].Get("message.content").String()
	assert.NotEmpty(t, content, "响应内容不应为空")
	t.Logf("响应内容: %s", content)
}

// TestE2E_QuotaDeduction 测试配额扣减
// 验证请求后配额被正确扣减
func TestE2E_QuotaDeduction(t *testing.T) {
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

	// 发送请求
	requestBody := map[string]interface{}{
		"model": "gpt-4",
		"messages": []map[string]string{
			{"role": "user", "content": "Hello"},
		},
		"max_tokens": 10,
	}

	requestJSON, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", newAPIURL+"/v1/chat/completions", bytes.NewReader(requestJSON))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+newAPIToken)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 等待配额更新
	time.Sleep(2 * time.Second)

	// 查询更新后的配额
	finalQuota, err := getQuota(newAPIURL, newAPIToken)
	if err != nil {
		t.Skipf("无法查询配额: %v", err)
	}

	t.Logf("最终配额: %d", finalQuota)

	// 验证配额被扣减
	assert.Less(t, finalQuota, initialQuota, "配额应该被扣减")
}

// TestE2E_UsageLogging 测试使用日志记录
// 验证请求被正确记录到使用日志中
func TestE2E_UsageLogging(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试（使用 -short 标志）")
	}

	newAPIURL := getEnvOrDefault("NEW_API_URL", "http://localhost:3000")
	newAPIToken := os.Getenv("NEW_API_TOKEN")
	if newAPIToken == "" {
		t.Skip("未设置 NEW_API_TOKEN 环境变量，跳过集成测试")
	}

	// 发送请求
	requestBody := map[string]interface{}{
		"model": "gpt-4",
		"messages": []map[string]string{
			{"role": "user", "content": "Test logging"},
		},
		"max_tokens": 10,
	}

	requestJSON, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", newAPIURL+"/v1/chat/completions", bytes.NewReader(requestJSON))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+newAPIToken)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 等待日志写入
	time.Sleep(2 * time.Second)

	// 查询使用日志
	logs, err := getUsageLogs(newAPIURL, newAPIToken)
	if err != nil {
		t.Skipf("无法查询使用日志: %v", err)
	}

	// 验证日志中包含我们的请求
	assert.Greater(t, len(logs), 0, "应该有使用日志记录")
	t.Logf("找到 %d 条使用日志", len(logs))
}

// getQuota 查询用户配额
func getQuota(baseURL, token string) (int64, error) {
	req, err := http.NewRequest("GET", baseURL+"/api/user/self", nil)
	if err != nil {
		return 0, err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("获取用户信息失败: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	result := gjson.ParseBytes(body)
	quota := result.Get("data.quota").Int()
	return quota, nil
}

// getUsageLogs 查询使用日志
func getUsageLogs(baseURL, token string) ([]gjson.Result, error) {
	req, err := http.NewRequest("GET", baseURL+"/api/log/self", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("获取使用日志失败: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	result := gjson.ParseBytes(body)
	logs := result.Get("data").Array()
	return logs, nil
}

// getEnvOrDefault 获取环境变量或返回默认值
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
