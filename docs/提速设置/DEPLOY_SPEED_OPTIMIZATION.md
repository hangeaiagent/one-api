# One-API 提速部署方案

## 概述

本方案包含两个方面的优化：
1. **提高限流速度**：减少限流限制，提高 API 请求速度
2. **429 错误优化**：已实现的智能重试和临时禁用机制

## 一、提高限流速度配置

### 当前默认配置

- `GLOBAL_API_RATE_LIMIT`: 480 次/3分钟（160 次/分钟）
- `GLOBAL_WEB_RATE_LIMIT`: 240 次/3分钟（80 次/分钟）

### 推荐提速配置

根据实际需求，可以选择以下配置：

#### 方案 1：中等提速（推荐）
```bash
GLOBAL_API_RATE_LIMIT=1000    # 1000 次/3分钟 ≈ 333 次/分钟
GLOBAL_WEB_RATE_LIMIT=500     # 500 次/3分钟 ≈ 167 次/分钟
```

#### 方案 2：大幅提速
```bash
GLOBAL_API_RATE_LIMIT=3000    # 3000 次/3分钟 = 1000 次/分钟
GLOBAL_WEB_RATE_LIMIT=1500    # 1500 次/3分钟 = 500 次/分钟
```

#### 方案 3：极高速度（需谨慎）
```bash
GLOBAL_API_RATE_LIMIT=10000   # 10000 次/3分钟 ≈ 3333 次/分钟
GLOBAL_WEB_RATE_LIMIT=5000    # 5000 次/3分钟 ≈ 1667 次/分钟
```

**注意**：过高的限流可能导致：
- 服务器负载过高
- 上游 API 触发 429 错误
- 数据库压力增大

## 二、部署方式

### 方式 1：Docker Compose 部署

编辑 `docker-compose.yml`，在 `environment` 部分添加限流配置：

```yaml
services:
  one-api:
    image: "${REGISTRY:-docker.io}/justsong/one-api:latest"
    container_name: one-api
    restart: always
    command: --log-dir /app/logs
    ports:
      - "3000:3000"
    volumes:
      - ./data/oneapi:/data
      - ./logs:/app/logs
    environment:
      - SQL_DSN=oneapi:123456@tcp(db:3306)/one-api
      - REDIS_CONN_STRING=redis://redis
      - SESSION_SECRET=random_string
      - TZ=Asia/Shanghai
      # 提速配置
      - GLOBAL_API_RATE_LIMIT=1000
      - GLOBAL_WEB_RATE_LIMIT=500
      # 429 错误优化配置（已实现，可选）
      - CHANNEL_429_AUTO_DISABLE=true
      - CHANNEL_429_DISABLE_DURATION=300
      - RETRY_BACKOFF_ENABLED=true
      - RETRY_BACKOFF_BASE=1
      - RETRY_BACKOFF_MAX=10
    depends_on:
      - redis
      - db
```

然后重启服务：
```bash
docker-compose down
docker-compose up -d
```

### 方式 2：直接运行部署

创建或编辑 `.env` 文件（如果使用环境变量文件）：

```bash
# 限流提速配置
GLOBAL_API_RATE_LIMIT=1000
GLOBAL_WEB_RATE_LIMIT=500

# 429 错误优化配置
CHANNEL_429_AUTO_DISABLE=true
CHANNEL_429_DISABLE_DURATION=300
RETRY_BACKOFF_ENABLED=true
RETRY_BACKOFF_BASE=1
RETRY_BACKOFF_MAX=10
```

或者在启动时直接设置环境变量：

```bash
export GLOBAL_API_RATE_LIMIT=1000
export GLOBAL_WEB_RATE_LIMIT=500
export CHANNEL_429_AUTO_DISABLE=true
export CHANNEL_429_DISABLE_DURATION=300
export RETRY_BACKOFF_ENABLED=true
export RETRY_BACKOFF_BASE=1
export RETRY_BACKOFF_MAX=10

./one-api --port 3000
```

### 方式 3：Systemd 服务部署

编辑 systemd 服务文件（如 `/etc/systemd/system/one-api.service`）：

```ini
[Unit]
Description=One API
After=network.target

[Service]
Type=simple
User=oneapi
WorkingDirectory=/opt/one-api
ExecStart=/opt/one-api/one-api --port 3000
Restart=always
RestartSec=5

# 限流提速配置
Environment="GLOBAL_API_RATE_LIMIT=1000"
Environment="GLOBAL_WEB_RATE_LIMIT=500"

# 429 错误优化配置
Environment="CHANNEL_429_AUTO_DISABLE=true"
Environment="CHANNEL_429_DISABLE_DURATION=300"
Environment="RETRY_BACKOFF_ENABLED=true"
Environment="RETRY_BACKOFF_BASE=1"
Environment="RETRY_BACKOFF_MAX=10"

[Install]
WantedBy=multi-user.target
```

然后重新加载并重启服务：

```bash
sudo systemctl daemon-reload
sudo systemctl restart one-api
```

## 三、429 错误优化配置说明

