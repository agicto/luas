# ZGO Skills Library

## 📚 什么是 Skills?

Skills 是模块化、可复用的工作流和最佳实践，用于指导 AI Agent 在 ZGO 项目中执行特定任务。

## 🎯 核心理念

### 渐进式披露架构 (Progressive Disclosure)

```
Level 1: 元数据层 → 启动时加载所有 skill 的名称和描述
Level 2: 指令层 → 需要时加载详细的 Markdown 内容
Level 3: 资源层 → 按需加载脚本、示例和模板
```

这种设计确保：
- ✅ AI 知道所有可用能力（轻量级元数据）
- ✅ 仅在需要时加载详细内容（节省上下文）
- ✅ 资源延迟加载（优化性能）

## 📁 目录结构

每个 skill 是一个独立文件夹，包含：

```
skill-name/
├── SKILL.md              # 必需：主指令文件
│   ├── YAML frontmatter  # name, description, 元数据
│   └── Markdown content  # 详细指导和步骤
├── scripts/              # 可选：辅助脚本
│   └── *.sh
├── examples/             # 可选：代码示例
│   └── *.go, *.md
└── resources/            # 可选：额外资源
    └── templates/
```

## 🎨 Available Skills

### Core Development Skills

| Skill | 描述 | 优先级 |
|-------|------|--------|
| [`architecture-principles`](./architecture-principles/) | `seam / depth / locality` 的上位架构规则 | P0 |
| [`module-creation`](./module-creation/) | 创建 starter-style 业务模块的完整流程 | P0 |
| [`coding-standards`](./coding-standards/) | 当前 scaffold 的代码规范、分层约束和 seam 判断 | P0 |
| [`api-development`](./api-development/) | route-owning 模块的 HTTP / REST 规范 | P0 |
| [`database-design`](./database-design/) | PO、索引、迁移和表生命周期设计 | P0 |
| [`kest-flow`](./kest-flow/) | Kest Flow API 测试框架 | P1 |
| [`logging-standards`](./logging-standards/) | 结构化日志与观测性规范 | P1 |
| [`code-review-guide`](./code-review-guide/) | Review 流程、检查点和反馈规范 | P1 |

### Quality & Testing

| Skill | 描述 | 优先级 |
|-------|------|--------|
| [`testing-strategy`](./testing-strategy/) | 测试层级、真实 seam 与 mock 策略 | P1 |

### Operations

| Skill | 描述 | 优先级 |
|-------|------|--------|
| [`deployment`](./deployment/) | 部署工作流和检查清单 | P1 |

## 🧭 ZGO 架构词汇

这些词汇和 `CONTEXT.md` 保持一致，避免文档和代码审查继续漂移：

- `core`: 所有 ZGO 应用都会复用的运行时和基础设施
- `starter`: 默认随新项目交付的业务起步模块，例如 `user`、`apikey`、`audit`
- `capability`: 技术能力模块，例如 `internal/capabilities/ai`
- `optional starter`: 质量达到 starter，但不默认装配，例如 `permission`
- `example`: 以演示为主的代码或文档
- `starter registry`: 决定默认 starter、迁移、seed 的装配 seam
- `command manifest`: 决定 CLI 命令集合的装配 seam
- `error contract`: route-owning modules return stable `error_code` values plus `request_id` for correlation

### 重要说明

- `8-file` 结构是 **starter-style route-owning 模块的默认模板**，不是所有代码形状的唯一合法答案。
- `capability` 不应为了套模板而强行拥有 `handler.go`、`routes.go`、`dto.go`。
- 默认先写 concrete implementation。只有当行为确实在 seam 上变化时，才引入额外 interface。

`architecture-principles` 是这些规则的单一事实源；其他 skills 负责把这些原则翻译成具体动作。

## 🚀 如何使用 Skills

### For AI Agents

1. **启动时**: 扫描所有 skill 目录，加载 YAML frontmatter 元数据
2. **意图识别**: 分析用户请求，匹配相关 skills
3. **动态加载**: 读取匹配的 SKILL.md 详细内容
4. **执行指导**: 按照 skill 中的步骤执行任务
5. **资源调用**: 需要时加载 scripts/examples

### For Developers

```bash
# 查看可用 skills
ls .agent/skills/

# 阅读某个 skill
cat .agent/skills/module-creation/SKILL.md

# 运行 skill 脚本
.agent/skills/module-creation/scripts/validate-module.sh blog
```

