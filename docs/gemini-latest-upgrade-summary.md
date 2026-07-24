# Gemini 最新模型升级实施总结

> 编制日期：2026-07-24
> 对应方案：[`gemini-latest-upgrade-plan.md`](./gemini-latest-upgrade-plan.md)
> 分支：`main`（已直接在主干实施）

---

## 一、实施结果

方案 4.2 中 P0/P1 的 6 个改造点全部完成，Live/Computer Use/Video 三条 P2 能力线按方案约定不在本次交付范围。

| 步骤 | 状态 | 关键文件 |
|---|---|---|
| 1. 扩充 ModelList | ✅ 完成 | `relay/adaptor/geminiv2/constants.go` |
| 2. Vertex 侧同步 | ✅ 完成 | `relay/adaptor/vertexai/gemini/adapter.go` |
| 3. API 版本判定重构 | ✅ 完成 | `relay/adaptor/gemini/adaptor.go` |
| 4. 能力矩阵集中化 | ✅ 完成 | `relay/adaptor/gemini/constants.go` |
| 5. 计费费率更新 | ✅ 完成 | `relay/billing/ratio/model.go` |
| 6. 前端 3 主题同步 | ⏭️ 无需改动 | 见下方说明 |
| 7. Live / Computer Use / Video | ⏭️ 独立立项 | 方案 4.2 步骤 7 |

**说明**：核查 `web/{default,berry,air}/src/constants/*.js` 后确认这些文件只维护「渠道类型下拉」（channel type dropdown），模型下拉是通过后端 `/api/channel/models` 从 `Adaptor.GetModelList()` 动态获取，因此扩充 Go 侧 `ModelList` 后前端自动同步，无需触碰前端源码。

---

## 二、新增模型清单

### GA 稳定版本（本次接入）

| Model ID | 定位 | 输入 $/1M | 输出 $/1M |
|---|---|---|---|
| `gemini-3.6-flash` | 2026-07-21 主力 workhorse | $1.50 | $7.50 |
| `gemini-3.5-flash` | 上一代主力 | $1.50 | $9.00 |
| `gemini-3.5-flash-lite` | 低成本 flash | $0.30 | $2.50 |
| `gemini-3.1-flash-lite` | 前沿性价比 | $0.25 | $1.50 |
| `gemini-2.5-pro` | 深度推理旗舰 | $1.25 (<=200k) | $10 (<=200k) |
| `gemini-2.5-flash` | 低延迟推理 | $0.30 | $2.50 |
| `gemini-2.5-flash-lite` | 最便宜多模态 | $0.10 | $0.40 |
| `gemini-embedding-2` | 多模态 embedding | $0.10 | — |

### Preview 版本（本次接入）

- `gemini-3.1-flash-tts`（预览 TTS）

### 已存在（无需改动）

`gemini-3-pro-preview` / `gemini-3-pro-image-preview` / `gemini-3-flash-preview` / `gemini-3.1-pro-preview` / `gemini-2.5-flash-preview-tts` / `gemini-2.5-pro-preview-tts` 及全部 1.x / 2.0 系列。

---

## 三、代码改造要点

### 3.1 版本判定重构

**Before**（`adaptor.go`）：

```go
if strings.Contains(meta.ActualModelName, "gemini-3.1") ||
   strings.Contains(meta.ActualModelName, "gemini-3.0") ||
   ...
```

**After**：

```go
var v1betaModelPrefixes = []string{
    "gemini-1.5", "gemini-2.0", "gemini-2.5",
    "gemini-3-", "gemini-3.0", "gemini-3.1",
    "gemini-3.5", "gemini-3.6",
}

func requiresV1Beta(modelName string) bool {
    for _, p := range v1betaModelPrefixes {
        if strings.HasPrefix(modelName, p) {
            return true
        }
    }
    return false
}
```

未来新增 `gemini-4` 只需追加一行；`Contains → HasPrefix` 避免罕见的误匹配。

### 3.2 能力矩阵集中化

**Before**：两个独立 slice（`ModelsSupportSystemInstruction`, `ModelsWithImageGeneration`）+ 各一个循环查找。

**After**：单个 `map[string]ModelCapability` 表 + 三个 O(1) 查询函数：

```go
type ModelCapability struct {
    SystemInstruction bool
    ImageGeneration   bool
    TTS               bool
}

func IsModelSupportSystemInstruction(model string) bool { return modelCapabilities[model].SystemInstruction }
func IsModelSupportImageGeneration(model string) bool  { return modelCapabilities[model].ImageGeneration }
func IsModelSupportTTS(model string) bool              { return modelCapabilities[model].TTS }  // 新增
```

新增模型只在一张表中登记全部能力，避免多处漏配。

### 3.3 Vertex 双维护消除

**Before**：`vertexai/gemini/ModelList` 是一份独立手抄，落后 geminiv2 好几代。

**After**：

```go
var vertexOnlyModels = []string{
    "gemini-pro-vision", "gemini-exp-1206",
    "gemini-1.5-pro-001", "gemini-1.5-pro-002",
    "gemini-1.5-flash-001", "gemini-1.5-flash-002",
    "gemini-2.0-flash-001",
}
var ModelList = append(append([]string{}, vertexOnlyModels...), geminiv2.ModelList...)
```

未来 Google 只要新增一个通用 Gemini 型号，Vertex 渠道自动继承，不再遗漏。

---

## 四、遗留技术债 & 后续 Issue 建议