以下配置已经在代码中实现，只需设置环境变量即可启用：

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `CHANNEL_429_AUTO_DISABLE` | `true` | 是否自动临时禁用返回 429 的渠道 |
| `CHANNEL_429_DISABLE_DURATION` | `300` | 429 错误临时禁用时间（秒），默认 5 分钟 |
| `RETRY_BACKOFF_ENABLED` | `true` | 是否启用指数退避重试 |
| `RETRY_BACKOFF_BASE` | `1` | 重试基础延迟（秒） |
| `RETRY_BACKOFF_MAX` | `10` | 重试最大延迟（秒） |

## 四、完整配置示例

### 推荐配置（平衡性能和稳定性）

```bash
# ========== 限流提速配置 ==========
# API 限流：1000 次/3分钟（约 333 次/分钟）
GLOBAL_API_RATE_LIMIT=1000

# Web 限流：500 次/3分钟（约 167 次/分钟）
GLOBAL_WEB_RATE_LIMIT=500

# ========== 429 错误优化配置 ==========
# 自动临时禁用返回 429 的渠道
CHANNEL_429_AUTO_DISABLE=true

# 429 错误临时禁用时间：5 分钟
CHANNEL_429_DISABLE_DURATION=300

# 启用指数退避重试
RETRY_BACKOFF_ENABLED=true

# 重试基础延迟：1 秒
RETRY_BACKOFF_BASE=1

# 重试最大延迟：10 秒
RETRY_BACKOFF_MAX=10

# ========== 其他可选配置 ==========
# 重试次数（如果未设置，使用系统默认值）
# RETRY_TIMES=3

# 启用监控（可选）
# ENABLE_METRIC=true
# METRIC_QUEUE_SIZE=10
# METRIC_SUCCESS_RATE_THRESHOLD=0.8
```

## 五、验证配置

### 1. 检查环境变量

```bash
# Docker 方式
docker exec one-api env | grep -E "GLOBAL_API_RATE_LIMIT|CHANNEL_429"

# 直接运行方式
env | grep -E "GLOBAL_API_RATE_LIMIT|CHANNEL_429"
```

### 2. 查看日志

检查启动日志，确认配置已加载：

```bash
# Docker 方式
docker logs one-api | grep -i "rate limit\|429"

# 直接运行方式
tail -f logs/one-api.log | grep -i "rate limit\|429"
```

### 3. 测试 API 速度

使用以下命令测试 API 限流是否已提高：

```bash
# 快速发送多个请求测试
for i in {1..10}; do
  curl -X POST http://localhost:3000/v1/chat/completions \
    -H "Authorization: Bearer YOUR_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"model":"gpt-3.5-turbo","messages":[{"role":"user","content":"test"}]}' &
done
wait
```

## 六、性能监控建议

### 监控指标

1. **限流触发频率**：监控 429 错误日志
2. **请求成功率**：监控成功请求 vs 失败请求
3. **响应时间**：监控平均响应时间
4. **服务器负载**：监控 CPU、内存使用率

### 告警设置

建议设置以下告警：
- 429 错误率 > 10%
- 请求成功率 < 90%
- 服务器 CPU 使用率 > 80%

## 七、注意事项

### ⚠️ 重要提示

1. **逐步调整**：不要一次性将限流设置得过高，建议逐步调整并观察效果
2. **监控上游 API**：提高限流可能导致上游 API 触发 429 错误，需要监控上游状态
3. **数据库性能**：高并发可能增加数据库压力，确保数据库性能足够
4. **Redis 性能**：如果使用 Redis，确保 Redis 性能足够支持高并发
5. **服务器资源**：确保服务器 CPU、内存、网络带宽足够

### 调优建议

1. **根据实际需求调整**：
   - 如果主要是单用户使用，可以设置较高限流
   - 如果是多用户共享，需要平衡各用户的需求

2. **观察上游 API 限制**：
   - 不同上游 API 有不同的限流策略
   - 建议根据上游 API 的限制来调整 one-api 的限流

3. **定期检查**：
   - 定期检查日志，查看是否有频繁的 429 错误
   - 根据实际情况调整配置

## 八、回滚方案

如果提速后出现问题，可以快速回滚：

### 回滚到默认配置

```bash
# 方式 1：移除环境变量（使用默认值）
# 在 docker-compose.yml 中删除 GLOBAL_API_RATE_LIMIT 等配置

# 方式 2：设置为默认值
GLOBAL_API_RATE_LIMIT=480
GLOBAL_WEB_RATE_LIMIT=240

# 然后重启服务
docker-compose restart one-api
# 或
sudo systemctl restart one-api
```

## 九、常见问题

### Q1: 提高限流后还是感觉慢？

**A**: 可能的原因：
- 上游 API 本身有限流
- 网络延迟
- 429 错误优化未生效（检查配置）

### Q2: 429 错误仍然频繁？

**A**: 检查：
- `CHANNEL_429_AUTO_DISABLE` 是否设置为 `true`
- 查看日志确认 429 错误是否被记录
- 检查上游 API 的限流策略

### Q3: 如何知道当前限流配置？

**A**: 
- 查看启动日志
- 检查环境变量
- 查看系统设置页面（如果有相关显示）

## 十、总结

通过以上配置，可以实现：
- ✅ **限流速度提升 2-10 倍**（根据配置）
- ✅ **429 错误自动处理**（临时禁用 + 智能重试）
- ✅ **更好的用户体验**（更快的响应速度）

建议从**方案 1（中等提速）**开始，根据实际效果逐步调整。