## 📝 创建新 Skill

### 1. 复制模板

```bash
cp -r .agent/skills/_template .agent/skills/your-skill-name
```

### 2. 更新 SKILL.md

```markdown
---
name: your-skill-name
description: Brief description of what this skill does
version: 1.0.0
category: development|testing|operations
tags: [tag1, tag2]
author: Your Name
updated: 2026-01-24
---

# Your Skill Name

## Purpose
[详细说明这个 skill 的目的]

## When to Use
[什么时候应该使用这个 skill]

## Prerequisites
- [ ] 前置条件 1
- [ ] 前置条件 2

## Workflow Steps

### Step 1: [步骤标题]
[详细说明]

### Step 2: [步骤标题]
[详细说明]

## Troubleshooting
[常见问题和解决方案]

## Examples
[提供示例]

## Related Skills
- `other-skill`: Description

## References
- [相关文档链接]
```

### 3. 添加资源

- **scripts/**: 添加验证、生成、检查脚本
- **examples/**: 提供代码示例和使用案例
- **resources/**: 放置模板、配置文件等

### 4. 测试

与 AI Agent 交互，确保 skill 被正确识别和使用。

## 🎯 Skill vs Workflow

| 维度 | Skills | Workflows |
|------|--------|-----------|
| **位置** | `.agent/skills/` | `.agent/workflows/` |
| **加载** | 渐进式 (3层) | 一次性 |
| **内容** | 通用方法论 + 最佳实践 | 具体行动步骤 |
| **复杂度** | 高 (脚本+示例+资源) | 低 (纯文本) |
| **复用性** | 跨项目复用 | 项目特定 |
| **示例** | "如何创建 DDD 模块" | "部署到 api.zgo.dev" |

**关系**: Skills 提供**通用方法**，Workflows 提供**具体步骤**。

## 📊 Best Practices

### Skill 设计原则

1. **单一职责**: 一个 skill 专注一个任务类型
2. **清晰描述**: description 精准，便于 AI 路由
3. **完整文档**: 包含示例、边界情况、错误处理
4. **可执行性**: 提供脚本和工具，不只是文档
5. **依赖明确**: 声明需要的工具和前置条件

### YAML Frontmatter 规范

```yaml
---
name: kebab-case-name          # 必需：唯一标识符
description: One-line summary  # 必需：简短描述 (用于AI路由)
version: 1.0.0                 # 必需：语义化版本
category: development          # 必需：分类
tags: [tag1, tag2]            # 可选：标签
author: Team Name             # 可选：作者
updated: 2026-01-24           # 可选：更新日期
---
```

### Markdown 内容结构

```markdown
# Skill Title

## Purpose          ← 核心目的
## When to Use      ← 使用场景
## Prerequisites    ← 前置条件 (checklist)
## Workflow Steps   ← 详细步骤
## Troubleshooting  ← 故障排除
## Examples         ← 实际示例
## Related Skills   ← 相关 skills
## References       ← 参考文档
```

## 🔒 安全规范

### 脚本执行权限

- ✅ **自动执行**: 只读检查脚本 (validate-*.sh, check-*.sh)
- ⚠️ **需确认**: 写操作脚本 (deploy-*.sh, migrate-*.sh)
- ❌ **禁止**: 危险命令 (rm -rf, DROP TABLE)

### 代码审查

所有新增 skills 需要经过 Code Review：
- [ ] 元数据正确且完整
- [ ] 文档清晰易懂
- [ ] 脚本经过测试
- [ ] 示例代码可运行
- [ ] 无安全风险

## 📈 维护和演进

- 优先保持 `SKILL.md` 简洁，细节放 `examples/` 或 `scripts/`。
- 不要为单个 skill 增加额外的 `README.md`、`CHANGELOG.md`、`INSTALLATION_GUIDE.md` 等辅助文档。
- 当架构规则变化时，先更新 `architecture-principles`，再更新承接它的实施型 skills。

## 🤝 贡献指南

1. Fork 项目
2. 创建 skill 分支: `git checkout -b skill/your-skill-name`
3. 按照模板创建 skill
4. 测试 AI Agent 能正确使用
5. 提交 PR，说明 skill 用途和使用场景

## 📞 支持

- 📖 文档：见各个 skill 的 SKILL.md
- 🐛 问题：提交 GitHub Issue
- 💡 建议：提交 GitHub Discussion

---

**Let's build a smarter AI Agent together! 🚀**
