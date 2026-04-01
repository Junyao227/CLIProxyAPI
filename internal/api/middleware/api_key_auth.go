// Package middleware provides HTTP middleware components for the API server.
package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
	log "github.com/sirupsen/logrus"
)

// APIKeyAuthMiddleware 创建一个 API Key 认证中间件
// 用于验证来自 new-api 的请求
//
// 中间件行为：
//   - 从 Authorization header 提取 Bearer token
//   - 验证 token 是否在配置的 API Keys 列表中
//   - 认证失败时返回 401 错误（OpenAI 兼容格式）
//   - 确保认证在 10ms 内完成
//
// 参数：
//   - apiKeyConfig: API Key 配置，包含启用状态、密钥列表和超时设置
//
// 返回：
//   - gin.HandlerFunc: Gin 中间件处理函数
func APIKeyAuthMiddleware(apiKeyConfig *config.APIKeyConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 如果配置为 nil 或未启用，则跳过认证（向后兼容）
		if apiKeyConfig == nil || !apiKeyConfig.Enabled {
			c.Next()
			return
		}

		// 设置认证超时
		startTime := time.Now()
		defer func() {
			elapsed := time.Since(startTime)
			if elapsed > time.Duration(apiKeyConfig.Timeout)*time.Millisecond {
				log.Warnf("API Key 认证超时: %v (限制: %dms)", elapsed, apiKeyConfig.Timeout)
			}
		}()

		// 从 Authorization header 提取 Bearer token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"message": "Missing Authorization header",
					"type":    "invalid_request_error",
					"code":    "invalid_api_key",
				},
			})
			return
		}

		// 验证 Bearer token 格式
		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"message": "Invalid Authorization header format. Expected 'Bearer <token>'",
					"type":    "invalid_request_error",
					"code":    "invalid_api_key",
				},
			})
			return
		}

		// 提取 token
		token := strings.TrimSpace(authHeader[len(bearerPrefix):])
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"message": "Empty API key in Authorization header",
					"type":    "invalid_request_error",
					"code":    "invalid_api_key",
				},
			})
			return
		}

		// 验证 API Key
		if !apiKeyConfig.ValidateAPIKey(token) {
			log.Warnf("API Key 认证失败: 无效的密钥 (长度: %d)", len(token))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"message": "Invalid API key",
					"type":    "invalid_request_error",
					"code":    "invalid_api_key",
				},
			})
			return
		}

		// 认证成功，继续处理请求
		c.Next()
	}
}
