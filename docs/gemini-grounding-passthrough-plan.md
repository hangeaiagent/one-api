# Gemini Grounding Metadata 透传 —— OneAPI 开发方案

> 编制日期：2026-07-31
> 目标：在 One API 的 `chat/completions` 响应中，将 Gemini `google_search` 工具返回的 `grounding_metadata` 原样透传给上层业务（TrueSource），使其能提取引用来源（citations）。
> 关联需求：见 `.specstory/history/2026-07-31_08-35-32Z-2-1-a-oneapi.md` 方案 A。

---

## 一、背景与问题

### 1.1 现状
- One API 的 Gemini 适配层（`relay/adaptor/gemini/`）**已支持** `google_search` 工具：
  - 请求端：`main.go:81-98` 会把 OpenAI 请求里 `tools[].type == "google_search"` 转换为 Gemini 的 `tools[].google_search: {}`。
  - 数据结构：`model.go:73-78` 定义了 `ChatTools.GoogleSearch`。
- 但 **响应端完全丢弃** Google 返回的 `candidates[0].groundingMetadata`：
  - `main.go:280-285` 的 `ChatCandidate` 结构体只保留 `content / finishReason / index / safetyRatings`。
  - `main.go:320-384` 的 `responseGeminiChat2OpenAI`、`main.go:386-462` 的 `streamResponseGeminiChat2OpenAI` 只搬运文本 / functionCall / inlineData，**从未读取 `groundingMetadata`**。

### 1.2 业务需求
TrueSource 需要 Gemini 联网搜索后的引用信息（URI、title、片段索引），当前只能拿到最终文本，没有出处，无法在 UI 上展示 citations。

### 1.3 Gemini 原生响应结构（v1beta，2026-07 官方文档）
```json
{
  "candidates": [{
    "content": { "parts": [{ "text": "..." }], "role": "model" },
    "finishReason": "STOP",
    "groundingMetadata": {
      "webSearchQueries": ["..."],
      "searchEntryPoint": { "renderedContent": "<div>...</div>" },
      "groundingChunks": [
        { "web": { "uri": "https://vertexaisearch.cloud.google.com/...", "title": "example.com" } }
      ],
      "groundingSupports": [
        {
          "segment": { "startIndex": 0, "endIndex": 42, "text": "被引用的原文片段" },
          "groundingChunkIndices": [0, 2],
          "confidenceScores": [0.98, 0.87]
        }
      ]
    }
  }]
}
```

---

## 二、设计方案

### 2.1 字段命名与承载位置

在 OneAPI 的 OpenAI 兼容响应中新增一个可选 **顶层字段**（不放在 `choices[]` 内，避免与 OpenAI 官方语义混淆）：

```json
{
  "id": "chatcmpl-xxx",
  "object": "chat.completion",
  "choices": [ ... ],
  "usage": { ... },
  "metadata": {
    "grounding": { /* 原样 Gemini groundingMetadata */ }
  }
}
```

**为什么用 `metadata.grounding` 而不是 `google_grounding`？**
- `metadata` 是一个 **provider-neutral 信封**，未来 Perplexity / Kimi / Doubao 等模型的联网元数据都能塞进同一层级（如 `metadata.perplexity_citations`），避免响应根节点污染。
- TrueSource 侧读取写法（与需求文档一致）：
  ```python
  grounding = resp.get("metadata", {}).get("grounding") or {}
  ```

> 也保留 `google_grounding` 作为 fallback 别名（同一份数据）以兼容需求文档 A 段落里给出的读取代码。上线后视调用方情况再决定是否删除别名。

### 2.2 数据结构原样透传（不做二次解析）

用 `map[string]any` 或 `json.RawMessage` 承载 —— 让 Google 后续新增字段（如 `citationMetadata` 的其它变体）自动透传，不需要 OneAPI 升级。

选择 **`json.RawMessage`**：
- 零解析开销：只做 byte 搬运。
- 顺序稳定：不会因 map 遍历乱序而改变 JSON 输出。
- 未知字段自然保留。

---

## 三、代码改动清单

改动集中在 **两个文件、约 60~80 行**，不涉及其它 provider。

### 3.1 `relay/adaptor/gemini/model.go`

给 `ChatCandidate` 加上 `GroundingMetadata` 字段（用 `json.RawMessage` 承载原始 JSON）：

