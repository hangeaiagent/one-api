#!/bin/bash

# One-API 服务测试脚本

SERVER="104.197.139.51"
PORT="3000"
BASE_URL="http://${SERVER}:${PORT}"

echo "=========================================="
echo "One-API 服务测试"
echo "服务器: ${SERVER}:${PORT}"
echo "=========================================="
echo ""

# 测试 1: 基本连接
echo "1. 测试基本连接..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout 5 "${BASE_URL}/api/status" 2>/dev/null)
if [ "$HTTP_CODE" = "200" ]; then
    echo "   ✅ 连接成功 (HTTP $HTTP_CODE)"
else
    echo "   ❌ 连接失败 (HTTP $HTTP_CODE)"
    exit 1
fi

# 测试 2: API 状态端点
echo ""
echo "2. 测试 API 状态端点..."
STATUS=$(curl -s --connect-timeout 5 "${BASE_URL}/api/status" 2>/dev/null)
if echo "$STATUS" | grep -q '"success":true'; then
    echo "   ✅ API 状态正常"
    echo "$STATUS" | python3 -m json.tool 2>/dev/null | head -10 || echo "$STATUS" | head -5
else
    echo "   ❌ API 状态异常"
    echo "$STATUS"
fi

# 测试 3: 检查服务信息
echo ""
echo "3. 检查服务信息..."
SYSTEM_NAME=$(echo "$STATUS" | grep -o '"system_name":"[^"]*"' | cut -d'"' -f4)
VERSION=$(echo "$STATUS" | grep -o '"version":"[^"]*"' | cut -d'"' -f4)
if [ -n "$SYSTEM_NAME" ]; then
    echo "   ✅ 系统名称: $SYSTEM_NAME"
    echo "   ✅ 版本: $VERSION"
else
    echo "   ⚠️  无法获取系统信息"
fi

# 测试 4: 检查端口监听
echo ""
echo "4. 检查远程服务状态..."
SSH_CMD="ssh -i ~/.ssh/id_rsa_google_longterm -o ConnectTimeout=10 -o StrictHostKeyChecking=no support@${SERVER}"
PROCESS=$($SSH_CMD "ps aux | grep -E 'one-api|oneapi' | grep -v grep | head -1" 2>/dev/null)
if [ -n "$PROCESS" ]; then
    echo "   ✅ 服务进程正在运行"
    echo "   $PROCESS" | awk '{print "   PID: "$2", CPU: "$3"%, MEM: "$4"%"}'
else
    echo "   ❌ 未找到运行中的服务进程"
fi

# 测试 5: 检查配置
echo ""
echo "5. 检查提速配置..."
ENV_CHECK=$($SSH_CMD "cd /mnt/disk-119/one-api && cat .env 2>/dev/null | grep -E 'GLOBAL_API_RATE_LIMIT|CHANNEL_429' | head -5" 2>/dev/null)
if [ -n "$ENV_CHECK" ]; then
    echo "   ✅ 提速配置已设置:"
    echo "$ENV_CHECK" | sed 's/^/      /'
else
    echo "   ⚠️  未找到配置文件或配置未设置"
fi

# 测试 6: 响应时间测试
echo ""
echo "6. 测试响应时间..."
for i in {1..3}; do
    TIME=$(curl -s -o /dev/null -w "%{time_total}" --connect-timeout 5 "${BASE_URL}/api/status" 2>/dev/null)
    if [ -n "$TIME" ]; then
        echo "   请求 $i: ${TIME} 秒"
    fi
done

echo ""
echo "=========================================="
echo "测试完成"
echo "=========================================="
echo ""
echo "服务地址: ${BASE_URL}"
echo "API 文档: ${BASE_URL}/docs (如果可用)"
echo ""
