#!/bin/bash
echo "=========================================="
echo "One-API 快速部署（macOS）"
echo "=========================================="
echo ""

# 获取脚本所在目录的父目录（项目根目录）
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$PROJECT_ROOT"

# 1. 创建 .env 文件
echo "1. 创建 .env 配置文件..."
cat > .env << 'ENVEOF'
GLOBAL_API_RATE_LIMIT=1000
GLOBAL_WEB_RATE_LIMIT=500
CHANNEL_429_AUTO_DISABLE=true
CHANNEL_429_DISABLE_DURATION=300
RETRY_BACKOFF_ENABLED=true
RETRY_BACKOFF_BASE=1
RETRY_BACKOFF_MAX=10
ENVEOF
echo "✅ .env 文件已创建"

# 2. 停止现有进程
echo ""
echo "2. 停止现有进程..."
pkill -f "one-api" 2>/dev/null && echo "✅ 已停止现有进程" || echo "ℹ️  没有运行中的进程"

# 3. 等待进程完全停止
sleep 1

# 4. 检查是否有编译好的二进制文件
if [ ! -f "./one-api" ]; then
    echo ""
    echo "3. 编译程序..."
    if command -v go &> /dev/null; then
        go build -o one-api
        echo "✅ 编译完成"
    else
        echo "❌ 错误: 未找到 Go 编译器，请先安装 Go 或使用已编译的二进制文件"
        exit 1
    fi
else
    echo ""
    echo "3. 使用已编译的二进制文件"
fi

# 5. 创建日志目录
mkdir -p logs

# 6. 启动服务
echo ""
echo "4. 启动服务..."
nohup ./one-api --port 3000 > logs/one-api.log 2>&1 &
PID=$!
echo "✅ 服务已启动 (PID: $PID)"

# 7. 等待启动
echo ""
echo "5. 等待服务启动..."
sleep 3

# 8. 检查进程
echo ""
echo "6. 检查服务状态..."
if ps -p $PID > /dev/null 2>&1; then
    echo "✅ 服务正在运行 (PID: $PID)"
    echo ""
    echo "7. 查看日志（最后 20 行）..."
    tail -20 logs/one-api.log
    echo ""
    echo "=========================================="
    echo "✅ 部署完成！"
    echo "=========================================="
    echo ""
    echo "服务信息:"
    echo "- PID: $PID"
    echo "- 端口: 3000"
    echo "- 日志: logs/one-api.log"
    echo ""
    echo "查看日志: tail -f logs/one-api.log"
    echo "停止服务: kill $PID"
    echo "测试 API: curl http://localhost:3000/api/status"
else
    echo "❌ 服务启动失败，请查看日志:"
    tail -50 logs/one-api.log
    exit 1
fi
