#!/bin/bash

# 服务器操作系统防火墙配置脚本（使用 iptables）
# 在服务器 104.197.139.51 上执行
# 用于允许 HTTP (80) 和 HTTPS (443) 端口

set -e

echo "=========================================="
echo "服务器防火墙配置 (iptables)"
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

echo "检查当前 iptables 规则..."
$SUDO iptables -L INPUT -n --line-numbers | head -10

echo ""
echo "配置防火墙规则..."

# 检查并添加 HTTP (80) 端口规则
echo "允许 HTTP (80) 端口..."
if $SUDO iptables -C INPUT -p tcp --dport 80 -j ACCEPT 2>/dev/null; then
    echo "  ✓ HTTP (80) 端口规则已存在"
else
    $SUDO iptables -A INPUT -p tcp --dport 80 -j ACCEPT -m comment --comment "Allow HTTP for One-API"
    echo "  ✓ HTTP (80) 端口规则已添加"
fi

# 检查并添加 HTTPS (443) 端口规则
echo "允许 HTTPS (443) 端口..."
if $SUDO iptables -C INPUT -p tcp --dport 443 -j ACCEPT 2>/dev/null; then
    echo "  ✓ HTTPS (443) 端口规则已存在"
else
    $SUDO iptables -A INPUT -p tcp --dport 443 -j ACCEPT -m comment --comment "Allow HTTPS for One-API"
    echo "  ✓ HTTPS (443) 端口规则已添加"
fi

echo ""
echo "=========================================="
echo "配置完成！"
echo "=========================================="
echo ""
echo "当前 INPUT 链规则："
$SUDO iptables -L INPUT -n --line-numbers | grep -E "(80|443|Chain|target)"

echo ""
echo "⚠ 注意: iptables 规则在重启后会丢失"
echo "如果需要持久化，请执行："
echo ""
echo "  # Ubuntu/Debian:"
echo "  sudo apt-get install -y iptables-persistent"
echo "  sudo netfilter-persistent save"
echo ""
echo "  # CentOS/RHEL:"
echo "  sudo service iptables save"
echo ""
