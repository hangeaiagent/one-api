#!/bin/bash

# 服务器操作系统防火墙配置脚本（使用 ufw）
# 在服务器 104.197.139.51 上执行
# 用于允许 HTTP (80) 和 HTTPS (443) 端口

set -e

echo "=========================================="
echo "服务器防火墙配置 (UFW)"
echo "=========================================="
echo ""

# 检查是否为 root 或有 sudo 权限
if [ "$EUID" -ne 0 ] && ! sudo -n true 2>/dev/null; then
    echo "此脚本需要 root 权限或 sudo 权限"
    echo "请使用: sudo $0"
    exit 1
fi

# 使用 sudo 执行命令的函数
SUDO=""
if [ "$EUID" -ne 0 ]; then
    SUDO="sudo"
fi

echo "检查 UFW 状态..."
$SUDO ufw status | head -5

echo ""
echo "配置防火墙规则..."

# 允许 HTTP (80) 端口
echo "允许 HTTP (80) 端口..."
if $SUDO ufw status | grep -q "80/tcp.*ALLOW"; then
    echo "  ✓ HTTP (80) 端口规则已存在"
else
    $SUDO ufw allow 80/tcp comment 'Allow HTTP for One-API'
    echo "  ✓ HTTP (80) 端口规则已添加"
fi

# 允许 HTTPS (443) 端口
echo "允许 HTTPS (443) 端口..."
if $SUDO ufw status | grep -q "443/tcp.*ALLOW"; then
    echo "  ✓ HTTPS (443) 端口规则已存在"
else
    $SUDO ufw allow 443/tcp comment 'Allow HTTPS for One-API'
    echo "  ✓ HTTPS (443) 端口规则已添加"
fi

# 如果 UFW 未启用，提示启用
if ! $SUDO ufw status | grep -q "Status: active"; then
    echo ""
    echo "UFW 防火墙未启用"
    read -p "是否现在启用 UFW 防火墙? (y/n, 默认 y): " ENABLE_UFW
    ENABLE_UFW=${ENABLE_UFW:-y}
    
    if [ "$ENABLE_UFW" = "y" ]; then
        echo "启用 UFW 防火墙..."
        $SUDO ufw --force enable
        echo "  ✓ UFW 已启用"
    else
        echo "  ⚠ UFW 未启用，规则将在启用后生效"
    fi
fi

echo ""
echo "=========================================="
echo "配置完成！"
echo "=========================================="
echo ""
echo "当前防火墙规则："
$SUDO ufw status numbered | grep -E "(80|443|Status)"

echo ""
echo "如果需要，还可以允许 SSH 端口（如果还没有）："
echo "  sudo ufw allow 22/tcp comment 'Allow SSH'"
echo ""
