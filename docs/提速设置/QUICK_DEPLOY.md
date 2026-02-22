# 快速部署提速方案

## 一键部署（Docker Compose）

### 步骤 1：更新配置

`docker-compose.yml` 已经更新，包含了提速配置：
- API 限流：1000 次/3分钟（提升 2 倍）
- Web 限流：500 次/3分钟（提升 2 倍）
- 429 错误自动优化（已启用）

### 步骤 2：重启服务

```bash
# 停止当前服务
docker-compose down

# 重新启动服务（会加载新配置）
docker-compose up -d

# 查看日志确认配置已加载
docker logs -f one-api
```

### 步骤 3：验证配置

```bash
# 检查环境变量
docker exec one-api env | grep -E "GLOBAL_API_RATE_LIMIT|CHANNEL_429"

# 应该看到：
# GLOBAL_API_RATE_LIMIT=1000
# CHANNEL_429_AUTO_DISABLE=true
```

## 自定义配置

如果需要调整限流速度，编辑 `docker-compose.yml` 中的以下行：

```yaml
- GLOBAL_API_RATE_LIMIT=1000  # 改为你需要的值，如 3000、5000 等
- GLOBAL_WEB_RATE_LIMIT=500   # 改为你需要的值
```

然后重启服务：
```bash
docker-compose restart one-api
```

## 配置说明

### 限流速度对比

| 配置 | 默认值 | 当前配置 | 提升倍数 |
|------|--------|----------|----------|
| API 限流 | 480 次/3分钟 | 1000 次/3分钟 | 2.08 倍 |
| Web 限流 | 240 次/3分钟 | 500 次/3分钟 | 2.08 倍 |

### 429 错误优化

以下优化已自动启用：
- ✅ 自动临时禁用返回 429 的渠道（5 分钟）
- ✅ 指数退避重试（1s → 2s → 4s → ...）
- ✅ 智能渠道选择（跳过被 429 阻止的渠道）

## 更高速度配置

如果需要更高速度，可以修改为：

```yaml
# 大幅提速（3 倍）
- GLOBAL_API_RATE_LIMIT=3000
- GLOBAL_WEB_RATE_LIMIT=1500

# 或极高速度（10 倍，需谨慎）
- GLOBAL_API_RATE_LIMIT=5000
- GLOBAL_WEB_RATE_LIMIT=2500
```

**注意**：过高的限流可能导致服务器负载过高或上游 API 触发 429 错误。

## 回滚到默认配置

如果需要回滚，编辑 `docker-compose.yml`，删除或注释掉以下行：

```yaml
# - GLOBAL_API_RATE_LIMIT=1000
# - GLOBAL_WEB_RATE_LIMIT=500
```

然后重启服务。

## 更多信息

详细说明请查看：`DEPLOY_SPEED_OPTIMIZATION.md`
