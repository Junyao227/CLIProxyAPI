# k6 负载测试指南

本目录包含使用 k6 进行负载测试的脚本和文档。

## 前置条件

### 安装 k6

**Windows:**
```powershell
choco install k6
```

或从 [k6 官网](https://k6.io/docs/getting-started/installation/) 下载安装包。

**macOS:**
```bash
brew install k6
```

**Linux:**
```bash
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6
```

### 环境准备

1. 确保 new-api 和 CLIProxyAPI 都在运行
2. 获取 new-api 的访问 token
3. 设置环境变量

## 运行负载测试

### 基本用法

```bash
# 设置环境变量
export NEW_API_URL="http://localhost:3000"
export NEW_API_TOKEN="your-token-here"

# 运行负载测试
k6 run load_test.js
```

### Windows PowerShell

```powershell
# 设置环境变量
$env:NEW_API_URL = "http://localhost:3000"
$env:NEW_API_TOKEN = "your-token-here"

# 运行负载测试
k6 run load_test.js
```

### 自定义测试场景

修改 `load_test.js` 中的 `options.stages` 配置：

```javascript
stages: [
  { duration: '30s', target: 10 },   // 30 秒内爬升到 10 并发
  { duration: '1m', target: 50 },    // 1 分钟内爬升到 50 并发
  { duration: '2m', target: 50 },    // 保持 50 并发 2 分钟
  { duration: '30s', target: 0 },    // 30 秒内降到 0
],
```

### 自定义性能阈值

修改 `load_test.js` 中的 `options.thresholds` 配置：

```javascript
thresholds: {
  'http_req_duration': ['p(95)<500'],  // 95% 的请求应在 500ms 内完成
  'errors': ['rate<0.01'],             // 错误率应小于 1%
  'http_req_duration': ['p(99)<1000'], // 99% 的请求应在 1000ms 内完成
  'http_req_duration': ['avg<300'],    // 平均响应时间应小于 300ms
},
```

## 测试场景

### 默认场景（load_test.js）

- **目标**: 验证系统在 100 并发下的性能
- **持续时间**: 约 7 分钟
- **阶段**:
  1. 1 分钟内爬升到 20 并发
  2. 1 分钟内爬升到 50 并发
  3. 1 分钟内爬升到 100 并发
  4. 保持 100 并发 3 分钟
  5. 1 分钟内降到 0

- **性能目标**:
  - P95 响应时间 < 500ms
  - P99 响应时间 < 1000ms
  - 平均响应时间 < 300ms
  - 错误率 < 1%

## 结果分析

### 输出文件

测试完成后会生成 `load_test_results.json` 文件，包含详细的测试结果。

### 关键指标

1. **http_reqs**: 总请求数和请求速率
2. **http_req_duration**: 响应时间统计（平均、最小、最大、P50、P90、P95、P99）
3. **errors**: 错误率
4. **successful_requests**: 成功请求数
5. **failed_requests**: 失败请求数

### 示例输出

```
负载测试摘要
==================================================

总请求数: 12543
请求速率: 29.87 req/s

响应时间统计:
  平均: 245.32 ms
  最小: 89.12 ms
  最大: 1234.56 ms
  P50: 223.45 ms
  P90: 387.65 ms
  P95: 456.78 ms
  P99: 789.01 ms

错误率: 0.23%

成功请求: 12514 (99.77%)
失败请求: 29

阈值检查:
  ✓ 状态码是 200: 12514/12543
  ✓ 响应包含 usage: 12514/12543
  ✓ 响应时间 < 500ms: 11892/12543
```

## 性能优化建议

### 如果 P95 > 500ms

1. 检查 CLIProxyAPI 实例数量，考虑增加实例
2. 检查 new-api 的负载均衡配置
3. 检查网络延迟
4. 检查上游 API（Claude、Gemini 等）的响应时间

### 如果错误率 > 1%

1. 检查 CLIProxyAPI 日志，查找错误原因
2. 检查 API Key 配置是否正确
3. 检查上游 API 的配额和限流
4. 检查网络连接稳定性

### 如果吞吐量 < 1000 req/s

1. 增加 CLIProxyAPI 实例数量
2. 优化 new-api 的 Channel 权重配置
3. 检查系统资源使用情况（CPU、内存、网络）
4. 考虑使用更高性能的硬件

## 高级用法

### 使用 k6 Cloud

```bash
k6 cloud load_test.js
```

### 生成 HTML 报告

```bash
k6 run --out json=results.json load_test.js
k6 report results.json --output report.html
```

### 分布式负载测试

使用多台机器同时运行测试：

```bash
# 机器 1
k6 run --tag instance=1 load_test.js

# 机器 2
k6 run --tag instance=2 load_test.js
```

## 故障排查

### 连接被拒绝

- 检查 new-api 是否在运行
- 检查 NEW_API_URL 是否正确
- 检查防火墙设置

### 认证失败

- 检查 NEW_API_TOKEN 是否正确
- 检查 token 是否过期
- 检查 token 权限

### 超时错误

- 增加 `params.timeout` 值
- 检查网络连接
- 检查服务器负载

## 参考资源

- [k6 官方文档](https://k6.io/docs/)
- [k6 性能测试最佳实践](https://k6.io/docs/testing-guides/test-types/)
- [k6 指标参考](https://k6.io/docs/using-k6/metrics/)
