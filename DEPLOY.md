# One-API 域名配置部署指南

## 配置信息
- **域名**: oneapi.gitagent.io
- **服务器 IP**: 104.197.139.51
- **服务地址**: http://104.197.139.51:3000/
- **配置文件**: nginx-oneapi.gitagent.io.conf

## 部署步骤

### 方法一：使用部署脚本（推荐）

```bash
cd /Users/a1/work/one-api
./deploy-nginx.sh
```

脚本会提示输入服务器用户名，然后自动完成部署。

### 方法二：手动部署

#### 1. 复制配置文件到服务器

```bash
# 替换 USERNAME 为你的服务器用户名（如 root, ubuntu 等）
scp nginx-oneapi.gitagent.io.conf USERNAME@104.197.139.51:/tmp/
```

#### 2. SSH 连接到服务器

```bash
ssh USERNAME@104.197.139.51
```

#### 3. 在服务器上执行以下命令

```bash
# 检查 nginx 是否安装
nginx -v

# 如果未安装，先安装 nginx
# Ubuntu/Debian:
sudo apt update && sudo apt install -y nginx

# CentOS/RHEL:
# sudo yum install -y nginx

# 移动配置文件到 nginx 配置目录
sudo mv /tmp/nginx-oneapi.gitagent.io.conf /etc/nginx/sites-available/

# 创建符号链接（如果使用 sites-available/sites-enabled 结构）
sudo ln -sf /etc/nginx/sites-available/nginx-oneapi.gitagent.io.conf /etc/nginx/sites-enabled/

# 或者如果使用 conf.d 结构，直接复制：
# sudo cp /tmp/nginx-oneapi.gitagent.io.conf /etc/nginx/conf.d/

# 测试 nginx 配置
sudo nginx -t

# 如果测试通过，重新加载 nginx
sudo systemctl reload nginx
# 或者
sudo service nginx reload
```

#### 4. 验证配置

```bash
# 检查 nginx 状态
sudo systemctl status nginx

# 检查配置是否生效
curl -I http://localhost
```

### 3. 配置 DNS 记录

在域名管理面板（gitagent.io 的 DNS 管理）添加 A 记录：

- **主机记录**: `oneapi`
- **记录类型**: `A`
- **记录值**: `104.197.139.51`
- **TTL**: 600（或默认值）

### 4. 配置 HTTPS（可选但推荐）

```bash
# SSH 连接到服务器后执行

# 安装 certbot（如果未安装）
sudo snap install --classic certbot
sudo ln -s /snap/bin/certbot /usr/bin/certbot

# 或者使用 apt 安装（Ubuntu/Debian）
sudo apt update
sudo apt install -y certbot python3-certbot-nginx

# 生成证书并自动配置 nginx
sudo certbot --nginx -d oneapi.gitagent.io

# 按照提示输入邮箱等信息
# certbot 会自动修改 nginx 配置文件并启用 HTTPS
```

### 5. 验证部署

1. **检查 DNS 解析**（可能需要几分钟到几小时）:
   ```bash
   nslookup oneapi.gitagent.io
   # 或
   dig oneapi.gitagent.io
   ```

2. **测试 HTTP 访问**:
   ```bash
   curl http://oneapi.gitagent.io
   ```

3. **在浏览器中访问**:
   - HTTP: http://oneapi.gitagent.io
   - HTTPS: https://oneapi.gitagent.io（配置证书后）

## 故障排查

### 如果无法访问：

1. **检查 nginx 是否运行**:
   ```bash
   sudo systemctl status nginx
   ```

2. **检查 nginx 配置**:
   ```bash
   sudo nginx -t
   ```

3. **检查防火墙**:
   ```bash
   # 确保 80 和 443 端口开放
   sudo ufw status
   # 或
   sudo iptables -L
   ```

4. **检查 one-api 服务**:
   ```bash
   # 确保 one-api 在 3000 端口运行
   curl http://localhost:3000
   ```

5. **查看 nginx 日志**:
   ```bash
   sudo tail -f /var/log/nginx/oneapi.gitagent.io.error.log
   sudo tail -f /var/log/nginx/oneapi.gitagent.io.access.log
   ```

## 注意事项

1. 确保服务器上的 one-api 服务正在运行在 3000 端口
2. 确保防火墙允许 80 和 443 端口
3. DNS 解析可能需要一些时间才能生效
4. 如果 one-api 和 nginx 在同一台服务器，proxy_pass 使用 `localhost:3000`
5. 如果 one-api 在其他服务器，需要修改 proxy_pass 为对应的 IP 和端口
