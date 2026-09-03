# Google Gemini 最新模型接入升级技术方案

> 编制日期：2026-07-24
> 目标：将 One API 项目中 Google Gemini 适配层升级到官方最新模型矩阵（含 Gemini 3.6 Flash 等 2026-07 GA 模型）。

---

## 一、GitHub 上游版本整合分析

### 1.1 仓库现状

| 项目 | 值 |
|---|---|
| 本地远端 `origin` | `https://github.com/hangeaiagent/one-api` |
| 上游 `upstream` | `https://github.com/songquanpeng/one-api.git` |
| 本地最新提交 | `d2b044f docs: 添加图片模型问题排查文档` |
| upstream 最新提交 | `8df4a26 docs: update ByteDance Doubao model link in README`（2025-02-21） |
| 分支关系 | 本地 **领先** upstream 10 个提交，**落后 0 个** |

### 1.2 与上游的差异内容（本地已有）

本地领先 upstream 的 10 个提交全部集中在 Google 生态，可视为「一份未合入 upstream 的 Google 能力增强补丁」：

```
d2b044f docs: 添加图片模型问题排查文档
3f50f26 docs: 添加 TTS 支持方案文档和 Chirp 使用指南
2efa5a5 fix: 为 Google Cloud TTS 渠道添加专用测试逻辑
5f6691b fix: 从完整语音名中自动提取语言代码
67f27a2 fix: 修正 Chirp 2 语音映射，使用模型特定的语音名称
c1a24d6 fix: 补充 berry 主题 Google Cloud TTS (Chirp) 渠道选项
e4fddf6 fix: 修正 Gemini TTS 模型名称为 gemini-2.5-flash-preview-tts / gemini-2.5-pro-preview-tts
86ee9c0 feat: 添加 Google TTS 模型支持 (Gemini 2.5 TTS + Chirp 系列)
c1153e6 feat: 添加 gemini-3.1-pro-preview 模型支持及性能优化
26ef823 feat: 添加 Google Gemini 3.0 模型支持
```

### 1.3 整合结论

- **上游无新增内容需合并**：`upstream/main` 停留在 2025-02，已 5 个月未更新，无需执行 `git merge upstream/main`。
- **本地是 Gemini 能力上的超集**：升级工作 **在本地分支直接推进** 即可。
- **可选后续动作**：若希望回馈社区，可将 10 个本地提交整理成一份 PR 提交 upstream。

---

## 二、Google Gemini 最新模型矩阵（截至 2026-07-24）

### 2.1 GA/稳定版本

| Model ID | 定位 | 上下文 | 输入 $/1M | 输出 $/1M | 备注 |
|---|---|---|---|---|---|
| `gemini-3.6-flash` | **主力 workhorse**，2026-07-21 GA | 1M | $1.50 | $7.50 | 相比 3.5 Flash 减少 17% 输出 token；知识截止 2026-03 |
| `gemini-3.5-flash` | 上一代主力 | 1M | $1.50 | $9.00 | 保留兼容 |
| `gemini-3.5-flash-lite` | 极低成本 flash | 1M | $0.30 | $2.50 | 2026-07-21 与 3.6 同批 GA |
| `gemini-3.1-flash-lite` | 前沿性价比 | 1M | $0.25/$0.50(audio) | $1.50 | |
| `gemini-2.5-pro` | 深度推理旗舰 | 2M | $1.25/$2.50 分档 | $10/$15 分档 | 200k tokens 为价格分界 |
| `gemini-2.5-flash` | 低延迟推理 | 1M | $0.30/$1.00(audio) | $2.50 | |
| `gemini-2.5-flash-lite` | 最便宜多模态 | 1M | $0.10/$0.30(audio) | $0.40 | |

### 2.2 Preview 版本

| Model ID | 能力 |
|---|---|
| `gemini-3.1-pro-preview` | 高级推理（本地已支持） |
| `gemini-3-flash` | 前沿 flash，逼近大模型 |
| `gemini-3.1-flash-live` | Live API，实时语音对话 |
| `gemini-3.1-flash-tts` | 低延迟语音合成 |
| `gemini-3.5-live-translate` | 70+ 语言实时同传 |
| `gemini-omni-flash` | 视频生成与编辑 |

### 2.3 专用模型

| Model ID | 能力 |
|---|---|
| `gemini-embedding-2` | 多模态 embedding |
| `computer-use-preview` | 浏览器/UI 自动化（Agent 能力） |
| Deep Research Models | 多步自主研究 Agent |

