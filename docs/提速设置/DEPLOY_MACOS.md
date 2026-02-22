# macOS 部署指南

## 当前环境

检测到您使用的是 macOS 系统。以下是针对 macOS 的部署步骤。

## 部署方式选择

### 方式 1：Docker Desktop（如果已安装）

如果已安装 Docker Desktop：

```bash
cd /Users/a1/work/one-api

# 检查 Docker 是否运行
docker ps

# 如果 Docker 未运行，启动 Docker Desktop 应用

# 停止现有服务
docker compose down

# 启动服务（加载新配置）
docker compose up -d

# 查看日志
docker compose logs -f one-api

# 验证配置
docker compose exec one-api env | grep -E "GLOBAL_API_RATE_LIMIT|CHANNEL_429"
```

### 方式 2：直接运行 Go 程序

如果直接运行编译后的程序：

#### 步骤 1：编译程序（如果还没有编译）

```bash
cd /Users/a1/work/one-api
go build -o one-api
```

#### 步骤 2：创建环境变量文件

```bash
cat > .env << 'EOF'
# 限流提速配置
GLOBAL_API_RATE_LIMIT=1000
GLOBAL_WEB_RATE_LIMIT=500

# 429 错误优化配置
CHANNEL_429_AUTO_DISABLE=true
CHANNEL_429_DISABLE_DURATION=300
RETRY_BACKOFF_ENABLED=true
RETRY_BACKOFF_BASE=1
RETRY_BACKOFF_MAX=10
EOF
```

#### 步骤 3：停止现有进程（如果有）

```bash
# 查找运行中的 one-api 进程
ps aux | grep one-api | grep -v grep

# 如果找到进程，停止它（替换 PID 为实际进程 ID）
kill <PID>
```

#### 步骤 4：启动服务

```bash
# 方式 A：前台运行（用于测试）
./one-api --port 3000

# 方式 B：后台运行
nohup ./one-api --port 3000 > logs/one-api.log 2>&1 &

# 方式 C：使用 launchd（macOS 推荐）
# 见下面的 launchd 配置
```

### 方式 3：使用 launchd（macOS 系统服务）

#### 创建 launchd plist 文件

```bash
cat > ~/Library/LaunchAgents/com.oneapi.plist << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.oneapi</string>
    <key>ProgramArguments</key>
    <array>
        <string>/Users/a1/work/one-api/one-api</string>
        <string>--port</string>
        <string>3000</string>
        <string>--log-dir</string>
        <string>/Users/a1/work/one-api/logs</string>
    </array>
    <key>WorkingDirectory</key>
    <string>/Users/a1/work/one-api</string>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/Users/a1/work/one-api/logs/one-api.out.log</string>
    <key>StandardErrorPath</key>
    <string>/Users/a1/work/one-api/logs/one-api.err.log</string>
    <key>EnvironmentVariables</key>
    <dict>
        <key>GLOBAL_API_RATE_LIMIT</key>
        <string>1000</string>
        <key>GLOBAL_WEB_RATE_LIMIT</key>
        <string>500</string>
        <key>CHANNEL_429_AUTO_DISABLE</key>
        <string>true</string>
        <key>CHANNEL_429_DISABLE_DURATION</key>
        <string>300</string>
        <key>RETRY_BACKOFF_ENABLED</key>
        <string>true</string>
        <key>RETRY_BACKOFF_BASE</key>
        <string>1</string>
        <key>RETRY_BACKOFF_MAX</key>
        <string>10</string>
    </dict>
</dict>
</plist>
EOF
```

#### 加载并启动服务

```bash
# 加载服务
launchctl load ~/Library/LaunchAgents/com.oneapi.plist

# 启动服务
launchctl start com.oneapi

# 检查状态
launchctl list | grep oneapi

# 查看日志
tail -f ~/work/one-api/logs/one-api.out.log
```

#### 重启服务（应用新配置）

```bash
# 卸载服务
launchctl unload ~/Library/LaunchAgents/com.oneapi.plist

# 重新加载（会应用新配置）
launchctl load ~/Library/LaunchAgents/com.oneapi.plist

# 启动服务
launchctl start com.oneapi
```

## 快速部署命令（一键执行）

### 如果使用 .env 文件直接运行

```bash
cd /Users/a1/work/one-api

# 1. 创建 .env 文件（如果还没有）
cat > .env << 'EOF'
GLOBAL_API_RATE_LIMIT=1000
GLOBAL_WEB_RATE_LIMIT=500
CHANNEL_429_AUTO_DISABLE=true
CHANNEL_429_DISABLE_DURATION=300
RETRY_BACKOFF_ENABLED=true
RETRY_BACKOFF_BASE=1
RETRY_BACKOFF_MAX=10
EOF

# 2. 停止现有进程
pkill -f "one-api" || true

# 3. 编译（如果需要）
go build -o one-api || echo "使用已编译的二进制文件"

# 4. 启动服务（后台运行）
nohup ./one-api --port 3000 > logs/one-api.log 2>&1 &

# 5. 等待启动
sleep 2

# 6. 检查进程
ps aux | grep one-api | grep -v grep

# 7. 查看日志
tail -f logs/one-api.log
```

## 验证部署

### 检查进程

```bash
ps aux | grep one-api | grep -v grep
```

### 检查端口

```bash
lsof -i :3000
```

### 测试 API

```bash
curl http://localhost:3000/api/status
```

### 检查环境变量（如果使用 .env）

```bash
# 查看 .env 文件
cat .env
```

## 常见问题

### Q: 如何查看日志？

```bash
# 如果使用 nohup
tail -f logs/one-api.log

# 如果使用 launchd
tail -f ~/work/one-api/logs/one-api.out.log
```

### Q: 如何停止服务？

```bash
# 方式 1：查找并停止进程
pkill -f "one-api"

# 方式 2：如果使用 launchd
launchctl stop com.oneapi
launchctl unload ~/Library/LaunchAgents/com.oneapi.plist
```

### Q: 如何重启服务？

```bash
# 停止
pkill -f "one-api"

# 启动
nohup ./one-api --port 3000 > logs/one-api.log 2>&1 &
```

## 下一步

1. 确认服务已启动：`ps aux | grep one-api`
2. 测试 API：`curl http://localhost:3000/api/status`
3. 查看日志确认配置已加载
4. 开始使用提速后的服务
