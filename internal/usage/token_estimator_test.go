package usage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewTokenEstimator 测试创建 token 估算器
func TestNewTokenEstimator(t *testing.T) {
	estimator := NewTokenEstimator()
	assert.NotNil(t, estimator)
	assert.NotNil(t, estimator.encoders)
	assert.Equal(t, 0, len(estimator.encoders))
}

// TestEstimatePromptTokens_EmptyMessages 测试空消息列表
func TestEstimatePromptTokens_EmptyMessages(t *testing.T) {
	estimator := NewTokenEstimator()
	tokens, err := estimator.EstimatePromptTokens("gpt-4", []Message{})
	require.NoError(t, err)
	assert.Equal(t, 0, tokens)
}

// TestEstimatePromptTokens_SingleMessage 测试单条消息
func TestEstimatePromptTokens_SingleMessage(t *testing.T) {
	estimator := NewTokenEstimator()
	messages := []Message{
		{Role: "user", Content: "Hello, how are you?"},
	}

	tokens, err := estimator.EstimatePromptTokens("gpt-4", messages)
	require.NoError(t, err)
	// "Hello, how are you?" 大约 5-6 个 token
	// 加上 role (1-2 token) 和格式开销 (4 token) 和基础开销 (3 token)
	// 总计约 13-15 个 token
	assert.Greater(t, tokens, 10)
	assert.Less(t, tokens, 20)
}

// TestEstimatePromptTokens_MultipleMessages 测试多条消息
func TestEstimatePromptTokens_MultipleMessages(t *testing.T) {
	estimator := NewTokenEstimator()
	messages := []Message{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: "What is the capital of France?"},
		{Role: "assistant", Content: "The capital of France is Paris."},
		{Role: "user", Content: "Thank you!"},
	}

	tokens, err := estimator.EstimatePromptTokens("gpt-4", messages)
	require.NoError(t, err)
	// 多条消息应该有更多 tokens
	assert.Greater(t, tokens, 30)
	assert.Less(t, tokens, 60)
}

// TestEstimatePromptTokens_GPTModel 测试 GPT 模型
func TestEstimatePromptTokens_GPTModel(t *testing.T) {
	estimator := NewTokenEstimator()
	messages := []Message{
		{Role: "user", Content: "Hello"},
	}

	models := []string{"gpt-4", "gpt-4-turbo", "gpt-3.5-turbo", "gpt-5"}
	for _, model := range models {
		tokens, err := estimator.EstimatePromptTokens(model, messages)
		require.NoError(t, err, "model: %s", model)
		assert.Greater(t, tokens, 0, "model: %s", model)
	}
}

// TestEstimatePromptTokens_ClaudeModel 测试 Claude 模型
func TestEstimatePromptTokens_ClaudeModel(t *testing.T) {
	estimator := NewTokenEstimator()
	messages := []Message{
		{Role: "user", Content: "Hello"},
	}

	models := []string{"claude-sonnet-4", "claude-opus-4-5", "claude-3-opus"}
	for _, model := range models {
		tokens, err := estimator.EstimatePromptTokens(model, messages)
		require.NoError(t, err, "model: %s", model)
		assert.Greater(t, tokens, 0, "model: %s", model)
	}
}

// TestEstimatePromptTokens_GeminiModel 测试 Gemini 模型
func TestEstimatePromptTokens_GeminiModel(t *testing.T) {
	estimator := NewTokenEstimator()
	messages := []Message{
		{Role: "user", Content: "Hello"},
	}

	models := []string{"gemini-2.5-pro", "gemini-pro", "gemini-1.5-flash"}
	for _, model := range models {
		tokens, err := estimator.EstimatePromptTokens(model, messages)
		require.NoError(t, err, "model: %s", model)
		assert.Greater(t, tokens, 0, "model: %s", model)
	}
}

// TestEstimatePromptTokens_LongContent 测试长文本内容
func TestEstimatePromptTokens_LongContent(t *testing.T) {
	estimator := NewTokenEstimator()
	
	// 创建一个长文本（约 1000 字符）
	longText := ""
	for i := 0; i < 100; i++ {
		longText += "This is a test sentence. "
	}
	
	messages := []Message{
		{Role: "user", Content: longText},
	}

	tokens, err := estimator.EstimatePromptTokens("gpt-4", messages)
	require.NoError(t, err)
	// 长文本应该有更多 tokens（大约 600-700 个）
	assert.Greater(t, tokens, 500)
	assert.Less(t, tokens, 1000)
}

