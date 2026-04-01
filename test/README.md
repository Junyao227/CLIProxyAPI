# 集成测试指南

本目录包含 CLIProxyAPI 与 new-api 集成的测试套件。

## 测试类型

### 1. 端到端测试 (integration_e2e_test.go)

测试完整的请求流程，从 new-api 到 CLIProxyAPI 再到上游 API。

**测试用例**:
- `TestE2E_ChatCompletion`: 测试聊天完成请求
- `TestE2E_QuotaDeduction`: 测试配额扣减
- `TestE2E_UsageLogging`: 测试使用日志记录

### 2. 故障转移测试 (integration_failover_test.go)

测试多实例部署的故障转移能力。

**测试用例**:
- `TestFailover_MultipleInstances`: 测试实例故障转移
- `TestFailover_GracefulDegradation`: 测试优雅降级
- `TestFailover_ResponseTime`: 测试故障转移响应时间

### 3. 负载均衡测试 (integration_loadbalancing_test.go)

测试请求在多个 CLIProxyAPI 实例之间的分配。

**测试用例**:
- `TestLoadBalancing_WeightDistribution`: 测试权重分配
- `TestLoadBalancing_ConcurrentRequests`: 测试并发请求
- `TestLoadBalancing_WeightRatio`: 测试权重比例
- `TestLoadBalancing_RoundRobin`: 测试轮询模式

### 4. 流式响应测试 (integration_streaming_test.go)

测试流式响应的正确性和性能。

**测试用例**:
- `TestStreaming_SSEFormat`: 测试 SSE 格式
- `TestStreaming_ContentAccumulation`: 测试内容累积
- `TestStreaming_QuotaDeduction`: 测试流式响应后的配额扣减
- `TestStreaming_ErrorHandling`: 测试错误处理
- `TestStreaming_LongResponse`: 测试长流式响应

## 运行测试

### 前置条件

1. **启动服务**

使用 Docker Compose 启动所有服务：

```bash
# 多实例部署
docker-compose -f docker-compose.integration.yml up -d

# 或单实例部署
docker-compose -f docker-compose.simple.yml up -d
```

2. **配置环境变量**

```bash
# Linux/macOS
export NEW_API_URL="http://localhost:3000"
export NEW_API_TOKEN="your-token-here"

# Windows PowerShell
$env:NEW_API_URL = "http://localhost:3000"
$env:NEW_API_TOKEN = "your-token-here"
```

3. **配置 new-api**

- 登录 new-api 管理界面
- 创建测试用户和 token
- 配置 CLIProxyAPI Channel
- 配置模型定价

### 运行所有测试

```bash
cd CLIProxyAPI

# 运行所有集成测试
go test -v ./test/...

# 跳过集成测试（使用 -short 标志）
go test -short -v ./test/...
```

### 运行特定测试

```bash
# 运行端到端测试
go test -v ./test/... -run TestE2E

# 运行故障转移测试（需要 Docker Compose）
export DOCKER_COMPOSE_TEST=1
go test -v ./test/... -run TestFailover

# 运行负载均衡测试
export DOCKER_COMPOSE_TEST=1
go test -v ./test/... -run TestLoadBalancing

# 运行流式响应测试
go test -v ./test/... -run TestStreaming
```

### 运行基准测试

```bash
# 运行负载均衡基准测试
export DOCKER_COMPOSE_TEST=1
go test -bench=BenchmarkLoadBalancing -benchtime=30s ./test/...

# 运行流式响应基准测试
go test -bench=BenchmarkStreaming -benchtime=30s ./test/...
```

## 测试配置

### 环境变量

| 变量 | 说明 | 默认值 | 必需 |
|------|------|--------|------|
| `NEW_API_URL` | new-api 的 URL | `http://localhost:3000` | 否 |
| `NEW_API_TOKEN` | new-api 的访问 token | - | 是 |
| `DOCKER_COMPOSE_TEST` | 启用 Docker Compose 测试 | - | 否* |

