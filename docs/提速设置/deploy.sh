#!/bin/bash

# One-API 提速部署脚本
# 支持多种部署方式：Docker Compose、Systemd、直接运行

set -e

echo "=========================================="
echo "One-API 提速部署脚本"
echo "=========================================="
echo ""

# 检测部署方式
DEPLOY_TYPE=""

# 检查 Docker Compose
if command -v docker-compose &> /dev/null || command -v docker &> /dev/null; then
    if [ -f "docker-compose.yml" ]; then
        DEPLOY_TYPE="docker"
    fi
fi

# 检查 Systemd
if systemctl list-units --type=service | grep -q "one-api" 2>/dev/null; then
    DEPLOY_TYPE="systemd"
fi

# 检查直接运行
if [ -f "one-api" ] && [ -x "one-api" ]; then
    DEPLOY_TYPE="binary"
fi

# 获取脚本所在目录的父目录（项目根目录）
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$PROJECT_ROOT"

# 如果没有检测到，尝试 Docker Compose（最常见）
if [ -z "$DEPLOY_TYPE" ] && [ -f "docker-compose.yml" ]; then
    echo "检测到 docker-compose.yml，尝试使用 Docker Compose 部署..."
    DEPLOY_TYPE="docker"
fi

case "$DEPLOY_TYPE" in
    docker)
        echo "使用 Docker Compose 部署..."
        echo ""
        
        # 检查使用 docker-compose 还是 docker compose
        if command -v docker-compose &> /dev/null; then
            DOCKER_CMD="docker-compose"
        elif docker compose version &> /dev/null; then
            DOCKER_CMD="docker compose"
        else
            echo "错误: 未找到 docker-compose 或 docker compose 命令"
            exit 1
        fi
        
        echo "1. 停止现有服务..."
        $DOCKER_CMD down
        
        echo ""
        echo "2. 启动服务（加载新配置）..."
        $DOCKER_CMD up -d
        
        echo ""
        echo "3. 等待服务启动..."
        sleep 3
        
        echo ""
        echo "4. 检查服务状态..."
        $DOCKER_CMD ps
        
        echo ""
        echo "5. 验证配置..."
        echo "检查环境变量:"
        $DOCKER_CMD exec one-api env | grep -E "GLOBAL_API_RATE_LIMIT|CHANNEL_429" || echo "（如果容器未完全启动，请稍后运行: $DOCKER_CMD exec one-api env | grep -E 'GLOBAL_API_RATE_LIMIT|CHANNEL_429'）"
        
        echo ""
        echo "6. 查看日志（最后 20 行）..."
        $DOCKER_CMD logs --tail=20 one-api
        
        echo ""
        echo "✅ 部署完成！"
        echo ""
        echo "查看完整日志: $DOCKER_CMD logs -f one-api"
        ;;
        
    systemd)
        echo "使用 Systemd 服务部署..."
        echo ""
        
        echo "1. 重新加载 systemd 配置..."
        sudo systemctl daemon-reload
        
        echo ""
        echo "2. 重启服务..."
        sudo systemctl restart one-api
        
        echo ""
        echo "3. 检查服务状态..."
        sudo systemctl status one-api --no-pager
        
        echo ""
        echo "✅ 部署完成！"
        echo ""
        echo "查看日志: sudo journalctl -u one-api -f"
        ;;
        
    binary)
        echo "检测到可执行文件，使用直接运行方式..."
        echo ""
        echo "⚠️  注意: 直接运行方式需要手动设置环境变量"
        echo ""
        echo "请设置以下环境变量后运行:"
        echo ""
        echo "export GLOBAL_API_RATE_LIMIT=1000"
        echo "export GLOBAL_WEB_RATE_LIMIT=500"
        echo "export CHANNEL_429_AUTO_DISABLE=true"
        echo "export CHANNEL_429_DISABLE_DURATION=300"
        echo "export RETRY_BACKOFF_ENABLED=true"
        echo "export RETRY_BACKOFF_BASE=1"
        echo "export RETRY_BACKOFF_MAX=10"
        echo ""
        echo "然后运行: ./one-api --port 3000"
        echo ""
        echo "或者创建 .env 文件并设置这些变量"
        ;;
        
    *)
        echo "⚠️  未检测到运行中的服务"
        echo ""
        echo "请选择部署方式:"
        echo ""
        echo "1. Docker Compose (推荐)"
        echo "   运行: docker-compose up -d"
        echo "   或: docker compose up -d"
        echo ""
        echo "2. Systemd 服务"
        echo "   编辑 one-api.service 文件，添加环境变量"
        echo "   然后运行: sudo systemctl daemon-reload && sudo systemctl restart one-api"
        echo ""
        echo "3. 直接运行"
        echo "   设置环境变量后运行: ./one-api --port 3000"
        echo ""
        echo "详细说明请查看: DEPLOY_SPEED_OPTIMIZATION.md"
        exit 1
        ;;
esac

echo ""
echo "=========================================="
echo "配置说明:"
echo "- API 限流: 1000 次/3分钟 (提升 2 倍)"
echo "- Web 限流: 500 次/3分钟 (提升 2 倍)"
echo "- 429 错误优化: 已启用"
echo "=========================================="
