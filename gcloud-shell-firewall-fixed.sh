#!/bin/bash
# Google Cloud Shell 防火墙配置脚本（修复版）
# 支持直接指定项目 ID 或使用当前项目

# 使用方法：
# 1. 使用当前项目: ./gcloud-shell-firewall-fixed.sh
# 2. 指定项目: PROJECT_ID="your-project-id" ./gcloud-shell-firewall-fixed.sh

set -e

# 如果通过环境变量指定了项目，使用它；否则尝试获取当前项目
if [ -z "$PROJECT_ID" ]; then
    PROJECT_ID=$(gcloud config get-value project 2>/dev/null | grep -v "^$" || echo "")
fi

# 如果还是没有项目 ID，提示用户
if [ -z "$PROJECT_ID" ]; then
    echo "=========================================="
    echo "错误: 未设置项目 ID"
    echo "=========================================="
    echo ""
    echo "请使用以下方法之一："
    echo ""
    echo "方法 1: 先设置项目，然后运行脚本"
    echo "  gcloud config set project YOUR_PROJECT_ID"
    echo "  ./gcloud-shell-firewall-fixed.sh"
    echo ""
    echo "方法 2: 在命令中直接指定项目"
    echo "  PROJECT_ID=\"your-project-id\" ./gcloud-shell-firewall-fixed.sh"
    echo ""
    echo "方法 3: 查看可用项目列表"
    echo "  gcloud projects list"
    echo ""
    exit 1
fi

echo "=========================================="
echo "Google Cloud 防火墙配置"
echo "=========================================="
echo "项目: $PROJECT_ID"
echo "配置端口: 80 (HTTP), 443 (HTTPS)"
echo ""

RULE_HTTP="allow-http-80-oneapi"
RULE_HTTPS="allow-https-443-oneapi"

# 创建 HTTP (80) 端口规则
echo "配置 HTTP (80) 端口..."
if gcloud compute firewall-rules describe "$RULE_HTTP" --project="$PROJECT_ID" &>/dev/null 2>&1; then
    echo "  ✓ 规则已存在，正在更新..."
    gcloud compute firewall-rules update "$RULE_HTTP" \
        --allow tcp:80 \
        --source-ranges 0.0.0.0/0 \
        --description "Allow HTTP traffic for One-API" \
        --project="$PROJECT_ID" \
        --quiet
    echo "  ✓ HTTP 规则已更新"
else
    gcloud compute firewall-rules create "$RULE_HTTP" \
        --allow tcp:80 \
        --source-ranges 0.0.0.0/0 \
        --description "Allow HTTP traffic for One-API" \
        --project="$PROJECT_ID" \
        --quiet
    echo "  ✓ HTTP 规则创建成功"
fi

echo ""

# 创建 HTTPS (443) 端口规则
echo "配置 HTTPS (443) 端口..."
if gcloud compute firewall-rules describe "$RULE_HTTPS" --project="$PROJECT_ID" &>/dev/null 2>&1; then
    echo "  ✓ 规则已存在，正在更新..."
    gcloud compute firewall-rules update "$RULE_HTTPS" \
        --allow tcp:443 \
        --source-ranges 0.0.0.0/0 \
        --description "Allow HTTPS traffic for One-API" \
        --project="$PROJECT_ID" \
        --quiet
    echo "  ✓ HTTPS 规则已更新"
else
    gcloud compute firewall-rules create "$RULE_HTTPS" \
        --allow tcp:443 \
        --source-ranges 0.0.0.0/0 \
        --description "Allow HTTPS traffic for One-API" \
        --project="$PROJECT_ID" \
        --quiet
    echo "  ✓ HTTPS 规则创建成功"
fi

echo ""
echo "=========================================="
echo "配置完成！"
echo "=========================================="
echo ""
echo "已配置的防火墙规则："
gcloud compute firewall-rules list \
    --filter="name=$RULE_HTTP OR name=$RULE_HTTPS" \
    --format="table(name,allowed,direction,sourceRanges)" \
    --project="$PROJECT_ID"

echo ""
echo "现在可以通过以下方式访问："
echo "  - HTTP:  http://oneapi.gitagent.io"
echo "  - HTTPS: https://oneapi.gitagent.io (配置 SSL 证书后)"
echo ""