// TestEstimatePromptTokens_CodeContent 测试包含代码的内容
func TestEstimatePromptTokens_CodeContent(t *testing.T) {
	estimator := NewTokenEstimator()
	messages := []Message{
		{Role: "user", Content: `Write a function in Go:
func add(a, b int) int {
    return a + b
}`},
	}

	tokens, err := estimator.EstimatePromptTokens("gpt-4", messages)
	require.NoError(t, err)
	assert.Greater(t, tokens, 20)
	assert.Less(t, tokens, 50)
}

// TestEstimatePromptTokens_MultilingualContent 测试多语言内容
func TestEstimatePromptTokens_MultilingualContent(t *testing.T) {
	estimator := NewTokenEstimator()
	messages := []Message{
		{Role: "user", Content: "你好，世界！Hello, World! Bonjour le monde!"},
	}

	tokens, err := estimator.EstimatePromptTokens("gpt-4", messages)
	require.NoError(t, err)
	assert.Greater(t, tokens, 10)
	assert.Less(t, tokens, 40)
}

// TestEstimateCompletionTokens_EmptyContent 测试空内容
func TestEstimateCompletionTokens_EmptyContent(t *testing.T) {
	estimator := NewTokenEstimator()
	tokens, err := estimator.EstimateCompletionTokens("gpt-4", "")
	require.NoError(t, err)
	assert.Equal(t, 0, tokens)
}

// TestEstimateCompletionTokens_SimpleText 测试简单文本
func TestEstimateCompletionTokens_SimpleText(t *testing.T) {
	estimator := NewTokenEstimator()
	content := "The capital of France is Paris."

	tokens, err := estimator.EstimateCompletionTokens("gpt-4", content)
	require.NoError(t, err)
	// "The capital of France is Paris." 大约 7-8 个 token
	assert.Greater(t, tokens, 5)
	assert.Less(t, tokens, 12)
}

// TestEstimateCompletionTokens_LongText 测试长文本
func TestEstimateCompletionTokens_LongText(t *testing.T) {
	estimator := NewTokenEstimator()
	
	// 创建一个长文本
	longText := ""
	for i := 0; i < 50; i++ {
		longText += "This is a response sentence. "
	}

	tokens, err := estimator.EstimateCompletionTokens("gpt-4", longText)
	require.NoError(t, err)
	assert.Greater(t, tokens, 200)
	assert.Less(t, tokens, 500)
}

// TestEstimateCompletionTokens_AllModels 测试所有支持的模型
func TestEstimateCompletionTokens_AllModels(t *testing.T) {
	estimator := NewTokenEstimator()
	content := "Hello, this is a test response."

	models := []string{
		"gpt-4",
		"gpt-3.5-turbo",
		"claude-sonnet-4",
		"claude-opus-4-5",
		"gemini-2.5-pro",
		"gemini-pro",
	}

	for _, model := range models {
		tokens, err := estimator.EstimateCompletionTokens(model, content)
		require.NoError(t, err, "model: %s", model)
		assert.Greater(t, tokens, 5, "model: %s", model)
		assert.Less(t, tokens, 15, "model: %s", model)
	}
}

// TestEstimateCompletionTokens_CodeResponse 测试代码响应
func TestEstimateCompletionTokens_CodeResponse(t *testing.T) {
	estimator := NewTokenEstimator()
	content := `Here is the function:

func add(a, b int) int {
    return a + b
}

This function adds two integers.`

	tokens, err := estimator.EstimateCompletionTokens("gpt-4", content)
	require.NoError(t, err)
	assert.Greater(t, tokens, 25)
	assert.Less(t, tokens, 60)
}

// TestEstimateCompletionTokens_MultilingualResponse 测试多语言响应
func TestEstimateCompletionTokens_MultilingualResponse(t *testing.T) {
	estimator := NewTokenEstimator()
	content := "你好！这是一个测试响应。Hello! This is a test response."

	tokens, err := estimator.EstimateCompletionTokens("gpt-4", content)
	require.NoError(t, err)
	assert.Greater(t, tokens, 10)
	assert.Less(t, tokens, 40)
}

