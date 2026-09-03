# Gemini 3.8 Flash 接入 & 生产升级记录

> 日期：2026-09-03
> 执行人：Claude Opus 4.7 (1M) + hangeaiagent
> 关联 commit：
> - `d493ba2` feat(gemini): 接入 Gemini 3.8 Flash（介绍价档）
> - `63e6a59` docs(gemini): 追加 Gemini 3.8 Flash 生产部署记录
> - `<本次 doc>` docs(升级): 归档 Gemini 3.8 Flash 升级过程

---

## 一、需求与背景

- 用户诉求：让 One API 支持 Google 最新的 `gemini-3.8-flash`，并同步升级线上部署。
- 参考文档：`docs/gemini-latest-upgrade-plan.md`（2026-07-24 制定的 Gemini 系列升级方案，此前已完成 3.6/3.5/2.5 GA 接入）。
- 定价来源（Google Cloud 官方）：
  - Standard-Global 介绍价（**2026-12-31 前**）：input $0.75/1M，cached $0.075/1M，output $3.75/1M
  - Standard-Global 全价（**2027-01-01 起**）：input $1.50/1M，cached $0.15/1M，output $7.50/1M
  - Priority（介绍价期间）：input $1.35，cached $0.135，output $6.75
  - Flex / Batch（介绍价期间）：input $0.375，cached $0.0375，output $1.875
  - 非-Global 区域一律加 10%

---

## 二、代码改动（commit `d493ba2`）

以最小改动方式接入，符合 3.6-flash 已建立的能力矩阵/费率/前缀路由模式，共 5 个文件、+38 行。

| 文件 | 改动 |
|---|---|
| `relay/adaptor/geminiv2/constants.go` | ModelList 追加 `gemini-3.8-flash`（Vertex 侧通过 `append(vertexOnlyModels, geminiv2.ModelList...)` 自动继承，无需二次维护）|
| `relay/adaptor/gemini/adaptor.go` | `v1betaModelPrefixes` 追加 `gemini-3.8`，让请求走 `v1beta` |
| `relay/adaptor/gemini/constants.go` | `modelCapabilities` 登记 `{SystemInstruction: true}` |
| `relay/billing/ratio/model.go` | 追加 `"gemini-3.8-flash": 0.375 * MILLI_USD`（介绍价档），注释里标记 2027-01-01 后需上调 |
| `docs/gemini-latest-upgrade-plan.md` | 追加「五之补」章节，记录接入决策、四档官方定价对照表、遗留技术债 |

**关键设计决策**

1. **暂按介绍价落费率**：`ratioValue * MILLI_USD` 在仓库约定下代表输入 $/1M，`0.375` 对应 $0.75/1M input（Global 介绍价）。选这个值而非 3.6-flash 的 `0.75`（$1.50/1M）是因为当前（2026-09-03）确实处于介绍价窗口。
2. **不加 server-override**：线上 `apply-server-pricing.py` 对 3.6-flash 也未做 override（保持仓库值），3.8-flash 沿用相同策略，保持一致性；如果后续统一 Gemini 3.x 走「值 = $/1M」约定，需要一并追加 3.6/3.8 flash 的 override + CompletionRatio。
3. **前端 3 主题无需改动**：`web/{default,berry,air}/src/constants/*.js` 只维护渠道类型下拉；模型下拉是通过 `/api/channel/models` 从 `Adaptor.GetModelList()` 动态获取，后端 ModelList 一改前端自动同步。

---

## 三、生产部署（服务器 `104.197.139.51`）

### 3.1 基本信息

| 项 | 值 |
|---|---|
| 服务器 | `support@104.197.139.51 -i ~/.ssh/google_compute_engine` |
| 主机名 | `instance-20251220-110506`（GCE, us-central1-c）|
| 代码目录 | `/mnt/disk-119/one-api` |
| 进程 | `nohup ./one-api --port 3000 --log-dir ./logs` |
| 入口 | nginx → localhost:3000 → oneapi.gitagent.io |
| Go 版本 | `/usr/local/go/bin/go`（1.22 系）|

### 3.2 服务器本地未提交改动的保留

服务器上有两个不进仓库的本地补丁，本次必须完整保留：

1. `relay/adaptor/gemini/adaptor.go`：`if requiresV1Beta(meta.ActualModelName) || meta.Mode == relaymode.Embeddings {` —— 让 embeddings 也走 v1beta。
2. `relay/billing/ratio/model.go`：定价 override + `gofmt -w` 排版差异（详见 `docs/gemini-latest-upgrade-summary.md` §6.1）。

### 3.3 部署 checklist（本次实际执行）

