# Google Cloud Shell 防火墙配置指南

## 快速开始

### 方法一：复制粘贴脚本内容（推荐）

1. 打开 [Google Cloud Shell](https://shell.cloud.google.com/)
2. 确保已选择正确的项目（或运行 `gcloud config set project YOUR_PROJECT_ID`）
3. 复制以下脚本内容并粘贴到 Cloud Shell 中执行：

```bash
PROJECT_ID=$(gcloud config get-value project) && \
echo "项目: $PROJECT_ID" && \
(gcloud compute firewall-rules describe allow-http-80-oneapi --project="$PROJECT_ID" &>/dev/null && \
 gcloud compute firewall-rules update allow-http-80-oneapi --allow tcp:80 --source-ranges 0.0.0.0/0 --description "Allow HTTP traffic for One-API" --project="$PROJECT_ID" --quiet && \
 echo "✓ HTTP 规则已更新" || \
 gcloud compute firewall-rules create allow-http-80-oneapi --allow tcp:80 --source-ranges 0.0.0.0/0 --description "Allow HTTP traffic for One-API" --project="$PROJECT_ID" --quiet && \
 echo "✓ HTTP 规则已创建") && \
(gcloud compute firewall-rules describe allow-https-443-oneapi --project="$PROJECT_ID" &>/dev/null && \
 gcloud compute firewall-rules update allow-https-443-oneapi --allow tcp:443 --source-ranges 0.0.0.0/0 --description "Allow HTTPS traffic for One-API" --project="$PROJECT_ID" --quiet && \
 echo "✓ HTTPS 规则已更新" || \
 gcloud compute firewall-rules create allow-https-443-oneapi --allow tcp:443 --source-ranges 0.0.0.0/0 --description "Allow HTTPS traffic for One-API" --project="$PROJECT_ID" --quiet && \
 echo "✓ HTTPS 规则已创建") && \
echo "" && \
echo "配置完成！查看规则：" && \
gcloud compute firewall-rules list --filter="name~oneapi" --format="table(name,allowed,direction,sourceRanges)" --project="$PROJECT_ID"
```

### 方法二：使用完整脚本

在 Google Cloud Shell 中执行：

```bash
# 创建脚本文件
cat > /tmp/setup-firewall.sh << 'EOF'
#!/bin/bash
set -e

PROJECT_ID=$(gcloud config get-value project 2>/dev/null)

if [ -z "$PROJECT_ID" ]; then
    echo "错误: 未设置项目，请先设置项目："
    echo "  gcloud config set project YOUR_PROJECT_ID"
    exit 1
fi

echo "当前项目: $PROJECT_ID"
echo "配置端口: 80 (HTTP), 443 (HTTPS)"
echo ""

RULE_HTTP="allow-http-80-oneapi"
RULE_HTTPS="allow-https-443-oneapi"

echo "配置 HTTP (80) 端口..."
if gcloud compute firewall-rules describe "$RULE_HTTP" --project="$PROJECT_ID" &>/dev/null 2>&1; then
    echo "  ✓ 规则已存在，正在更新..."
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
    echo "  ✓ 规则创建成功"
fi

echo ""
echo "配置 HTTPS (443) 端口..."
if gcloud compute firewall-rules describe "$RULE_HTTPS" --project="$PROJECT_ID" &>/dev/null 2>&1; then
    echo "  ✓ 规则已存在，正在更新..."
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
    echo "  ✓ 规则创建成功"
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
EOF

# 执行脚本
chmod +x /tmp/setup-firewall.sh
/tmp/setup-firewall.sh
```

### 方法三：分步执行（手动命令）

如果你更喜欢手动执行每个命令：

```bash
# 1. 确认当前项目
gcloud config get-value project

# 2. 创建 HTTP (80) 规则
gcloud compute firewall-rules create allow-http-80-oneapi \
    --allow tcp:80 \
    --source-ranges 0.0.0.0/0 \
    --description "Allow HTTP traffic for One-API"

# 3. 创建 HTTPS (443) 规则
gcloud compute firewall-rules create allow-https-443-oneapi \
    --allow tcp:443 \
    --source-ranges 0.0.0.0/0 \
    --description "Allow HTTPS traffic for One-API"

# 4. 查看创建的规则
gcloud compute firewall-rules list --filter="name~oneapi"
```

## 验证配置

配置完成后，可以使用以下命令验证：

```bash
# 查看所有相关规则
gcloud compute firewall-rules list \
    --filter="name~oneapi" \
    --format="table(name,allowed,direction,sourceRanges,targetTags)"

# 查看规则详情
gcloud compute firewall-rules describe allow-http-80-oneapi
gcloud compute firewall-rules describe allow-https-443-oneapi
```

## 如果规则已存在

如果规则已存在，脚本会自动更新。你也可以手动更新：

```bash
# 更新 HTTP 规则
gcloud compute firewall-rules update allow-http-80-oneapi \
    --allow tcp:80 \
    --source-ranges 0.0.0.0/0

# 更新 HTTPS 规则
gcloud compute firewall-rules update allow-https-443-oneapi \
    --allow tcp:443 \
    --source-ranges 0.0.0.0/0
```

## 删除规则（如果需要）

```bash
# 删除 HTTP 规则
gcloud compute firewall-rules delete allow-http-80-oneapi

# 删除 HTTPS 规则
gcloud compute firewall-rules delete allow-https-443-oneapi
```

## 注意事项

1. **项目权限**：确保你的账户有 `compute.firewalls.create` 和 `compute.firewalls.update` 权限
2. **生效时间**：防火墙规则通常在几秒内生效
3. **实例防火墙**：Google Cloud 防火墙规则只控制网络层面的访问，还需要确保实例操作系统层面的防火墙（如 ufw/iptables）也允许这些端口

## 故障排查

### 问题：权限不足

```bash
# 检查当前账户权限
gcloud projects get-iam-policy $(gcloud config get-value project) \
    --flatten="bindings[].members" \
    --filter="bindings.members:$(gcloud config get-value account)"
```

### 问题：规则创建失败

检查是否有冲突的规则：
```bash
gcloud compute firewall-rules list --filter="allowed.ports:80 OR allowed.ports:443"
```

### 问题：端口仍然无法访问

1. 检查实例是否运行
2. 检查实例上的服务是否监听正确端口
3. 检查实例操作系统防火墙

```bash
# SSH 到实例检查
ssh support@104.197.139.51

# 检查服务
sudo systemctl status nginx
sudo netstat -tlnp | grep -E ':(80|443)'

# 检查防火墙
sudo ufw status
```
