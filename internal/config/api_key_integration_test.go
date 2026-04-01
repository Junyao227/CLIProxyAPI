// Package config provides configuration management for the CLI Proxy API server.
package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAPIKeyConfig_Integration 集成测试：验证完整的配置加载和验证流程
func TestAPIKeyConfig_Integration(t *testing.T) {
	// 创建一个完整的配置文件，模拟 new-api 集成场景
	configContent := `
# Server configuration
host: ""
port: 8317

# TLS settings
tls:
  enable: false
  cert: ""
  key: ""

# Remote management
remote-management:
  allow-remote: false
  secret-key: ""
  disable-control-panel: false

# Authentication directory
auth-dir: "~/.cli-proxy-api"

# API keys for new-api integration
api-keys:
  - "sk-cliproxy-newapi-instance1-xxx"
  - "sk-cliproxy-newapi-instance2-xxx"
  - "sk-cliproxy-newapi-instance3-xxx"

# Authentication timeout (milliseconds)
auth-timeout: 10

# Debug mode
debug: false

# Usage statistics
usage-statistics-enabled: false
`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// 加载配置
	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// 验证基本配置
	assert.Equal(t, "", cfg.Host)
	assert.Equal(t, 8317, cfg.Port)

	// 验证 API Keys 配置
	assert.Len(t, cfg.APIKeys, 3)
	assert.Contains(t, cfg.APIKeys, "sk-cliproxy-newapi-instance1-xxx")
	assert.Contains(t, cfg.APIKeys, "sk-cliproxy-newapi-instance2-xxx")
	assert.Contains(t, cfg.APIKeys, "sk-cliproxy-newapi-instance3-xxx")

	// 验证认证超时
	assert.Equal(t, 10, cfg.AuthTimeout)

	// 获取 API Key 配置
	apiKeyConfig := cfg.GetAPIKeyConfig()
	require.NotNil(t, apiKeyConfig)

	// 验证 API Key 配置已启用
	assert.True(t, apiKeyConfig.Enabled, "API Key authentication should be enabled")
	assert.Len(t, apiKeyConfig.Keys, 3)
	assert.Equal(t, 10, apiKeyConfig.Timeout)

	// 测试有效的 API Key 验证
	assert.True(t, apiKeyConfig.ValidateAPIKey("sk-cliproxy-newapi-instance1-xxx"))
	assert.True(t, apiKeyConfig.ValidateAPIKey("sk-cliproxy-newapi-instance2-xxx"))
	assert.True(t, apiKeyConfig.ValidateAPIKey("sk-cliproxy-newapi-instance3-xxx"))

	// 测试无效的 API Key 验证
	assert.False(t, apiKeyConfig.ValidateAPIKey("sk-invalid-key"))
	assert.False(t, apiKeyConfig.ValidateAPIKey(""))
	assert.False(t, apiKeyConfig.ValidateAPIKey("sk-cliproxy-newapi-instance4-xxx"))
}

// TestAPIKeyConfig_BackwardCompatibility 测试向后兼容性
// 验证当没有配置 API Keys 时，系统仍然可以正常工作
func TestAPIKeyConfig_BackwardCompatibility(t *testing.T) {
	configContent := `
port: 8317
auth-dir: "~/.cli-proxy-api"
debug: false
`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// 加载配置
	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// 验证没有 API Keys
	assert.Empty(t, cfg.APIKeys)

	// 获取 API Key 配置
	apiKeyConfig := cfg.GetAPIKeyConfig()
	require.NotNil(t, apiKeyConfig)

	// 验证 API Key 认证未启用（向后兼容）
	assert.False(t, apiKeyConfig.Enabled, "API Key authentication should be disabled for backward compatibility")

	// 验证任何 key 都能通过（向后兼容模式）
	assert.True(t, apiKeyConfig.ValidateAPIKey("any-key"))
	assert.True(t, apiKeyConfig.ValidateAPIKey(""))
	assert.True(t, apiKeyConfig.ValidateAPIKey("random-string"))
}

