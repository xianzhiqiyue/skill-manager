# Skill Home 技能发布分类与标签强制化实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 Skill Home 建立强制性的 `category + official tags` 发布约束，并让 CLI、Web、server 与 `skill-home-manager` 都围绕同一套 taxonomy 工作。

**Architecture:** 采用“单一 taxonomy 源文件 + 生成脚本 + 各端本地生成产物”的结构。顶层 `config/skill-taxonomy.json` 作为唯一真相来源，脚本把它同步成 CLI/server 可嵌入的 JSON、副本和 `skill-home-manager` 可读的参考文档；CLI 和 Web 负责顺手补齐，server 负责硬校验，`skill-home-manager` 负责让 OpenClaw 在 `push` 前主动整理元数据。

**Tech Stack:** Go 1.21、Cobra、Viper、GORM、Gin、React 18、TypeScript、Vite、Vitest、Testing Library、仓库内 `skills/skill-home-manager`

---

## 文件结构映射

### 新增文件

- Create: `config/skill-taxonomy.json`
  - 仓库级 taxonomy 唯一来源，定义 `categories`、`official_tags`、`aliases` 与说明。
- Create: `scripts/generate_skill_taxonomy.py`
  - 从顶层 taxonomy 生成各端消费产物。
- Create: `skill-home-cli/internal/taxonomy/taxonomy.go`
  - CLI 端 taxonomy 加载与校验辅助。
- Create: `skill-home-cli/internal/taxonomy/taxonomy_test.go`
  - 覆盖 taxonomy 加载、合法性与别名归一化。
- Create: `skill-home-cli/internal/taxonomy/taxonomy.json`
  - 由生成脚本输出，供 CLI 内嵌。
- Create: `skill-home-server/internal/taxonomy/taxonomy.go`
  - server 端 taxonomy 加载与校验辅助。
- Create: `skill-home-server/internal/taxonomy/taxonomy_test.go`
  - 覆盖 server taxonomy 解析与别名逻辑。
- Create: `skill-home-server/internal/taxonomy/taxonomy.json`
  - 由生成脚本输出，供 server 内嵌。
- Create: `skill-home-web/src/generated/skillTaxonomy.json`
  - 由生成脚本输出，供 Web 表单渲染分类和标签选项。
- Create: `skill-home-web/src/pages/PublishNewPage.test.tsx`
  - 覆盖发布页 `category/tags` 必填交互。
- Create: `skill-home-web/src/pages/settings/SkillGeneralSettingsPage.test.tsx`
  - 覆盖 owner 编辑 `category/tags`。
- Create: `skills/skill-home-manager/references/publish-taxonomy.md`
  - 面向 OpenClaw/用户的官方 taxonomy 说明与示例。

### 需要修改的现有文件

- Modify: `skill-home-cli/internal/skill/parser.go`
- Modify: `skill-home-cli/internal/skill/parser_test.go`
  - manifest 新增 `Category` 字段，并解析 frontmatter。
- Modify: `skill-home-cli/internal/cmd/init.go`
- Modify: `skill-home-cli/internal/cmd/create.go`
- Modify: `skill-home-cli/internal/cmd/validate.go`
- Modify: `skill-home-cli/internal/cmd/push.go`
- Modify: `skill-home-cli/internal/cmd/push_test.go`
- Modify: `skill-home-cli/internal/cmd/registry_helpers.go`
- Modify: `skill-home-cli/internal/cmd/registry_test_helpers_test.go`
  - 让 CLI 模板、校验、推送与测试都理解 taxonomy。
- Modify: `skill-home-cli/internal/registry/types.go`
- Modify: `skill-home-cli/internal/registry/client.go`
- Modify: `skill-home-cli/internal/registry/client_test.go`
  - 发布请求增加 `category/tags` 字段并写入 multipart form。
- Modify: `skill-home-cli/internal/cmd/import_codex.go`
- Modify: `skill-home-cli/internal/cmd/import_cursor.go`
- Modify: `skill-home-cli/internal/import/github/importer.go`
  - 导入链路生成合法的分类骨架或受控默认值，避免立刻产出不合规 skill。
