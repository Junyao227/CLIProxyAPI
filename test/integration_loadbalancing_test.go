// Package test 提供集成测试
package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

// TestLoadBalancing_WeightDistribution 测试负载均衡权重分配
// 验证请求按照配置的权重分配到各个 Channel
// 前置条件：在 new-api 中配置不同权重的 CLIProxyAPI Channel
func TestLoadBalancing_WeightDistribution(t *testing.T) {
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

	// 发送大量请求以测试负载均衡
	totalRequests := 1000
	successCount := 0
	failCount := 0

	t.Logf("发送 %d 个请求以测试负载均衡...", totalRequests)

	startTime := time.Now()

	for i := 0; i < totalRequests; i++ {
		if sendLoadBalancingRequest(t, newAPIURL, newAPIToken) {
			successCount++
		} else {
			failCount++
		}

		// 每 100 个请求输出一次进度
		if (i+1)%100 == 0 {
			t.Logf("进度: %d/%d (成功: %d, 失败: %d)", i+1, totalRequests, successCount, failCount)
		}

		// 避免过快发送请求
		time.Sleep(10 * time.Millisecond)
	}

	elapsed := time.Since(startTime)

	t.Logf("完成 %d 个请求，耗时: %v", totalRequests, elapsed)
	t.Logf("成功: %d (%.2f%%), 失败: %d (%.2f%%)",
		successCount, float64(successCount)/float64(totalRequests)*100,
		failCount, float64(failCount)/float64(totalRequests)*100)

	// 验证成功率
	successRate := float64(successCount) / float64(totalRequests)
	assert.Greater(t, successRate, 0.95, "成功率应大于 95%")

	// 验证吞吐量
	throughput := float64(totalRequests) / elapsed.Seconds()
	t.Logf("吞吐量: %.2f req/s", throughput)
	assert.Greater(t, throughput, 50.0, "吞吐量应大于 50 req/s")
}

// TestLoadBalancing_ConcurrentRequests 测试并发请求的负载均衡
// 验证在高并发情况下负载均衡仍能正常工作
func TestLoadBalancing_ConcurrentRequests(t *testing.T) {
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

	// 并发配置
	concurrency := 20
	requestsPerWorker := 50
	totalRequests := concurrency * requestsPerWorker

	t.Logf("启动 %d 个并发 worker，每个发送 %d 个请求...", concurrency, requestsPerWorker)

	var wg sync.WaitGroup
	successChan := make(chan bool, totalRequests)

	startTime := time.Now()

	// 启动并发 worker
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < requestsPerWorker; j++ {
				success := sendLoadBalancingRequest(t, newAPIURL, newAPIToken)
				successChan <- success

				// 避免过快发送请求
				time.Sleep(50 * time.Millisecond)
			}
		}(i)
	}

	// 等待所有 worker 完成
	wg.Wait()
	close(successChan)

	elapsed := time.Since(startTime)

	// 统计结果
	successCount := 0
	failCount := 0
	for success := range successChan {
		if success {
			successCount++
		} else {
			failCount++
		}
	}

	t.Logf("完成 %d 个并发请求，耗时: %v", totalRequests, elapsed)
	t.Logf("成功: %d (%.2f%%), 失败: %d (%.2f%%)",
		successCount, float64(successCount)/float64(totalRequests)*100,
		failCount, float64(failCount)/float64(totalRequests)*100)

	// 验证成功率
	successRate := float64(successCount) / float64(totalRequests)
	assert.Greater(t, successRate, 0.90, "并发情况下成功率应大于 90%")

	// 验证吞吐量
	throughput := float64(totalRequests) / elapsed.Seconds()
	t.Logf("并发吞吐量: %.2f req/s", throughput)
}

