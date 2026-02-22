#!/bin/bash

# One-API Nginx 配置部署脚本
# 服务器: 104.197.139.51
# 域名: oneapi.gitagent.io

SERVER_IP="104.197.139.51"
CONFIG_FILE="nginx-oneapi.gitagent.io.conf"
NGINX_CONF_DIR="/etc/nginx/sites-available"
NGINX_ENABLED_DIR="/etc/nginx/sites-enabled"

echo "开始部署 Nginx 配置到服务器 $SERVER_IP..."

# 检查配置文件是否存在
if [ ! -f "$CONFIG_FILE" ]; then
    echo "错误: 配置文件 $CONFIG_FILE 不存在"
    exit 1
fi

# 提示输入服务器用户名
read -p "请输入服务器用户名 (默认: root): " SERVER_USER
SERVER_USER=${SERVER_USER:-root}

echo "使用用户名: $SERVER_USER"

# 复制配置文件到服务器
echo "正在复制配置文件到服务器..."
scp "$CONFIG_FILE" "$SERVER_USER@$SERVER_IP:/tmp/"

if [ $? -ne 0 ]; then
    echo "错误: 无法复制文件到服务器，请检查 SSH 连接"
    exit 1
fi

echo "配置文件已复制到服务器"

# 在服务器上执行配置命令
echo "正在配置 Nginx..."
ssh "$SERVER_USER@$SERVER_IP" << 'ENDSSH'
    # 检查 nginx 是否安装
    if ! command -v nginx &> /dev/null; then
        echo "错误: Nginx 未安装，请先安装 Nginx"
        exit 1
    fi

    # 移动配置文件到 nginx 配置目录
    sudo mv /tmp/nginx-oneapi.gitagent.io.conf /etc/nginx/sites-available/
    
    # 创建符号链接（如果使用 sites-available/sites-enabled 结构）
    if [ -d /etc/nginx/sites-enabled ]; then
        sudo ln -sf /etc/nginx/sites-available/nginx-oneapi.gitagent.io.conf /etc/nginx/sites-enabled/
    fi
    
    # 或者直接复制到 conf.d（如果使用 conf.d 结构）
    # sudo cp /etc/nginx/sites-available/nginx-oneapi.gitagent.io.conf /etc/nginx/conf.d/
    
    # 测试 nginx 配置
    echo "测试 Nginx 配置..."
    sudo nginx -t
    
    if [ $? -eq 0 ]; then
        echo "Nginx 配置测试通过"
        # 重新加载 nginx
        echo "重新加载 Nginx..."
        sudo systemctl reload nginx || sudo service nginx reload
        echo "Nginx 已重新加载"
        echo "配置部署完成！"
    else
        echo "错误: Nginx 配置测试失败，请检查配置文件"
        exit 1
    fi
ENDSSH

if [ $? -eq 0 ]; then
    echo ""
    echo "=========================================="
    echo "部署成功！"
    echo "=========================================="
    echo "域名: oneapi.gitagent.io"
    echo "服务器: $SERVER_IP"
    echo ""
    echo "下一步："
    echo "1. 确保 DNS 记录已配置：oneapi.gitagent.io -> $SERVER_IP"
    echo "2. 测试访问: http://oneapi.gitagent.io"
    echo "3. 配置 HTTPS (可选): sudo certbot --nginx -d oneapi.gitagent.io"
    echo "=========================================="
else
    echo "部署过程中出现错误"
    exit 1
fi
