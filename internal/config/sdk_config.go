// Package config provides configuration management for the CLI Proxy API server.
// It handles loading and parsing YAML configuration files, and provides structured
// access to application settings including server port, authentication directory,
// debug settings, proxy configuration, and API keys.
package config

import "crypto/subtle"

// SDKConfig represents the application's configuration, loaded from a YAML file.
type SDKConfig struct {
	// ProxyURL is the URL of an optional proxy server to use for outbound requests.
	ProxyURL string `yaml:"proxy-url" json:"proxy-url"`

	// ForceModelPrefix requires explicit model prefixes (e.g., "teamA/gemini-3-pro-preview")
	// to target prefixed credentials. When false, unprefixed model requests may use prefixed
	// credentials as well.
	ForceModelPrefix bool `yaml:"force-model-prefix" json:"force-model-prefix"`

	// RequestLog enables or disables detailed request logging functionality.
	RequestLog bool `yaml:"request-log" json:"request-log"`

	// APIKeys is a list of keys for authenticating clients to this proxy server.
	APIKeys []string `yaml:"api-keys" json:"api-keys"`

	// AuthTimeout 定义 API Key 认证的超时时间（毫秒）
	// 默认值为 10 毫秒，用于确保认证操作快速完成
	AuthTimeout int `yaml:"auth-timeout,omitempty" json:"auth-timeout,omitempty"`

	// PassthroughHeaders controls whether upstream response headers are forwarded to downstream clients.
	// Default is false (disabled).
	PassthroughHeaders bool `yaml:"passthrough-headers" json:"passthrough-headers"`

	// Streaming configures server-side streaming behavior (keep-alives and safe bootstrap retries).
	Streaming StreamingConfig `yaml:"streaming" json:"streaming"`

	// NonStreamKeepAliveInterval controls how often blank lines are emitted for non-streaming responses.
	// <= 0 disables keep-alives. Value is in seconds.
	NonStreamKeepAliveInterval int `yaml:"nonstream-keepalive-interval,omitempty" json:"nonstream-keepalive-interval,omitempty"`
}

// StreamingConfig holds server streaming behavior configuration.
type StreamingConfig struct {
	// KeepAliveSeconds controls how often the server emits SSE heartbeats (": keep-alive\n\n").
	// <= 0 disables keep-alives. Default is 0.
	KeepAliveSeconds int `yaml:"keepalive-seconds,omitempty" json:"keepalive-seconds,omitempty"`

	// BootstrapRetries controls how many times the server may retry a streaming request before any bytes are sent,
	// to allow auth rotation / transient recovery.
	// <= 0 disables bootstrap retries. Default is 0.
	BootstrapRetries int `yaml:"bootstrap-retries,omitempty" json:"bootstrap-retries,omitempty"`
}

// APIKeyConfig 表示 API Key 认证的配置
// 用于 new-api 集成场景，支持将 CLIProxyAPI 作为上游 Channel 使用
type APIKeyConfig struct {
	// Enabled 控制是否启用 API Key 认证
	// 当为 true 时，所有请求必须提供有效的 API Key
	// 当为 false 时，使用原有的认证机制（向后兼容）
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Keys 是允许访问的 API Key 列表
	// 从配置文件的 api-keys 字段加载
	Keys []string `yaml:"keys" json:"keys"`

	// Timeout 定义认证操作的超时时间（毫秒）
	// 默认值为 10 毫秒，确保认证快速完成
	Timeout int `yaml:"timeout" json:"timeout"`
}

// GetAPIKeyConfig 从 SDKConfig 构建 APIKeyConfig
// 这个方法用于将配置转换为认证中间件所需的格式
func (c *SDKConfig) GetAPIKeyConfig() *APIKeyConfig {
	if c == nil {
		return &APIKeyConfig{
			Enabled: false,
			Keys:    []string{},
			Timeout: 10,
		}
	}

	// 如果没有配置 API Keys，则认为未启用 API Key 认证
	enabled := len(c.APIKeys) > 0

	// 设置默认超时时间
	timeout := c.AuthTimeout
	if timeout <= 0 {
		timeout = 10 // 默认 10 毫秒
	}

	return &APIKeyConfig{
		Enabled: enabled,
		Keys:    c.APIKeys,
		Timeout: timeout,
	}
}

// ValidateAPIKey 检查给定的 key 是否在允许的 API Keys 列表中
// 使用常量时间比较防止时序攻击
func (cfg *APIKeyConfig) ValidateAPIKey(key string) bool {
	if cfg == nil || !cfg.Enabled {
		return true // 未启用时允许所有请求（向后兼容）
	}

	if key == "" {
		return false
	}

	// 使用常量时间比较防止时序攻击
	for _, validKey := range cfg.Keys {
		if len(key) == len(validKey) && subtle.ConstantTimeCompare([]byte(key), []byte(validKey)) == 1 {
			return true
		}
	}

	return false
}
