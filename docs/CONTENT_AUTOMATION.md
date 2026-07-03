# AGI01 内容自动化后台设计

这份文档定义 AGI01 营销内容引擎的第一版产品切片：

```text
热点源 -> AI 筛选 -> PostgreSQL 记忆库 -> skill 写作工作流 -> 可审核发布包
```

## 产品定位

AGI01 不需要做一个通用 CMS。它更应该像一个内部编辑驾驶舱，服务一个 builder 型 AI 账号：

- 提前发现开发者和 AI 圈热点
- 判断哪些热点值得写
- 把选中的热点变成有研究、有观点、有风格的文章
- 稳定套用 AGI01 的写作 skills
- 发布前保留人工审核

第一版建议做半自动。系统可以抓取、评分、起草、翻译和打包，但选题卡、骨架、终稿、发布动作都应该有人确认。

## Skill 仓库

来源：`https://github.com/happycto/agi01-skills`

这个仓库不是业务代码，而是内容工作流资产。系统里应该把它当成可版本化的 prompt / 规则材料。

| Skill | 产品作用 | 系统产物 |
|---|---|---|
| `research-companion` | 验证热点、收集论据 | 带引用的 `research_brief` |
| `general-writer` | 长文写作骨架 | `topic_card`、`outline`、`draft`、`quality_report` |
| `shudong-writer` | AGI01 / Stark 个人风格层 | 中文主稿的风格约束 |
| `title-generator` | 公众号标题候选 | `title_candidates` |
| `illustration-companion` | 封面和段中配图规划 | `image_plan` 和生图 prompt |

每篇文章都要记录使用的 skill 快照：仓库 URL、commit SHA、skill 名、内容 hash、引用文件。这样后续 skill 升级后，仍然能复盘一篇文章是按哪一版规则生成的。

## 第一版闭环

先做这个最小可用版本：

1. 接入 `daily.dev/highlights` 作为热点源。
2. 把热点标准化后写入 PostgreSQL。
3. 对新热点跑 AI 评分，输出 HKR 和品牌匹配度。
4. 在后台展示内部选题池。
5. 运营人员选中一个热点，创建文章项目。
6. 按顺序生成这些 artifact：
   - 研究 brief
   - 选题卡
   - 带图位的骨架
   - AGI01 风格中文稿
   - 英文适配稿
   - 标题候选
   - 配图规划
   - 质量检查报告
7. 人工审核后，才允许进入待发布状态。

第一版不做：

- 自动发布到公众号
- 自动发社媒
- 复杂付费源管理
- 多角色权限体系
- 完整生图流水线
- 全量查重系统

## 后端模块

建议在 `api/internal/modules/` 下拆三个模块：

| 模块 | 职责 |
|---|---|
| `trend` | 热点源、抓取、去重、评分、选中 |
| `content` | 文章项目、生成 run、artifact、审核状态 |
| `skill` | skill 仓库同步、快照解析、prompt 组装 |

复用现有能力：

- `api/internal/capabilities/ai`：文本生成
- `api/internal/infra/http`：daily.dev 和后续外部源 adapter
- `api/internal/infra/schedule`：定时抓取和扫描
- `api/internal/infra/queue`：文章生成任务

不要让 `web/` 直接读取 skill 文件。前端只通过 API 获取已入库的 artifact 和状态。

## 后台页面

内部 console 建议做五个工作区：

| 页面 | 作用 |
|---|---|
| Trend Radar | 新热点、频道、AI 分数、选中状态 |
| Source Settings | daily.dev/API/RSS 源配置、轮询状态、错误 |
| Skill Library | 已同步 skills、commit SHA、快照历史、当前启用版本 |
| Article Studio | 文章项目时间线、artifact、审核、重新生成 |
| Publish Queue | 待发布包、标题、封面规划、正文、checklist |

AI 评分必须展示原因。只有分数、没有理由，对编辑决策没有价值。

## 自动化流程

推荐任务：

| Job | 频率 | 说明 |
|---|---:|---|
| `sync_skills` | 手动 + 每日 | 拉取 `happycto/agi01-skills`，生成不可变快照 |
| `poll_daily_dev_highlights` | 每 60 分钟 | 解析 daily.dev highlights；后续优先切官方 API |
| `score_new_trends` | 每 15 分钟 | 对未评分热点跑规则 + AI 评分 |
| `send_daily_shortlist` | 每天 09:00 | 给运营展示 Top 5-10 候选 |
| `continue_article_runs` | 每 5 分钟 | 推进排队中的文章生成阶段 |
| `quality_gate` | artifact 生成后 | 扫 AI 味、引用覆盖、截图/图位缺口 |