### 2.4 关键结论

- Google 最新旗舰是 **Gemini 3.6 Flash**（2026-07-21 GA），成为主力推荐模型。
- 3.5 Pro 因质量未达标被跳过发布，Gemini Pro 线仍以 **2.5 Pro** 为最新稳定旗舰，`3.1-pro-preview` 为预览。
- 出现了三条新的独立能力线：**Live API**、**Computer Use**、**Video（Omni Flash）**，需要单独适配。

---

## 三、本地 Gemini 适配层现状盘点

### 3.1 关键文件

| 文件 | 作用 |
|---|---|
| `relay/adaptor/gemini/adaptor.go` | 主适配器，路由 URL 拼装（含 v1beta 判定） |
| `relay/adaptor/gemini/constants.go` | 系统指令支持列表、图像生成支持列表 |
| `relay/adaptor/gemini/main.go` | 请求转换、SSE 流处理 |
| `relay/adaptor/gemini/tts.go` | Gemini TTS 转换/处理 |
| `relay/adaptor/geminiv2/constants.go` | **ModelList 主定义位置** |
| `relay/adaptor/vertexai/gemini/adapter.go` | Vertex AI 侧的 Gemini 模型清单（老旧） |
| `relay/adaptor/vertexai/registry.go` | Vertex 适配器注册 |
| `relay/billing/ratio/model.go` | 计费费率表 |
| `web/{default,berry,air}/src/constants/*.js` | 前端渠道选项 |

### 3.2 当前已支持的 Gemini 3.x 模型

```go
// relay/adaptor/geminiv2/constants.go
"gemini-3-pro-preview",
"gemini-3-pro-image-preview",
"gemini-3-flash-preview",
"gemini-3.1-pro-preview",
"gemini-2.5-flash-preview-tts",
"gemini-2.5-pro-preview-tts",
```

### 3.3 与官方最新模型的差异（缺口）

| 缺失模型 | 类型 | 优先级 |
|---|---|---|
| `gemini-3.6-flash` | GA 主力 | **P0** |
| `gemini-3.5-flash` | GA | **P0** |
| `gemini-3.5-flash-lite` | GA | **P0** |
| `gemini-3.1-flash-lite` | GA | **P1** |
| `gemini-2.5-pro` / `gemini-2.5-flash` / `gemini-2.5-flash-lite`（纯名版） | GA | **P0** |
| `gemini-3.1-flash-tts` | Preview | P1 |
| `gemini-3.1-flash-live` / `gemini-3.5-live-translate` | Preview / Live API | P2（能力线） |
| `gemini-omni-flash` | Preview / Video | P2（能力线） |
| `gemini-embedding-2` | GA | P1 |
| `computer-use-preview` | Preview / Agent | P2（能力线） |

### 3.4 已识别的技术债

1. **`geminiv2/ModelList` 与 `vertexai/gemini/ModelList` 双维护**——两处清单不同步，Vertex 侧只到 `gemini-2.0-flash-*`，缺 3.x 全部。
2. **API 版本判断硬编码**（`adaptor.go:28-34`）——每个新版本前缀（`gemini-3.6`, `gemini-3.7`…）都要手动添加到 `strings.Contains` 链，容易遗漏。
3. **系统指令 / 图像生成能力列表**是手工维护数组（`constants.go`）——新增模型时容易漏配。
4. **费率表既有 `MILLI_USD` 又有分档（<200k / >200k）需求**——`ratio/model.go` 当前是单一浮点，无法表达分档定价。
5. **前端渠道选项分散在 3 套主题**——`default`、`berry`、`air` 均需同步。

---

## 四、升级技术方案

### 4.1 升级目标（Scope）

- **必须（本次交付）**：接入 2.5/3.5/3.6 系列所有 GA 模型 + 更新费率 + 前端渠道选项。
- **可选（本次评估）**：接入 3.x preview TTS、Embedding 2。
- **后续（独立立项）**：Live API、Computer Use、Video 生成 —— 需扩展 `relaymode` 与新适配器。

### 4.2 分层改造

#### 步骤 1：扩充 ModelList（P0）

**文件**：`relay/adaptor/geminiv2/constants.go`

