// Package main 演示如何使用 usage.EnsureUsageField 函数
// 确保 API 响应包含完整的 usage 字段
package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/usage"
)

func main() {
	fmt.Println("=== Usage Field Ensurer 示例 ===\n")

	// 示例 1: 响应已包含完整的 usage 字段
	example1()

	// 示例 2: 响应缺失 usage 字段
	example2()

	// 示例 3: 响应包含不完整的 usage 字段
	example3()

	// 示例 4: 多条消息的处理
	example4()
}

// example1 演示当响应已包含完整 usage 字段时的处理
func example1() {
	fmt.Println("示例 1: 响应已包含完整的 usage 字段")
	fmt.Println("----------------------------------------")

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

	messages := []usage.Message{
		{Role: "user", Content: "Hello"},
	}

	result, err := usage.EnsureUsageField(responseJSON, "gpt-4", messages)
	if err != nil {
		log.Fatalf("错误: %v", err)
	}

	printResult(result)
	fmt.Println()
}

// example2 演示当响应缺失 usage 字段时的处理
func example2() {
	fmt.Println("示例 2: 响应缺失 usage 字段（将自动估算）")
	fmt.Println("----------------------------------------")

	responseJSON := []byte(`{
		"id": "chatcmpl-456",
		"object": "chat.completion",
		"created": 1234567890,
		"model": "claude-sonnet-4",
		"choices": [{
			"index": 0,
			"message": {
				"role": "assistant",
				"content": "I can help you with that. What would you like to know?"
			},
			"finish_reason": "stop"
		}]
	}`)

	messages := []usage.Message{
		{Role: "user", Content: "Can you help me?"},
	}

	result, err := usage.EnsureUsageField(responseJSON, "claude-sonnet-4", messages)
	if err != nil {
		log.Fatalf("错误: %v", err)
	}

	printResult(result)
	fmt.Println()
}

// example3 演示当响应包含不完整 usage 字段时的处理
func example3() {
	fmt.Println("示例 3: 响应包含不完整的 usage 字段（将重新估算）")
	fmt.Println("----------------------------------------")

	responseJSON := []byte(`{
		"id": "chatcmpl-789",
		"object": "chat.completion",
		"created": 1234567890,
		"model": "gemini-2.5-pro",
		"choices": [{
			"index": 0,
			"message": {
				"role": "assistant",
				"content": "Gemini is ready to assist you."
			},
			"finish_reason": "stop"
		}],
		"usage": {
			"prompt_tokens": 15
		}
	}`)

	messages := []usage.Message{
		{Role: "user", Content: "Tell me about Gemini"},
	}

	result, err := usage.EnsureUsageField(responseJSON, "gemini-2.5-pro", messages)
	if err != nil {
		log.Fatalf("错误: %v", err)
	}

	printResult(result)
	fmt.Println()
}

// example4 演示多条消息的处理
func example4() {
	fmt.Println("示例 4: 多条消息的 token 估算")
	fmt.Println("----------------------------------------")

	responseJSON := []byte(`{
		"id": "chatcmpl-abc",
		"object": "chat.completion",
		"created": 1234567890,
		"model": "gpt-4",
		"choices": [{
			"index": 0,
			"message": {
				"role": "assistant",
				"content": "Based on your requirements, I recommend using Go for backend development. It offers excellent performance and built-in concurrency support."
			},
			"finish_reason": "stop"
		}]
	}`)

	messages := []usage.Message{
		{Role: "system", Content: "You are a helpful programming assistant."},
		{Role: "user", Content: "What programming language should I use for my backend?"},
		{Role: "assistant", Content: "Could you tell me more about your project requirements?"},
		{Role: "user", Content: "I need high performance and good concurrency support."},
	}

	result, err := usage.EnsureUsageField(responseJSON, "gpt-4", messages)
	if err != nil {
		log.Fatalf("错误: %v", err)
	}

	printResult(result)
	fmt.Println()
}

// printResult 格式化打印结果
func printResult(jsonData []byte) {
	var formatted map[string]interface{}
	if err := json.Unmarshal(jsonData, &formatted); err != nil {
		log.Fatalf("解析 JSON 失败: %v", err)
	}

	// 只打印关键字段
	fmt.Printf("模型: %s\n", formatted["model"])
	
	if choices, ok := formatted["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].(string); ok {
					fmt.Printf("响应内容: %s\n", content)
				}
			}
		}
	}

	if usageData, ok := formatted["usage"].(map[string]interface{}); ok {
		fmt.Println("Usage 字段:")
		fmt.Printf("  - prompt_tokens: %.0f\n", usageData["prompt_tokens"])
		fmt.Printf("  - completion_tokens: %.0f\n", usageData["completion_tokens"])
		fmt.Printf("  - total_tokens: %.0f\n", usageData["total_tokens"])
	}
}
