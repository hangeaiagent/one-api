#!/bin/bash

# Google Cloud Shell 防火墙配置脚本
# 直接在 Google Cloud Shell 中执行
# 用于允许 HTTP (80) 和 HTTPS (443) 端口

set -e

echo "=========================================="
echo "Google Cloud 防火墙配置"
echo "=========================================="
echo ""

# 获取当前项目
PROJECT_ID=$(gcloud config get-value project 2>/dev/null | grep -v "^$")

if [ -z "$PROJECT_ID" ]; then
    echo "错误: 未设置项目 ID"
    echo ""
    echo "请先设置项目，可以使用以下方法："
    echo ""
    echo "方法 1: 查看可用项目列表"
    echo "  gcloud projects list"
    echo ""
    echo "方法 2: 设置项目"
    echo "  gcloud config set project YOUR_PROJECT_ID"
    echo ""
    echo "方法 3: 在脚本中指定项目（修改下面的 PROJECT_ID 变量）"
    exit 1
fi

echo "当前项目: $PROJECT_ID"
echo "配置端口: 80 (HTTP), 443 (HTTPS)"
echo ""

# 规则名称
RULE_HTTP="allow-http-80-oneapi"
RULE_HTTPS="allow-https-443-oneapi"

# 创建 HTTP (80) 端口规则
echo "配置 HTTP (80) 端口..."
if gcloud compute firewall-rules describe "$RULE_HTTP" --project="$PROJECT_ID" &>/dev/null 2>&1; then
    echo "  ✓ 规则 $RULE_HTTP 已存在"
    # 更新规则以确保配置正确
    gcloud compute firewall-rules update "$RULE_HTTP" \
        --allow tcp:80 \
        --source-ranges 0.0.0.0/0 \
        --description "Allow HTTP traffic for One-API" \
        --project="$PROJECT_ID" \
        --quiet
    echo "  ✓ 规则已更新"
else
    gcloud compute firewall-rules create "$RULE_HTTP" \
        --allow tcp:80 \
        --source-ranges 0.0.0.0/0 \
        --description "Allow HTTP traffic for One-API" \
        --project="$PROJECT_ID" \
        --quiet
    echo "  ✓ 规则 $RULE_HTTP 创建成功"
fi

echo ""

# 创建 HTTPS (443) 端口规则
echo "配置 HTTPS (443) 端口..."
if gcloud compute firewall-rules describe "$RULE_HTTPS" --project="$PROJECT_ID" &>/dev/null 2>&1; then
    echo "  ✓ 规则 $RULE_HTTPS 已存在"
    # 更新规则以确保配置正确
    gcloud compute firewall-rules update "$RULE_HTTPS" \
        --allow tcp:443 \
        --source-ranges 0.0.0.0/0 \
        --description "Allow HTTPS traffic for One-API" \
        --project="$PROJECT_ID" \
        --quiet
    echo "  ✓ 规则已更新"
else
    gcloud compute firewall-rules create "$RULE_HTTPS" \
        --allow tcp:443 \
        --source-ranges 0.0.0.0/0 \
        --description "Allow HTTPS traffic for One-API" \
        --project="$PROJECT_ID" \
        --quiet
    echo "  ✓ 规则 $RULE_HTTPS 创建成功"
fi

echo ""
echo "=========================================="
echo "配置完成！"
echo "=========================================="
echo ""

# 显示创建的规则
echo "已配置的防火墙规则："
echo ""
gcloud compute firewall-rules list \
    --filter="name=$RULE_HTTP OR name=$RULE_HTTPS" \
    --format="table(
        name,
        allowed[].map().firewall_rule().list():label=ALLOW,
        direction,
        sourceRanges.list():label=SRC_RANGES,
        targetTags.list():label=TARGET_TAGS
    )" \
    --project="$PROJECT_ID"

echo ""
echo "=========================================="
echo "验证信息"
echo "=========================================="
echo ""
echo "现在可以通过以下方式访问："
echo "  - HTTP:  http://oneapi.gitagent.io"
echo "  - HTTPS: https://oneapi.gitagent.io (配置 SSL 证书后)"
echo ""
echo "如果无法访问，请检查："
echo "  1. DNS 记录是否已配置 (oneapi.gitagent.io -> 104.197.139.51)"
echo "  2. 实例上的 nginx 服务是否运行"
echo "  3. 实例操作系统防火墙是否允许端口 80 和 443"
echo ""
