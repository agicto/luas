# ZGO 代码规范与 Skills 系统 - 完整方案

## 📌 问题解答

### Q: 有了 Skills，AGENTS.md 是否还有用？

**A: 两者定位不同，互补而非替代！**

### 📖 AGENTS.md - 快速参考手册

**定位**: 一站式快速查询，日常开发必备

**内容**:
- ✅ 项目结构和常用命令
- ✅ **代码规范（新增!）**
- ✅ 快速示例
- ✅ 工具使用指南

**使用场景**:
```
开发者: "JSON 字段用什么格式？"
      → 打开 AGENTS.md → 快速找到 "snake_case"

开发者: "Handler 怎么解析参数？"
      → AGENTS.md → handler.ParseID() 示例
```

**特点**: 简洁、快速、一页式

---

### 🎯 Skills - 深度工作流指南

**定位**: 完整任务执行指南，从入门到精通

**内容**:
- ✅ 15+ 步完整流程
- ✅ 所有代码示例
- ✅ 自动化验证脚本
- ✅ 故障排除

**使用场景**:
```
AI Agent: "创建一个 Blog 模块"
        → 加载 module-creation skill
        → 执行 15 步完整工作流
        → 运行验证脚本

开发者: "如何确保代码符合规范？"
      → 阅读 coding-standards skill
      → 运行 verify-standards.sh
```

**特点**: 详细、完整、可执行

---

### 关系图示

```
                    ZGO 开发工作流
                          │
          ┌───────────────┼───────────────┐
          │                               │
    快速查询/参考                    完整任务执行
          │                               │
          ▼                               ▼
    ┌─────────────┐               ┌─────────────┐
    │ AGENTS.md   │               │   Skills    │
    │             │               │   System    │
    ├─────────────┤               ├─────────────┤
    │• 命令手册    │               │• 工作流      │
    │• 代码规范    │◄─────关联────►│• 脚本       │
    │• 快速示例    │               │• 深度指南    │
    │• 工具用法    │               │• 故障排除    │
    └─────────────┘               └─────────────┘
          │                               │
          │                               │
    适用: 日常开发                  适用: 复杂任务
    例子: 查命令                    例子: 创建模块
          查规范                          代码审查
          查 API                          学习最佳实践
```

---

## ✅ 已完成的工作

### 1. AGENTS.md 增强

#### 新增章节: 📋 代码规范（必须遵守）

包含 6 大类规范：

1. **命名规范** (完整)
   - 包名、文件名、类型名
   - 函数名、变量名、JSON 标签
   - 每种都有 ✅/❌ 对比示例

2. **架构规范** (完整)
   - 8-file 模块结构（强制）
   - 分层架构（禁止跨层访问）
   - 数据流转规范

3. **文件组织规范** (完整)
   - 每个文件的详细规范
   - model.go、dto.go、repository.go、service.go、handler.go、routes.go、provider.go
   - 包含完整代码示例

4. **错误处理规范**
   - 自定义错误定义
   - 错误包装和上下文
   - 错误处理最佳实践

5. **安全规范**
   - 敏感数据保护
   - 输入验证
   - 密码处理

6. **测试规范**
   - Mock 使用
   - 测试组织
   - 覆盖率要求

**字数**: 新增约 800 行详细规范

#### 新增章节: 定位说明

明确说明了 AGENTS.md vs Skills 的关系和使用场景。

---

### 2. 新增 coding-standards Skill

**位置**: `.agent/skills/coding-standards/`

**内容**:

#### SKILL.md (完整工作流)
- 8 个验证级别
- 每级别有自动化检查命令
- 完整的检查清单
- 快速验证脚本

#### 验证脚本 (verify-standards.sh)
```bash
# 自动化检查8个级别:
✓ Level 1: 文件结构
✓ Level 2: 命名约定
✓ Level 3: 架构合规
✓ Level 4: 必需内容
✓ Level 5: 安全检查
✓ Level 6: Wire DI
✓ Level 7: 测试
✓ Level 8: 代码质量
```

**使用**:
```bash
.agent/skills/coding-standards/scripts/verify-standards.sh user
```

---

## 📊 对比：AGENTS.md vs Skills

| 维度 | AGENTS.md | Skills |
|------|-----------|--------|
| **位置** | 根目录 | `.agent/skills/` |
| **长度** | 适中 (~1,200 行) | 详细 (每个 skill 500-1000 行) |
| **内容** | 规范 + 快速示例 | 完整流程 + 脚本 + 示例 |
| **更新频率** | 稳定 | 持续迭代 |
| **查询方式** | Ctrl+F 查找 | 加载相关 skill |
| **典型问题** | "这个怎么写？" | "怎么完成这个任务？" |
| **学习曲线** | 平缓 | 渐进 |

