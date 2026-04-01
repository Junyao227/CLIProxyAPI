// Package main 演示如何使用 token 估算器
package main

import (
	"fmt"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/usage"
)

func main() {
	fmt.Println("=== Token 估算器示例 ===\n")

	// 创建 token 估算器
	estimator := usage.NewTokenEstimator()

	// 示例 1: 简单对话
	fmt.Println("示例 1: 简单对话")
	simpleMessages := []usage.Message{
		{Role: "user", Content: "Hello, how are you?"},
	}
	demonstrateEstimation(estimator, "gpt-4", simpleMessages, "I'm doing well, thank you!")

	// 示例 2: 多轮对话
	fmt.Println("\n示例 2: 多轮对话")
	multiMessages := []usage.Message{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: "What is the capital of France?"},
		{Role: "assistant", Content: "The capital of France is Paris."},
		{Role: "user", Content: "What about Germany?"},
	}
	demonstrateEstimation(estimator, "gpt-4", multiMessages, "The capital of Germany is Berlin.")

	// 示例 3: 不同模型对比
	fmt.Println("\n示例 3: 不同模型对比")
	testMessages := []usage.Message{
		{Role: "user", Content: "Explain quantum computing in simple terms."},
	}
	models := []string{"gpt-4", "claude-sonnet-4", "gemini-2.5-pro"}
	for _, model := range models {
		promptTokens, _ := estimator.EstimatePromptTokens(model, testMessages)
		fmt.Printf("  %s: %d prompt tokens\n", model, promptTokens)
	}

	// 示例 4: 长文本估算
	fmt.Println("\n示例 4: 长文本估算")
	longText := generateLongText(100)
	longMessages := []usage.Message{
		{Role: "user", Content: longText},
	}
	demonstrateEstimation(estimator, "gpt-4", longMessages, "I understand your request.")

	// 示例 5: 代码内容估算
	fmt.Println("\n示例 5: 代码内容估算")
	codeMessages := []usage.Message{
		{Role: "user", Content: "Write a function to calculate fibonacci numbers"},
	}
	codeResponse := `Here's a function to calculate Fibonacci numbers:

func fibonacci(n int) int {
    if n <= 1 {
        return n
    }
    return fibonacci(n-1) + fibonacci(n-2)
}`
	demonstrateEstimation(estimator, "gpt-4", codeMessages, codeResponse)

	// 示例 6: 多语言内容
	fmt.Println("\n示例 6: 多语言内容")
	multilingualMessages := []usage.Message{
		{Role: "user", Content: "你好！请用中文、英文和法文说'欢迎'"},
	}
	multilingualResponse := "中文：欢迎\nEnglish: Welcome\nFrançais: Bienvenue"
	demonstrateEstimation(estimator, "gpt-4", multilingualMessages, multilingualResponse)

	fmt.Println("\n=== 演示完成 ===")
}

// demonstrateEstimation 演示 token 估算过程
func demonstrateEstimation(estimator *usage.TokenEstimator, model string, messages []usage.Message, completion string) {
	// 估算 prompt tokens
	promptTokens, err := estimator.EstimatePromptTokens(model, messages)
	if err != nil {
		fmt.Printf("  错误: %v\n", err)
		return
	}

	// 估算 completion tokens
	completionTokens, err := estimator.EstimateCompletionTokens(model, completion)
	if err != nil {
		fmt.Printf("  错误: %v\n", err)
		return
	}

	// 计算总 tokens
	totalTokens := promptTokens + completionTokens

	// 显示结果
	fmt.Printf("  模型: %s\n", model)
	fmt.Printf("  Prompt tokens: %d\n", promptTokens)
	fmt.Printf("  Completion tokens: %d\n", completionTokens)
	fmt.Printf("  Total tokens: %d\n", totalTokens)
}

// generateLongText 生成长文本用于测试
func generateLongText(sentences int) string {
	text := ""
	for i := 0; i < sentences; i++ {
		text += "This is a test sentence for token estimation. "
	}
	return text
}