- Modify: `skill-home-cli/README.md`
  - 文档同步新规则和推送行为。
- Modify: `skill-home-server/internal/models/skill.go`
- Modify: `skill-home-server/internal/api/handlers/skill.go`
- Modify: `skill-home-server/internal/api/handlers/helpers.go`
- Modify: `skill-home-server/internal/api/handlers/handler_integration_test.go`
  - 增加 `category` 存储、接口校验与集成测试。
- Modify: `API.md`
  - 补充新字段和发布/更新接口要求。
- Modify: `skill-home-web/src/api.ts`
- Modify: `skill-home-web/src/hooks/useRegistryApp.ts`
- Modify: `skill-home-web/src/hooks/useRegistryApp.test.tsx`
- Modify: `skill-home-web/src/pages/PublishNewPage.tsx`
- Modify: `skill-home-web/src/pages/settings/SkillGeneralSettingsPage.tsx`
- Modify: `skill-home-web/src/App.test.tsx`
  - Web 端发布和管理表单升级为受控分类录入。
- Modify: `skills/skill-home-manager/SKILL.md`
- Modify: `skills/skill-home-manager/README.md`
- Modify: `skills/skill-home-manager/references/cli-workflows.md`
- Modify: `skills/skill-home-manager/scripts/create-local-skill.sh`
  - 让 OpenClaw 代理和 bundled script 都把分类元数据视为发布前必做步骤。

---

### Task 1: 固化 taxonomy 源文件与生成链路

**Files:**
- Create: `config/skill-taxonomy.json`
- Create: `scripts/generate_skill_taxonomy.py`
- Create: `skill-home-cli/internal/taxonomy/taxonomy.json`
- Create: `skill-home-server/internal/taxonomy/taxonomy.json`
- Create: `skill-home-web/src/generated/skillTaxonomy.json`
- Create: `skills/skill-home-manager/references/publish-taxonomy.md`

- [ ] **Step 1: 先写 taxonomy 结构与生成脚本的失败用例**
  - 为 CLI 和 server 新建 `taxonomy_test.go`，先断言以下事实：
    - 能加载 8 个一级分类
    - 能识别 `ci-cd`、`deployment` 等官方标签
    - `pipeline` 会被归一化到 `ci-cd`
  - 为生成脚本准备一个最小 smoke 命令，要求执行后 4 份产物都存在。

- [ ] **Step 2: 运行失败测试确认缺口**

Run:
- `cd skill-home-cli && go test ./internal/taxonomy`
- `cd skill-home-server && go test ./internal/taxonomy`

Expected: FAIL，因为 taxonomy 包与生成产物尚不存在。

- [ ] **Step 3: 写最小实现**
  - 新建 `config/skill-taxonomy.json`，至少包含：

```json
{
  "categories": [
    { "id": "development", "label": "Development", "description": "代码生成、重构、调试、工程辅助" },
    { "id": "testing", "label": "Testing", "description": "测试、验证、评审、质量保障" }
  ],
  "official_tags": [
    { "id": "ci-cd", "description": "CI/CD 与 pipeline 场景" },
    { "id": "deployment", "description": "部署与发布" }
  ],
  "aliases": {
    "ci": "ci-cd",
    "pipeline": "ci-cd",
    "deploy": "deployment"
  }
}
```

  - 新建 `scripts/generate_skill_taxonomy.py`，读取顶层 JSON 并输出：
    - `skill-home-cli/internal/taxonomy/taxonomy.json`
    - `skill-home-server/internal/taxonomy/taxonomy.json`
    - `skill-home-web/src/generated/skillTaxonomy.json`
    - `skills/skill-home-manager/references/publish-taxonomy.md`

- [ ] **Step 4: 运行生成与单元测试**

Run:
- `python3 scripts/generate_skill_taxonomy.py`
- `cd skill-home-cli && go test ./internal/taxonomy`
- `cd skill-home-server && go test ./internal/taxonomy`