---

## 🎯 实际使用场景

### 场景 1: 开发者日常开发

```
开发者正在写代码...

Question: "Update DTO 的字段应该用什么类型？"

Action:
1. 打开 AGENTS.md
2. Ctrl+F 搜索 "Update DTO"
3. 找到示例:
   type UpdateUserRequest struct {
       Username *string  // ← 指针表示可选
   }
4. 30秒内解决

Cost: 低
Speed: 快
```

---

### 场景 2: AI Agent 创建新模块

```
User: "创建一个 Product 模块"

AI Process:
1. 识别任务类型 → 模块创建
2. 匹配 skill → module-creation
3. 加载完整工作流 (15 步)
4. 逐步执行:
   - 创建 8 个文件
   - 每个文件使用 AGENTS.md 中的规范
   - 运行验证脚本
5. 完成并报告

Cost: 中
Completeness: 100%
```

---

### 场景 3: Code Review / PR 提交前

```
开发者准备提交 PR...

Action:
1. 运行验证脚本:
   .agent/skills/coding-standards/scripts/verify-standards.sh blog

2. 查看报告:
   ✅ 文件结构完整
   ✅ 命名符合规范
   ❌ 发现 camelCase JSON tag
   ✅ 架构合规
   ⚠️  缺少一些单元测试

3. 修复问题

4. 重新验证

5. 提交 PR

Cost: 低（自动化）
Quality: 高
```

---

## 💡 最佳实践建议

### 对于开发者

1. **日常开发**: 
   - 📖 AGENTS.md 作为浏览器书签
   - 🔍 快速查询命令和规范

2. **创建模块**: 
   - 🎯 先阅读 module-creation skill
   - 📝 按步骤执行
   - ✅ 运行验证脚本

3. **代码审查**:
   - 🔒 使用 coding-standards skill
   - 🧪 运行 verify-standards.sh
   - 📋 对照检查清单

### 对于 AI Agent

1. **快速查询**:
   - 从 AGENTS.md 获取规范和示例
   - 直接应用到代码生成

2. **复杂任务**:
   - 加载相关 skill
   - 按完整工作流执行
   - 调用自动化脚本

3. **质量保证**:
   - 创建代码后自动运行验证
   - 参考 AGENTS.md 修正问题

---

## 📈 未来扩展

### Phase 1: 已完成 ✅

- [x] AGENTS.md 增加代码规范
- [x] 明确 AGENTS.md vs Skills 定位
- [x] 创建 module-creation skill
- [x] 创建 coding-standards skill

### Phase 2: 计划中

- [ ] api-development skill (API 开发模式)
- [ ] testing-strategy skill (测试策略)
- [ ] wire-di skill (DI 高级用法)
- [ ] 更多验证脚本

### Phase 3: 长期

- [ ] 集成到 CI/CD
- [ ] 自动化 PR 检查
- [ ] 代码规范分析报告
- [ ] 团队知识库持续扩充

---

## 🎓 总结

### AGENTS.md 的价值

✅ **保留并加强**：
- 作为快速参考手册，无可替代
- 新增的代码规范章节是**强制遵守的标准**
- 开发者日常必备

### Skills 的价值

✅ **补充并深化**：
- 提供完整的工作流指导
- 包含自动化工具和脚本
- AI Agent 的执行指南
- 新人学习的完整路径

### 两者关系

```
AGENTS.md (规范) + Skills (执行) = 完整的开发体系

AGENTS.md: "是什么" "怎么写"
Skills:    "怎么做" "如何验证"
```

---

## 🚀 立即行动

### 开发者

1. **熟悉 AGENTS.md**:
   ```bash
   # 阅读代码规范章节
   cat AGENTS.md | grep -A 100 "代码规范"
   ```

2. **运行验证**:
   ```bash
   # 验证现有模块
   .agent/skills/coding-standards/scripts/verify-standards.sh user
   ```

3. **创建新模块时**:
   ```bash
   # 参考 module-creation skill
   cat .agent/skills/module-creation/SKILL.md
   ```

### AI Agent

1. **启动时加载 AGENTS.md**
   - 获取核心规范和工具

2. **识别任务类型**
   - 查询 → AGENTS.md
   - 创建 → module-creation skill
   - 审查 → coding-standards skill

3. **执行和验证**
   - 按 skill 工作流执行
   - 运行自动化脚本
   - 符合 AGENTS.md 规范

---

**Status**: ✅ 完成  
**Impact**: 显著提升代码质量和开发效率  
**Ready**: 立即可用

🎉 **ZGO 现在拥有完整的代码规范和质量保障体系！**
