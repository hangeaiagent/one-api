#!/bin/bash

# 在服务器上通过 gcloud 配置项目防火墙规则
# 在服务器 104.197.139.51 上执行
# 需要服务器上已安装 gcloud CLI

set -e

echo "=========================================="
echo "Google Cloud 防火墙配置（从服务器执行）"
echo "=========================================="
echo ""

# 检查 gcloud 是否安装
if ! command -v gcloud &> /dev/null; then
    echo "错误: gcloud CLI 未安装"
    echo ""
    echo "安装方法："
    echo "  # 添加 Google Cloud 官方源"
    echo "  echo \"deb [signed-by=/usr/share/keyrings/cloud.google.gpg] https://packages.cloud.google.com/apt cloud-sdk main\" | sudo tee -a /etc/apt/sources.list.d/google-cloud-sdk.list"
    echo "  curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key --keyring /usr/share/keyrings/cloud.google.gpg add -"
    echo "  sudo apt-get update && sudo apt-get install google-cloud-sdk"
    echo ""
    echo "或者访问: https://cloud.google.com/sdk/docs/install"
    exit 1
fi

# 检查是否已登录
if ! gcloud auth list --filter=status:ACTIVE --format="value(account)" | grep -q .; then
    echo "未检测到活动的 gcloud 认证"
    echo "请先登录:"
    echo "  gcloud auth login"
    exit 1
fi

# 获取当前项目
PROJECT_ID=$(gcloud config get-value project 2>/dev/null | grep -v "^$" || echo "")

if [ -z "$PROJECT_ID" ]; then
    echo "错误: 未设置项目 ID"
    echo ""
    echo "请先设置项目："
    echo "  gcloud config set project YOUR_PROJECT_ID"
    echo ""
    echo "或查看可用项目："
    echo "  gcloud projects list"
    exit 1
fi

echo "当前项目: $PROJECT_ID"
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
