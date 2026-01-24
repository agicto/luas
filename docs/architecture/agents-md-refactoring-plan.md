# AGENTS.md Content Analysis - What to Move to Skills

## 📊 Current AGENTS.md Structure Analysis

### ✅ Keep in AGENTS.md (Quick Reference)

These sections are perfect for AGENTS.md - concise, frequently referenced:

1. **Project Overview** (Lines 5-8) ✅
   - Brief description
   - **Keep**: Essential context

2. **AGENTS.md vs Skills - Positioning** (Lines 9-52) ✅
   - Explains the system
   - **Keep**: Meta-information

3. **AI Agent Skills List** (Lines 55-76) ✅
   - Table of available skills
   - **Keep**: Navigation/discovery

4. **Directory Structure** (Lines 101-127) ✅
   - Project layout
   - **Keep**: Quick reference for structure

5. **Common Commands** (Lines 129-137) ✅
   - Make targets
   - **Keep**: Daily use commands

6. **Module Structure Table** (Lines 139-149) ✅
   - 8-file responsibilities
   - **Keep**: Quick lookup

---

### 🔄 Should Move to Skills

These sections have detailed content that belongs in skills:

#### 1. **Capabilities Layer** (Lines 151-189)
**Current**: 38 lines explaining capabilities layer
**Move to**: New skill `capabilities-guide` OR expand `module-creation`
**Reason**: Detailed usage patterns, guidelines
**Keep in AGENTS.md**: 2-3 line summary + link to skill

```markdown
## Capabilities Layer

Reusable technical capabilities in `internal/capabilities/` (idgen, crypto, etc.)

📚 **Full Details**: See [`.agent/skills/capabilities-guide/`](./.agent/skills/capabilities-guide/SKILL.md)

Quick usage:
\`\`\`go
id := idgen.UUID()
hash, _ := crypto.HashPassword("password")
\`\`\`
```

---

#### 2. **Domain Layer** (Lines 190-206)
**Current**: Detailed explanation with examples
**Move to**: Expand `module-creation` skill
**Reason**: Part of DDD module design
**Keep in AGENTS.md**: Brief mention

```markdown
## Domain Layer

Core business entities in `internal/domain/` with JSON tags.

📚 **Details**: See [module-creation skill](./.agent/skills/module-creation/SKILL.md#domain-entities)
```

---

#### 3. **Handler Utilities** (Lines 207-233)
**Current**: 26 lines of handler package examples
**Move to**: `api-development` skill (already created!)
**Reason**: Part of API development
**Action**: Already covered in `api-development` skill
**Keep in AGENTS.md**: 3-4 line summary

```markdown
## Handler Utilities

\`\`\`go
id, ok := handler.Parse ID(c, "id")         // Parse params
userID, ok := handler.GetUserID(c)        // Get auth user
if !handler.BindJSON(c, &req) { return }  // Bind + validate
\`\`\`

📚 **Full API**: See [`api-development` skill](./.agent/skills/api-development/)
```

---

#### 4. **Unified Response** (Lines 235-266)
**Current**: 31 lines of response package examples
**Move to**: `api-development` skill (already partially there)
**Reason**: Part of API standards
**Action**: Already in `api-development` Section 2-3
**Keep in AGENTS.md**: 5 line quick reference

```markdown
## Unified Response

\`\`\`go
response.Success(c, data)              // 200 OK
response.Created(c, data)              // 201 Created
response.NotFound(c, "msg", err)       // 404
response.HandleError(c, "msg", err)    // Auto-map
\`\`\`

📚 **Full Guide**: See [`api-development` skill](./.agent/skills/api-development/)
```

---

#### 5. **Pagination** (Lines 268-327)
**Current**: 59 lines of detailed pagination examples
**Move to**: `api-development` skill (already there!)
**Reason**: Core API standard
**Action**: Already in `api-development` Section 1
**Keep in AGENTS.md**: 5-8 lines

```markdown
## Pagination

\`\`\`go
// One-liner (recommended)
users, paginator, _ := pagination.PaginateFromContext[*domain.User](c, h.db)
response.Success(c, paginator)  // Auto-includes meta + links
\`\`\`

📚 **Full Guide**: See [`api-development` skill](./.agent/skills/api-development/SKILL.md#pagination)
```

---

