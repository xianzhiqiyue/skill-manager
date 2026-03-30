# 命令行目录缓存 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 `skill-home list --remote` 和 `skill-home search` 接入基于 `catalog_version` 的远程目录缓存，并同步更新 `skill-home-manager` 对应说明。

**Architecture:** 在 `skill-home-cli` 内新增一层轻量的目录缓存模块，负责注册中心隔离、目录版本校验、查询结果持久化和失败回退。命令层只负责参数解析、调用缓存层和输出提示；`skill-home-manager` 只更新文档，不新增脚本。

**Tech Stack:** Go 1.21、Cobra、Viper、现有 registry client、现有 CLI 命令测试、Codex 技能文档

---

## 文件结构映射

### 现有文件

- 修改：`skill-home-cli/internal/cmd/list.go`
  - 让 `list --remote` 通过缓存层读取远程目录。
- 修改：`skill-home-cli/internal/cmd/search.go`
  - 让 `search` 通过缓存层读取远程目录。
- 修改：`skill-home-cli/internal/cmd/registry_helpers.go`
  - 为命令层暴露目录版本读取能力，并集中创建缓存依赖。
- 修改：`skill-home-cli/internal/cmd/delete_test.go`
  - 扩展 `fakeRegistryClient` 以支持目录版本读取。
- 修改：`/mnt/c/Users/zhuyu/.codex/skills/skill-home-manager/SKILL.md`
  - 说明 `list --remote` / `search` 的缓存与回退行为。
- 修改：`/mnt/c/Users/zhuyu/.codex/skills/skill-home-manager/references/cli-workflows.md`
  - 补远程发现与排障文案。

### 新文件

- 新建：`skill-home-cli/internal/cmd/remote_catalog_cache.go`
  - 放缓存 key、状态文件、查询文件、命中/回退流程。
- 新建：`skill-home-cli/internal/cmd/remote_catalog_cache_test.go`
  - 覆盖缓存层的注册中心隔离、一致性校验、失败回退。
- 新建：`skill-home-cli/internal/cmd/list_test.go`
  - 覆盖 `list --remote` 命中缓存、刷新缓存、失败回退的命令层行为。
- 新建：`skill-home-cli/internal/cmd/search_test.go`
  - 覆盖 `search` 命中缓存、刷新缓存、失败回退的命令层行为。

---

### Task 1: 先把目录缓存层设计成可测模块

**Files:**
- Create: `skill-home-cli/internal/cmd/remote_catalog_cache.go`
- Create: `skill-home-cli/internal/cmd/remote_catalog_cache_test.go`
- Modify: `skill-home-cli/internal/cmd/registry_helpers.go`
- Modify: `skill-home-cli/internal/cmd/delete_test.go`

- [ ] **Step 1: 写失败测试，锁定缓存状态与查询文件语义**

在 `skill-home-cli/internal/cmd/remote_catalog_cache_test.go` 增加测试，至少覆盖：

- 同一注册中心、相同查询参数会生成稳定缓存 key
- 切换 `registry.endpoint` 后不会命中旧注册中心缓存
- `state.json` 与查询文件版本不一致时不会被当作“新鲜缓存”
- 远端失败时，只要查询文件来源正确且结构完整，就允许作为过期结果回退
- 查询文件缺少来源信息或结构损坏时，不允许回退

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/cmd -run 'TestRemoteCatalogCache' -count=1`
Expected: FAIL，因为当前还没有缓存模块和相关类型。

- [ ] **Step 3: 写最小实现**

在 `skill-home-cli/internal/cmd/remote_catalog_cache.go` 中实现：

- 缓存根目录解析（放在 `~/.config/skill-home/cache/remote-catalog/<registry-hash>/`）
- `state.json` 结构
- 查询缓存文件结构
- 查询 key 构造
- “新鲜命中”校验
- “远端失败 -> 过期回退”校验
- 原子写入辅助逻辑

在 `skill-home-cli/internal/cmd/registry_helpers.go` 中：

- 给 `registryClient` 接口增加 `GetCatalogVersion()`
- 保留现有 `newRegistryClient()` 入口

在 `skill-home-cli/internal/cmd/delete_test.go` 中：

- 给 `fakeRegistryClient` 增加 `GetCatalogVersion()` 实现

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/cmd -run 'TestRemoteCatalogCache' -count=1`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add skill-home-cli/internal/cmd/remote_catalog_cache.go skill-home-cli/internal/cmd/remote_catalog_cache_test.go skill-home-cli/internal/cmd/registry_helpers.go skill-home-cli/internal/cmd/delete_test.go
git commit -m "feat(cli): add remote catalog cache core"
```

### Task 2: 把 `list --remote` 接到缓存层

**Files:**
- Modify: `skill-home-cli/internal/cmd/list.go`
- Create: `skill-home-cli/internal/cmd/list_test.go`

- [ ] **Step 1: 写失败测试**

在 `skill-home-cli/internal/cmd/list_test.go` 增加命令层测试，至少覆盖：

- 远端目录版本未变化时，`list --remote` 直接返回缓存结果
- 远端目录版本变化时，会请求真实 `ListSkills` 并刷新缓存
- 远端失败但本地有缓存时，会回退并输出“结果可能过期”
- 本地无缓存且远端失败时，返回错误

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/cmd -run 'TestListRemoteUsesCatalogCache' -count=1`
Expected: FAIL，因为 `list --remote` 目前直接请求远端。