```go
var ModelList = []string{
    // ...历史模型保留...

    // Gemini 2.5 GA (新增，纯名版)
    "gemini-2.5-pro",
    "gemini-2.5-flash",
    "gemini-2.5-flash-lite",

    // Gemini 3.5 GA (新增)
    "gemini-3.5-flash",
    "gemini-3.5-flash-lite",

    // Gemini 3.6 GA - 2026-07-21 主力 (新增)
    "gemini-3.6-flash",

    // Gemini 3.1 flash-lite GA (新增)
    "gemini-3.1-flash-lite",

    // Preview（可选）
    "gemini-3.1-flash-tts",

    // Embedding
    "gemini-embedding-2",
}
```

#### 步骤 2：同步 Vertex AI 清单

**文件**：`relay/adaptor/vertexai/gemini/adapter.go:17-25`

将 Vertex 侧 `ModelList` 由手抄改为「复用 geminiv2.ModelList + Vertex 特有模型」：

```go
import "github.com/songquanpeng/one-api/relay/adaptor/geminiv2"

var ModelList = append(
    []string{
        // Vertex 特有别名 / region 版本
        "gemini-1.5-pro-001", "gemini-1.5-pro-002",
        "gemini-1.5-flash-001", "gemini-1.5-flash-002",
        "gemini-2.0-flash-001",
    },
    geminiv2.ModelList...,
)
```

> 这样彻底消除双维护，只有 Vertex 独有的 `-001/-002` 版本需要手动列出。

#### 步骤 3：重构 API 版本判定（消除硬编码）

**文件**：`relay/adaptor/gemini/adaptor.go:26-51`

将 `strings.Contains` 判断改为「白名单集中管理 + 前缀正则」：

```go
// v1betaModelPrefixes 需要走 v1beta 的模型前缀
var v1betaModelPrefixes = []string{
    "gemini-1.5", "gemini-2.0", "gemini-2.5",
    "gemini-3-", "gemini-3.0", "gemini-3.1",
    "gemini-3.5", "gemini-3.6",
}

func requiresV1Beta(model string) bool {
    for _, p := range v1betaModelPrefixes {
        if strings.HasPrefix(model, p) {
            return true
        }
    }
    return false
}
```

以后新增 `gemini-3.7`、`gemini-4` 只需在切片增加一行；同时把 `Contains` 换成 `HasPrefix`，避免误命中。

#### 步骤 4：能力矩阵改为 map 集中表

**文件**：`relay/adaptor/gemini/constants.go`

```go
type ModelCapability struct {
    SystemInstruction bool
    ImageGeneration   bool
    TTS               bool
    Live              bool
}

var modelCapabilities = map[string]ModelCapability{
    "gemini-3.6-flash":            {SystemInstruction: true},
    "gemini-3.5-flash":            {SystemInstruction: true},
    "gemini-3.5-flash-lite":       {SystemInstruction: true},
    "gemini-2.5-pro":              {SystemInstruction: true},
    "gemini-2.5-flash":            {SystemInstruction: true},
    "gemini-2.5-flash-lite":       {SystemInstruction: true},
    "gemini-3.1-flash-lite":       {SystemInstruction: true},
    "gemini-3-pro-preview":        {SystemInstruction: true},
    "gemini-3-pro-image-preview":  {SystemInstruction: true, ImageGeneration: true},
    "gemini-3-flash-preview":      {SystemInstruction: true},
    "gemini-3.1-pro-preview":      {SystemInstruction: true},
    "gemini-2.0-flash":            {SystemInstruction: true},
    "gemini-2.0-flash-exp":        {SystemInstruction: true, ImageGeneration: true},
    "gemini-2.5-flash-preview-tts":{TTS: true},
    "gemini-2.5-pro-preview-tts":  {TTS: true},
    "gemini-3.1-flash-tts":        {TTS: true},
}
```

原 `IsModelSupportSystemInstruction`、`IsModelSupportImageGeneration` 改为读取该表；新增 `IsModelSupportTTS`、`IsModelSupportLive` 供后续扩展。

#### 步骤 5：更新计费费率

**文件**：`relay/billing/ratio/model.go:128-150`