1. ✅ 备份：`/tmp/model.go.serverlocal.20260903-164947`、`/tmp/adaptor.go.serverlocal.20260903-164947`、`one-api.bak-20260903-164947`（38.29 MB）
2. ✅ `git stash push -m 'pre-gemini38-upgrade-20260903-164947' -- relay/billing/ratio/model.go relay/adaptor/gemini/adaptor.go`
3. ✅ `git pull origin main` fast-forward 到 `d493ba2`（+38 行 / 5 文件）
4. ✅ `python3 /tmp/apply-server-pricing.py` 幂等重放：7 条定价 override + 2 条 embedding 新增 + CompletionRatio Gemini 段
5. ✅ Python 就地补丁重放 embeddings-v1beta 修复（sed 因 `||` 与分隔符冲突失败后改用 python）
6. ✅ `/usr/local/go/bin/gofmt -w relay/billing/ratio/model.go relay/adaptor/gemini/adaptor.go`
7. ✅ `/usr/local/go/bin/go build -ldflags '-s -w' -o one-api.new ./`（46.7s，产物 38286232 B）
8. ✅ `mv one-api.new one-api`
9. ✅ 精确 `kill 1826877`（老 PID），1s 后进程消失
10. ✅ `setsid nohup ./one-api --port 3000 --log-dir ./logs > logs/one-api-boot-20260903.log 2>&1 & disown` → **新 PID 2665594**
11. ✅ DB migration 78s（logs 表 819,543 行，rebuild 6 个索引）
12. ✅ 探活：
    - 内网 `curl http://localhost:3000/api/status` HTTP 200（124µs）
    - 直连 IP `curl http://104.197.139.51:3000/api/status` HTTP 200 干净 JSON
    - 外网 `curl http://oneapi.gitagent.io/api/status` HTTP 200（~171ms，nginx gzip）
    - `strings one-api | grep gemini-3.8-flash` 命中 ✓
13. ⏸️ 真实 API key 冒烟 `POST /v1/chat/completions` model=`gemini-3.8-flash` —— 待运维/QA 使用生效渠道触发

### 3.4 关键 PID / 时间线

| 时刻 (UTC) | 事件 |
|---|---|
| 16:49:47 | 备份完成 |
| 16:49:47+ | git stash / git pull / apply-server-pricing / gofmt |
| 16:51:47 | 新二进制构建完成 |
| 16:51:47 | kill 1826877，启动 setsid nohup |
| 16:51:47 – 16:53:02 | DB migration（logs 表 rebuild 索引）|
| 16:53:02 | server started on http://localhost:3000 |
| 16:53:05 | 首次 `/api/status` 200 |

---

## 四、⚠️ 需要日历提醒的技术债

### 4.1 主时间炸弹：2026-12-31 定价调整

**On or before 2026-12-31**：把 `relay/billing/ratio/model.go` 里的

```go
"gemini-3.8-flash": 0.375 * MILLI_USD,
```

上调为

```go
"gemini-3.8-flash": 0.75 * MILLI_USD,
```

否则从 2027-01-01 起会持续按介绍价计费，**服务器每收 1 美元 Google 实收 2 美元**。已同步以下持久化提醒：

- Claude 项目记忆：`~/.claude/projects/-Users-a1-work-one-api/memory/gemini_3_8_flash_intro_pricing_bump.md`
- 本文档 §4.1（当前）
- 主升级 summary：`docs/gemini-latest-upgrade-summary.md` §6.3

**执行步骤（约 2026-12-15 触发）**：
1. 编辑 `relay/billing/ratio/model.go`，把 0.375 改成 0.75，同时删掉「intro pricing until 2026-12-31」注释
2. commit + push
3. 服务器执行本文 §3.3 的部署 checklist（不需要新增 apply-server-pricing.py 条目，因为 3.8-flash 沿用仓库值）

### 4.2 其它待办

1. **分档定价缺失**：`gemini-3.8-flash` 有 Standard / Priority / Flex-Batch / cached-input / 非-Global 加价共 5 组价格，ratio 表当前是单一浮点，只能落一档。建议扩展方案里已有的 `TieredRatio{Small, Large, Threshold}` 到「多档 + 计费入口按 mode 选档」。
2. **服务器本地补丁易失**：`apply-server-pricing.py` 只覆盖 ratio；`adaptor.go` 的 embeddings-v1beta 修复靠 `git stash` 保留，如果哪次 stash pop 出问题就会丢失。建议把这两个改动收敛成一份 upstream PR 或至少长驻仓库的 branch patch。
3. **上游合并**：`upstream/main` 仍停留在 `8df4a26` (2025-02-21)，本地累计领先 20 个提交。若打算回馈社区，可将 20 个提交整理成 upstream PR（不阻塞任何本地工作）。

---

## 五、变更文件清单

```
新增：
docs/升级/gemini-3.8-flash-2026-09-03.md       # 本文

修改（commit d493ba2）：
relay/adaptor/geminiv2/constants.go              # +3
relay/adaptor/gemini/adaptor.go                  # +1
relay/adaptor/gemini/constants.go                # +3
relay/billing/ratio/model.go                     # +3
docs/gemini-latest-upgrade-plan.md               # +28

修改（commit 63e6a59）：
docs/gemini-latest-upgrade-summary.md            # +32（部署记录 §6.3）
```

---

## 六、参考

- Google 官方模型页：<https://ai.google.dev/gemini-api/docs/models/gemini-3.8-flash>
- Google 官方定价：<https://cloud.google.com/vertex-ai/generative-ai/pricing>
- 前置升级方案：`docs/gemini-latest-upgrade-plan.md`
- 前置升级 summary：`docs/gemini-latest-upgrade-summary.md`
- 服务器定价重放脚本：`/tmp/apply-server-pricing.py`（服务器本地，未进仓库）