#### 6. **Complete Handler Examples** (Lines 329-401)
**Current**: 72 lines of full handler code
**Move to**: `api-development` skill
**Reason**: Complete examples belong in skills
**Action**: Similar examples already in `api-development/examples/complete-crud-handler.go`
**Keep in AGENTS.md**: Reference to skill

```markdown
## Complete Examples

See [`.agent/skills/api-development/examples/complete-crud-handler.go`](./.agent/skills/api-development/examples/complete-crud-handler.go)
```

---

#### 7. **Wire Dependency Injection** (Lines 403-426)
**Current**: 23 lines of Wire examples
**Move to**: New skill `wire-di` OR expand `module-creation`
**Reason**: Specialized topic deserving deep dive
**Recommendation**: Add to `module-creation` skill Section on provider.go
**Keep in AGENTS.md**: 3-4 lines

```markdown
## Wire Dependency Injection

\`\`\`go
var ProviderSet = wire.NewSet(
    NewRepository,
    wire.Bind(new(Repository), new(*repository)),
)
\`\`\`

Run: `cd internal/wiring && wire`

📚 **Full Guide**: See [module-creation skill - Wire DI section]
```

---

### 📋 Coding Standards (Lines 428-1240)

**Current Status**: ~800 lines in AGENTS.md
**Already Moved**: Section 7 (API Development Standards) → `api-development` skill ✅

**Remaining Content**:
- Section 1-6: Naming, Architecture, File Organization, Error Handling, Security, Testing
- These ARE in AGENTS.md correctly as "quick reference"
- BUT: Can be EXPANDED in `coding-standards` skill

**Recommendation**:
- **Keep Sections 1-6 in AGENTS.md** (200-300 lines) - These are core standards
- **Expand `coding-standards` skill** with:
  - Advanced error handling patterns (Circuit Breaker, Retry, Timeout)
  - Error aggregation
  - Graceful degradation
  - Error monitoring integration

---

## 🎯 Recommended Actions

### Immediate (This Session):

1. **✅ Simplify AGENTS.md** - Remove verbose examples, keep quick reference
   - Pagination: 59 lines → 8 lines
   - Handler Utils: 26 lines → 8 lines
   - Unified Response: 31 lines → 8 lines
   - Complete Examples: 72 lines → 2 lines (link)
   - Wire DI: 23 lines → 6 lines
   - **Total Reduction**: ~180 lines

2. **✅ Expand `coding-standards` skill**:
   - Add Section: Advanced Error Handling Patterns
     - Circuit Breaker Pattern
     - Retry with Exponential Backoff
     - Timeout Patterns
     - Error Aggregation
     - Graceful Degradation
   - Add validation script section
   - Add complete examples

### Future (Next Phase):

3. **Consider `wire-di` skill**:
   - Advanced Wire patterns
   - Provider organization
   - Testing with DI
   - Troubleshooting

4. **Consider `capabilities-guide` skill**:
   - How to create new capabilities
   - Capability patterns
   - Testing capabilities

---

## 📐 Size Goals

| Section | Current (AGENTS.md) | Target (AGENTS.md) | Moved To |
|---------|---------------------|--------------------| ---------|
| Handler Utils | 26 lines | 8 lines | api-development |
| Unified Response | 31 lines | 8 lines | api-development |
| Pagination | 59 lines | 8 lines | api-development |
| Complete Examples | 72 lines | 2 lines | api-development/examples |
| Wire DI | 23 lines | 6 lines | module-creation (future) |
| **Total Reduction** | **~180 lines** | | |

**Target AGENTS.md Size**: ~1000 lines (from current 1264)

---

## ✅ Implementation Plan

### Step 1: Simplify AGENTS.md (10 min)
```markdown
1. Remove verbose pagination examples → keep 1 example
2. Remove verbose handler util examples → keep 2-3 examples
3. Remove verbose response examples → keep core functions
4. Remove complete handler code → add link to skill
5. Add "📚 Full Details: See skill..." references
```

### Step 2: Expand coding-standards Skill (30 min)
```markdown
1. Add "Advanced Error Handling Patterns" section
2. Create error-handling-patterns.go example
3. Add Circuit Breaker implementation
4. Add Retry with backoff
5. Add Error aggregation pattern
6. Update validation script
```

### Step 3: Update Module Creation Skill (future)
```markdown
1. Add Wire DI deep dive
2. Add provider.go patterns
3. Add DI troubleshooting
```

---

**Ready to proceed?**
- ✅ Simplify AGENTS.md
- ✅ Expand coding-standards with advanced error handling
