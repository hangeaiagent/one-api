#!/bin/bash

# Google Cloud 防火墙配置脚本
# 用于允许 HTTP (80) 和 HTTPS (443) 端口
# 服务器 IP: 104.197.139.51

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Google Cloud 防火墙配置脚本${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# 检查 gcloud 是否安装
if ! command -v gcloud &> /dev/null; then
    echo -e "${RED}错误: gcloud CLI 未安装${NC}"
    echo "请访问 https://cloud.google.com/sdk/docs/install 安装 gcloud CLI"
    exit 1
fi

# 检查是否已登录
if ! gcloud auth list --filter=status:ACTIVE --format="value(account)" | grep -q .; then
    echo -e "${YELLOW}未检测到活动的 gcloud 认证，正在登录...${NC}"
    gcloud auth login
fi

# 获取当前项目
CURRENT_PROJECT=$(gcloud config get-value project 2>/dev/null)

if [ -z "$CURRENT_PROJECT" ]; then
    echo -e "${YELLOW}未设置默认项目，请输入项目 ID:${NC}"
    read -p "项目 ID: " PROJECT_ID
    gcloud config set project "$PROJECT_ID"
else
    echo -e "${GREEN}当前项目: ${CURRENT_PROJECT}${NC}"
    read -p "是否使用当前项目? (y/n, 默认 y): " USE_CURRENT
    USE_CURRENT=${USE_CURRENT:-y}
    
    if [ "$USE_CURRENT" != "y" ]; then
        read -p "请输入项目 ID: " PROJECT_ID
        gcloud config set project "$PROJECT_ID"
    else
        PROJECT_ID="$CURRENT_PROJECT"
    fi
fi

echo ""
echo -e "${GREEN}使用项目: ${PROJECT_ID}${NC}"
echo ""

# 获取实例信息
echo "正在查找实例 104.197.139.51..."
INSTANCE_INFO=$(gcloud compute instances list --filter="EXTERNAL_IP=104.197.139.51" --format="json" 2>/dev/null || echo "[]")

if [ "$INSTANCE_INFO" = "[]" ] || [ -z "$INSTANCE_INFO" ]; then
    echo -e "${YELLOW}未找到 IP 为 104.197.139.51 的实例${NC}"
    echo "将创建适用于所有实例的防火墙规则"
    TARGET_TAG=""
else
    INSTANCE_NAME=$(echo "$INSTANCE_INFO" | grep -o '"name":"[^"]*' | head -1 | cut -d'"' -f4)
    ZONE=$(echo "$INSTANCE_INFO" | grep -o '"zone":"[^"]*' | head -1 | cut -d'"' -f4 | awk -F'/' '{print $NF}')
    TAGS=$(gcloud compute instances describe "$INSTANCE_NAME" --zone="$ZONE" --format="get(tags.items)" 2>/dev/null || echo "")
    
    echo -e "${GREEN}找到实例: ${INSTANCE_NAME} (区域: ${ZONE})${NC}"
    if [ -n "$TAGS" ]; then
        echo -e "${GREEN}实例标签: ${TAGS}${NC}"
    fi
    
    read -p "是否为此实例创建特定防火墙规则? (y/n, 默认 n): " CREATE_SPECIFIC
    CREATE_SPECIFIC=${CREATE_SPECIFIC:-n}
    
    if [ "$CREATE_SPECIFIC" = "y" ] && [ -n "$TAGS" ]; then
        echo "可用的标签: $TAGS"
        read -p "请输入要使用的标签 (留空使用默认规则): " TARGET_TAG
    fi
fi

echo ""
echo -e "${GREEN}开始配置防火墙规则...${NC}"
echo ""

# 函数：创建或更新防火墙规则
create_firewall_rule() {
    local RULE_NAME=$1
    local PORT=$2
    local PROTOCOL=$3
    local DESCRIPTION=$4
    local TARGET=$5
    
    echo -e "${YELLOW}配置规则: ${RULE_NAME} (端口 ${PORT})${NC}"
    
    # 检查规则是否已存在
    if gcloud compute firewall-rules describe "$RULE_NAME" --project="$PROJECT_ID" &>/dev/null; then
        echo -e "${YELLOW}规则 ${RULE_NAME} 已存在，正在更新...${NC}"
        
        if [ -n "$TARGET" ]; then
            gcloud compute firewall-rules update "$RULE_NAME" \
                --allow tcp:"$PORT" \
                --source-ranges 0.0.0.0/0 \
                --description "$DESCRIPTION" \
                --target-tags "$TARGET" \
                --project="$PROJECT_ID"
        else
            gcloud compute firewall-rules update "$RULE_NAME" \
                --allow tcp:"$PORT" \
                --source-ranges 0.0.0.0/0 \
                --description "$DESCRIPTION" \
                --project="$PROJECT_ID"
        fi
        
        echo -e "${GREEN}规则 ${RULE_NAME} 更新成功${NC}"
    else
        echo -e "${YELLOW}创建新规则: ${RULE_NAME}${NC}"
        
        if [ -n "$TARGET" ]; then
            gcloud compute firewall-rules create "$RULE_NAME" \
                --allow tcp:"$PORT" \
                --source-ranges 0.0.0.0/0 \
                --description "$DESCRIPTION" \
                --target-tags "$TARGET" \
                --project="$PROJECT_ID"
        else
            gcloud compute firewall-rules create "$RULE_NAME" \
                --allow tcp:"$PORT" \
                --source-ranges 0.0.0.0/0 \
                --description "$DESCRIPTION" \
                --project="$PROJECT_ID"
        fi
        
        echo -e "${GREEN}规则 ${RULE_NAME} 创建成功${NC}"
    fi
}

# 配置 HTTP (80) 端口
if [ -n "$TARGET_TAG" ]; then
    create_firewall_rule "allow-http-80-oneapi" "80" "tcp" "Allow HTTP traffic for One-API" "$TARGET_TAG"
else
    create_firewall_rule "allow-http-80-oneapi" "80" "tcp" "Allow HTTP traffic for One-API" ""
fi

# 配置 HTTPS (443) 端口
if [ -n "$TARGET_TAG" ]; then
    create_firewall_rule "allow-https-443-oneapi" "443" "tcp" "Allow HTTPS traffic for One-API" "$TARGET_TAG"
else
    create_firewall_rule "allow-https-443-oneapi" "443" "tcp" "Allow HTTPS traffic for One-API" ""
fi

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}防火墙配置完成！${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "已配置的规则："
echo "  - allow-http-80-oneapi: 允许 HTTP (80) 端口"
echo "  - allow-https-443-oneapi: 允许 HTTPS (443) 端口"
echo ""

# 显示当前防火墙规则
echo -e "${YELLOW}当前项目中的相关防火墙规则:${NC}"
gcloud compute firewall-rules list --filter="name~oneapi OR name~http OR name~https" --format="table(name,allowed[].map().firewall_rule().list():label=ALLOW,direction,sourceRanges.list():label=SRC_RANGES,targetTags.list():label=TARGET_TAGS)" --project="$PROJECT_ID"

echo ""
echo -e "${GREEN}配置完成！现在可以通过以下方式访问:${NC}"
echo "  - HTTP:  http://oneapi.gitagent.io"
echo "  - HTTPS: https://oneapi.gitagent.io (配置 SSL 证书后)"
echo ""
