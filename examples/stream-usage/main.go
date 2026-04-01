// Package main 演示如何使用流式响应 usage 字段处理器
package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/usage"
)

func main() {
	r := gin.Default()

	// 模拟流式聊天完成端点
	r.POST("/v1/chat/completions", handleChatCompletions)

	fmt.Println("Server starting on :8080")
	fmt.Println("Try: curl -X POST http://localhost:8080/v1/chat/completions -H 'Content-Type: application/json' -d '{\"model\":\"gpt-4\",\"messages\":[{\"role\":\"user\",\"content\":\"Hello\"}],\"stream\":true}'")
	r.Run(":8080")
}

func handleChatCompletions(c *gin.Context) {
	var req struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
		Stream bool `json:"stream"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	if !req.Stream {
		c.JSON(400, gin.H{"error": "This example only supports streaming"})
		return
	}

	// 转换消息格式
	messages := make([]usage.Message, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = usage.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// 处理流式响应
	handleStreamingResponse(c, req.Model, messages)
}

func handleStreamingResponse(c *gin.Context, model string, messages []usage.Message) {
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(500, gin.H{"error": "Streaming not supported"})
		return
	}

	// 设置 SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// 创建 usage 累积器
	accumulator := usage.NewStreamUsageAccumulator(model, messages)

	// 模拟流式响应数据块
	chunks := []string{
		`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":null}]}`,
		`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":"!"},"finish_reason":null}]}`,
		`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":" How"},"finish_reason":null}]}`,
		`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":" can"},"finish_reason":null}]}`,
		`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":" I"},"finish_reason":null}]}`,
		`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":" help"},"finish_reason":null}]}`,
		`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":" you"},"finish_reason":null}]}`,
		`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":" today"},"finish_reason":null}]}`,
		`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":"?"},"finish_reason":null}]}`,
	}

	// 发送所有内容块
	for _, chunk := range chunks {
		// 累积内容用于 token 估算
		accumulator.AccumulateChunk([]byte(chunk))

		// 发送数据块到客户端
		fmt.Fprintf(c.Writer, "data: %s\n\n", chunk)
		flusher.Flush()
	}

	// 生成并发送包含 usage 的最后数据块
	usageChunk, err := accumulator.GenerateUsageChunk()
	if err != nil {
		fmt.Printf("Error generating usage chunk: %v\n", err)
	} else {
		fmt.Fprintf(c.Writer, "data: %s\n\n", string(usageChunk))
		flusher.Flush()
	}

	// 发送 [DONE] 标记
	fmt.Fprint(c.Writer, "data: [DONE]\n\n")
	flusher.Flush()
}
