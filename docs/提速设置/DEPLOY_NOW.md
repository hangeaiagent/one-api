# 立即部署指南

## 快速部署

### 方式 1：使用部署脚本（推荐）

```bash
cd /Users/a1/work/one-api
./deploy.sh
```

脚本会自动检测部署方式并执行相应的操作。

### 方式 2：Docker Compose

如果使用 Docker Compose 部署：

```bash
cd /Users/a1/work/one-api

# 停止服务
docker-compose down
# 或
docker compose down

# 启动服务（加载新配置）
docker-compose up -d
# 或
docker compose up -d

# 查看日志
docker-compose logs -f one-api
# 或
docker compose logs -f one-api

# 验证配置
docker-compose exec one-api env | grep -E "GLOBAL_API_RATE_LIMIT|CHANNEL_429"
# 或
docker compose exec one-api env | grep -E "GLOBAL_API_RATE_LIMIT|CHANNEL_429"
```

### 方式 3：Systemd 服务

如果使用 Systemd 服务：

1. **编辑服务文件**（如果还没有添加环境变量）：

```bash
sudo nano /etc/systemd/system/one-api.service
```

添加环境变量：

```ini
[Service]
# ... 其他配置 ...
Environment="GLOBAL_API_RATE_LIMIT=1000"
Environment="GLOBAL_WEB_RATE_LIMIT=500"
Environment="CHANNEL_429_AUTO_DISABLE=true"
Environment="CHANNEL_429_DISABLE_DURATION=300"
Environment="RETRY_BACKOFF_ENABLED=true"
Environment="RETRY_BACKOFF_BASE=1"
Environment="RETRY_BACKOFF_MAX=10"
```

2. **重新加载并重启**：

```bash
sudo systemctl daemon-reload
sudo systemctl restart one-api
sudo systemctl status one-api
```

3. **查看日志**：

```bash
sudo journalctl -u one-api -f
```

### 方式 4：直接运行

如果直接运行二进制文件：

1. **设置环境变量**：

```bash
export GLOBAL_API_RATE_LIMIT=1000
export GLOBAL_WEB_RATE_LIMIT=500
export CHANNEL_429_AUTO_DISABLE=true
export CHANNEL_429_DISABLE_DURATION=300
export RETRY_BACKOFF_ENABLED=true
export RETRY_BACKOFF_BASE=1
export RETRY_BACKOFF_MAX=10
```

2. **或者创建 .env 文件**：

```bash
cat > .env << EOF
GLOBAL_API_RATE_LIMIT=1000
GLOBAL_WEB_RATE_LIMIT=500
CHANNEL_429_AUTO_DISABLE=true
CHANNEL_429_DISABLE_DURATION=300
RETRY_BACKOFF_ENABLED=true
RETRY_BACKOFF_BASE=1
RETRY_BACKOFF_MAX=10
EOF
```

3. **运行服务**：

```bash
./one-api --port 3000
```

## 验证部署

### 检查配置是否生效

**Docker Compose**:
```bash
docker-compose exec one-api env | grep -E "GLOBAL_API_RATE_LIMIT|CHANNEL_429"
```

**Systemd**:
```bash
sudo systemctl show one-api | grep -E "GLOBAL_API_RATE_LIMIT|CHANNEL_429"
```

**直接运行**:
```bash
env | grep -E "GLOBAL_API_RATE_LIMIT|CHANNEL_429"
```

### 检查服务状态

**Docker Compose**:
```bash
docker-compose ps
docker-compose logs --tail=50 one-api
```

**Systemd**:
```bash
sudo systemctl status one-api
```

### 测试 API

```bash
# 快速测试（替换 YOUR_TOKEN 为实际 token）
curl -X POST http://localhost:3000/v1/chat/completions \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-3.5-turbo","messages":[{"role":"user","content":"test"}]}'
```

## 预期结果

部署成功后，你应该看到：

1. ✅ 服务正常运行
2. ✅ 环境变量已加载：
   - `GLOBAL_API_RATE_LIMIT=1000`
   - `CHANNEL_429_AUTO_DISABLE=true`
3. ✅ API 限流速度提升 2 倍（从 480 到 1000 次/3分钟）
4. ✅ 429 错误自动优化已启用

## 故障排查

### 如果服务无法启动

1. **检查日志**：
   ```bash
   # Docker Compose
   docker-compose logs one-api
   
   # Systemd
   sudo journalctl -u one-api -n 50
   ```

2. **检查配置语法**：
   - Docker Compose: `docker-compose config`
   - Systemd: `sudo systemctl daemon-reload` 后查看错误

3. **检查端口占用**：
   ```bash
   lsof -i :3000
   ```

### 如果配置未生效

1. **确认环境变量已设置**（见上面的验证步骤）
2. **重启服务**（确保新配置被加载）
3. **检查配置文件**（docker-compose.yml 或 systemd 服务文件）

## 需要帮助？

- 详细文档：`DEPLOY_SPEED_OPTIMIZATION.md`
- 快速指南：`QUICK_DEPLOY.md`
- 429 错误优化：`429_ERROR_SUMMARY.md`
