# Gemini 图片生成模型修复说明

## 问题描述

通过 One API 网关的 `/v1/chat/completions` 接口调用 Gemini 图片生成模型时，无法返回图片内容：

- `gemini-2.0-pro-exp-02-05` 返回 404（模型已被 Google 下线）
- `gemini-3-flash-preview` 仅返回文本，无图片（该模型本身不支持图片生成）
- `gemini-3-pro-image-preview` 仅返回文本，图片数据被丢弃

## 根因分析

### 1. 请求缺少 `responseModalities` 参数

Gemini 图片生成模型要求在请求的 `generationConfig` 中设置：
```json
{
  "generationConfig": {
    "responseModalities": ["TEXT", "IMAGE"]
  }
}
```
One API 的 Gemini 适配器没有为图片生成模型设置此参数，导致 Gemini API 默认只返回文本。

### 2. 响应转换忽略 InlineData

Gemini API 返回图片时，图片以 `inlineData`（base64）形式嵌入在 response parts 中：
```json
{
  "candidates": [{
    "content": {
      "parts": [
        {"text": "描述文字"},
        {"inlineData": {"mimeType": "image/jpeg", "data": "/9j/4AAQ..."}}
      ]
    }
  }]
}
```

但 `responseGeminiChat2OpenAI()` 函数在转换响应时只提取了 `part.Text`，`InlineData` 被完全忽略。

### 3. 流式响应同样丢弃图片

`streamResponseGeminiChat2OpenAI()` 和 `GetResponseText()` 也只处理文本部分。

## 修复内容

### 文件修改清单

| 文件 | 修改内容 |
|------|---------|
| `relay/adaptor/gemini/model.go` | `ChatGenerationConfig` 添加 `ResponseModalities []string` 字段 |
| `relay/adaptor/gemini/constants.go` | 添加图片生成模型列表 `ModelsWithImageGeneration` 和检测函数 |
| `relay/adaptor/gemini/main.go` | 3 处修改：请求设置 responseModalities、非流式/流式响应处理 InlineData |
| `relay/billing/ratio/model.go` | 添加 gemini-3-pro-preview、gemini-3-pro-image-preview、gemini-3-flash-preview 的定价 |

### 核心修改详解

#### 1. 请求自动设置 responseModalities（main.go ConvertRequest）

对识别为图片生成的模型，自动在 `generationConfig` 中添加 `responseModalities: ["TEXT", "IMAGE"]`：

```go
if IsModelSupportImageGeneration(textRequest.Model) {
    geminiRequest.GenerationConfig.ResponseModalities = []string{"TEXT", "IMAGE"}
}
```

#### 2. 响应转换支持多模态内容（main.go responseGeminiChat2OpenAI）

检测响应 parts 中是否包含 `InlineData`，如果有则转为 OpenAI 多模态格式：

```json
{
  "content": [
    {"type": "text", "text": "描述文字"},
    {"type": "image_url", "image_url": {"url": "data:image/jpeg;base64,/9j/4AAQ..."}}
  ]
}
```

如果响应中没有图片，则保持原有的纯文本字符串格式，不影响非图片模型。

#### 3. 流式响应同样处理（main.go streamResponseGeminiChat2OpenAI）

流式响应的 delta 内容也支持多模态格式，与非流式处理逻辑一致。

### 支持图片生成的模型

当前在 `ModelsWithImageGeneration` 列表中的模型：

| 模型 | 说明 |
|------|------|
| `gemini-2.0-flash-exp` | Gemini 2.0 实验版，支持图片生成 |
| `gemini-3-pro-image-preview` | Gemini 3.0 专用图片生成模型 |

如需添加新的图片生成模型，编辑 `relay/adaptor/gemini/constants.go` 中的 `ModelsWithImageGeneration` 列表。

### 不支持图片生成的模型（请勿用于图片生成）

| 模型 | 原因 |
|------|------|
| `gemini-2.0-pro-exp-02-05` | 已被 Google 下线，返回 404 |
| `gemini-3-flash-preview` | 仅支持文本生成 |
| `gemini-3-pro-preview` | 仅支持文本生成 |
| `gemini-3.1-pro-preview` | 仅支持文本生成 |

## 部署说明

### 编译

```bash
# 本机需要 CGO（SQLite 依赖）
CGO_ENABLED=1 go build -ldflags "-s -w" -o one-api .

# 如果用 Linux 交叉编译（需要 musl 或在目标服务器上编译）
# 不能使用 CGO_ENABLED=0，否则 SQLite 无法工作
```

### 服务器部署（104.197.139.51）

```bash
# SSH 连接
ssh -i ~/.ssh/google_compute_engine a1@104.197.139.51

# 二进制路径
/mnt/disk-119/one-api/one-api

# 备份 + 替换 + 重启
cd /mnt/disk-119/one-api
sudo -u support cp one-api one-api.bak-$(date +%Y%m%d-%H%M%S)
sudo cp /path/to/new-binary one-api
sudo chmod +x one-api
sudo chown support:support one-api
sudo pkill -f './one-api --port 3000'
sleep 2
sudo -u support bash -c 'cd /mnt/disk-119/one-api && nohup ./one-api --port 3000 --log-dir ./logs > logs/one-api.log 2>&1 &'
```

**注意**：服务器根盘空间很小（10G，经常满），SQLite 迁移时可能需要设置 `TMPDIR` 到数据盘：
```bash
export TMPDIR=/mnt/disk-119/one-api/logs
```

## 测试验证

### 测试命令

```bash
curl http://104.197.139.51:3000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "model": "gemini-3-pro-image-preview",
    "messages": [{"role": "user", "content": "Generate a simple image of a red circle"}],
    "max_tokens": 4096
  }'
```

### 预期响应格式

```json
{
  "id": "chatcmpl-xxx",
  "object": "chat.completion",
  "choices": [{
    "index": 0,
    "message": {
      "role": "assistant",
      "content": [
        {"type": "image_url", "image_url": {"url": "data:image/jpeg;base64,/9j/4AAQ..."}}
      ]
    },
    "finish_reason": "stop"
  }]
}
```

### 测试结果（2026-03-23）

- `gemini-3-pro-image-preview` 图片生成：**通过** - 返回 JPEG 图片 (base64, ~539KB)
- 多模态响应（文字+图片）：**通过** - 正确返回 content 数组

## 修复日期

2026-03-23
