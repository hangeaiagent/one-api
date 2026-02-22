# Google Cloud 防火墙配置指南

## 概述

本目录包含用于配置 Google Cloud 防火墙规则的脚本，用于允许 HTTP (80) 和 HTTPS (443) 端口访问 One-API 服务。

## 脚本说明

### 1. `gcloud-firewall-setup.sh` - 完整配置脚本

**功能**：
- 交互式配置，自动检测项目信息
- 支持查找实例并配置特定标签
- 详细的输出和错误处理
- 自动检测和更新已存在的规则

**使用方法**：
```bash
./gcloud-firewall-setup.sh
```

**特点**：
- 自动检测当前 gcloud 项目
- 查找 IP 为 104.197.139.51 的实例
- 支持为特定实例标签创建规则
- 显示详细的配置信息

### 2. `gcloud-firewall-quick.sh` - 快速配置脚本

**功能**：
- 快速创建防火墙规则
- 支持环境变量配置
- 简洁的输出

**使用方法**：

**方法一：使用环境变量**
```bash
export GCP_PROJECT_ID="your-project-id"
./gcloud-firewall-quick.sh
```

**方法二：直接运行（会提示输入项目 ID）**
```bash
./gcloud-firewall-quick.sh
```

**方法三：一行命令（指定项目 ID）**
```bash
GCP_PROJECT_ID="your-project-id" ./gcloud-firewall-quick.sh
```

## 前置要求

### 1. 安装 Google Cloud SDK

**macOS**:
```bash
# 使用 Homebrew
brew install --cask google-cloud-sdk

# 或下载安装包
# https://cloud.google.com/sdk/docs/install
```

**Linux**:
```bash
# 添加 Google Cloud 官方源
echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] https://packages.cloud.google.com/apt cloud-sdk main" | sudo tee -a /etc/apt/sources.list.d/google-cloud-sdk.list

# 安装
curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key --keyring /usr/share/keyrings/cloud.google.gpg add -
sudo apt-get update && sudo apt-get install google-cloud-sdk
```

**Windows**:
下载并运行安装程序：https://cloud.google.com/sdk/docs/install

### 2. 认证和初始化

```bash
# 登录 Google Cloud
gcloud auth login

# 设置默认项目（可选）
gcloud config set project YOUR_PROJECT_ID

# 验证配置
gcloud config list
```

## 使用步骤

### 步骤 1: 获取项目 ID

如果你不知道项目 ID，可以通过以下方式获取：

```bash
# 列出所有项目
gcloud projects list

# 或查看当前项目
gcloud config get-value project
```

### 步骤 2: 运行配置脚本

**推荐使用快速脚本**：
```bash
cd /Users/a1/work/one-api
GCP_PROJECT_ID="your-project-id" ./gcloud-firewall-quick.sh
```

**或使用完整脚本**：
```bash
cd /Users/a1/work/one-api
./gcloud-firewall-setup.sh
```

### 步骤 3: 验证配置

```bash
# 查看创建的防火墙规则
gcloud compute firewall-rules list \
    --filter="name~oneapi" \
    --format="table(name,allowed[].map().firewall_rule().list():label=ALLOW,direction,sourceRanges.list():label=SRC_RANGES)"

# 测试端口是否开放
curl -I http://104.197.139.51
```

## 手动配置（不使用脚本）

如果你更喜欢手动配置，可以使用以下命令：

### 创建 HTTP 规则
```bash
gcloud compute firewall-rules create allow-http-80-oneapi \
    --allow tcp:80 \
    --source-ranges 0.0.0.0/0 \
    --description "Allow HTTP traffic for One-API" \
    --project YOUR_PROJECT_ID
```

### 创建 HTTPS 规则
```bash
gcloud compute firewall-rules create allow-https-443-oneapi \
    --allow tcp:443 \
    --source-ranges 0.0.0.0/0 \
    --description "Allow HTTPS traffic for One-API" \
    --project YOUR_PROJECT_ID
```

### 针对特定实例标签（可选）

如果你只想为特定标签的实例开放端口：

```bash
# 首先查看实例的标签
gcloud compute instances describe INSTANCE_NAME \
    --zone=ZONE \
    --format="get(tags.items)"

# 创建针对标签的规则
gcloud compute firewall-rules create allow-http-80-oneapi \
    --allow tcp:80 \
    --source-ranges 0.0.0.0/0 \
    --target-tags http-server \
    --description "Allow HTTP traffic for One-API" \
    --project YOUR_PROJECT_ID
```

## 故障排查

### 问题 1: 权限不足

**错误信息**：
```
ERROR: (gcloud.compute.firewall-rules.create) User [xxx] does not have permission
```

**解决方法**：
确保你的账户具有以下权限：
- `compute.firewalls.create`
- `compute.firewalls.update`
- `compute.firewalls.get`

或者使用具有 `Compute Admin` 角色的账户。

### 问题 2: 规则已存在

**错误信息**：
```
ERROR: (gcloud.compute.firewall-rules.create) Resource in use
```

**解决方法**：
脚本会自动检测并更新已存在的规则。如果手动创建，可以先删除旧规则：
```bash
gcloud compute firewall-rules delete RULE_NAME --project YOUR_PROJECT_ID
```

### 问题 3: 端口仍然无法访问

**检查清单**：
1. ✅ 防火墙规则已创建
2. ✅ 规则允许的端口正确（80/443）
3. ✅ 源 IP 范围包含你的 IP（0.0.0.0/0 表示所有 IP）
4. ✅ 实例上的服务正在运行
5. ✅ 实例的操作系统防火墙（如 ufw/iptables）已配置

**检查实例防火墙**：
```bash
# SSH 到实例
ssh -i ~/.ssh/id_rsa_google_longterm support@104.197.139.51

# 检查 ufw（如果使用）
sudo ufw status
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# 检查 iptables
sudo iptables -L -n
```

## 安全建议

1. **限制源 IP 范围**（可选）：
   如果只需要从特定 IP 访问，可以修改 `source-ranges`：
   ```bash
   --source-ranges 1.2.3.4/32,5.6.7.8/32
   ```

2. **使用实例标签**：
   为需要开放端口的实例添加标签，然后只针对这些标签创建规则。

3. **定期审查规则**：
   ```bash
   gcloud compute firewall-rules list --format="table(name,allowed,direction,sourceRanges)"
   ```

## 相关资源

- [Google Cloud 防火墙文档](https://cloud.google.com/vpc/docs/firewalls)
- [gcloud 命令参考](https://cloud.google.com/sdk/gcloud/reference/compute/firewall-rules)
- [One-API 部署文档](./DEPLOY.md)

## 脚本创建的规则

脚本会创建以下防火墙规则：

| 规则名称 | 端口 | 协议 | 描述 |
|---------|------|------|------|
| `allow-http-80-oneapi` | 80 | TCP | 允许 HTTP 流量 |
| `allow-https-443-oneapi` | 443 | TCP | 允许 HTTPS 流量 |

这些规则允许来自任何 IP 地址（0.0.0.0/0）的流量访问指定的端口。