```go
import "encoding/json"

type ChatCandidate struct {
    Content           ChatContent        `json:"content"`
    FinishReason      string             `json:"finishReason"`
    Index             int64              `json:"index"`
    SafetyRatings     []ChatSafetyRating `json:"safetyRatings"`
    GroundingMetadata json.RawMessage    `json:"groundingMetadata,omitempty"` // NEW
}
```

### 3.2 `relay/adaptor/openai/model.go`

给 `TextResponse` 和 `ChatCompletionsStreamResponse` 各加一个可选 `Metadata` 字段：

```go
type TextResponse struct {
    Id          string               `json:"id"`
    Model       string               `json:"model,omitempty"`
    Object      string               `json:"object"`
    Created     int64                `json:"created"`
    Choices     []TextResponseChoice `json:"choices"`
    model.Usage `json:"usage"`
    Metadata    map[string]json.RawMessage `json:"metadata,omitempty"` // NEW
}

type ChatCompletionsStreamResponse struct {
    Id       string                                `json:"id"`
    Object   string                                `json:"object"`
    Created  int64                                 `json:"created"`
    Model    string                                `json:"model"`
    Choices  []ChatCompletionsStreamResponseChoice `json:"choices"`
    Usage    *model.Usage                          `json:"usage,omitempty"`
    Metadata map[string]json.RawMessage            `json:"metadata,omitempty"` // NEW
}
```

> `omitempty` 是关键 —— 未启用 grounding 的普通请求响应体完全不变，对其它 provider / OpenAI SDK 完全无感。

### 3.3 `relay/adaptor/gemini/main.go`

新增一个提取函数，并在两条转换路径注入 metadata。

#### 3.3.1 提取函数（放在 `responseGeminiChat2OpenAI` 上方）

```go
// buildMetadata assembles the top-level metadata envelope carried on the
// OpenAI-compatible response. Currently only surfaces Gemini grounding
// data; extend here when other vendor-neutral fields are needed.
func buildMetadata(response *ChatResponse) map[string]json.RawMessage {
    if response == nil || len(response.Candidates) == 0 {
        return nil
    }
    raw := response.Candidates[0].GroundingMetadata
    if len(raw) == 0 || string(raw) == "null" {
        return nil
    }
    return map[string]json.RawMessage{
        "grounding":        raw,
        "google_grounding": raw, // temporary alias for backward compatibility
    }
}
```

#### 3.3.2 非流式路径

在 `responseGeminiChat2OpenAI`（`main.go:320`）返回前加一行：

```go
fullTextResponse.Metadata = buildMetadata(response)
return &fullTextResponse
```

#### 3.3.3 流式路径

Gemini SSE 只在 **最后一个 chunk**（`finishReason` 非空）才带 `groundingMetadata`。改动 `streamResponseGeminiChat2OpenAI`（`main.go:386`）尾部：

```go
response.Choices = []openai.ChatCompletionsStreamResponseChoice{choice}
if md := buildMetadata(geminiResponse); md != nil {
    response.Metadata = md
}
return &response
```

同时 `StreamHandler`（`main.go:481`）里的 `render.ObjectData(c, response)` 已经把整个 chunk 序列化到 SSE 事件，无需额外改动 —— metadata 会自然出现在包含 `finishReason` 的那一帧。

---

## 四、协议与兼容性

### 4.1 对下游（TrueSource 等业务方）
- **未使用 google_search 的请求**：响应完全不变（`omitempty` 生效）。
- **使用 google_search 的请求**：多出 `metadata.grounding`（及别名 `metadata.google_grounding`）。
- 别名策略：初期同时输出两个 key，一个月观察期后（约 2026-08 底）保留 `metadata.grounding`，删除 `google_grounding`。

### 4.2 对上游 OneAPI 主干
- 只增加可选字段，不修改任何现有字段。可作为独立 PR 回馈 upstream（本仓已多次向 upstream 提交 Gemini 相关增强，参考 `docs/gemini-latest-upgrade-plan.md`）。

### 4.3 对其它 provider adaptor
- 零影响。`Metadata` 只在 Gemini 主动写入时才有值。

---

## 五、测试计划

### 5.1 单元测试
新增 `relay/adaptor/gemini/main_test.go`（若已存在则追加）：