Expected: PASS

- [ ] **Step 5: 自查**
  - 确认生成脚本可重复执行且结果稳定
  - 确认 skill 参考文档使用中文说明，但分类 id/tag id 保持英文原值

### Task 2: 让 CLI manifest、模板和校验理解 `category/tags`

**Files:**
- Create: `skill-home-cli/internal/taxonomy/taxonomy.go`
- Create: `skill-home-cli/internal/taxonomy/taxonomy_test.go`
- Modify: `skill-home-cli/internal/skill/parser.go`
- Modify: `skill-home-cli/internal/skill/parser_test.go`
- Modify: `skill-home-cli/internal/cmd/init.go`
- Modify: `skill-home-cli/internal/cmd/create.go`
- Modify: `skill-home-cli/internal/cmd/validate.go`

- [ ] **Step 1: 写失败测试**
  - 在 `parser_test.go` 中新增断言：frontmatter 中的 `category` 会被解析到 manifest。
  - 在 `validate.go` 附近新增测试文件或扩展现有测试，覆盖：
    - 缺少 `category` 时失败
    - `tags` 为空时失败
    - 非法 tag 与超出 4 个 tag 时失败
  - 在 `create`/`init` 对应测试中断言模板里包含：

```yaml
category: ""
tags:
  - ""
```

- [ ] **Step 2: 运行测试确认失败**

Run:
- `cd skill-home-cli && go test ./internal/skill`
- `cd skill-home-cli && go test ./internal/cmd -run 'Test.*(Validate|Init|Create|Category|Tags)'`

Expected: FAIL，因为 parser、模板和 validate 还不知道 `category`。

- [ ] **Step 3: 写最小实现**
  - 给 CLI manifest 增加字段：

```go
type Manifest struct {
    Name        string   `yaml:"name"`
    Version     string   `yaml:"version"`
    Description string   `yaml:"description"`
    Category    string   `yaml:"category,omitempty"`
    Tags        []string `yaml:"tags,omitempty"`
}
```

  - `init` 和 `create` 模板统一输出 `category` 和 `tags` 骨架。
  - `validate` 接入 taxonomy 包，执行：
    - `category` 必填且必须存在
    - `tags` 数量 1-4
    - tag 必须是受控值或可折叠别名

- [ ] **Step 4: 重新运行测试**

Run:
- `cd skill-home-cli && go test ./internal/skill`
- `cd skill-home-cli && go test ./internal/cmd -run 'Test.*(Validate|Init|Create|Category|Tags)'`

Expected: PASS

- [ ] **Step 5: 自查**
  - 确认 `skill-home init` 生成的新 skill 能立即看出“发布前必须补齐分类元数据”
  - 确认 `validate` 的报错文案会直接指向该改什么，而不是只说 “invalid input”

### Task 3: 改造 CLI `push`，让交互终端能补齐并真正上传分类元数据

**Files:**
- Modify: `skill-home-cli/internal/cmd/push.go`
- Modify: `skill-home-cli/internal/cmd/push_test.go`
- Modify: `skill-home-cli/internal/cmd/registry_helpers.go`
- Modify: `skill-home-cli/internal/cmd/registry_test_helpers_test.go`
- Modify: `skill-home-cli/internal/registry/types.go`
- Modify: `skill-home-cli/internal/registry/client.go`
- Modify: `skill-home-cli/internal/registry/client_test.go`

- [ ] **Step 1: 先补测试支架并写失败测试**
  - 让 `registryClient` 接口支持 `Publish`，让 `runPush` 能被 fake client 驱动。
  - 在 `push_test.go` 中新增：
    - `runPush` 会把 `category/tags` 带到 `PublishRequest`
    - 非交互模式下缺少 `category` 时直接失败
    - 非交互模式下非法 tag 时直接失败
  - 在 `client_test.go` 中新增断言：multipart form 包含 `category` 和逗号拼接后的 `tags`。

- [ ] **Step 2: 运行测试确认失败**