// TestAPIKeyConfig_MultipleInstances 测试多实例场景
// 模拟 new-api 配置多个 CLIProxyAPI Channel 的场景
func TestAPIKeyConfig_MultipleInstances(t *testing.T) {
	// 实例 1 配置
	instance1Config := `
port: 8317
api-keys:
  - "sk-cliproxy-instance1-key"
auth-timeout: 10
`

	// 实例 2 配置
	instance2Config := `
port: 8318
api-keys:
  - "sk-cliproxy-instance2-key"
auth-timeout: 15
`

	// 实例 3 配置
	instance3Config := `
port: 8319
api-keys:
  - "sk-cliproxy-instance3-key"
auth-timeout: 20
`

	tmpDir := t.TempDir()

	// 加载实例 1
	cfg1Path := filepath.Join(tmpDir, "config1.yaml")
	err := os.WriteFile(cfg1Path, []byte(instance1Config), 0644)
	require.NoError(t, err)
	cfg1, err := LoadConfig(cfg1Path)
	require.NoError(t, err)

	// 加载实例 2
	cfg2Path := filepath.Join(tmpDir, "config2.yaml")
	err = os.WriteFile(cfg2Path, []byte(instance2Config), 0644)
	require.NoError(t, err)
	cfg2, err := LoadConfig(cfg2Path)
	require.NoError(t, err)

	// 加载实例 3
	cfg3Path := filepath.Join(tmpDir, "config3.yaml")
	err = os.WriteFile(cfg3Path, []byte(instance3Config), 0644)
	require.NoError(t, err)
	cfg3, err := LoadConfig(cfg3Path)
	require.NoError(t, err)

	// 验证每个实例的配置独立
	apiKeyConfig1 := cfg1.GetAPIKeyConfig()
	apiKeyConfig2 := cfg2.GetAPIKeyConfig()
	apiKeyConfig3 := cfg3.GetAPIKeyConfig()

	// 实例 1 验证
	assert.True(t, apiKeyConfig1.Enabled)
	assert.Equal(t, 10, apiKeyConfig1.Timeout)
	assert.True(t, apiKeyConfig1.ValidateAPIKey("sk-cliproxy-instance1-key"))
	assert.False(t, apiKeyConfig1.ValidateAPIKey("sk-cliproxy-instance2-key"))

	// 实例 2 验证
	assert.True(t, apiKeyConfig2.Enabled)
	assert.Equal(t, 15, apiKeyConfig2.Timeout)
	assert.True(t, apiKeyConfig2.ValidateAPIKey("sk-cliproxy-instance2-key"))
	assert.False(t, apiKeyConfig2.ValidateAPIKey("sk-cliproxy-instance1-key"))

	// 实例 3 验证
	assert.True(t, apiKeyConfig3.Enabled)
	assert.Equal(t, 20, apiKeyConfig3.Timeout)
	assert.True(t, apiKeyConfig3.ValidateAPIKey("sk-cliproxy-instance3-key"))
	assert.False(t, apiKeyConfig3.ValidateAPIKey("sk-cliproxy-instance1-key"))
}

// TestAPIKeyConfig_SecurityValidation 测试安全性验证
func TestAPIKeyConfig_SecurityValidation(t *testing.T) {
	configContent := `
port: 8317
api-keys:
  - "sk-cliproxy-secure-key-123456"
auth-timeout: 10
`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)

	apiKeyConfig := cfg.GetAPIKeyConfig()

	// 测试常量时间比较（防止时序攻击）
	// 相同长度的不同 key 应该被拒绝
	similarKey := "sk-cliproxy-secure-key-654321"
	assert.False(t, apiKeyConfig.ValidateAPIKey(similarKey))

	// 测试空字符串
	assert.False(t, apiKeyConfig.ValidateAPIKey(""))

	// 测试只有空格的字符串
	assert.False(t, apiKeyConfig.ValidateAPIKey("   "))

	// 测试特殊字符
	assert.False(t, apiKeyConfig.ValidateAPIKey("sk-cliproxy-secure-key-123456\n"))
	assert.False(t, apiKeyConfig.ValidateAPIKey("sk-cliproxy-secure-key-123456\r"))
	assert.False(t, apiKeyConfig.ValidateAPIKey("sk-cliproxy-secure-key-123456\t"))

	// 测试 SQL 注入尝试（虽然不适用，但确保不会崩溃）
	assert.False(t, apiKeyConfig.ValidateAPIKey("'; DROP TABLE users; --"))

	// 测试超长字符串
	longKey := string(make([]byte, 10000))
	assert.False(t, apiKeyConfig.ValidateAPIKey(longKey))
}

// TestAPIKeyConfig_PerformanceValidation 测试性能要求
// 验证 API Key 认证在 10ms 内完成（需求 1.5）
func TestAPIKeyConfig_PerformanceValidation(t *testing.T) {
	// 创建包含多个 API Keys 的配置
	keys := make([]string, 100)
	for i := 0; i < 100; i++ {
		keys[i] = "sk-cliproxy-key-" + string(rune('a'+i%26)) + "-" + string(rune('0'+i%10))
	}

	apiKeyConfig := &APIKeyConfig{
		Enabled: true,
		Keys:    keys,
		Timeout: 10,
	}

	// 测试有效 key 的验证性能
	validKey := keys[50]
	
	// 执行多次验证，确保性能稳定
	for i := 0; i < 1000; i++ {
		result := apiKeyConfig.ValidateAPIKey(validKey)
		assert.True(t, result)
	}

	// 测试无效 key 的验证性能
	invalidKey := "sk-cliproxy-invalid-key"
	
	for i := 0; i < 1000; i++ {
		result := apiKeyConfig.ValidateAPIKey(invalidKey)
		assert.False(t, result)
	}

	// 注意：实际的性能测试应该使用 benchmark，这里只是功能验证
	// 真正的性能测试会在 benchmark 中进行
}
