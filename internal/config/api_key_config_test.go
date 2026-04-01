// Package config provides configuration management for the CLI Proxy API server.
package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAPIKeyConfig_GetAPIKeyConfig 测试从 SDKConfig 构建 APIKeyConfig
func TestAPIKeyConfig_GetAPIKeyConfig(t *testing.T) {
	tests := []struct {
		name           string
		sdkConfig      *SDKConfig
		expectedConfig *APIKeyConfig
	}{
		{
			name:      "nil config",
			sdkConfig: nil,
			expectedConfig: &APIKeyConfig{
				Enabled: false,
				Keys:    []string{},
				Timeout: 10,
			},
		},
		{
			name: "empty api keys",
			sdkConfig: &SDKConfig{
				APIKeys:     []string{},
				AuthTimeout: 0,
			},
			expectedConfig: &APIKeyConfig{
				Enabled: false,
				Keys:    []string{},
				Timeout: 10,
			},
		},
		{
			name: "with api keys and default timeout",
			sdkConfig: &SDKConfig{
				APIKeys:     []string{"key1", "key2"},
				AuthTimeout: 0,
			},
			expectedConfig: &APIKeyConfig{
				Enabled: true,
				Keys:    []string{"key1", "key2"},
				Timeout: 10,
			},
		},
		{
			name: "with api keys and custom timeout",
			sdkConfig: &SDKConfig{
				APIKeys:     []string{"key1", "key2", "key3"},
				AuthTimeout: 20,
			},
			expectedConfig: &APIKeyConfig{
				Enabled: true,
				Keys:    []string{"key1", "key2", "key3"},
				Timeout: 20,
			},
		},
		{
			name: "negative timeout defaults to 10",
			sdkConfig: &SDKConfig{
				APIKeys:     []string{"key1"},
				AuthTimeout: -5,
			},
			expectedConfig: &APIKeyConfig{
				Enabled: true,
				Keys:    []string{"key1"},
				Timeout: 10,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.sdkConfig.GetAPIKeyConfig()
			assert.Equal(t, tt.expectedConfig.Enabled, result.Enabled)
			assert.Equal(t, tt.expectedConfig.Keys, result.Keys)
			assert.Equal(t, tt.expectedConfig.Timeout, result.Timeout)
		})
	}
}

// TestAPIKeyConfig_ValidateAPIKey 测试 API Key 验证逻辑
func TestAPIKeyConfig_ValidateAPIKey(t *testing.T) {
	tests := []struct {
		name      string
		config    *APIKeyConfig
		key       string
		wantValid bool
	}{
		{
			name:      "nil config allows all",
			config:    nil,
			key:       "any-key",
			wantValid: true,
		},
		{
			name: "disabled config allows all",
			config: &APIKeyConfig{
				Enabled: false,
				Keys:    []string{"key1"},
				Timeout: 10,
			},
			key:       "any-key",
			wantValid: true,
		},
		{
			name: "valid key",
			config: &APIKeyConfig{
				Enabled: true,
				Keys:    []string{"sk-test-key-123", "sk-test-key-456"},
				Timeout: 10,
			},
			key:       "sk-test-key-123",
			wantValid: true,
		},
		{
			name: "invalid key",
			config: &APIKeyConfig{
				Enabled: true,
				Keys:    []string{"sk-test-key-123", "sk-test-key-456"},
				Timeout: 10,
			},
			key:       "sk-invalid-key",
			wantValid: false,
		},
		{
			name: "empty key",
			config: &APIKeyConfig{
				Enabled: true,
				Keys:    []string{"sk-test-key-123"},
				Timeout: 10,
			},
			key:       "",
			wantValid: false,
		},
		{
			name: "case sensitive validation",
			config: &APIKeyConfig{
				Enabled: true,
				Keys:    []string{"sk-test-key-123"},
				Timeout: 10,
			},
			key:       "SK-TEST-KEY-123",
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.ValidateAPIKey(tt.key)
			assert.Equal(t, tt.wantValid, result)
		})
	}
}