Run:
- `cd skill-home-cli && go test ./internal/registry -run 'Test.*Publish'`
- `cd skill-home-cli && go test ./internal/cmd -run 'Test.*Push'`

Expected: FAIL，因为请求结构、client form 和 push 流程都还没覆盖这些字段。

- [ ] **Step 3: 写最小实现**
  - `PublishRequest` 增加：

```go
type PublishRequest struct {
    Namespace   string   `json:"namespace,omitempty"`
    Name        string   `json:"name,omitempty"`
    Version     string   `json:"version,omitempty"`
    Description string   `json:"description,omitempty"`
    Category    string   `json:"category,omitempty"`
    Tags        []string `json:"tags,omitempty"`
    License     string   `json:"license,omitempty"`
}
```

  - `runPush` 在调用 `Publish` 前先执行 taxonomy 校验。
  - 如果 `stdin/stdout` 是交互终端且缺失字段，则进入轻量补齐向导，写回 `SKILL.md` 后继续。
  - `client.Publish()` 在 multipart form 中追加 `category`、`tags`。

- [ ] **Step 4: 重新运行测试**

Run:
- `cd skill-home-cli && go test ./internal/registry -run 'Test.*Publish'`
- `cd skill-home-cli && go test ./internal/cmd -run 'Test.*Push'`

Expected: PASS

- [ ] **Step 5: 自查**
  - 确认已有 skill 版本追加发布路径不会丢掉 `category/tags`
  - 确认 `push` 错误文案能区分“需补齐元数据”和“远端发布失败”

### Task 4: 收口导入链路，避免生成天然不合规的 skill

**Files:**
- Modify: `skill-home-cli/internal/cmd/import_codex.go`
- Modify: `skill-home-cli/internal/cmd/import_cursor.go`
- Modify: `skill-home-cli/internal/import/github/importer.go`
- Modify: `skill-home-cli/internal/cmd/import.go`

- [ ] **Step 1: 写失败测试**
  - 针对 Codex/Cursor/GitHub 导入结果新增断言：
    - frontmatter 至少包含 `category` 字段骨架
    - 默认 tags 不再使用 `imported` 这种受控词表外的值
    - 导入完成后运行 `skill-home validate` 不会因字段完全缺失而直接失败

- [ ] **Step 2: 运行测试确认失败**

Run:
- `cd skill-home-cli && go test ./internal/cmd -run 'Test.*Import'`
- `cd skill-home-cli && go test ./internal/import/...`

Expected: FAIL，因为当前导入模板仍输出 `tags: [imported, ...]` 且没有 `category`。

- [ ] **Step 3: 写最小实现**
  - 导入生成的 `SKILL.md` 改为输出：

```yaml
category: productivity
tags:
  - workflow
```

  - 如果导入源已经带有可用 tags，则通过 taxonomy 归一化后保留合法值。
  - 对明显不合法的导入 tag，转成注释提示而不是写入正式 `tags`。

- [ ] **Step 4: 重新运行测试**

Run:
- `cd skill-home-cli && go test ./internal/cmd -run 'Test.*Import'`
- `cd skill-home-cli && go test ./internal/import/...`

Expected: PASS

- [ ] **Step 5: 自查**
  - 确认导入体验不会因为 taxonomy 收紧而退化成“导入成功但立刻不可用”

### Task 5: server 端持久化 `category` 并做统一硬校验

**Files:**
- Create: `skill-home-server/internal/taxonomy/taxonomy.go`
- Create: `skill-home-server/internal/taxonomy/taxonomy_test.go`
- Modify: `skill-home-server/internal/models/skill.go`
- Modify: `skill-home-server/internal/api/handlers/helpers.go`
- Modify: `skill-home-server/internal/api/handlers/skill.go`
- Modify: `skill-home-server/internal/api/handlers/handler_integration_test.go`
- Modify: `API.md`

