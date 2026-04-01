// Package test 提供集成测试
package test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// TestFailover_MultipleInstances 测试多实例故障转移
// 前置条件：使用 Docker Compose 启动多个 CLIProxyAPI 实例
// 测试步骤：
// 1. 启动多个 CLIProxyAPI 实例
// 2. 停止其中一个实例
// 3. 发送请求验证自动转移到其他实例
// 4. 重启实例验证自动恢复
func TestFailover_MultipleInstances(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试（使用 -short 标志）")
	}

	// 检查是否在 Docker Compose 环境中运行
	if os.Getenv("DOCKER_COMPOSE_TEST") == "" {
		t.Skip("需要设置 DOCKER_COMPOSE_TEST=1 环境变量来运行此测试")
	}

	newAPIURL := getEnvOrDefault("NEW_API_URL", "http://localhost:3000")
	newAPIToken := os.Getenv("NEW_API_TOKEN")
	if newAPIToken == "" {
		t.Skip("未设置 NEW_API_TOKEN 环境变量，跳过集成测试")
	}

	// 步骤 1: 验证所有实例都在运行
	t.Log("验证所有 CLIProxyAPI 实例都在运行...")
	successCount := 0
	for i := 0; i < 10; i++ {
		if sendTestRequest(t, newAPIURL, newAPIToken) {
			successCount++
		}
		time.Sleep(100 * time.Millisecond)
	}
	assert.Greater(t, successCount, 7, "大部分请求应该成功")

	// 步骤 2: 停止一个实例
	t.Log("停止 cliproxy-api-1 实例...")
	cmd := exec.Command("docker", "compose", "stop", "cliproxy-api-1")
	err := cmd.Run()
	require.NoError(t, err, "停止实例应该成功")

	// 等待实例完全停止
	time.Sleep(2 * time.Second)

	// 步骤 3: 验证请求自动转移到其他实例
	t.Log("验证请求自动转移到其他实例...")
	successCount = 0
	for i := 0; i < 10; i++ {
		if sendTestRequest(t, newAPIURL, newAPIToken) {
			successCount++
		}
		time.Sleep(100 * time.Millisecond)
	}
	assert.Greater(t, successCount, 7, "即使一个实例停止，大部分请求仍应成功")

	// 步骤 4: 重启实例
	t.Log("重启 cliproxy-api-1 实例...")
	cmd = exec.Command("docker", "compose", "start", "cliproxy-api-1")
	err = cmd.Run()
	require.NoError(t, err, "重启实例应该成功")

	// 等待实例完全启动
	time.Sleep(5 * time.Second)

	// 步骤 5: 验证实例恢复后正常工作
	t.Log("验证实例恢复后正常工作...")
	successCount = 0
	for i := 0; i < 10; i++ {
		if sendTestRequest(t, newAPIURL, newAPIToken) {
			successCount++
		}
		time.Sleep(100 * time.Millisecond)
	}
	assert.Greater(t, successCount, 8, "实例恢复后，所有请求应该成功")
}

// TestFailover_GracefulDegradation 测试优雅降级
// 验证当部分实例不可用时，系统仍能提供服务
func TestFailover_GracefulDegradation(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试（使用 -short 标志）")
	}

	if os.Getenv("DOCKER_COMPOSE_TEST") == "" {
		t.Skip("需要设置 DOCKER_COMPOSE_TEST=1 环境变量来运行此测试")
	}

	newAPIURL := getEnvOrDefault("NEW_API_URL", "http://localhost:3000")
	newAPIToken := os.Getenv("NEW_API_TOKEN")
	if newAPIToken == "" {
		t.Skip("未设置 NEW_API_TOKEN 环境变量，跳过集成测试")
	}

	// 测试场景：逐步停止实例，验证系统仍能工作
	instances := []string{"cliproxy-api-3", "cliproxy-api-2"}

	for _, instance := range instances {
		t.Logf("停止实例: %s", instance)
		cmd := exec.Command("docker", "compose", "stop", instance)
		err := cmd.Run()
		require.NoError(t, err)

		time.Sleep(2 * time.Second)

		// 验证系统仍能响应
		success := sendTestRequest(t, newAPIURL, newAPIToken)
		assert.True(t, success, "即使部分实例停止，系统仍应能响应")
	}

	// 恢复所有实例
	t.Log("恢复所有实例...")
	for _, instance := range instances {
		cmd := exec.Command("docker", "compose", "start", instance)
		_ = cmd.Run()
	}

	time.Sleep(5 * time.Second)
}

