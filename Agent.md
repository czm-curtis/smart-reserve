## 🎫 业务场景：2026 跨年演唱会抢票系统

**周杰伦「嘉年华」世界巡回演唱会 — 上海站门票开售**

- 10万张门票，400万+用户同时抢购
- 黄牛脚本每秒数万次刷接口
- Kafka 消息中间件磁盘 IO 打满，写入超时

## 🏗️ 微服务治理四大核心能力

### 1. 注册中心 — etcd 服务注册发现

```
                    ┌─────────────────────┐
                    │   API Gateway :8888  │
                    │  etcd 自动发现 RPC   │
                    └──────────┬──────────┘
                               │ etcd://appointment.rpc
               ┌───────────────┼───────────────┐
               ▼               ▼               ▼
    ┌──────────────┐ ┌──────────────┐ ┌──────────────┐
    │ RPC :9090    │ │ RPC :9091    │ │ RPC :9092    │
    │ (稳定版 #1)  │ │ (灰度版)     │ │ (稳定版 #2)  │
    └──────────────┘ └──────────────┘ └──────────────┘
```

**演示步骤：**
1. 启动多个 RPC 实例，各自注册到 etcd
2. 网关通过 `etcd://127.0.0.1:2379/appointment.rpc` 自动发现
3. 停掉一个实例，观察流量自动转移到剩余实例

### 2. 限流 — 双层自适应限流（反黄牛）

- **IP 级别**：单 IP 每秒最多 5 次 → 返回 429
- **用户级别**：单用户每秒最多 10 次 → 防止单用户高频刷接口

**触发方式：**
```bash
# 用压测工具高并发发送请求即可触发
curl -X POST http://localhost:8888/v1/appointment \
  -H "Content-Type: application/json" \
  -d '{"userId":1001,"scheduleId":99}'

# 高频重复发送 → 返回 429
# {"code":429,"msg":"抢购太火爆了！您的IP请求过于频繁，请稍后再试"}
```

### 3. 熔断 — Kafka 写入熔断器

**状态机：**
```
  CLOSED ──连续3次失败──▶ OPEN ──冷却30s──▶ HALF_OPEN
     ▲                                        │
     └──────────试探成功────────────┘   试探失败 → 回到 OPEN
```

**演示步骤（手动触发）：**
```bash
# Step 1: 查看系统状态
curl http://localhost:8888/v1/admin/status

# Step 2: 开启故障模拟（模拟 Kafka 写入失败）
curl -X POST http://localhost:8888/v1/admin/simulate/failure

# Step 3: 发送抢票请求，观察日志输出
# 请求会返回: "预约成功(降级模式:订单排队处理中)"
# 日志: 🔴 [熔断] 熔断器已打开，降级写入 Redis 延迟队列

# Step 4: 再次查看状态（降级队列长度会增加）
curl http://localhost:8888/v1/admin/status

# Step 5: 恢复 Kafka
curl -X POST http://localhost:8888/v1/admin/simulate/recovery

# 等待 30s 冷却期后，熔断器自动恢复，降级补偿 Worker 开始投递
```

### 4. 降级 — Redis 延迟队列兜底

当 Kafka 熔断器打开时：
- 订单不丢弃 → 写入 Redis 降级队列 `degradation:orders`
- 用户端正常返回 "预约成功(降级模式)"
- 后台补偿 Worker 每 5 秒检查，熔断恢复后自动投递

## 🚀 快速开始

### 前置条件
```bash
# 1. 启动基础设施
docker compose up -d mysql-reserve redis-reserve kafka-reserve kafka-init etcd-reserve

# 2. 等待 MySQL 就绪（约 10s）
docker logs mysql-reserve

# 3. 预热数据库：给场次 99 充值名额
# （init.sql 已创建场次 99，如需重置可用 Redis）
redis-cli SET reserve:slots:99 10000
```

### 启动应用服务（本地开发模式）
```bash
# 终端 1: 启动 RPC 稳定版 (9090)
cd apps/appointment/rpc
go run appointment.go -f etc/appointment.yaml

# 终端 2: 启动 RPC 灰度版 (9091)
cd apps/appointment/rpc
go run appointment.go -f etc/appointment-canary.yaml

# 终端 3: 启动 API 网关 (8888)
cd apps/appointment/api
go run appointment.go -f etc/appointment-api.yaml
```

### Docker Compose 一键启动
```bash
docker compose up -d --build
```

### 验证服务注册
```bash
# etcd 中应能看到注册的 RPC 服务
docker exec etcd-reserve etcdctl get --prefix "" --keys-only
# /appointment.rpc/xxx
# /appointment-canary.rpc/xxx
```

## 📊 监控端点

| 端点 | 方法 | 说明 |
|------|------|------|
| `/v1/admin/status` | GET | 查看熔断器状态、降级队列长度 |
| `/v1/admin/simulate/failure` | POST | 开启 Kafka 故障模拟 |
| `/v1/admin/simulate/recovery` | POST | 关闭故障模拟 |
| `/v1/admin/breaker/reset` | POST | 手动重置网关熔断器 |
| `/v1/admin/preload` | POST | 预热场次缓存（原有） |

## 🔧 配置说明

### API 网关 (`appointment-api.yaml`)
```yaml
RateLimit:
  Period: 1      # 限流周期（秒）
  IpQuota: 5     # IP 每秒最多 5 次
  UserQuota: 10  # 用户每秒最多 10 次

GatewayBreaker:
  Threshold: 5   # 连续 5 次 5xx 响应后熔断
  Cooldown: 30s  # 冷却 30 秒后试探恢复
```

### RPC 服务 (`appointment.yaml`)
```yaml
KafkaBreaker:
  Threshold: 3   # 连续 3 次 Kafka 写入失败后熔断
  Cooldown: 30s  # 冷却 30 秒后试探恢复
```

## 📁 项目结构

```
apps/appointment/
├── api/                          # API 网关层
│   ├── internal/
│   │   ├── middleware/
│   │   │   ├── canaryMiddleware.go    # 灰度路由（10% 染色）
│   │   │   ├── ratelimitMiddleware.go # 双层限流（IP + 用户）
│   │   │   └── breakerMiddleware.go   # 网关熔断器
│   │   └── handler/
│   │       └── adminHandler.go        # 管理端点（状态/模拟/恢复）
├── rpc/                          # RPC 业务服务层
│   ├── internal/
│   │   ├── breaker/
│   │   │   ├── kafkaBreaker.go        # Kafka 写入熔断器（Closed→Open→HalfOpen）
│   │   │   └── degradationQueue.go    # Redis 降级队列 + 补偿机制
│   │   └── logic/
│   │       └── createAppointmentLogic.go  # 核心抢票逻辑 + 熔断降级
└── rmq/                          # Kafka 消费者（异步落库）
```

## 🧪 压测验证

```bash
# K6 压测（需要 k6 已安装）
k6 run stress-test.js

# 或使用 Docker
docker compose run k6-stress
```