1. **分档费率（≤200k / >200k）**：`gemini-2.5-pro`、`gemini-3.1-pro-preview` 官方分档定价，本次按 ≤200k 档一律计费。建议开 issue 扩展 `ratio` 结构为 `TieredRatio{Small, Large, Threshold}`，并在 `relay/billing` 按 `promptTokens` 选档。
2. **Live API 能力线**（`gemini-3.1-flash-live` / `gemini-3.5-live-translate`）：需要 `relay/relaymode` 新增 `RealtimeAudio` 常量，并实现 WebSocket 通道。
3. **Computer Use Preview**：需要新的 tool call schema + 屏幕截图上传通道。
4. **Video 生成**（`gemini-omni-flash`）：参考 Veo 系列，新增独立 endpoint 与配额结构。

---

## 五、验证情况

| 项目 | 结果 |
|---|---|
| Go 语法审查 | ✅ 通过 |
| `go build` | ✅ 通过（服务器 Go 1.22.5 编译，16.4s，产物 38.28 MB） |
| `gofmt -l` | ✅ 通过 |
| `/api/status` 探活 | ✅ 内网 HTTP 200 (0.001s)，外网 http://oneapi.gitagent.io HTTP 200 (1.66s) |
| 新模型嵌入验证 | ✅ `strings one-api` 命中 gemini-3.6-flash / gemini-3.5-flash / gemini-2.5-pro / gemini-3.1-flash-lite / gemini-embedding-2 |
| 真实 API key 冒烟 | ⏸️ 待运维/QA 触发 `gemini-3.6-flash` 与 `gemini-2.5-pro` 实调 |

## 六、生产部署记录

- **服务器**：`support@104.197.139.51`（GCE `instance-20251220-110506` / us-central1-c）
- **代码目录**：`/mnt/disk-119/one-api`
- **进程**：`nohup ./one-api --port 3000 --log-dir ./logs`，新 PID **1593093**（老 PID 1355243 已 kill）
- **旧二进制备份**：`one-api.bak-20260724-003938`（38.27 MB, 2026-07-14 15:05 构建版）
- **入口**：nginx → localhost:3000 → oneapi.gitagent.io

### 6.1 服务器保留的生产定价（未提交到 GitHub，仅存在于服务器）

服务器 `ratio/model.go` 相对 commit `427c85c` 额外应用了以下覆盖，包含一个关键的 169× 图像计费修复。这些改动**保留在服务器本地**，未推送到仓库：

| 模型 | commit 值 | 服务器覆盖值 | 备注 |
|---|---|---|---|
| `gemini-2.0-flash-exp` | 0.075 | **0.15** | 与 `gemini-2.0-flash` 一致 |
| `gemini-3-pro-preview` | 1.25 | **2.00** | Gemini 2.5 Pro tier |
| `gemini-3-pro-image-preview` | 1.25 | **211** | **169× 图像 token 未计费修复** |
| `gemini-3-flash-preview` | 0.15 | **0.50** | Gemini 2.5 Flash tier |
| `gemini-3.1-pro-preview` | 1.25 | **2.00** | |
| `gemini-3.5-flash` | 0.75 | **1.50** | 6× 输出倍率对齐 |
| `gemini-embedding-2` | 0.05 | **0.20** | |
| `gemini-embedding-001` | — | **0.15** | 服务器新增 |
| `gemini-embedding-2-preview` | — | **0.20** | 服务器新增 |

外加 `CompletionRatio` 表新增：
- `gemini-2.0-flash*`: 4×
- `gemini-3-*-preview`, `gemini-3.1-pro-preview`, `gemini-3.5-flash`: 6×
- `gemini-embedding-*`: 0

**下次升级注意**：`git pull` 前先 `git stash` 保留本地定价，pull 后用 `/tmp/apply-server-pricing.py`（已保留在服务器）幂等重放。

### 6.2 部署 checklist（本次实际执行）

1. ✅ 服务器 SSH `support@104.197.139.51 -i ~/.ssh/google_compute_engine`
2. ✅ 备份 `/tmp/model.go.serverlocal.20260724-083509`, `/tmp/adaptor.go.serverlocal.20260724-083509`
3. ✅ `git stash push -m 'pre-gemini36-upgrade-server-local-pricing'`
4. ✅ `git pull origin main`（fast-forward 到 `427c85c`）
5. ✅ `python3 /tmp/apply-server-pricing.py` 重放服务器定价
6. ✅ `/usr/local/go/bin/gofmt -w relay/billing/ratio/model.go`
7. ✅ `/usr/local/go/bin/go build -ldflags '-s -w' -o one-api.new ./`（16.4s）
8. ✅ `cp one-api one-api.bak-20260724-003938 && mv one-api.new one-api`
9. ✅ `kill 1355243`（PID 精确 kill，避免 `pkill -f` 的自我竞态）
10. ✅ `setsid nohup ./one-api --port 3000 --log-dir ./logs ...`
11. ✅ 等待 DB migration 完成（10s，logs 表加索引）
12. ✅ 探活：内网 200 / 外网 200 / 二进制包含新模型 ID

---

## 六、GitHub 上游情况回顾

- 上游 `songquanpeng/one-api` 仍停留在 2025-02-21，本次升级 **未 merge upstream**（无内容可 merge）。
- 本地 `origin/main` 累计相对 upstream 领先 11 个提交（10 个历史 + 本次 5 个）。
- 后续如需回馈社区，可将 11 个提交整理为 upstream PR。

---

## 七、变更文件清单

```
relay/adaptor/geminiv2/constants.go              # 扩充模型
relay/adaptor/gemini/constants.go                # 能力矩阵集中化
relay/adaptor/gemini/adaptor.go                  # 版本判定重构
relay/adaptor/vertexai/gemini/adapter.go         # 复用 geminiv2
relay/billing/ratio/model.go                     # 新费率
docs/gemini-latest-upgrade-plan.md               # 方案（前置）
docs/gemini-latest-upgrade-summary.md            # 本文
```