*: 故障转移和负载均衡测试需要此变量

### 跳过测试

某些测试在特定条件下会自动跳过：

1. **使用 `-short` 标志**: 跳过所有集成测试
2. **未设置 `NEW_API_TOKEN`**: 跳过需要认证的测试
3. **未设置 `DOCKER_COMPOSE_TEST`**: 跳过需要 Docker Compose 的测试

## 测试结果

### 成功示例

```
=== RUN   TestE2E_ChatCompletion
    integration_e2e_test.go:45: 响应: {"id":"chatcmpl-123",...}
    integration_e2e_test.go:78: 响应内容: Hello, integration test!
--- PASS: TestE2E_ChatCompletion (2.34s)
```

### 失败示例

```
=== RUN   TestE2E_ChatCompletion
    integration_e2e_test.go:42: 应该返回 200 OK
        Expected: 200
        Actual:   401
--- FAIL: TestE2E_ChatCompletion (0.12s)
```

## 故障排查

### 连接被拒绝

**错误**: `connection refused`

**原因**: 服务未启动或 URL 不正确

**解决方案**:
1. 检查服务是否在运行: `docker-compose ps`
2. 验证 URL: `curl http://localhost:3000/api/status`
3. 检查防火墙设置

### 认证失败

**错误**: `401 Unauthorized`

**原因**: Token 无效或过期

**解决方案**:
1. 验证 token: `echo $NEW_API_TOKEN`
2. 在 new-api 中重新生成 token
3. 检查 token 权限

### 配额不足

**错误**: `insufficient quota`

**原因**: 测试账户配额耗尽

**解决方案**:
1. 在 new-api 中增加配额
2. 使用新的测试账户
3. 等待配额重置

### 超时错误

**错误**: `context deadline exceeded`

**原因**: 请求超时

**解决方案**:
1. 检查网络连接
2. 检查服务器负载
3. 增加超时时间（修改测试代码）

### Docker Compose 测试失败

**错误**: `需要设置 DOCKER_COMPOSE_TEST=1`

**原因**: 未启用 Docker Compose 测试

**解决方案**:
```bash
export DOCKER_COMPOSE_TEST=1
go test -v ./test/... -run TestFailover
```

## 最佳实践

### 1. 使用测试账户

不要使用生产账户进行测试，创建专门的测试账户。

### 2. 清理测试数据

测试完成后清理生成的数据：

```bash
# 停止服务
docker-compose -f docker-compose.integration.yml down

# 清理卷（可选）
docker-compose -f docker-compose.integration.yml down -v
```

### 3. 并行测试

避免并行运行集成测试，因为它们共享相同的服务实例：

```bash
# 不要使用 -parallel 标志
go test -v ./test/...
```

### 4. 测试隔离

每个测试应该是独立的，不依赖其他测试的状态。

### 5. 合理的超时

设置合理的超时时间，避免测试挂起：

```go
client := &http.Client{Timeout: 30 * time.Second}
```

## 持续集成

### GitHub Actions 示例

```yaml
name: Integration Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    
    steps:
      - uses: actions/checkout@v2
      
      - name: Set up Go
        uses: actions/setup-go@v2
        with:
          go-version: 1.21
      
      - name: Start services
        run: docker-compose -f docker-compose.integration.yml up -d
      
      - name: Wait for services
        run: sleep 30
      
      - name: Run integration tests
        env:
          NEW_API_URL: http://localhost:3000
          NEW_API_TOKEN: ${{ secrets.NEW_API_TOKEN }}
          DOCKER_COMPOSE_TEST: 1
        run: go test -v ./test/...
      
      - name: Stop services
        run: docker-compose -f docker-compose.integration.yml down
```

## 参考资源

- [Go 测试最佳实践](https://golang.org/doc/tutorial/add-a-test)
- [Docker Compose 测试](https://docs.docker.com/compose/reference/)
- [集成测试模式](https://martinfowler.com/articles/practical-test-pyramid.html)