- [ ] **Step 3: 写最小实现**

在 `skill-home-cli/internal/cmd/list.go` 中：

- 把 `listRemoteSkills()` 改成通过缓存层获取结果
- 保持现有 `json/table` 输出不变
- 只有在回退旧缓存时打印提示，不污染 JSON 主体

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/cmd -run 'TestListRemoteUsesCatalogCache' -count=1`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add skill-home-cli/internal/cmd/list.go skill-home-cli/internal/cmd/list_test.go
git commit -m "feat(cli): cache remote list results"
```

### Task 3: 把 `search` 接到缓存层

**Files:**
- Modify: `skill-home-cli/internal/cmd/search.go`
- Create: `skill-home-cli/internal/cmd/search_test.go`

- [ ] **Step 1: 写失败测试**

在 `skill-home-cli/internal/cmd/search_test.go` 增加命令层测试，至少覆盖：

- 目录版本未变化时，`search` 直接返回缓存结果
- 目录版本变化时，`search` 刷新缓存
- 远端失败但本地有缓存时，`search` 回退并提示结果可能过期
- 搜索条件变化（`query`、`tags`、`namespace`、`page`、`perPage`）会使用不同缓存 key

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/cmd -run 'TestSearchUsesCatalogCache' -count=1`
Expected: FAIL，因为 `search` 目前直接请求远端。

- [ ] **Step 3: 写最小实现**

在 `skill-home-cli/internal/cmd/search.go` 中：

- 把远程搜索入口改成通过缓存层读取
- 保持现有输出格式和提示文案风格

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/cmd -run 'TestSearchUsesCatalogCache' -count=1`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add skill-home-cli/internal/cmd/search.go skill-home-cli/internal/cmd/search_test.go
git commit -m "feat(cli): cache remote search results"
```

### Task 4: 更新 `skill-home-manager` 文档

**Files:**
- Modify: `/mnt/c/Users/zhuyu/.codex/skills/skill-home-manager/SKILL.md`
- Modify: `/mnt/c/Users/zhuyu/.codex/skills/skill-home-manager/references/cli-workflows.md`

- [ ] **Step 1: 写文档改动**

在 `SKILL.md` 和 `references/cli-workflows.md` 中补充：

- `list --remote` / `search` 现在会优先利用目录缓存
- 注册中心失败时会自动回退旧缓存并提示“结果可能过期”
- 该缓存只覆盖目录结构，不保证下载量和评分等动态统计字段实时

- [ ] **Step 2: 自查文档一致性**

检查两份文档的表述是否一致，且都使用中文。

- [ ] **Step 3: 提交**

```bash
git add /mnt/c/Users/zhuyu/.codex/skills/skill-home-manager/SKILL.md /mnt/c/Users/zhuyu/.codex/skills/skill-home-manager/references/cli-workflows.md
git commit -m "docs(skill): describe catalog cache behavior"
```

### Task 5: 统一回归并做本地验收

**Files:**
- Modify: `skill-home-cli/internal/cmd/remote_catalog_cache_test.go`
- Modify: `skill-home-cli/internal/cmd/list_test.go`
- Modify: `skill-home-cli/internal/cmd/search_test.go`
- Modify: 其他本轮实际触达测试文件（如有）

- [ ] **Step 1: 跑完整 CLI 命令层测试**

Run: `go test ./internal/cmd -count=1`
Expected: PASS

- [ ] **Step 2: 跑 registry client 测试，确认目录版本读取未回归**

Run: `go test ./internal/registry -count=1`
Expected: PASS

- [ ] **Step 3: 做一次手工本地验收**

建议最少验证：

- 先造一份本地缓存
- 在 mock 远端版本不变时，`list --remote` / `search` 直接读缓存
- 在 mock 远端失败时，CLI 输出“结果可能过期”
- 删除缓存后，远端失败时命令明确报错

- [ ] **Step 4: 提交收尾改动（如有）**

```bash
git add skill-home-cli/internal/cmd
git commit -m "test(cli): cover catalog cache flows"
```
