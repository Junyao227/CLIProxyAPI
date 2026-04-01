// k6 负载测试脚本 - CLIProxyAPI 与 new-api 集成
// 使用方法: k6 run load_test.js
// 环境变量:
// - NEW_API_URL: new-api 的 URL（默认 http://localhost:3000）
// - NEW_API_TOKEN: new-api 的访问 token

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// 自定义指标
const errorRate = new Rate('errors');
const requestDuration = new Trend('request_duration');
const successfulRequests = new Counter('successful_requests');
const failedRequests = new Counter('failed_requests');

// 测试配置
export const options = {
  // 测试场景：爬升到 100 并发，保持 3 分钟
  stages: [
    { duration: '1m', target: 20 },   // 1 分钟内爬升到 20 并发
    { duration: '1m', target: 50 },   // 1 分钟内爬升到 50 并发
    { duration: '1m', target: 100 },  // 1 分钟内爬升到 100 并发
    { duration: '3m', target: 100 },  // 保持 100 并发 3 分钟
    { duration: '1m', target: 0 },    // 1 分钟内降到 0
  ],

  // 性能阈值
  thresholds: {
    // 95% 的请求应在 500ms 内完成
    'http_req_duration': ['p(95)<500'],
    
    // 错误率应小于 1%
    'errors': ['rate<0.01'],
    
    // 99% 的请求应在 1000ms 内完成
    'http_req_duration': ['p(99)<1000'],
    
    // 平均响应时间应小于 300ms
    'http_req_duration': ['avg<300'],
  },
};

// 从环境变量读取配置
const NEW_API_URL = __ENV.NEW_API_URL || 'http://localhost:3000';
const NEW_API_TOKEN = __ENV.NEW_API_TOKEN;

if (!NEW_API_TOKEN) {
  throw new Error('必须设置 NEW_API_TOKEN 环境变量');
}

// 测试数据
const testMessages = [
  'Hello, how are you?',
  'What is the weather like today?',
  'Tell me a joke.',
  'Explain quantum computing in simple terms.',
  'What are the benefits of exercise?',
  'How do I learn programming?',
  'What is artificial intelligence?',
  'Recommend a good book.',
  'How to make coffee?',
  'What is the meaning of life?',
];

// 主测试函数
export default function () {
  // 随机选择一条测试消息
  const message = testMessages[Math.floor(Math.random() * testMessages.length)];

  // 构建请求
  const payload = JSON.stringify({
    model: 'gpt-4',
    messages: [
      { role: 'user', content: message },
    ],
    max_tokens: 50,
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${NEW_API_TOKEN}`,
    },
    timeout: '30s',
  };

  // 发送请求
  const startTime = Date.now();
  const response = http.post(`${NEW_API_URL}/v1/chat/completions`, payload, params);
  const duration = Date.now() - startTime;

  // 记录请求时长
  requestDuration.add(duration);

  // 检查响应
  const success = check(response, {
    '状态码是 200': (r) => r.status === 200,
    '响应包含 id': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.id !== undefined;
      } catch (e) {
        return false;
      }
    },
    '响应包含 usage': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.usage !== undefined &&
               body.usage.prompt_tokens !== undefined &&
               body.usage.completion_tokens !== undefined &&
               body.usage.total_tokens !== undefined;
      } catch (e) {
        return false;
      }
    },
    '响应包含 choices': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.choices !== undefined && body.choices.length > 0;
      } catch (e) {
        return false;
      }
    },
    '响应时间 < 500ms': () => duration < 500,
  });

  // 更新指标
  if (success) {
    successfulRequests.add(1);
    errorRate.add(0);
  } else {
    failedRequests.add(1);
    errorRate.add(1);
    console.error(`请求失败: ${response.status} - ${response.body}`);
  }

  // 模拟真实用户行为，请求之间有间隔
  sleep(Math.random() * 2 + 1); // 1-3 秒随机间隔
}

// 测试设置阶段
export function setup() {
  console.log('开始负载测试...');
  console.log(`目标 URL: ${NEW_API_URL}`);
  console.log('测试场景: 爬升到 100 并发，保持 3 分钟');
  
  // 验证 API 可访问性
  const params = {
    headers: {
      'Authorization': `Bearer ${NEW_API_TOKEN}`,
    },
  };
  
  const response = http.get(`${NEW_API_URL}/v1/models`, params);
  if (response.status !== 200) {
    throw new Error(`API 不可访问: ${response.status}`);
  }
  
  console.log('API 可访问性验证通过');
  return { startTime: Date.now() };
}

// 测试清理阶段
export function teardown(data) {
  const duration = (Date.now() - data.startTime) / 1000;
  console.log(`测试完成，总耗时: ${duration.toFixed(2)} 秒`);
}

// 处理摘要数据
export function handleSummary(data) {
  return {
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
    'load_test_results.json': JSON.stringify(data, null, 2),
  };
}

// 文本摘要函数
function textSummary(data, options) {
  const indent = options.indent || '';
  const enableColors = options.enableColors || false;
  
  let summary = '\n';
  summary += `${indent}负载测试摘要\n`;
  summary += `${indent}${'='.repeat(50)}\n\n`;
  
  // 请求统计
  const httpReqs = data.metrics.http_reqs;
  if (httpReqs) {
    summary += `${indent}总请求数: ${httpReqs.values.count}\n`;
    summary += `${indent}请求速率: ${httpReqs.values.rate.toFixed(2)} req/s\n\n`;
  }
  
  // 响应时间统计
  const httpReqDuration = data.metrics.http_req_duration;
  if (httpReqDuration) {
    summary += `${indent}响应时间统计:\n`;
    summary += `${indent}  平均: ${httpReqDuration.values.avg.toFixed(2)} ms\n`;
    summary += `${indent}  最小: ${httpReqDuration.values.min.toFixed(2)} ms\n`;
    summary += `${indent}  最大: ${httpReqDuration.values.max.toFixed(2)} ms\n`;
    summary += `${indent}  P50: ${httpReqDuration.values['p(50)'].toFixed(2)} ms\n`;
    summary += `${indent}  P90: ${httpReqDuration.values['p(90)'].toFixed(2)} ms\n`;
    summary += `${indent}  P95: ${httpReqDuration.values['p(95)'].toFixed(2)} ms\n`;
    summary += `${indent}  P99: ${httpReqDuration.values['p(99)'].toFixed(2)} ms\n\n`;
  }
  
  // 错误率
  const errors = data.metrics.errors;
  if (errors) {
    const errorRate = (errors.values.rate * 100).toFixed(2);
    summary += `${indent}错误率: ${errorRate}%\n\n`;
  }
  
  // 成功/失败统计
  const successful = data.metrics.successful_requests;
  const failed = data.metrics.failed_requests;
  if (successful && failed) {
    const total = successful.values.count + failed.values.count;
    const successRate = ((successful.values.count / total) * 100).toFixed(2);
    summary += `${indent}成功请求: ${successful.values.count} (${successRate}%)\n`;
    summary += `${indent}失败请求: ${failed.values.count}\n\n`;
  }
  
  // 阈值检查
  summary += `${indent}阈值检查:\n`;
  const thresholds = data.root_group.checks;
  if (thresholds) {
    for (const [name, result] of Object.entries(thresholds)) {
      const status = result.passes === result.fails + result.passes ? '✓' : '✗';
      summary += `${indent}  ${status} ${name}: ${result.passes}/${result.passes + result.fails}\n`;
    }
  }
  
  summary += `\n${indent}${'='.repeat(50)}\n`;
  
  return summary;
}