```go
// Gemini 3.6 - 2026-07-21 GA
"gemini-3.6-flash":       0.75 * MILLI_USD,   // $1.50/1M input
"gemini-3.5-flash":       0.75 * MILLI_USD,   // $1.50/1M input
"gemini-3.5-flash-lite":  0.15 * MILLI_USD,   // $0.30/1M input

// Gemini 3.1 flash-lite
"gemini-3.1-flash-lite":  0.125 * MILLI_USD,  // $0.25/1M input (text)

// Gemini 2.5 GA (纯名版)
"gemini-2.5-pro":         0.625 * MILLI_USD,  // $1.25/1M input (<200k)
"gemini-2.5-flash":       0.15 * MILLI_USD,   // $0.30/1M input (text)
"gemini-2.5-flash-lite":  0.05 * MILLI_USD,   // $0.10/1M input (text)

// Embedding
"gemini-embedding-2":     0.05 * MILLI_USD,
```

> **重要**：`gemini-2.5-pro`、`gemini-3.1-pro-preview` 官方按「输入 token ≤200k / >200k」分档定价，当前 `ratioMap` 是单值 `float64`，短期先按 **≤200k 档位** 计费；后续如需精准按输入 token 长度动态选择费率，需要扩展 `ratio` 结构为：
>
> ```go
> type TieredRatio struct {
>     Small float64 // <= threshold
>     Large float64 // > threshold
>     Threshold int
> }
> ```
>
> 并在 `relay/billing` 计费入口按 `promptTokens` 选档。属于**独立技术债 issue**，本次不必阻塞发布。

#### 步骤 6：前端渠道选项同步

**文件**（3 处主题需同步）：
- `web/default/src/constants/channel.constants.js`
- `web/berry/src/constants/ChannelConstants.js`
- `web/air/src/constants/channel.constants.js`

搜索 `gemini-3` / `gemini-2.5`，将步骤 1 中的新模型 ID 加入 Google Gemini 渠道的默认候选列表；确保「测试模型」下拉能选到 `gemini-3.6-flash`。

#### 步骤 7：Live / Computer Use / Video（后续独立立项）

这三条属于新的能力线，与 OpenAI 兼容协议差异较大：

| 能力 | 需扩展的位置 |
|---|---|
| Live API（`gemini-3.1-flash-live`, `gemini-3.5-live-translate`） | `relay/relaymode/` 新增 `RealtimeAudio` 常量；新增 `relay/adaptor/gemini/live.go`；WebSocket 通道 |
| Computer Use（`computer-use-preview`） | 需要 tool call schema 转换 + 屏幕截图上传通道 |
| Video（`gemini-omni-flash`） | 参考 Veo 系列适配思路，独立 endpoint |

**建议**：本次升级不覆盖，产出 3 个独立 issue，评估 UI/协议影响后再排期。

### 4.3 兼容性 & 回归风险

| 风险 | 缓解措施 |
|---|---|
| 修改 `requiresV1Beta` 从 `Contains` 换成 `HasPrefix` 可能让「`my-gemini-3.5` 自定义模型名」不再匹配 | 保留旧函数为 fallback；或在切片中同时加 `"gemini-3.5"` 前缀模式 |
| Vertex ModelList 合并 geminiv2.ModelList 后，会出现 Vertex 上不存在的 preview 名 | 用户创建 Vertex 渠道时未启用的模型会 404，属预期；由用户按需勾选 |
| `gemini-2.5-pro` 费率单值可能少收费 | 短期文档提示；长期实现 `TieredRatio` |

### 4.4 测试计划

1. **单元测试**：新增 `TestRequiresV1Beta`、`TestModelCapabilities` 覆盖新模型。
2. **手工冒烟**（真实 API key）：
   - `POST /v1/chat/completions` model=`gemini-3.6-flash`，验证 200 + stream。
   - model=`gemini-2.5-pro` 长文本（>200k）验证走 v1beta 且计费产生。
   - model=`gemini-3.1-flash-tts` 走 `/v1/audio/speech`。
3. **回归**：现存 `gemini-1.5-*`、`gemini-2.0-*`、Chirp TTS 用例保持通过。
4. **前端**：3 套主题渠道创建/编辑页均能看到并勾选新模型。

### 4.5 交付清单

| 交付物 | 文件数（预估） | 交付形态 |
|---|---|---|
| ModelList 更新 | 2 | 单 commit `feat: 添加 Gemini 3.6/3.5/2.5 GA 模型` |
| Vertex 侧模型同步 | 1 | 单 commit `refactor: Vertex Gemini 复用 geminiv2 ModelList` |
| API 版本判定重构 | 1 | 单 commit `refactor: Gemini API 版本判定改为前缀白名单` |
| 能力矩阵重构 | 1 | 单 commit `refactor: Gemini 模型能力表集中化` |
| 计费费率 | 1 | 单 commit `feat: 添加 Gemini 3.6/3.5/2.5 GA 费率` |
| 前端 3 主题 | 3 | 单 commit `feat: 前端同步 Gemini 新模型选项` |
| 文档 | 本文件 | 已生成 |

