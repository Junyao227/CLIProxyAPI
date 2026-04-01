// Package auth provides authentication mechanisms for the CLI Proxy API server.
package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/api/middleware"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
	"github.com/stretchr/testify/assert"
)

// Feature: cliproxyapi-newapi-integration, Property 1: API Key 认证正确性
// 验证需求: 1.1, 1.2, 1.3
//
// 属性：对于任意带有 Authorization header 的请求，如果 API Key 在配置的密钥列表中，
// 则请求应该被转发（不返回 401）；如果 API Key 不在列表中或格式无效，则应该返回 401 错误响应。
func TestProperty_APIKeyAuthCorrectness(t *testing.T) {
	gin.SetMode(gin.TestMode)

	properties := gopter.NewProperties(nil)

	properties.Property("有效 API Key 不返回 401，无效 API Key 返回 401",
		prop.ForAll(
			func(validKey string, invalidKey string) bool {
				// 确保 validKey 和 invalidKey 不同
				if validKey == invalidKey {
					return true // 跳过相同的情况
				}

				// 创建配置，只包含 validKey
				apiKeyConfig := &config.APIKeyConfig{
					Enabled: true,
					Keys:    []string{validKey},
					Timeout: 10,
				}

				// 测试有效 API Key
				validRouter := gin.New()
				validRouter.Use(middleware.APIKeyAuthMiddleware(apiKeyConfig))
				validRouter.GET("/test", func(c *gin.Context) {
					c.JSON(http.StatusOK, gin.H{"message": "success"})
				})

				validReq := httptest.NewRequest("GET", "/test", nil)
				validReq.Header.Set("Authorization", "Bearer "+validKey)
				validResp := httptest.NewRecorder()
				validRouter.ServeHTTP(validResp, validReq)

				// 有效 key 应该不返回 401
				validOK := validResp.Code != http.StatusUnauthorized

				// 测试无效 API Key
				invalidRouter := gin.New()
				invalidRouter.Use(middleware.APIKeyAuthMiddleware(apiKeyConfig))
				invalidRouter.GET("/test", func(c *gin.Context) {
					c.JSON(http.StatusOK, gin.H{"message": "success"})
				})

				invalidReq := httptest.NewRequest("GET", "/test", nil)
				invalidReq.Header.Set("Authorization", "Bearer "+invalidKey)
				invalidResp := httptest.NewRecorder()
				invalidRouter.ServeHTTP(invalidResp, invalidReq)

				// 无效 key 应该返回 401
				invalidOK := invalidResp.Code == http.StatusUnauthorized

				return validOK && invalidOK
			},
			genValidAPIKey(),
			genValidAPIKey(),
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: cliproxyapi-newapi-integration, Property 2: Bearer Token 格式支持
// 验证需求: 1.4
//
// 属性：对于任意使用 `Bearer <token>` 格式的 Authorization header，
// CLIProxyAPI 应该正确提取 token 并进行验证。
func TestProperty_BearerTokenFormatSupport(t *testing.T) {
	gin.SetMode(gin.TestMode)

	properties := gopter.NewProperties(nil)

	properties.Property("Bearer token 格式正确提取和验证",
		prop.ForAll(
			func(apiKey string) bool {
				apiKeyConfig := &config.APIKeyConfig{
					Enabled: true,
					Keys:    []string{apiKey},
					Timeout: 10,
				}

				router := gin.New()
				router.Use(middleware.APIKeyAuthMiddleware(apiKeyConfig))
				router.GET("/test", func(c *gin.Context) {
					c.JSON(http.StatusOK, gin.H{"message": "success"})
				})

				// 测试标准 Bearer 格式
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("Authorization", "Bearer "+apiKey)
				resp := httptest.NewRecorder()
				router.ServeHTTP(resp, req)

				// 应该成功通过认证
				return resp.Code == http.StatusOK
			},
			genValidAPIKey(),
		))

	properties.Property("非 Bearer 格式被拒绝",
		prop.ForAll(
			func(apiKey string, invalidPrefix string) bool {
				// 确保不是 "Bearer " 前缀
				if invalidPrefix == "Bearer " {
					return true // 跳过
				}

				apiKeyConfig := &config.APIKeyConfig{
					Enabled: true,
					Keys:    []string{apiKey},
					Timeout: 10,
				}

				router := gin.New()
				router.Use(middleware.APIKeyAuthMiddleware(apiKeyConfig))
				router.GET("/test", func(c *gin.Context) {
					c.JSON(http.StatusOK, gin.H{"message": "success"})
				})

				// 测试非 Bearer 格式
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("Authorization", invalidPrefix+apiKey)
				resp := httptest.NewRecorder()
				router.ServeHTTP(resp, req)

				// 应该返回 401
				return resp.Code == http.StatusUnauthorized
			},
			genValidAPIKey(),
			gen.OneConstOf("Basic ", "Token ", "ApiKey ", ""),
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: cliproxyapi-newapi-integration, Property 1: API Key 认证正确性（扩展）
// 验证需求: 1.1, 1.2, 1.3
//
// 属性：缺失 Authorization header 应该返回 401 错误
func TestProperty_MissingAuthorizationHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	properties := gopter.NewProperties(nil)

	properties.Property("缺失 Authorization header 返回 401",
		prop.ForAll(
			func(apiKey string) bool {
				apiKeyConfig := &config.APIKeyConfig{
					Enabled: true,
					Keys:    []string{apiKey},
					Timeout: 10,
				}

				router := gin.New()
				router.Use(middleware.APIKeyAuthMiddleware(apiKeyConfig))
				router.GET("/test", func(c *gin.Context) {
					c.JSON(http.StatusOK, gin.H{"message": "success"})
				})

				// 不设置 Authorization header
				req := httptest.NewRequest("GET", "/test", nil)
				resp := httptest.NewRecorder()
				router.ServeHTTP(resp, req)

				// 应该返回 401
				return resp.Code == http.StatusUnauthorized
			},
			genValidAPIKey(),
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: cliproxyapi-newapi-integration, Property 1: API Key 认证正确性（多密钥）
// 验证需求: 1.1, 1.2, 1.3
//
// 属性：当配置多个有效 API Key 时，任何一个有效 key 都应该通过认证
func TestProperty_MultipleValidAPIKeys(t *testing.T) {
	gin.SetMode(gin.TestMode)

	properties := gopter.NewProperties(nil)

	properties.Property("多个有效 API Key 中的任意一个都能通过认证",
		prop.ForAll(
			func(keys []string) bool {
				// 至少需要 2 个不同的 key
				if len(keys) < 2 {
					return true // 跳过
				}

				// 去重
				uniqueKeys := make(map[string]bool)
				for _, k := range keys {
					uniqueKeys[k] = true
				}
				if len(uniqueKeys) < 2 {
					return true // 跳过
				}

				// 转换为切片
				keyList := make([]string, 0, len(uniqueKeys))
				for k := range uniqueKeys {
					keyList = append(keyList, k)
				}

				apiKeyConfig := &config.APIKeyConfig{
					Enabled: true,
					Keys:    keyList,
					Timeout: 10,
				}

				// 测试每个 key 都能通过认证
				for _, key := range keyList {
					router := gin.New()
					router.Use(middleware.APIKeyAuthMiddleware(apiKeyConfig))
					router.GET("/test", func(c *gin.Context) {
						c.JSON(http.StatusOK, gin.H{"message": "success"})
					})

					req := httptest.NewRequest("GET", "/test", nil)
					req.Header.Set("Authorization", "Bearer "+key)
					resp := httptest.NewRecorder()
					router.ServeHTTP(resp, req)

					if resp.Code != http.StatusOK {
						return false
					}
				}

				return true
			},
			gen.SliceOfN(5, genValidAPIKey()),
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: cliproxyapi-newapi-integration, Property 19: 向后兼容性
// 验证需求: 9.1, 9.3, 11.3
//
// 属性：当 API Key 认证被禁用时，应该允许所有请求通过（不检查 Authorization header）
func TestProperty_BackwardCompatibility(t *testing.T) {
	gin.SetMode(gin.TestMode)

	properties := gopter.NewProperties(nil)

	properties.Property("禁用认证时允许所有请求",
		prop.ForAll(
			func(hasAuth bool, authValue string) bool {
				apiKeyConfig := &config.APIKeyConfig{
					Enabled: false,
					Keys:    []string{"some-key"},
					Timeout: 10,
				}

				router := gin.New()
				router.Use(middleware.APIKeyAuthMiddleware(apiKeyConfig))
				router.GET("/test", func(c *gin.Context) {
					c.JSON(http.StatusOK, gin.H{"message": "success"})
				})

				req := httptest.NewRequest("GET", "/test", nil)
				if hasAuth {
					req.Header.Set("Authorization", authValue)
				}
				resp := httptest.NewRecorder()
				router.ServeHTTP(resp, req)

				// 禁用认证时，无论是否有 Authorization header，都应该通过
				return resp.Code == http.StatusOK
			},
			gen.Bool(),
			gen.AnyString(),
		))

	properties.Property("nil 配置时允许所有请求",
		prop.ForAll(
			func(hasAuth bool, authValue string) bool {
				router := gin.New()
				router.Use(middleware.APIKeyAuthMiddleware(nil))
				router.GET("/test", func(c *gin.Context) {
					c.JSON(http.StatusOK, gin.H{"message": "success"})
				})

				req := httptest.NewRequest("GET", "/test", nil)
				if hasAuth {
					req.Header.Set("Authorization", authValue)
				}
				resp := httptest.NewRecorder()
				router.ServeHTTP(resp, req)

				// nil 配置时，无论是否有 Authorization header，都应该通过
				return resp.Code == http.StatusOK
			},
			gen.Bool(),
			gen.AnyString(),
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// genValidAPIKey 生成有效的 API Key 字符串
// API Key 格式：sk-cliproxy-{随机字符串}
func genValidAPIKey() gopter.Gen {
	return gen.Identifier().Map(func(id string) string {
		return "sk-cliproxy-" + id
	})
}

// TestProperty_EmptyTokenRejection 测试空 token 被拒绝的属性
// Feature: cliproxyapi-newapi-integration, Property 1: API Key 认证正确性
// 验证需求: 1.1, 1.2, 1.3
func TestProperty_EmptyTokenRejection(t *testing.T) {
	gin.SetMode(gin.TestMode)

	properties := gopter.NewProperties(nil)

	properties.Property("空 token 被拒绝",
		prop.ForAll(
			func(apiKey string) bool {
				apiKeyConfig := &config.APIKeyConfig{
					Enabled: true,
					Keys:    []string{apiKey},
					Timeout: 10,
				}

				router := gin.New()
				router.Use(middleware.APIKeyAuthMiddleware(apiKeyConfig))
				router.GET("/test", func(c *gin.Context) {
					c.JSON(http.StatusOK, gin.H{"message": "success"})
				})

				// 测试空 token（只有 "Bearer " 前缀）
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("Authorization", "Bearer ")
				resp := httptest.NewRecorder()
				router.ServeHTTP(resp, req)

				// 应该返回 401
				return resp.Code == http.StatusUnauthorized
			},
			genValidAPIKey(),
		))

	properties.Property("只有空格的 token 被拒绝",
		prop.ForAll(
			func(apiKey string, spaces int) bool {
				if spaces < 1 || spaces > 10 {
					return true // 限制空格数量
				}

				apiKeyConfig := &config.APIKeyConfig{
					Enabled: true,
					Keys:    []string{apiKey},
					Timeout: 10,
				}

				router := gin.New()
				router.Use(middleware.APIKeyAuthMiddleware(apiKeyConfig))
				router.GET("/test", func(c *gin.Context) {
					c.JSON(http.StatusOK, gin.H{"message": "success"})
				})

				// 生成只有空格的 token
				spaceToken := ""
				for i := 0; i < spaces; i++ {
					spaceToken += " "
				}

				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("Authorization", "Bearer "+spaceToken)
				resp := httptest.NewRecorder()
				router.ServeHTTP(resp, req)

				// 应该返回 401
				return resp.Code == http.StatusUnauthorized
			},
			genValidAPIKey(),
			gen.IntRange(1, 10),
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty_CaseSensitiveValidation 测试 API Key 验证的大小写敏感性
// Feature: cliproxyapi-newapi-integration, Property 1: API Key 认证正确性
// 验证需求: 1.1, 1.2, 1.3
func TestProperty_CaseSensitiveValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 手动测试大小写敏感性（因为随机生成可能不会产生大小写差异）
	testCases := []struct {
		name      string
		configKey string
		testKey   string
		shouldPass bool
	}{
		{
			name:      "完全匹配",
			configKey: "sk-cliproxy-TestKey123",
			testKey:   "sk-cliproxy-TestKey123",
			shouldPass: true,
		},
		{
			name:      "大小写不匹配",
			configKey: "sk-cliproxy-TestKey123",
			testKey:   "sk-cliproxy-testkey123",
			shouldPass: false,
		},
		{
			name:      "前缀大小写不匹配",
			configKey: "sk-cliproxy-TestKey123",
			testKey:   "SK-CLIPROXY-TestKey123",
			shouldPass: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			apiKeyConfig := &config.APIKeyConfig{
				Enabled: true,
				Keys:    []string{tc.configKey},
				Timeout: 10,
			}

			router := gin.New()
			router.Use(middleware.APIKeyAuthMiddleware(apiKeyConfig))
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", "Bearer "+tc.testKey)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			if tc.shouldPass {
				assert.Equal(t, http.StatusOK, resp.Code, "应该通过认证")
			} else {
				assert.Equal(t, http.StatusUnauthorized, resp.Code, "应该拒绝认证")
			}
		})
	}
}