// TestCodecCaching 测试编码器缓存
func TestCodecCaching(t *testing.T) {
	estimator := NewTokenEstimator()
	
	// 第一次调用应该创建编码器
	_, err := estimator.EstimatePromptTokens("gpt-4", []Message{{Role: "user", Content: "Hello"}})
	require.NoError(t, err)
	assert.Equal(t, 1, len(estimator.encoders))

	// 第二次调用相同模型应该使用缓存
	_, err = estimator.EstimatePromptTokens("gpt-4", []Message{{Role: "user", Content: "Hi"}})
	require.NoError(t, err)
	assert.Equal(t, 1, len(estimator.encoders))

	// 调用不同模型应该创建新编码器
	_, err = estimator.EstimatePromptTokens("claude-sonnet-4", []Message{{Role: "user", Content: "Hello"}})
	require.NoError(t, err)
	assert.Equal(t, 2, len(estimator.encoders))
}

// TestModelNameNormalization 测试模型名称标准化
func TestModelNameNormalization(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"GPT-4", "gpt-4"},
		{"  Claude-Sonnet-4  ", "claude-sonnet-4"},
		{"GEMINI-2.5-PRO", "gemini-2.5-pro"},
		{"gpt-4-turbo", "gpt-4-turbo"},
	}

	for _, tt := range tests {
		result := normalizeModelName(tt.input)
		assert.Equal(t, tt.expected, result, "input: %s", tt.input)
	}
}

// TestModelTypeDetection 测试模型类型检测
func TestModelTypeDetection(t *testing.T) {
	tests := []struct {
		model    string
		isGPT    bool
		isClaude bool
		isGemini bool
	}{
		{"gpt-4", true, false, false},
		{"gpt-3.5-turbo", true, false, false},
		{"o1-preview", true, false, false},
		{"claude-sonnet-4", false, true, false},
		{"claude-opus-4-5", false, true, false},
		{"gemini-2.5-pro", false, false, true},
		{"gemini-pro", false, false, true},
		{"unknown-model", false, false, false},
	}

	for _, tt := range tests {
		normalized := normalizeModelName(tt.model)
		assert.Equal(t, tt.isGPT, isGPTModel(normalized), "model: %s, isGPT", tt.model)
		assert.Equal(t, tt.isClaude, isClaudeModel(normalized), "model: %s, isClaude", tt.model)
		assert.Equal(t, tt.isGemini, isGeminiModel(normalized), "model: %s, isGemini", tt.model)
	}
}

// TestEstimateTokens_Consistency 测试估算一致性
// 相同的输入应该产生相同的输出
func TestEstimateTokens_Consistency(t *testing.T) {
	estimator := NewTokenEstimator()
	messages := []Message{
		{Role: "user", Content: "What is the meaning of life?"},
	}

	// 多次估算应该得到相同结果
	tokens1, err1 := estimator.EstimatePromptTokens("gpt-4", messages)
	require.NoError(t, err1)

	tokens2, err2 := estimator.EstimatePromptTokens("gpt-4", messages)
	require.NoError(t, err2)

	tokens3, err3 := estimator.EstimatePromptTokens("gpt-4", messages)
	require.NoError(t, err3)

	assert.Equal(t, tokens1, tokens2)
	assert.Equal(t, tokens2, tokens3)
}

// TestEstimateTokens_Accuracy 测试估算准确性
// 验证估算值在合理范围内
func TestEstimateTokens_Accuracy(t *testing.T) {
	estimator := NewTokenEstimator()
	
	// 已知的测试用例（基于 OpenAI 的 token 计数）
	testCases := []struct {
		content      string
		minTokens    int
		maxTokens    int
		description  string
	}{
		{"Hello", 1, 2, "单个单词"},
		{"Hello, world!", 3, 5, "简单句子"},
		{"The quick brown fox jumps over the lazy dog", 9, 12, "标准句子"},
		{"1234567890", 2, 4, "数字"},
		{"你好世界", 4, 8, "中文"},
	}

	for _, tc := range testCases {
		tokens, err := estimator.EstimateCompletionTokens("gpt-4", tc.content)
		require.NoError(t, err, "content: %s", tc.description)
		assert.GreaterOrEqual(t, tokens, tc.minTokens, "content: %s", tc.description)
		assert.LessOrEqual(t, tokens, tc.maxTokens, "content: %s", tc.description)
	}
}
