# ZGO 项目 Skill 系统集成方案

## 📌 方案概述

基于 Claude Skill 的核心设计理念，为 ZGO 项目打造一个**渐进式披露、模块化、可扩展**的 AI Agent Skill 系统。

## 🎯 集成目标

1. **提升 AI Agent 效率**: 通过预定义工作流减少重复指令
2. **知识库标准化**: 将团队最佳实践固化为 skills
3. **降低上下文成本**: 采用渐进式加载策略
4. **模块化管理**: 每个 skill 独立维护和版本控制
5. **跨项目复用**: skills 可在多个项目间共享

## 🏗️ 目录结构设计

```
zgo/
├── .agent/
│   ├── skills/              # Skill 库根目录
│   │   ├── README.md        # Skill 使用指南
│   │   ├── _template/       # Skill 模板
│   │   │   ├── SKILL.md
│   │   │   ├── scripts/
│   │   │   └── examples/
│   │   │
│   │   ├── module-creation/ # 创建新模块的完整工作流
│   │   │   ├── SKILL.md
│   │   │   ├── scripts/
│   │   │   │   ├── validate-module.sh
│   │   │   │   └── post-create-check.sh
│   │   │   └── examples/
│   │   │       └── user-module-example.md
│   │   │
│   │   ├── api-development/ # API 开发最佳实践
│   │   │   ├── SKILL.md
│   │   │   ├── examples/
│   │   │   │   ├── crud-handler.go
│   │   │   │   ├── pagination-example.go
│   │   │   │   └── error-handling.go
│   │   │   └── resources/
│   │   │       └── response-patterns.md
│   │   │
│   │   ├── testing-strategy/ # 测试策略和模式
│   │   │   ├── SKILL.md
│   │   │   ├── scripts/
│   │   │   │   └── run-test-suite.sh
│   │   │   └── examples/
│   │   │       ├── unit-test-template.go
│   │   │       └── integration-test-template.go
│   │   │
│   │   ├── deployment/      # 部署工作流
│   │   │   ├── SKILL.md
│   │   │   ├── scripts/
│   │   │   │   ├── pre-deploy-check.sh
│   │   │   │   └── health-check.sh
│   │   │   └── examples/
│   │   │       └── deployment-checklist.md
│   │   │
│   │   ├── wire-di/         # Wire 依赖注入指南
│   │   │   ├── SKILL.md
│   │   │   ├── scripts/
│   │   │   │   └── wire-gen-check.sh
│   │   │   └── examples/
│   │   │       └── provider-pattern.go
│   │   │
│   │   ├── database-migration/ # 数据库迁移最佳实践
│   │   │   ├── SKILL.md
│   │   │   ├── scripts/
│   │   │   │   └── migration-validator.sh
│   │   │   └── examples/
│   │   │       └── migration-template.go
│   │   │
│   │   └── swagger-docs/    # Swagger 文档生成
│   │       ├── SKILL.md
│   │       ├── scripts/
│   │       │   └── swagger-gen.sh
│   │       └── examples/
│   │           └── annotation-examples.go
│   │
│   └── workflows/           # 现有工作流目录（保持不变）
│       └── ...
│
└── docs/
    └── architecture/
        ├── claude-skill-strategy-analysis.md  # 已创建
        └── skill-integration-proposal.md       # 本文件
```

## 📝 SKILL.md 标准格式

```markdown
---
name: module-creation
description: Complete workflow for creating a new DDD module following ZGO's 8-file standard
version: 1.0.0
category: development
tags: [module, ddd, scaffolding]
author: ZGO Team
updated: 2026-01-24
---

# Module Creation Skill

## 📋 Purpose

This skill guides you through creating a standardized DDD module in the ZGO framework,
ensuring adherence to the 8-file structure and best practices.

## 🎯 When to Use

- Creating a new business module (e.g., User, Blog, Product)
- Need to follow ZGO's DDD layered architecture
- Want to ensure all necessary files and patterns are included

## ⚙️ Prerequisites

- [ ] ZGO project environment set up
- [ ] Go 1.21+ installed
- [ ] Wire tool available (`go install github.com/google/wire/cmd/wire@latest`)
- [ ] Understanding of DDD concepts

## 🚀 Workflow Steps

### Step 1: Define Module Scope

Before creating any files, clarify:
- **Module name**: Use PascalCase (e.g., `BlogPost`, `UserProfile`)
- **Domain entities**: What core business objects are involved?
- **API endpoints**: What operations are needed? (CRUD, custom actions)
- **Database tables**: What tables will be created?

**Example**:
```
Module: Blog
Domain: BlogPost
Table: blog_posts
Endpoints: GET /api/blogs, POST /api/blogs, GET /api/blogs/:id, etc.
```

### Step 2: Create Directory Structure

```bash
# Create module directory
mkdir -p internal/modules/blog
cd internal/modules/blog
```

### Step 3: Create 8 Standard Files

#### 3.1 model.go - Database Entity

```go
package blog