- [ ] **Step 1: 写失败测试**
  - 在 `handler_integration_test.go` 中新增：
    - 创建 skill 时缺少 `category` 返回 `400`
    - 创建 skill 时 `tags` 为空返回 `400`
    - 别名 tag 会被归一化后持久化
    - 更新 skill 时非法 `category` 或 tag 会被拒绝
  - 扩展现有 “persist official tags” 测试为同时断言 `category`。

- [ ] **Step 2: 运行测试确认失败**

Run:
- `cd skill-home-server && go test ./internal/api/handlers -run 'Test(CreateSkill|UpdateSkill).*'`

Expected: FAIL，因为模型和 handler 还没有 `category` 字段与校验。

- [ ] **Step 3: 写最小实现**
  - `Skill` 模型增加：

```go
Category string `gorm:"size:64;index" json:"category,omitempty"`
```

  - `CreateSkillRequest` / `UpdateSkillRequest` 增加 `Category string`.
  - 新增 `normalizeCategory` / `validateOfficialTags` 之类的帮助函数，统一使用 taxonomy 包。
  - 创建、更新时都执行同一套校验并返回明确 `INVALID_INPUT` 文案。

- [ ] **Step 4: 重新运行测试**

Run:
- `cd skill-home-server && go test ./internal/taxonomy`
- `cd skill-home-server && go test ./internal/api/handlers -run 'Test(CreateSkill|UpdateSkill).*'`

Expected: PASS

- [ ] **Step 5: 自查**
  - 确认 AutoMigrate 能在开发环境补齐新列
  - 在计划注释中记录生产环境暂不立刻加 `NOT NULL`，先依赖代码层约束和存量补录

### Task 6: 升级 Web 发布页与 owner 管理页

**Files:**
- Modify: `skill-home-web/src/api.ts`
- Modify: `skill-home-web/src/hooks/useRegistryApp.ts`
- Modify: `skill-home-web/src/hooks/useRegistryApp.test.tsx`
- Modify: `skill-home-web/src/pages/PublishNewPage.tsx`
- Create: `skill-home-web/src/pages/PublishNewPage.test.tsx`
- Modify: `skill-home-web/src/pages/settings/SkillGeneralSettingsPage.tsx`
- Create: `skill-home-web/src/pages/settings/SkillGeneralSettingsPage.test.tsx`
- Modify: `skill-home-web/src/App.test.tsx`
- Create: `skill-home-web/src/generated/skillTaxonomy.json`

- [ ] **Step 1: 写失败测试**
  - 在 `useRegistryApp.test.tsx` 中新增：
    - 发布时会提交 `category` 和受控 `tags`
    - 管理页保存时也会提交 `category`
  - 在页面级测试中新增：
    - 发布页未选 `category` 时不可提交
    - `tags` 选择数超过 4 时给出表单内错误
    - owner 管理页能编辑 `category/tags`

- [ ] **Step 2: 运行测试确认失败**

Run:
- `cd skill-home-web && npm test -- --run src/hooks/useRegistryApp.test.tsx`
- `cd skill-home-web && npm test -- --run src/pages/PublishNewPage.test.tsx src/pages/settings/SkillGeneralSettingsPage.test.tsx`

Expected: FAIL，因为 publish/manage form 还没有 `category`，`tags` 仍是自由文本。

- [ ] **Step 3: 写最小实现**
  - `SkillSummary` / `SkillDetail` / `PublishPayload` / `UpdateSkillPayload` 增加 `category`。
  - `useRegistryApp` 的 `publishForm`、`manageForm` 改为：

```ts
{
  category: '',
  tags: [] as string[]
}
```

  - 发布页和管理页改用：
    - `category` 下拉
    - `tags` 受控多选或 checkbox token 选择
  - 第一版列表和详情不强制展示 `category`，但前端类型先完整支持。

- [ ] **Step 4: 重新运行测试**

Run:
- `cd skill-home-web && npm test -- --run src/hooks/useRegistryApp.test.tsx`
- `cd skill-home-web && npm test -- --run src/pages/PublishNewPage.test.tsx src/pages/settings/SkillGeneralSettingsPage.test.tsx`