当前已实现第一条本地同步器：

```bash
cd api

# 单次同步：抓取 daily.dev highlights -> upsert trend_items -> 规则评分 -> 写入 score_trend job
go run ./cmd/trend-sync --once

# 长驻同步：立即跑一次，然后每 10 分钟跑一次
go run ./cmd/trend-sync --interval=10m
```

本地启动为后台进程时：

```bash
cd api
go build -o storage/bin/trend-sync ./cmd/trend-sync
nohup storage/bin/trend-sync --interval=10m > storage/logs/trend-sync.log 2>&1 &
echo $! > storage/trend-sync.pid
```

在 macOS 上更推荐用用户级 `launchd` 托管，避免普通 shell 退出后后台进程被清理。当前本地服务名：

```bash
# 查看状态
launchctl print gui/$(id -u)/com.agi01.trend-sync

# 停止
launchctl bootout gui/$(id -u) ~/Library/LaunchAgents/com.agi01.trend-sync.plist

# 重启
launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/com.agi01.trend-sync.plist
launchctl kickstart -k gui/$(id -u)/com.agi01.trend-sync
```

状态机：

```text
trend.new
  -> trend.candidate
  -> trend.selected
  -> article.researching
  -> article.topic_review
  -> article.outline_review
  -> article.drafting
  -> article.final_review
  -> article.ready_to_publish
  -> article.published
```

被拒绝的热点不要删除，保留在 PostgreSQL 里用于去重和模型反馈。

## 中英文策略

建议中文是 AGI01 的主版本。

英文稿不要做逐字翻译，而是做英文适配：

- 事实和引用保持一致
- 减少公众号语境里的表达
- 替换只适合中文读者的固定尾部或行动召唤
- 保留 builder 口吻

推荐 artifact：

- `draft`，`language = zh-CN`
- `translation` 或 `draft`，`language = en-US`
- 每个语言各有一个 `quality_report`

## 热点评分

使用混合分数：

```text
final_score =
  source_score
  + recency_score
  + engagement_score
  + hkr_score
  + brand_fit_score
  - risk_score
```

AI 评估输出：

- `h_score`：钩子强度
- `k_score`：知识密度
- `r_score`：读者共鸣
- `brand_fit`：是否适合 AGI01 / builder 读者
- `risk_notes`：事实、版权、过热、误导风险
- `recommended_angle`：推荐写作角度
- `target_audience`：AI 创业者、程序员、产品/运营或混合
- `skill_plan`：应该调用哪些 skills，以及为什么

## 质量门

发布包进入 ready 前必须满足：

- 中文终稿 AI 味黑名单零命中
- 每个硬事实都有来源，或明确标记为个人判断
- 生成内容不能复写来源摘要
- 真实截图位置必须标记为真实截图，不能用生图伪造
- 标题候选有风险分级
- 标题和封面互补，不重复同一信息
- 中文公众号版本带 AGI01 固定尾部

## API 草案

所有接口遵循 `contracts/README.md` 里的 Luas response envelope。

```text
GET    /api/trends
POST   /api/trends/ingest
POST   /api/trends/{id}/evaluate
POST   /api/trends/{id}/select

GET    /api/skills
POST   /api/skills/sync
GET    /api/skills/{id}/snapshots

POST   /api/articles
GET    /api/articles
GET    /api/articles/{id}
POST   /api/articles/{id}/runs
POST   /api/articles/{id}/artifacts/{artifact_id}/approve
POST   /api/articles/{id}/artifacts/{artifact_id}/request-changes
GET    /api/articles/{id}/publish-package
```

## 本地 PostgreSQL

本地库名：

```bash
createdb -h localhost agi01_content_local
psql -h localhost -d agi01_content_local -f api/database/schema/content_pipeline.sql
```

如果 Homebrew PostgreSQL 没启动：

```bash
brew services start postgresql@14
pg_isready -h localhost -p 5432
```

当前 schema 文件先作为产品设计库存在。等这个切片确认后，再转换成 Luas 的 Go migration，放到 `api/database/migrations/`，并在新模块里注册 GORM model。