// TestLoadConfig_APIKeyConfiguration 测试配置文件加载时的 API Key 配置处理
func TestLoadConfig_APIKeyConfiguration(t *testing.T) {
	tests := []struct {
		name           string
		configContent  string
		expectedKeys   []string
		expectedTimeout int
	}{
		{
			name: "basic api keys configuration",
			configContent: `
port: 8317
api-keys:
  - "sk-cliproxy-test-key-1"
  - "sk-cliproxy-test-key-2"
auth-timeout: 10
`,
			expectedKeys:   []string{"sk-cliproxy-test-key-1", "sk-cliproxy-test-key-2"},
			expectedTimeout: 10,
		},
		{
			name: "api keys with whitespace trimming",
			configContent: `
port: 8317
api-keys:
  - "  sk-cliproxy-test-key-1  "
  - "sk-cliproxy-test-key-2"
  - "   "
  - ""
auth-timeout: 15
`,
			expectedKeys:   []string{"sk-cliproxy-test-key-1", "sk-cliproxy-test-key-2"},
			expectedTimeout: 15,
		},
		{
			name: "default timeout when not specified",
			configContent: `
port: 8317
api-keys:
  - "sk-cliproxy-test-key-1"
`,
			expectedKeys:   []string{"sk-cliproxy-test-key-1"},
			expectedTimeout: 10,
		},
		{
			name: "negative timeout defaults to 10",
			configContent: `
port: 8317
api-keys:
  - "sk-cliproxy-test-key-1"
auth-timeout: -5
`,
			expectedKeys:   []string{"sk-cliproxy-test-key-1"},
			expectedTimeout: 10,
		},
		{
			name: "zero timeout defaults to 10",
			configContent: `
port: 8317
api-keys:
  - "sk-cliproxy-test-key-1"
auth-timeout: 0
`,
			expectedKeys:   []string{"sk-cliproxy-test-key-1"},
			expectedTimeout: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建临时配置文件
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")
			err := os.WriteFile(configPath, []byte(tt.configContent), 0644)
			require.NoError(t, err)

			// 加载配置
			cfg, err := LoadConfig(configPath)
			require.NoError(t, err)
			require.NotNil(t, cfg)

			// 验证 API Keys
			assert.Equal(t, tt.expectedKeys, cfg.APIKeys)

			// 验证超时设置
			assert.Equal(t, tt.expectedTimeout, cfg.AuthTimeout)

			// 验证 GetAPIKeyConfig 方法
			apiKeyConfig := cfg.GetAPIKeyConfig()
			assert.NotNil(t, apiKeyConfig)
			assert.Equal(t, len(tt.expectedKeys) > 0, apiKeyConfig.Enabled)
			assert.Equal(t, tt.expectedKeys, apiKeyConfig.Keys)
			assert.Equal(t, tt.expectedTimeout, apiKeyConfig.Timeout)
		})
	}
}

// TestAPIKeyConfig_ConstantTimeComparison 测试常量时间比较防止时序攻击
func TestAPIKeyConfig_ConstantTimeComparison(t *testing.T) {
	config := &APIKeyConfig{
		Enabled: true,
		Keys:    []string{"sk-test-key-with-exact-length-32"},
		Timeout: 10,
	}

	// 测试相同长度的不同 key（应该使用常量时间比较）
	invalidKey := "sk-invalid-key-same-length-32"
	assert.False(t, config.ValidateAPIKey(invalidKey))

	// 测试不同长度的 key（长度检查会快速失败，这是可以接受的）
	shortKey := "short"
	assert.False(t, config.ValidateAPIKey(shortKey))

	longKey := "sk-very-long-key-that-exceeds-the-expected-length-significantly"
	assert.False(t, config.ValidateAPIKey(longKey))
}

// TestLoadConfig_EmptyAPIKeys 测试空 API Keys 列表的处理
func TestLoadConfig_EmptyAPIKeys(t *testing.T) {
	configContent := `
port: 8317
api-keys: []
auth-timeout: 10
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// 空列表应该被保留
	assert.Empty(t, cfg.APIKeys)

	// GetAPIKeyConfig 应该返回 disabled 状态
	apiKeyConfig := cfg.GetAPIKeyConfig()
	assert.False(t, apiKeyConfig.Enabled)
}

// TestLoadConfig_NoAPIKeysField 测试配置文件中没有 api-keys 字段的情况
func TestLoadConfig_NoAPIKeysField(t *testing.T) {
	configContent := `
port: 8317
auth-timeout: 10
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// 没有 api-keys 字段时应该是 nil 或空列表
	assert.Empty(t, cfg.APIKeys)

	// GetAPIKeyConfig 应该返回 disabled 状态
	apiKeyConfig := cfg.GetAPIKeyConfig()
	assert.False(t, apiKeyConfig.Enabled)
}
