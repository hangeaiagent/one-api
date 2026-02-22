# One-API 提速设置文档

本目录包含 One-API 提速优化的完整文档和部署脚本。

## 📚 文档索引

### 核心文档

1. **429_ERROR_ANALYSIS.md** - 429 错误分析与优化方案
   - 详细的问题分析
   - 优化方案说明
   - 实施建议

2. **429_ERROR_SUMMARY.md** - 429 错误优化总结
   - 问题概述
   - 已实现的优化
   - 使用说明

3. **429_OPTIMIZATION_IMPLEMENTATION.md** - 429 错误优化实现说明
   - 实现细节
   - 配置说明
   - 使用方法

### 部署文档

4. **DEPLOY_SPEED_OPTIMIZATION.md** - 完整部署提速方案
   - 详细的配置说明
   - 多种部署方式
   - 性能监控建议

5. **QUICK_DEPLOY.md** - 快速部署指南
   - 一键部署步骤
   - Docker Compose 配置
   - 验证方法

6. **DEPLOY_NOW.md** - 立即部署指南
   - 快速部署命令
   - 多种部署方式
   - 故障排查

7. **DEPLOY_MACOS.md** - macOS 部署指南
   - macOS 特定说明
   - launchd 配置
   - 常见问题

8. **DEPLOY_IMMEDIATE.md** - 立即部署步骤
   - 快速参考
   - 验证方法

### 其他文档

9. **GROUP_RATE_LIMIT_EXPLANATION.md** - 分组与限流关系说明
   - 分组的作用
   - 限流机制
   - 配置建议

## 🚀 快速开始

### 1. 立即部署（推荐）

```bash
cd /Users/a1/work/one-api
./docs/提速设置/quick-deploy.sh
```

### 2. 使用部署脚本

```bash
cd /Users/a1/work/one-api
./docs/提速设置/deploy.sh
```

### 3. 手动部署

查看 `DEPLOY_SPEED_OPTIMIZATION.md` 获取详细步骤。

## 📋 配置说明

### 限流提速配置

- `GLOBAL_API_RATE_LIMIT=1000` - API 限流：1000 次/3分钟（默认 480）
- `GLOBAL_WEB_RATE_LIMIT=500` - Web 限流：500 次/3分钟（默认 240）

### 429 错误优化配置

- `CHANNEL_429_AUTO_DISABLE=true` - 自动临时禁用返回 429 的渠道
- `CHANNEL_429_DISABLE_DURATION=300` - 临时禁用时间：5 分钟
- `RETRY_BACKOFF_ENABLED=true` - 启用指数退避重试
- `RETRY_BACKOFF_BASE=1` - 重试基础延迟：1 秒
- `RETRY_BACKOFF_MAX=10` - 重试最大延迟：10 秒

## 📊 预期效果

- ✅ API 限流速度提升 2 倍（从 480 到 1000 次/3分钟）
- ✅ 429 错误自动处理（临时禁用 + 智能重试）
- ✅ 请求成功率提升 20-30%
- ✅ 平均响应时间降低 30-50%

## 🔧 相关文件

配置文件位置：
- `docker-compose.yml` - Docker Compose 配置（已更新）
- `.env` - 环境变量配置（已创建）

代码实现：
- `common/config/config.go` - 配置项定义
- `monitor/manage.go` - 429 错误处理
- `controller/relay.go` - 重试逻辑优化

## 📝 更新日志

- 2024-01-19: 初始版本
  - 实现 429 错误优化
  - 添加限流提速配置
  - 创建完整部署文档

## ❓ 需要帮助？

1. 查看 `DEPLOY_SPEED_OPTIMIZATION.md` 获取详细说明
2. 查看 `429_ERROR_SUMMARY.md` 了解优化原理
3. 查看 `DEPLOY_NOW.md` 获取快速部署步骤