Expected: PASS

- [ ] **Step 5: 自查**
  - 确认未登录发布仍沿用现有登录回跳
  - 确认设置页能作为存量 skill 补齐分类信息的入口

### Task 7: 更新 `skill-home-manager`，让 OpenClaw 默认先补元数据再发布

**Files:**
- Modify: `skills/skill-home-manager/SKILL.md`
- Modify: `skills/skill-home-manager/README.md`
- Modify: `skills/skill-home-manager/references/cli-workflows.md`
- Modify: `skills/skill-home-manager/scripts/create-local-skill.sh`
- Create: `skills/skill-home-manager/references/publish-taxonomy.md`

- [ ] **Step 1: 写出要验证的行为清单**
  - `SKILL.md` 中默认工作流明确加入：
    - 发布前检查 `category/tags`
    - 缺失时先基于内容推荐并补齐
    - 只有补齐后才继续 `validate/pack/push`
  - `cli-workflows.md` 明确说明非交互 `push` 会因缺少分类元数据失败。

- [ ] **Step 2: 修改 skill 文档与脚本**
  - 在 [`skills/skill-home-manager/SKILL.md`](/mnt/d/code/soul-store/skill-manager/skills/skill-home-manager/SKILL.md) 中把“发布 skill”流程改写为：
    - 检查 `SKILL.md`
    - 参考 `references/publish-taxonomy.md`
    - 自动补建议值或在歧义时只问一个短问题
    - 再执行 `validate/pack/push`
  - `README.md` 和 `cli-workflows.md` 同步这一行为。
  - `create-local-skill.sh` 的 usage 与输出文案提示“创建后需补齐分类元数据”。

- [ ] **Step 3: 自查**
  - 确认 skill 文档默认面向 OpenClaw/Codex 的语气是“代理先帮做，再提最少问题”
  - 确认不把 taxonomy 规则藏在 README，而是放到 skill 可直接读取的 reference 文件里

### Task 8: 文档、回归与上线前验证

**Files:**
- Modify: `API.md`
- Modify: `skill-home-cli/README.md`
- Modify: `skills/skill-home-manager/README.md`
- Modify: `docs/superpowers/specs/2026-04-01-skill-publish-classification-design.md`
- Modify: `docs/superpowers/plans/2026-04-01-skill-publish-classification-plan.md`

- [ ] **Step 1: 运行 CLI 测试**

Run:
- `cd skill-home-cli && go test ./internal/taxonomy ./internal/skill ./internal/registry ./internal/cmd`

Expected: PASS

- [ ] **Step 2: 运行 server 测试**

Run:
- `cd skill-home-server && go test ./internal/taxonomy ./internal/api/handlers ./internal/models ./internal/storage`

Expected: PASS

- [ ] **Step 3: 运行 Web 测试与构建**

Run:
- `cd skill-home-web && npm test`
- `cd skill-home-web && VITE_APP_BASE_PATH=/skill-home node node_modules/typescript/bin/tsc -b`
- `cd skill-home-web && VITE_APP_BASE_PATH=/skill-home node node_modules/vite/bin/vite.js build --emptyOutDir`

Expected: PASS

- [ ] **Step 4: 手工验收清单**
  - 使用 `skill-home init` 新建 skill，确认模板包含 `category/tags`
  - 使用 `skill-home validate` 验证缺失/非法 taxonomy 时的错误文案
  - 使用 `skill-home push` 在交互终端下补齐元数据并成功发包
  - Web 发布页无法提交空 `category/tags`
  - owner 能在 General 设置页补齐存量 skill 的分类元数据
  - `skill-home-manager` 文档足以引导 OpenClaw 在 `push` 前先补齐元数据

- [ ] **Step 5: 上线备注**
  - 记录生产环境先不上数据库 `NOT NULL` 强约束
  - 上线后观察新增 skill 是否全部带 `category/tags`
  - 为存量 skill 准备单独 backfill 清单，但不阻塞本次功能发布