// TestFailover_ResponseTime 测试故障转移的响应时间
// 验证故障转移不会显著增加响应时间
func TestFailover_ResponseTime(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试（使用 -short 标志）")
	}

	if os.Getenv("DOCKER_COMPOSE_TEST") == "" {
		t.Skip("需要设置 DOCKER_COMPOSE_TEST=1 环境变量来运行此测试")
	}

	newAPIURL := getEnvOrDefault("NEW_API_URL", "http://localhost:3000")
	newAPIToken := os.Getenv("NEW_API_TOKEN")
	if newAPIToken == "" {
		t.Skip("未设置 NEW_API_TOKEN 环境变量，跳过集成测试")
	}

	// 测量正常情况下的响应时间
	normalTimes := make([]time.Duration, 10)
	for i := 0; i < 10; i++ {
		start := time.Now()
		sendTestRequest(t, newAPIURL, newAPIToken)
		normalTimes[i] = time.Since(start)
		time.Sleep(100 * time.Millisecond)
	}

	avgNormalTime := average(normalTimes)
	t.Logf("正常情况下平均响应时间: %v", avgNormalTime)

	// 停止一个实例
	cmd := exec.Command("docker", "compose", "stop", "cliproxy-api-1")
	_ = cmd.Run()
	time.Sleep(2 * time.Second)

	// 测量故障转移情况下的响应时间
	failoverTimes := make([]time.Duration, 10)
	for i := 0; i < 10; i++ {
		start := time.Now()
		sendTestRequest(t, newAPIURL, newAPIToken)
		failoverTimes[i] = time.Since(start)
		time.Sleep(100 * time.Millisecond)
	}

	avgFailoverTime := average(failoverTimes)
	t.Logf("故障转移情况下平均响应时间: %v", avgFailoverTime)

	// 验证故障转移不会显著增加响应时间（允许增加 50%）
	assert.Less(t, avgFailoverTime, avgNormalTime*3/2, "故障转移不应显著增加响应时间")

	// 恢复实例
	cmd = exec.Command("docker", "compose", "start", "cliproxy-api-1")
	_ = cmd.Run()
	time.Sleep(5 * time.Second)
}

// sendTestRequest 发送测试请求并返回是否成功
func sendTestRequest(t *testing.T, baseURL, token string) bool {
	requestBody := map[string]interface{}{
		"model": "gpt-4",
		"messages": []map[string]string{
			{"role": "user", "content": "Test"},
		},
		"max_tokens": 5,
	}

	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		t.Logf("序列化请求失败: %v", err)
		return false
	}

	req, err := http.NewRequest("POST", baseURL+"/v1/chat/completions", bytes.NewReader(requestJSON))
	if err != nil {
		t.Logf("创建请求失败: %v", err)
		return false
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Logf("发送请求失败: %v", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Logf("请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
		return false
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Logf("读取响应失败: %v", err)
		return false
	}

	// 验证响应包含 usage 字段
	result := gjson.ParseBytes(body)
	if !result.Get("usage").Exists() {
		t.Logf("响应缺少 usage 字段")
		return false
	}

	return true
}

// average 计算平均值
func average(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	var total time.Duration
	for _, d := range durations {
		total += d
	}

	return total / time.Duration(len(durations))
}
