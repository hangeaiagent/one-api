# 立即部署 - 完整步骤

## ✅ 已完成的配置

1. ✅ `docker-compose.yml` 已更新（包含提速配置）
2. ✅ `.env` 文件已创建（包含所有配置）

## 🚀 立即部署步骤

### 情况 1：使用 Docker（如果已安装 Docker Desktop）

```bash
cd /Users/a1/work/one-api

# 启动 Docker Desktop 应用（如果还没启动）

# 部署
docker compose up -d

# 查看日志
docker compose logs -f one-api
```

### 情况 2：已有编译好的二进制文件

```bash
cd /Users/a1/work/one-api

# 1. 停止现有进程（如果有）
pkill -f "one-api" || true

# 2. 启动服务（.env 文件会自动加载）
./one-api --port 3000 > logs/one-api.log 2>&1 &

# 3. 查看日志
tail -f logs/one-api.log
```

### 情况 3：需要编译（如果有 Go 环境）

```bash
cd /Users/a1/work/one-api

# 1. 编译
go build -o one-api

# 2. 停止现有进程
pkill -f "one-api" || true

# 3. 启动服务
./one-api --port 3000 > logs/one-api.log 2>&1 &

# 4. 查看日志
tail -f logs/one-api.log
```

### 情况 4：使用其他部署方式

请查看：
- `DEPLOY_MACOS.md` - macOS 详细部署指南
- `DEPLOY_SPEED_OPTIMIZATION.md` - 完整部署文档

## 📋 当前配置

`.env` 文件已包含以下配置：

```
GLOBAL_API_RATE_LIMIT=1000        # API 限流：1000 次/3分钟（提升 2 倍）
GLOBAL_WEB_RATE_LIMIT=500         # Web 限流：500 次/3分钟（提升 2 倍）
CHANNEL_429_AUTO_DISABLE=true     # 自动处理 429 错误
CHANNEL_429_DISABLE_DURATION=300  # 429 错误临时禁用 5 分钟
RETRY_BACKOFF_ENABLED=true        # 启用指数退避重试
RETRY_BACKOFF_BASE=1              # 重试基础延迟 1 秒
RETRY_BACKOFF_MAX=10              # 重试最大延迟 10 秒
```

## ✅ 验证部署

### 检查服务是否运行

```bash
# 检查进程
ps aux | grep one-api | grep -v grep

# 检查端口
lsof -i :3000

# 测试 API
curl http://localhost:3000/api/status
```

### 检查配置是否生效

查看日志中是否有相关配置信息，或测试 API 响应速度。

## 📝 下一步

1. 根据您的部署方式选择上面的步骤
2. 启动服务
3. 验证配置已生效
4. 开始使用提速后的服务

## ❓ 需要帮助？

- 详细文档：`DEPLOY_SPEED_OPTIMIZATION.md`
- macOS 指南：`DEPLOY_MACOS.md`
- 快速指南：`QUICK_DEPLOY.md`
