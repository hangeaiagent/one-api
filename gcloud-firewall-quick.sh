#!/bin/bash

# Google Cloud 防火墙快速配置脚本
# 快速配置 HTTP (80) 和 HTTPS (443) 端口

set -e

# 配置参数（可根据需要修改）
PROJECT_ID="${GCP_PROJECT_ID:-}"  # 从环境变量获取，或手动设置
INSTANCE_IP="104.197.139.51"
RULE_NAME_HTTP="allow-http-80-oneapi"
RULE_NAME_HTTPS="allow-https-443-oneapi"

# 如果没有设置项目 ID，尝试获取当前项目
if [ -z "$PROJECT_ID" ]; then
    PROJECT_ID=$(gcloud config get-value project 2>/dev/null || echo "")
fi

# 如果还是没有，提示输入
if [ -z "$PROJECT_ID" ]; then
    echo "请输入 Google Cloud 项目 ID:"
    read -p "项目 ID: " PROJECT_ID
    gcloud config set project "$PROJECT_ID"
fi

echo "使用项目: $PROJECT_ID"
echo "配置实例 IP: $INSTANCE_IP"
echo ""

# 创建 HTTP 规则
echo "配置 HTTP (80) 端口..."
if gcloud compute firewall-rules describe "$RULE_NAME_HTTP" --project="$PROJECT_ID" &>/dev/null 2>&1; then
    echo "规则已存在，跳过创建"
else
    gcloud compute firewall-rules create "$RULE_NAME_HTTP" \
        --allow tcp:80 \
        --source-ranges 0.0.0.0/0 \
        --description "Allow HTTP traffic for One-API" \
        --project="$PROJECT_ID" \
        --quiet
    echo "HTTP 规则创建成功"
fi

# 创建 HTTPS 规则
echo "配置 HTTPS (443) 端口..."
if gcloud compute firewall-rules describe "$RULE_NAME_HTTPS" --project="$PROJECT_ID" &>/dev/null 2>&1; then
    echo "规则已存在，跳过创建"
else
    gcloud compute firewall-rules create "$RULE_NAME_HTTPS" \
        --allow tcp:443 \
        --source-ranges 0.0.0.0/0 \
        --description "Allow HTTPS traffic for One-API" \
        --project="$PROJECT_ID" \
        --quiet
    echo "HTTPS 规则创建成功"
fi

echo ""
echo "=========================================="
echo "防火墙配置完成！"
echo "=========================================="
echo ""
echo "已创建的规则："
gcloud compute firewall-rules list \
    --filter="name=allow-http-80-oneapi OR name=allow-https-443-oneapi" \
    --format="table(name,allowed[].map().firewall_rule().list():label=ALLOW,direction,sourceRanges.list():label=SRC_RANGES)" \
    --project="$PROJECT_ID"