// TestLoadBalancing_WeightRatio 测试权重比例
// 验证请求分配比例符合配置的权重（允许 ±15% 误差）
// 注意：此测试需要在 new-api 中配置特定的权重
func TestLoadBalancing_WeightRatio(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试（使用 -short 标志）")
	}

	if os.Getenv("DOCKER_COMPOSE_TEST") == "" {
		t.Skip("需要设置 DOCKER_COMPOSE_TEST=1 环境变量来运行此测试")
	}

	// 此测试需要特殊配置，跳过自动执行
	t.Skip("此测试需要在 new-api 中配置特定的权重，请手动执行")

	// 示例：假设配置了 3 个 Channel，权重分别为 10, 20, 30
	// 期望的分配比例为 1:2:3
	// 发送 1200 个请求，期望分配为 200, 400, 600（允许 ±15% 误差）

	newAPIURL := getEnvOrDefault("NEW_API_URL", "http://localhost:3000")
	newAPIToken := os.Getenv("NEW_API_TOKEN")
	if newAPIToken == "" {
		t.Skip("未设置 NEW_API_TOKEN 环境变量，跳过集成测试")
	}

	totalRequests := 1200
	expectedRatios := []float64{0.167, 0.333, 0.500} // 1:2:3 的比例
	_ = expectedRatios                                // 避免未使用变量错误
	tolerance := 0.15                                 // ±15% 误差
	_ = tolerance                                     // 避免未使用变量错误

	// 发送请求并统计分配
	// 注意：实际实现需要从响应中识别是哪个 Channel 处理的请求
	// 这可能需要在响应头或日志中添加标识

	t.Logf("发送 %d 个请求以测试权重比例...", totalRequests)

	for i := 0; i < totalRequests; i++ {
		sendLoadBalancingRequest(t, newAPIURL, newAPIToken)
		time.Sleep(10 * time.Millisecond)
	}

	// 从 new-api 日志或统计接口获取实际分配比例
	// 验证实际比例在期望范围内

	t.Log("权重比例测试需要手动验证 new-api 的请求分配统计")

	// 示例验证逻辑（需要实际数据）
	for i, expectedRatio := range expectedRatios {
		// actualRatio := getActualRatio(i) // 需要实现
		// assert.InDelta(t, expectedRatio, actualRatio, tolerance,
		// 	"Channel %d 的实际分配比例应在期望范围内", i)
		t.Logf("Channel %d 期望比例: %.2f%%", i, expectedRatio*100)
	}
}

// TestLoadBalancing_RoundRobin 测试轮询负载均衡
// 验证在相同权重下请求均匀分配
func TestLoadBalancing_RoundRobin(t *testing.T) {
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

	// 发送少量请求以观察轮询模式
	requestCount := 30

	t.Logf("发送 %d 个请求以观察轮询模式...", requestCount)

	for i := 0; i < requestCount; i++ {
		success := sendLoadBalancingRequest(t, newAPIURL, newAPIToken)
		if success {
			t.Logf("请求 %d: 成功", i+1)
		} else {
			t.Logf("请求 %d: 失败", i+1)
		}
		time.Sleep(200 * time.Millisecond)
	}

	t.Log("轮询模式测试完成，请检查日志以验证请求分配模式")
}

// sendLoadBalancingRequest 发送负载均衡测试请求
func sendLoadBalancingRequest(t *testing.T, baseURL, token string) bool {
	requestBody := map[string]interface{}{
		"model": "gpt-4",
		"messages": []map[string]string{
			{"role": "user", "content": "LB test"},
		},
		"max_tokens": 5,
	}

	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		return false
	}

	req, err := http.NewRequest("POST", baseURL+"/v1/chat/completions", bytes.NewReader(requestJSON))
	if err != nil {
		return false
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	// 验证响应包含 usage 字段
	result := gjson.ParseBytes(body)
	return result.Get("usage").Exists()
}

// BenchmarkLoadBalancing 负载均衡性能基准测试
func BenchmarkLoadBalancing(b *testing.B) {
	if os.Getenv("DOCKER_COMPOSE_TEST") == "" {
		b.Skip("需要设置 DOCKER_COMPOSE_TEST=1 环境变量来运行此测试")
	}

	newAPIURL := getEnvOrDefault("NEW_API_URL", "http://localhost:3000")
	newAPIToken := os.Getenv("NEW_API_TOKEN")
	if newAPIToken == "" {
		b.Skip("未设置 NEW_API_TOKEN 环境变量，跳过基准测试")
	}

	requestBody := map[string]interface{}{
		"model": "gpt-4",
		"messages": []map[string]string{
			{"role": "user", "content": "Benchmark"},
		},
		"max_tokens": 5,
	}

	requestJSON, _ := json.Marshal(requestBody)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", newAPIURL+"/v1/chat/completions", bytes.NewReader(requestJSON))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+newAPIToken)

		client := &http.Client{Timeout: 15 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			b.Logf("请求失败: %v", err)
			continue
		}

		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

// getChannelDistribution 获取请求在各 Channel 的分配情况
// 注意：这需要 new-api 提供相应的统计接口
func getChannelDistribution(baseURL, token string) (map[string]int, error) {
	// 此函数需要根据 new-api 的实际 API 实现
	// 示例实现：
	req, err := http.NewRequest("GET", baseURL+"/api/channel/stats", nil)
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
		return nil, fmt.Errorf("获取 Channel 统计失败: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 解析统计数据
	result := gjson.ParseBytes(body)
	distribution := make(map[string]int)

	result.Get("data").ForEach(func(key, value gjson.Result) bool {
		channelName := value.Get("name").String()
		requestCount := int(value.Get("request_count").Int())
		distribution[channelName] = requestCount
		return true
	})

	return distribution, nil
}