import (
    "time"
    "gorm.io/gorm"
)

// BlogPostPO is the persistent object for blog posts
type BlogPostPO struct {
    ID        uint           `gorm:"primaryKey"`
    Title     string         `gorm:"size:255;not null"`
    Content   string         `gorm:"type:text"`
    AuthorID  uint           `gorm:"index;not null"`
    Status    string         `gorm:"size:20;default:'draft'"` // draft, published
    CreatedAt time.Time
    UpdatedAt time.Time
    DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (BlogPostPO) TableName() string {
    return "blog_posts"
}
```

#### 3.2 dto.go - DTOs and Mappers

[... 完整示例代码 ...]

### Step 4: Wire Dependency Injection

[... 详细步骤 ...]

### Step 5: Register Routes

[... 详细步骤 ...]

### Step 6: Create Database Migration

[... 详细步骤 ...]

### Step 7: Validation and Testing

**Run validation script**:
```bash
# From skill directory
./scripts/validate-module.sh blog
```

**Manual checks**:
- [ ] All 8 files created
- [ ] Wire generation successful
- [ ] Routes registered
- [ ] Migration applied
- [ ] Tests passing

## 🔍 Troubleshooting

### Common Error: Wire Generation Fails

**Symptom**: `wire: generate failed`

**Solution**:
1. Check `provider.go` ProviderSet includes all constructors
2. Ensure interfaces are correctly bound
3. Run `cd internal/wiring && wire` with verbose output

[... 更多故障排除 ...]

## 📚 Examples

See `examples/user-module-example.md` for a complete walkthrough of creating the User module.

## 🔗 Related Skills

- `api-development`: For handler patterns
- `testing-strategy`: For writing tests
- `wire-di`: For advanced DI scenarios

## 📖 References

- [ZGO AGENTS.md](../../AGENTS.md)
- [DDD Layered Architecture](../resources/ddd-guide.md)
- [Wire User Guide](https://github.com/google/wire)
```

## 🎨 核心 Skills 规划

### Phase 1: 基础 Skills (立即实施)

| Skill | 优先级 | 说明 |
|-------|-------|------|
| `module-creation` | P0 | 创建新模块的完整流程 |
| `api-development` | P0 | API 开发模式和最佳实践 |
| `wire-di` | P0 | 依赖注入配置 |
| `testing-strategy` | P1 | 测试编写指南 |
| `deployment` | P1 | 部署检查清单 |

### Phase 2: 进阶 Skills (按需添加)

| Skill | 说明 |
|-------|------|
| `database-optimization` | 数据库性能优化 |
| `swagger-docs` | API 文档生成 |
| `error-handling` | 错误处理模式 |
| `middleware-development` | 中间件开发 |
| `capability-creation` | 创建技术能力包 |

## 🔄 Skills 加载机制

### Level 1: 启动时元数据加载

AI Agent 在启动时扫描 `.agent/skills/` 目录：

```python
# 伪代码
skill_registry = {}
for skill_dir in scan('.agent/skills/'):
    metadata = parse_yaml_frontmatter(f"{skill_dir}/SKILL.md")
    skill_registry[metadata['name']] = {
        'description': metadata['description'],
        'category': metadata['category'],
        'tags': metadata['tags'],
        'path': skill_dir,
        'loaded': False  # 内容未加载
    }

# 将元数据注入 System Prompt
system_prompt += f"""
Available Skills:
{format_skills_list(skill_registry)}
"""
```

**元数据示例输出**:
```
Available Skills:
- module-creation: Complete workflow for creating a new DDD module
- api-development: Best practices for API development in ZGO
- wire-di: Dependency injection configuration guide
- testing-strategy: Testing patterns and strategies
- deployment: Deployment workflow and checklist
```

### Level 2: 按需加载详细内容

当 AI Agent 判断需要使用某个 skill 时：

```python
def handle_request(user_request):
    # 1. 意图分析
    if "创建" in user_request and "模块" in user_request:
        skill = skill_registry['module-creation']
        
        # 2. Level 2: 加载 SKILL.md 详细内容
        if not skill['loaded']:
            content = read_file(f"{skill['path']}/SKILL.md")
            skill['content'] = parse_markdown(content)
            skill['loaded'] = True
        
        # 3. 使用 skill 指导工作
        return execute_with_skill(skill, user_request)
```

### Level 3: 按需加载资源

```python
def execute_with_skill(skill, request):
    # 如果需要脚本
    if skill.needs_validation():
        load_script(f"{skill['path']}/scripts/validate-module.sh")
    
    # 如果需要示例
    if user_asks_for_example():
        load_example(f"{skill['path']}/examples/user-module-example.md")
```

## 🛠️ 实施步骤

### Step 1: 创建基础设施

```bash
# 创建 skills 目录结构
mkdir -p .agent/skills/_template/{scripts,examples,resources}

# 创建模板文件
cat > .agent/skills/_template/SKILL.md << 'EOF'
---
name: template-skill
description: Template for creating new skills
version: 1.0.0
category: meta
tags: [template]
---

# Skill Template

## Purpose
[Describe what this skill does]

## When to Use
[When should this skill be invoked]

## Prerequisites
- [ ] Requirement 1
- [ ] Requirement 2

## Workflow Steps

### Step 1: [Title]
[Detailed instructions]

## Troubleshooting
[Common issues and solutions]

## Examples
[Provide examples]

## References
[Related documentation]
EOF

# 创建 README
cat > .agent/skills/README.md << 'EOF'
# ZGO Skills Library

## What are Skills?

Skills are modular, reusable workflows and best practices that guide AI agents
in performing specific tasks within the ZGO project.

## Directory Structure

Each skill is a self-contained folder with:
- `SKILL.md`: Main instruction file (required)
- `scripts/`: Helper scripts (optional)
- `examples/`: Code examples (optional)
- `resources/`: Additional resources (optional)

## Available Skills

See individual skill directories for detailed documentation.

## Creating a New Skill

1. Copy the `_template` directory
2. Rename to your skill name (kebab-case)
3. Update SKILL.md with your content
4. Add scripts and examples as needed
5. Test with AI agent

## Skill Naming Conventions

- Use kebab-case: `module-creation`, `api-development`
- Be descriptive but concise
- Avoid abbreviations unless widely known
EOF
```

### Step 2: 创建第一个核心 Skill

```bash
# 创建 module-creation skill
mkdir -p .agent/skills/module-creation/{scripts,examples}

# 创建主文件（将使用上面详细的模板）
# ... (这里会创建完整的 SKILL.md)
```

### Step 3: 更新 AGENTS.md

在 `AGENTS.md` 中添加 Skills 使用说明：

```markdown
## Using Skills

This project includes AI Agent Skills in `.agent/skills/` directory.

### Available Skills

- `module-creation`: Complete workflow for creating new modules
- `api-development`: API best practices and patterns
- `wire-di`: Dependency injection configuration

### How Skills Work

1. **Metadata Loading**: AI loads skill descriptions at startup
2. **Dynamic Content**: Detailed instructions loaded only when needed
3. **Progressive Resources**: Scripts and examples loaded on demand

### When to Use Skills

AI agents should:
- Check available skills before starting complex tasks
- Load relevant skills for guidance
- Follow skill workflows for consistency
- Update skills when better patterns are discovered
```

### Step 4: 集成到 AI Agent 工作流

在项目根目录的 `.agent/` 下创建配置文件：

```yaml
# .agent/config.yml
skills:
  enabled: true
  directory: .agent/skills
  auto_load_metadata: true
  lazy_load_content: true
  
  # Skill 优先级
  priorities:
    module-creation: high
    api-development: high
    wire-di: high
    testing-strategy: medium
    deployment: medium
  
  # Skill 路由规则
  routing:
    keywords:
      - keywords: [创建, 新建, 模块, module, create]
        skill: module-creation
      - keywords: [API, handler, endpoint, 接口]
        skill: api-development
      - keywords: [测试, test, 单元测试]
        skill: testing-strategy
      - keywords: [部署, deploy, 发布]
        skill: deployment
```

## 📊 Skills 与 Workflows 的区别

| 维度 | Skills | Workflows |
|------|--------|-----------|
| **位置** | `.agent/skills/` | `.agent/workflows/` |
| **格式** | YAML + Markdown + 脚本 | 纯 Markdown |
| **加载方式** | 渐进式 (3层) | 一次性加载 |
| **内容** | 完整指导 + 最佳实践 + 示例 | 具体步骤 |
| **复杂度** | 高 (包含脚本和资源) | 低 (纯文本) |
| **适用场景** | 通用工作流、跨项目复用 | 项目特定任务 |
| **例子** | "如何创建 DDD 模块" | "部署到生产环境的具体步骤" |

**关系**: Skills 提供**通用方法论**，Workflows 提供**具体行动步骤**。

## 🎯 Skills 使用示例

### 场景 1: 用户要求创建新模块

```
User: "帮我创建一个 Blog 模块，包含 CRUD 操作"

AI Process:
1. [Level 1] 检查元数据，发现 `module-creation` skill 相关
2. [Level 2] 加载 `module-creation/SKILL.md` 详细内容
3. [执行] 按照 skill 中的 8-file 标准创建模块
4. [Level 3] 如需验证，加载 scripts/validate-module.sh
5. [完成] 输出创建的文件清单 + 验证结果

AI Response:
"我将使用 module-creation skill 为您创建 Blog 模块，遵循 ZGO 的 8-file DDD 标准。

正在创建以下文件：
✓ model.go - BlogPostPO 实体
✓ dto.go - DTOs 和转换函数
✓ repository.go - 数据访问层
✓ service.go - 业务逻辑层
✓ handler.go - HTTP 处理器
✓ routes.go - 路由注册
✓ provider.go - Wire DI
✓ service_test.go - 单元测试

正在运行验证...
✓ Wire 生成成功
✓ 所有文件符合规范
✓ 路由注册完成

下一步建议：
1. 创建数据库迁移 (使用 database-migration skill)
2. 编写集成测试 (使用 testing-strategy skill)
3. 生成 Swagger 文档 (使用 swagger-docs skill)
"
```

### 场景 2: 用户询问 API 最佳实践

```
User: "ZGO 项目中如何正确处理分页？"

AI Process:
1. [Level 1] 识别关键词 "API"、"分页"
2. [Level 2] 加载 `api-development/SKILL.md`
3. [Level 3] 加载 examples/pagination-example.go
4. [执行] 提取相关部分回答

AI Response:
"根据 api-development skill，ZGO 有两种分页方式：

**方式1: Handler 直接查询 (最简单，AI推荐)**
[显示示例代码]

**方式2: 通过 Service 层 (DDD标准)**
[显示示例代码]

详细示例请查看 .agent/skills/api-development/examples/pagination-example.go
"
```

## 🔐 Skill 安全和权限

### 脚本执行权限

```yaml
# .agent/skills-security.yml
scripts:
  # 允许自动执行的脚本（只读操作）
  auto_run:
    - validate-module.sh
    - health-check.sh
    - wire-gen-check.sh
  
  # 需要用户确认的脚本（写操作）
  require_approval:
    - migration-validator.sh  # 涉及数据库
    - pre-deploy-check.sh     # 涉及部署
  
  # 禁止执行的脚本
  blocked:
    - rm -rf *
    - DROP TABLE
```

## 📈 Skill 效果评估

### 成功指标

1. **使用频率**: 统计每个 skill 被调用的次数
2. **任务完成率**: 使用 skill 后任务一次性完成的比例
3. **代码质量**: 使用 skill 生成代码的 lint 通过率
4. **时间节省**: 对比使用前后完成相同任务的时间

### 优化迭代

```markdown
# Skill Update Log

## module-creation v1.1.0 (2026-02-01)
- Added: Migration file auto-generation
- Fixed: Wire provider binding pattern
- Improved: Validation script now checks all 8 files

## api-development v1.0.1 (2026-01-28)
- Added: Error handling examples
- Improved: Pagination documentation
```

## 🚀 下一步行动

### 立即执行 (本周)

1. [ ] 创建 `.agent/skills/` 基础目录结构
2. [ ] 创建 `_template` 模板
3. [ ] 实现第一个 skill: `module-creation`
4. [ ] 更新 `AGENTS.md` 添加 Skills 说明
5. [ ] 测试 AI Agent 加载和使用 skill

### 短期目标 (本月)

1. [ ] 完成 5 个核心 skills (P0-P1)
2. [ ] 为每个 skill 创建示例和脚本
3. [ ] 编写 Skills 使用文档
4. [ ] 收集使用反馈并优化

### 长期目标 (季度)

1. [ ] 建立 Skill 贡献流程
2. [ ] 实现 Skill 版本管理
3. [ ] 跨项目 Skills 库
4. [ ] Skill 效果分析仪表板

## 🎓 总结

通过实施 Skill 系统，ZGO 项目将获得：

1. **标准化开发流程**: 所有模块遵循统一模式
2. **知识沉淀**: 最佳实践固化为可复用资源
3. **提升效率**: AI Agent 更智能，开发速度更快
4. **降低成本**: 减少上下文 token 使用
5. **易于维护**: 模块化设计便于更新和扩展

这是一个**渐进式架构**，可以从最核心的 1-2 个 skills 开始，逐步扩展到完整的知识库系统。