---

## 五、执行 Checklist

- [ ] 拉取本地分支并创建 `feature/gemini-latest` 分支
- [ ] 步骤 1：`geminiv2/constants.go` 新增模型
- [ ] 步骤 2：`vertexai/gemini/adapter.go` 合并 ModelList
- [ ] 步骤 3：`gemini/adaptor.go` 重构版本判定
- [ ] 步骤 4：`gemini/constants.go` 能力矩阵化
- [ ] 步骤 5：`ratio/model.go` 新增费率
- [ ] 步骤 6：前端 3 主题 `channel.constants.js` 同步
- [ ] 运行 `go test ./relay/adaptor/gemini/... ./relay/adaptor/vertexai/...`
- [ ] 真实 API key 冒烟 `gemini-3.6-flash` + `gemini-2.5-pro`
- [ ] 三主题前端手工验证渠道编辑页
- [ ] 提交 PR，评审后合入 `main`
- [ ] （可选）将本地 10 个提交 + 本次升级整理成 upstream PR

---

## 五之补：Gemini 3.8 Flash 追加（2026-09-04）

Google 于 2026 年下半年推出 **Gemini 3.8 Flash**（页面：<https://ai.google.dev/gemini-api/docs/models/gemini-3.8-flash>），本次以最小改动方式接入：

**改动点**

- `relay/adaptor/geminiv2/constants.go` 追加 `gemini-3.8-flash`（Vertex 侧通过 `append(vertexOnlyModels, geminiv2.ModelList...)` 自动继承）。
- `relay/adaptor/gemini/adaptor.go` `v1betaModelPrefixes` 追加 `gemini-3.8`。
- `relay/adaptor/gemini/constants.go` `modelCapabilities` 追加 `{SystemInstruction: true}`。
- `relay/billing/ratio/model.go` 追加 `"gemini-3.8-flash": 0.375 * MILLI_USD`。

**定价来源（Google Cloud 官方页面）**

| 档位 | 输入 $/1M | 缓存输入 $/1M | 输出 $/1M | 备注 |
|---|---|---|---|---|
| Standard (Global) — 介绍价，2026-12-31 前 | 0.75 | 0.075 | 3.75 | **本次采用** |
| Standard (Global) — 2027-01-01 起 | 1.50 | 0.15 | 7.50 | 到期后需改为 `0.75 * MILLI_USD` |
| Standard (非 Global) — 介绍价 | 0.825 | 0.0825 | 4.125 | 未区分区域 |
| Priority（介绍价） | 1.35 | 0.135 | 6.75 | 未区分档位 |
| Flex / Batch（介绍价） | 0.375 | 0.0375 | 1.875 | 未区分档位 |

**技术债**

- ratio 目前是单一浮点，无法表达 Standard / Priority / Flex-Batch 分档，也无法自动跨过 2026-12-31 提价日；到期前需要人工修改并部署。
- 缓存输入、非 Global 区域加价均未反映，如后续需要精算需扩展 `TieredRatio` 结构（原方案第 4.2.5 节已埋伏笔）。

## 六、参考资料

- Google 官方模型页：<https://ai.google.dev/gemini-api/docs/models>
- Google 官方定价：<https://ai.google.dev/gemini-api/docs/pricing>
- Gemini API Changelog：<https://ai.google.dev/gemini-api/docs/changelog>
- Gemini 3.6 Flash 发布报道（TechCrunch, 2026-07-21）：<https://techcrunch.com/2026/07/21/google-releases-three-new-gemini-models-but-no-3-5-pro/>
- Gemini 3.6 Flash 报道（9to5Google, 2026-07-21）：<https://9to5google.com/2026/07/21/gemini-3-6-flash-launch/>
- Gemini 3.6 Flash Model ID 与规格（kie.ai）：<https://kie.ai/blog/what-is-gemini-3-6-flash>
- Gemini 3.8 Flash 官方模型页：<https://ai.google.dev/gemini-api/docs/models/gemini-3.8-flash>
- Gemini 3.8 Flash 官方定价（Google Cloud）：<https://cloud.google.com/vertex-ai/generative-ai/pricing>
