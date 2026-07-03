# 创建长任务提示词: AGI01 热点内容运营引擎

下面这段提示词用于创建一个长任务。执行者必须依赖以下两个文件作为产品和验收来源：

- `docs/CONTENT_AUTOMATION_PRODUCT_PLAN.md`
- `docs/CONTENT_AUTOMATION_USER_STORIES.md`

不要只依赖聊天上下文。如果聊天上下文和这两个文件冲突，以文件为准。

## Long Task Prompt

```text
你是 Codex，高级全栈工程师和产品型技术负责人。请在当前仓库中继续实现 AGI01 热点内容运营引擎。

必须先阅读并遵守：
- AGENTS.md
- api/AGENTS.md
- web/AGENTS.md
- CONTEXT.md
- contracts/README.md
- docs/CONTENT_AUTOMATION_PRODUCT_PLAN.md
- docs/CONTENT_AUTOMATION_USER_STORIES.md

目标：
按 docs/CONTENT_AUTOMATION_PRODUCT_PLAN.md 和 docs/CONTENT_AUTOMATION_USER_STORIES.md 实现完整产品闭环：
热点来源 -> 抓取入库 -> AI 去重/评分/筛选 -> 创建选题 -> skill 写作 -> 中英文母稿 -> 多平台适配 -> 审核 -> 导出/发布 -> 数据反馈。

第一优先级：
先完成 MVP 范围，不要跳到自动发布和复杂权限。MVP 范围见 docs/CONTENT_AUTOMATION_USER_STORIES.md 的 “MVP 覆盖范围”。

关键约束：
1. 通义千问 DashScope 是默认大模型 provider，但业务逻辑必须通过 api/internal/capabilities/ai 抽象调用。
2. 不要把真实 API Key 写入代码、测试、文档、日志、commit 或错误响应。
3. DashScope 接口使用 OpenAI compatible chat completions，base URL 通过 DASHSCOPE_BASE_URL 配置。
4. 当前 OpenAI provider 使用 /responses，不要用替换 base URL 的方式冒充 DashScope。
5. Web 只能通过 HTTP/BFF 调 API，不允许直接读取 api 文件或 skill 文件。
6. 所有 HTTP 响应遵循 contracts/README.md 的 envelope 和 error_code 规范。
7. 每个可验证功能必须独立 commit。不要把多个阶段混成一个巨大 commit。

实施顺序：

阶段 0: 重新确认现状
- 运行 git status，确认工作区状态。
- 阅读现有 trend module、ai capability、content pipeline migration、web trends feature。
- 确认当前已有测试命令和 Kest flow。

阶段 1: DashScope provider
- 新增 dashscope provider。
- 扩展 AI config 和 .env.example。
- 增加 provider 测试接口或 CLI 验证路径。
- 覆盖 key 缺失、模型缺失、provider 不可用、空响应、HTTP 错误。
- 新增或更新 Kest flow。
- 运行 API 测试和 Kest flow。
- commit: feat(ai): add dashscope provider

阶段 2: AI 模型设置页面和接口
- 实现 ai_providers / ai_model_profiles 所需 schema 或配置存储。
- 实现 /v1/ai/providers 和 /v1/ai/model-profiles 系列接口。
- Web 增加 /console/settings/ai 页面。
- 浏览器打开页面测试配置、空状态、错误状态。
- 新增 Kest flow。
- commit: feat(ai): add model profile management

阶段 3: 来源中心
- 增加 source_fetch_runs 和 source_raw_items。
- 完善 content_sources CRUD、pause、resume、poll、runs、raw-items。
- 增加 /console/sources 页面。
- 让 daily.dev 现有抓取迁移到新的 fetch run 记录。
- 单个来源失败不得影响其他来源。
- 新增 Kest flow。
- 浏览器测试来源列表、手动同步、错误展示。
- commit: feat(source): add source management

阶段 4: 热点详情和 AI 评分
- 增加趋势详情接口和页面。
- 用 DashScope 生成结构化评分，规则评分作为 fallback。
- 展示评分理由、推荐角度、风险、目标读者、评分历史。
- 实现 select、reject、archive。
- 新增 trend_decisions，如需要增加 trend_clusters 的最小实现。
- 新增 Kest flow 覆盖列表、详情、选择、拒绝。
- 浏览器测试 /console/trends 和 /console/trends/[id]。
- commit: feat(trend): add AI evaluation and trend detail

阶段 5: Skill 库
- 实现 skill repository sync。
- 拉取或读取 happycto/agi01-skills，生成不可变 skill_snapshots。
- Web 增加 /console/skills 页面。
- 每次文章生成必须能引用 skill_snapshot_ids。
- 同步失败时保留上一版可用快照。
- 新增 Kest flow。
- 浏览器测试 skill 列表、同步、详情。
- commit: feat(skill): add skill snapshot sync

阶段 6: 文章项目和工作台
- 实现 articles CRUD、从热点创建文章、手动创建文章。
- 实现 article runs、artifacts、timeline。
- 生成 research_brief 和 topic_card。
- Web 增加 /console/articles 和 /console/articles/[id]。
- 支持 artifact 查看、编辑、保存、重新生成。
- 新增 Kest flow 覆盖创建文章、启动 run、查看 artifact。
- 浏览器测试从热点创建文章到生成 brief 的完整链路。
- commit: feat(content): add article projects and research workflow

阶段 7: 写作流水线和质量门
- 生成 outline、draft zh-CN、draft en-US、title_candidates、image_plan、quality_report。
- 实现 approve、request-changes、版本保留。
- 增加 article_citations 和 quality_checks，如 schema 尚未存在。
- 未通过质量门不能进入 ready_to_publish。
- 新增 Kest flow 覆盖审批和打回。
- 浏览器测试文章工作台全部 artifact 状态。
- commit: feat(content): add drafting and quality gates

阶段 8: 多平台适配
- 增加 platform_profiles 和 platform_renditions。
- 内置默认 profile: wechat_mp, zhihu, feishu, xiaohongshu, x, linkedin。
- 实现生成、编辑、审批、导出平台稿。
- Web 增加 /console/articles/[id]/platforms。
- 平台稿不能新增未经验证事实。
- 新增 Kest flow。
- 浏览器测试多个平台 tab、导出和限制提示。
- commit: feat(platform): add platform renditions

阶段 9: 发布队列
- 增加 publication_targets 和 publish_jobs。
- 实现发布队列、导出、标记已发布、标记失败、重试。
- Web 增加 /console/publish。
- 未审核平台稿不能进入发布队列。
- 新增 Kest flow。
- 浏览器测试发布队列状态和导出。
- commit: feat(publish): add publish queue

阶段 10: 数据反馈
- 增加 content_metrics。
- 实现录入和查看文章/平台表现。
- 增加最小分析汇总接口和 UI。
- commit: feat(metrics): add content performance metrics

每个阶段必须做的验证：
1. 运行相关 Go 测试，例如：
   cd api && go test ./internal/capabilities/ai ./internal/modules/trend ./internal/starter
2. API 变更必须运行 Kest flow：
   cd api && make test-kest
   如果新增了专用 flow，也要运行：
   cd api && ./tests/kest/run_local.sh tests/kest/<new-flow>.flow.md
3. Web 变更必须运行：
   cd web && pnpm type-check
   cd web && pnpm lint
   cd web && pnpm build
4. UI 变更必须启动 dev server 或目标环境，用 Codex 内置浏览器打开相关页面，完成真实交互验证。至少检查：
   - 页面能加载
   - 数据能显示
   - 按钮能触发请求
   - 错误和空状态可见
   - 浏览器控制台无新增错误
5. 跨 API/Web 功能必须验证完整链路：浏览器 -> BFF/API -> PostgreSQL -> UI 回显。
6. migration 变更必须在 scratch DB 或测试 DB 上验证 up/down。
7. 每个阶段完成并验证后立即 commit。commit 前运行 git diff --check。commit 后再进入下一阶段。

提交策略：
- 每个功能 slice 一个 commit。
- commit message 使用 conventional commits。
- 不要在一个 commit 里混入无关重构。
- 如果发现用户已有未提交改动，不要回滚，先理解并绕开或协同。
- 每次 commit 前列出变更文件并确认没有 secret。

回归要求：
- 阶段性完成后运行完整回归：
  cd api && make test
  cd api && make test-kest
  cd web && pnpm type-check
  cd web && pnpm lint
  cd web && pnpm build
- 最终必须使用浏览器验证这些页面：
  /console
  /console/trends
  /console/trends/[id]
  /console/sources
  /console/skills
  /console/articles
  /console/articles/[id]
  /console/articles/[id]/platforms
  /console/publish
  /console/settings/ai

最终交付：
- 所有 MVP user stories 有实现或明确标记为 deferred。
- 所有新增接口有 Kest flow 或等价 API 测试。
- 所有新增页面可在浏览器打开并完成核心操作。
- 后台 worker 可运行并推进任务。
- PostgreSQL 数据真实写入并可在 UI 回显。
- 代码已分阶段 commit。
- 最终报告包含：
  - 完成的阶段和 commit 列表
  - 未完成或 deferred 项
  - 测试命令和结果
  - 浏览器验证页面
  - 部署/运行说明
  - 风险和下一步建议
```

## 使用说明

创建长任务时，把上面的 `Long Task Prompt` 整段复制进去。不要删掉文件依赖和验证要求。

如果执行过程中需求不清楚，先回到：

- `docs/CONTENT_AUTOMATION_PRODUCT_PLAN.md`
- `docs/CONTENT_AUTOMATION_USER_STORIES.md`

只有当这两个文件没有答案，才向用户提问。

