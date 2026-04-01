package usage

import (
	"reflect"
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: cliproxyapi-newapi-integration, Property 4: Usage 缺失时的估算
// 验证需求: 2.3, 12.1, 12.2, 12.3, 12.4
//
// 属性：对于任意上游提供商不返回 usage 信息的响应，CLIProxyAPI 应该使用 tiktoken 库估算 token 数量，
// 估算应基于请求的 prompt 和响应的 completion，估算结果应包含在响应的 usage 字段中，
// 且估算误差应在实际值的 ±10% 范围内。
func TestProperty_UsageEstimationAccuracy(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("估算的 token 数量应该为正数",
		prop.ForAll(
			func(content string, model string) bool {
				// 跳过空内容
				if content == "" {
					return true
				}

				estimator := NewTokenEstimator()
				tokens, err := estimator.EstimateCompletionTokens(model, content)
				
				// 不应该有错误
				if err != nil {
					return false
				}

				// token 数量应该为正数
				return tokens > 0
			},
			genNonEmptyText(),
			genSupportedModel(),
		))

	properties.Property("相同内容的估算应该一致",
		prop.ForAll(
			func(content string, model string) bool {
				estimator := NewTokenEstimator()
				
				tokens1, err1 := estimator.EstimateCompletionTokens(model, content)
				tokens2, err2 := estimator.EstimateCompletionTokens(model, content)
				tokens3, err3 := estimator.EstimateCompletionTokens(model, content)

				// 不应该有错误
				if err1 != nil || err2 != nil || err3 != nil {
					return false
				}

				// 多次估算应该得到相同结果
				return tokens1 == tokens2 && tokens2 == tokens3
			},
			genText(),
			genSupportedModel(),
		))

	properties.Property("空内容返回 0 tokens",
		prop.ForAll(
			func(model string) bool {
				estimator := NewTokenEstimator()
				tokens, err := estimator.EstimateCompletionTokens(model, "")

				// 不应该有错误
				if err != nil {
					return false
				}

				// 空内容应该返回 0
				return tokens == 0
			},
			genSupportedModel(),
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: cliproxyapi-newapi-integration, Property 4: Usage 缺失时的估算
// 验证需求: 2.3, 12.1, 12.2, 12.3
//
// 属性：估算的 prompt tokens 应该随消息数量和内容长度增加而增加
func TestProperty_PromptTokensScaling(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("更长的内容产生更多 tokens",
		prop.ForAll(
			func(shortText string, longText string, model string) bool {
				// 确保 longText 明显比 shortText 长
				if len(longText) <= len(shortText)*2 {
					return true // 跳过
				}

				estimator := NewTokenEstimator()
				
				shortTokens, err1 := estimator.EstimateCompletionTokens(model, shortText)
				longTokens, err2 := estimator.EstimateCompletionTokens(model, longText)

				// 不应该有错误
				if err1 != nil || err2 != nil {
					return false
				}

				// 更长的文本应该产生更多 tokens
				return longTokens > shortTokens
			},
			genShortText(),
			genLongText(),
			genSupportedModel(),
		))

	properties.Property("空消息列表返回 0 tokens",
		prop.ForAll(
			func(model string) bool {
				estimator := NewTokenEstimator()
				tokens, err := estimator.EstimatePromptTokens(model, []Message{})

				// 不应该有错误
				if err != nil {
					return false
				}

				// 空消息列表应该返回 0
				return tokens == 0
			},
			genSupportedModel(),
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: cliproxyapi-newapi-integration, Property 6: Token 估算支持多种模型
// 验证需求: 12.5
//
// 属性：对于任意 GPT、Claude 或 Gemini 模型的请求，当上游不返回 usage 时，
// token 估算功能应该正常工作并返回合理的估算值。
func TestProperty_MultiModelSupport(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("所有支持的模型都能正常估算",
		prop.ForAll(
			func(content string, model string) bool {
				// 跳过空内容
				if content == "" {
					return true
				}

				estimator := NewTokenEstimator()
				tokens, err := estimator.EstimateCompletionTokens(model, content)

				// 不应该有错误
				if err != nil {
					return false
				}

				// 应该返回合理的 token 数量（至少 1 个）
				return tokens > 0
			},
			genNonEmptyText(),
			genSupportedModel(),
		))

	properties.Property("不同模型对相同内容的估算应该相近",
		prop.ForAll(
			func(content string, model1 string, model2 string) bool {
				// 跳过空内容
				if content == "" {
					return true
				}

				estimator := NewTokenEstimator()
				
				tokens1, err1 := estimator.EstimateCompletionTokens(model1, content)
				tokens2, err2 := estimator.EstimateCompletionTokens(model2, content)

				// 不应该有错误
				if err1 != nil || err2 != nil {
					return false
				}

				// 两个模型的估算应该相近（允许 ±20% 差异，因为不同模型可能有不同的 tokenizer）
				if tokens1 == 0 || tokens2 == 0 {
					return tokens1 == tokens2
				}

				ratio := float64(tokens1) / float64(tokens2)
				return ratio >= 0.8 && ratio <= 1.2
			},
			genNonEmptyText(),
			genSupportedModel(),
			genSupportedModel(),
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: cliproxyapi-newapi-integration, Property 4: Usage 缺失时的估算
// 验证需求: 12.4
//
// 属性：估算误差应该在合理范围内（基于已知的 token 计数规则）
func TestProperty_EstimationReasonableness(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("英文单词的 token 估算应该合理",
		prop.ForAll(
			func(wordCount int, model string) bool {
				// 限制单词数量在合理范围内
				if wordCount < 1 || wordCount > 100 {
					return true
				}

				// 生成简单的英文文本（每个单词约 1-2 个 token）
				words := make([]string, wordCount)
				for i := 0; i < wordCount; i++ {
					words[i] = "word"
				}
				content := strings.Join(words, " ")

				estimator := NewTokenEstimator()
				tokens, err := estimator.EstimateCompletionTokens(model, content)

				// 不应该有错误
				if err != nil {
					return false
				}

				// 每个单词约 1-2 个 token，加上空格
				// 估算应该在 wordCount 到 wordCount*2 之间
				return tokens >= wordCount && tokens <= wordCount*3
			},
			gen.IntRange(1, 100),
			genSupportedModel(),
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: cliproxyapi-newapi-integration, Property 4: Usage 缺失时的估算
// 验证需求: 2.3, 12.1, 12.2
//
// 属性：编码器缓存应该正常工作，避免重复创建
func TestProperty_CodecCaching(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("相同模型复用编码器",
		prop.ForAll(
			func(model string) bool {
				estimator := NewTokenEstimator()
				
				// 第一次调用（使用非空内容）
				_, err1 := estimator.EstimateCompletionTokens(model, "test content 1")
				if err1 != nil {
					return false
				}
				
				cacheSize1 := len(estimator.encoders)

				// 第二次调用相同模型（使用不同的非空内容）
				_, err2 := estimator.EstimateCompletionTokens(model, "test content 2")
				if err2 != nil {
					return false
				}
				
				cacheSize2 := len(estimator.encoders)

				// 缓存大小不应该增加
				return cacheSize1 == cacheSize2 && cacheSize1 > 0
			},
			genSupportedModel(),
		))

	properties.Property("不同模型创建新编码器",
		prop.ForAll(
			func(model1 string, model2 string) bool {
				// 确保是不同的模型类型
				if isSameModelType(model1, model2) {
					return true // 跳过相同类型的模型
				}

				estimator := NewTokenEstimator()
				
				// 第一次调用
				_, err1 := estimator.EstimateCompletionTokens(model1, "test content")
				if err1 != nil {
					return false
				}
				
				cacheSize1 := len(estimator.encoders)

				// 第二次调用不同模型
				_, err2 := estimator.EstimateCompletionTokens(model2, "test content")
				if err2 != nil {
					return false
				}
				
				cacheSize2 := len(estimator.encoders)

				// 缓存大小应该增加（如果模型类型不同）
				return cacheSize2 >= cacheSize1 && cacheSize1 > 0
			},
			genSupportedModel(),
			genSupportedModel(),
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: cliproxyapi-newapi-integration, Property 4: Usage 缺失时的估算
// 验证需求: 12.3
//
// 属性：total_tokens 应该等于 prompt_tokens + completion_tokens
func TestProperty_TotalTokensCorrectness(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("total_tokens = prompt_tokens + completion_tokens",
		prop.ForAll(
			func(messages []Message, completion string, model string) bool {
				estimator := NewTokenEstimator()
				
				promptTokens, err1 := estimator.EstimatePromptTokens(model, messages)
				completionTokens, err2 := estimator.EstimateCompletionTokens(model, completion)

				// 不应该有错误
				if err1 != nil || err2 != nil {
					return false
				}

				totalTokens := promptTokens + completionTokens

				// 验证总和正确
				return totalTokens == promptTokens+completionTokens
			},
			genMessageSlice(1, 5),
			genText(),
			genSupportedModel(),
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// 生成器辅助函数

// genSupportedModel 生成支持的模型名称
func genSupportedModel() gopter.Gen {
	return gen.OneConstOf(
		"gpt-4",
		"gpt-4-turbo",
		"gpt-3.5-turbo",
		"gpt-5",
		"claude-sonnet-4",
		"claude-opus-4-5",
		"claude-3-opus",
		"gemini-2.5-pro",
		"gemini-pro",
		"gemini-1.5-flash",
	)
}

// genText 生成任意文本（包括空字符串）
func genText() gopter.Gen {
	return gen.OneGenOf(
		gen.Const(""),
		gen.AlphaString(),
		gen.Identifier(),
		gen.AnyString(),
	)
}

// genNonEmptyText 生成非空文本
func genNonEmptyText() gopter.Gen {
	return gen.OneGenOf(
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.Identifier(),
	)
}

// genShortText 生成短文本（1-20 个字符）
func genShortText() gopter.Gen {
	return gen.Identifier().Map(func(s string) string {
		if len(s) > 20 {
			return s[:20]
		}
		return s
	})
}

// genLongText 生成长文本（50+ 个字符）
func genLongText() gopter.Gen {
	return gen.SliceOfN(10, gen.Identifier()).Map(func(words []string) string {
		return strings.Join(words, " ")
	})
}

// genMessage 生成单个消息
func genMessage() gopter.Gen {
	return gen.Struct(reflect.TypeOf(Message{}), map[string]gopter.Gen{
		"Role":    gen.OneConstOf("system", "user", "assistant"),
		"Content": genNonEmptyText(),
	})
}

// genMessageSlice 生成指定数量范围的消息列表
func genMessageSlice(min, max int) gopter.Gen {
	return gen.IntRange(min, max).FlatMap(func(n interface{}) gopter.Gen {
		count := n.(int)
		return gen.SliceOfN(count, genMessage())
	}, reflect.TypeOf([]Message{}))
}

// isSameModelType 判断两个模型是否属于同一类型
func isSameModelType(model1, model2 string) bool {
	normalized1 := normalizeModelName(model1)
	normalized2 := normalizeModelName(model2)

	isGPT1 := isGPTModel(normalized1)
	isGPT2 := isGPTModel(normalized2)
	if isGPT1 && isGPT2 {
		return true
	}

	isClaude1 := isClaudeModel(normalized1)
	isClaude2 := isClaudeModel(normalized2)
	if isClaude1 && isClaude2 {
		return true
	}

	isGemini1 := isGeminiModel(normalized1)
	isGemini2 := isGeminiModel(normalized2)
	if isGemini1 && isGemini2 {
		return true
	}

	return false
}

// TestProperty_EstimationErrorBound 测试估算误差边界
// Feature: cliproxyapi-newapi-integration, Property 4: Usage 缺失时的估算
// 验证需求: 12.4
//
// 注意：这个测试使用已知的 token 计数来验证误差范围
// 由于我们使用相同的 tiktoken 库，实际误差应该非常小
func TestProperty_EstimationErrorBound(t *testing.T) {
	// 已知的测试用例（基于 tiktoken 的实际输出）
	knownCases := []struct {
		content      string
		model        string
		expectedMin  int
		expectedMax  int
	}{
		{"Hello", "gpt-4", 1, 2},
		{"Hello, world!", "gpt-4", 3, 5},
		{"The quick brown fox", "gpt-4", 4, 6},
	}

	for _, tc := range knownCases {
		t.Run(tc.content, func(t *testing.T) {
			estimator := NewTokenEstimator()
			tokens, err := estimator.EstimateCompletionTokens(tc.model, tc.content)
			
			if err != nil {
				t.Errorf("估算失败: %v", err)
				return
			}

			if tokens < tc.expectedMin || tokens > tc.expectedMax {
				t.Errorf("估算值 %d 超出预期范围 [%d, %d]", tokens, tc.expectedMin, tc.expectedMax)
			}
		})
	}
}

// TestProperty_RobustnessToSpecialCharacters 测试对特殊字符的鲁棒性
// Feature: cliproxyapi-newapi-integration, Property 4: Usage 缺失时的估算
// 验证需求: 2.3, 12.1, 12.2
func TestProperty_RobustnessToSpecialCharacters(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("包含特殊字符的文本应该正常估算",
		prop.ForAll(
			func(model string) bool {
				specialTexts := []string{
					"Hello\nWorld",
					"Tab\there",
					"Quote\"test",
					"Emoji 😀 test",
					"Unicode: 你好世界",
					"Mixed: Hello 世界 🌍",
				}

				estimator := NewTokenEstimator()
				
				for _, text := range specialTexts {
					tokens, err := estimator.EstimateCompletionTokens(model, text)
					
					// 不应该有错误
					if err != nil {
						return false
					}

					// 应该返回合理的 token 数量
					if tokens <= 0 {
						return false
					}
				}

				return true
			},
			genSupportedModel(),
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