| 测试 | 期望 |
|---|---|
| `TestResponseGeminiChat2OpenAI_NoGrounding` | 输入不含 `groundingMetadata` 的响应 → 输出 `Metadata == nil` |
| `TestResponseGeminiChat2OpenAI_WithGrounding` | 输入含 `groundingMetadata` → 输出 `Metadata["grounding"]` 字节级等于原始 raw JSON |
| `TestStreamResponseGeminiChat2OpenAI_LastChunk` | 最后一个 chunk 携带 grounding → 序列化后包含 `"metadata":{...}` |
| `TestStreamResponseGeminiChat2OpenAI_MiddleChunk` | 中间 chunk 不含 grounding → 序列化后无 `metadata` 字段 |

### 5.2 端到端手测
1. 用 `curl` 打到本地 OneAPI：
   ```bash
   curl http://localhost:3000/v1/chat/completions \
     -H "Authorization: Bearer sk-xxx" \
     -H "Content-Type: application/json" \
     -d '{
       "model": "gemini-3.6-flash",
       "messages": [{"role":"user","content":"2026 年欧洲杯冠军是谁？请引用来源"}],
       "tools": [{"type":"google_search"}]
     }'
   ```
   预期响应体末尾出现 `"metadata": {"grounding": {...}, "google_grounding": {...}}`。

2. 流式版本加 `"stream": true`，观察最后一个 `data:` 帧包含 metadata。

3. 反向验证：把 `tools` 去掉，响应体无 `metadata` 字段。

### 5.3 回归
- 运行 `go test ./relay/adaptor/gemini/... -cover`。
- 对现有的非 google_search 请求做一次冒烟，确认响应字节数与改动前一致（可对比 md5）。

---

## 六、实施步骤

| # | 动作 | 预计时长 | 负责人 |
|---|---|---|---|
| 1 | 拉分支 `feat/gemini-grounding-passthrough` | 1 min | 后端 |
| 2 | 修改 `model.go` + `main.go` + `openai/model.go` | 30 min | 后端 |
| 3 | 补齐单测（5.1） | 40 min | 后端 |
| 4 | 本地端到端手测（5.2） | 20 min | 后端 |
| 5 | 提交 PR，附本方案文档链接 | 5 min | 后端 |
| 6 | 与 TrueSource 联调（读取 `metadata.grounding`） | 30 min | 后端 + 业务方 |
| 7 | 上生产、观察一周 | — | — |
| 8 | 一个月后（约 2026-08-31）移除 `google_grounding` 别名 | 5 min | 后端 |

**总人力：约 2 小时开发 + 联调。**

---

## 七、风险与备选

| 风险 | 影响 | 应对 |
|---|---|---|
| Gemini 未来变更 `groundingMetadata` 字段结构 | 业务侧解析失败 | 用 `json.RawMessage` 原样透传，OneAPI 不做 schema 假设；业务侧做 defensive parsing |
| 上游 OneAPI 主干合入时字段命名冲突 | rebase 冲突 | `metadata` 是通用词但当前 upstream 无此字段（已确认），冲突概率低 |
| 部分老版 OpenAI SDK 严格校验响应字段 | 反序列化失败 | 已用 `omitempty`，未启用 grounding 时不会出现该字段；启用方一般是自定义客户端，可控 |
| Gemini 关闭旧价格搜索工具 | grounding 消失 | 只影响业务效果，不影响 OneAPI 本身；届时通过监控告警 |

**备选方案（若本方案受阻）**：
- **方案 B（在 TrueSource 侧直连 Gemini）**：绕过 OneAPI，成本最高、失去了统一网关的价值。
- **方案 C（把 grounding 序列化后塞到 `choices[0].message.content` 的末尾特殊标记块）**：污染文本输出，业务解析成本高，不推荐。

本方案 A 是三者中改动最少、扩展性最好的路径。

---

## 八、文件改动摘要

```
relay/adaptor/openai/model.go     +2  行  (import + Metadata 字段 ×2)
relay/adaptor/gemini/model.go     +2  行  (import + GroundingMetadata 字段)
relay/adaptor/gemini/main.go     +18  行  (buildMetadata + 两处调用)
relay/adaptor/gemini/main_test.go +80 行  (新增或追加)
```

净增代码 ~100 行，无删除、无 breaking change。
