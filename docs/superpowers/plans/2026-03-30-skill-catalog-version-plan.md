# Skill Home 技能目录版本号实施计划

> **给代理执行者的要求：** 必须使用 `superpowers:subagent-driven-development`（推荐）或 `superpowers:executing-plans` 按任务逐项执行。所有步骤使用 `- [ ]` 复选框跟踪。

**Goal:** 为 Skill Home 增加单调递增的技能目录版本号接口，并在所有影响公开目录结构的成功写操作中统一递增，同时为 CLI 提供读取该版本号的客户端能力。

**Architecture:** 服务端新增一张单例状态表保存 `catalog_version` 与 `updated_at`，通过一个公开只读接口暴露当前目录版本。所有会改变公开目录结构的写操作都在现有数据库事务中原子递增该版本号，避免成功写入与版本状态分叉。该版本号不覆盖 `download_count`、`rating`、`rating_count` 等动态统计字段。CLI 侧只补充读取目录版本的 registry client 能力，不在这一阶段引入完整的本地缓存策略。

**Tech Stack:** Go 1.21、Gin、GORM、PostgreSQL、现有 Skill Home server handlers、现有 skill-home CLI registry client

---

## 文件结构映射

### 现有文件

- 修改：`skill-home-server/internal/storage/database.go`
  - 把新的目录状态模型纳入 `AutoMigrate`。
- 修改：`skill-home-server/cmd/server/main.go`
  - 注册新的 `GET /api/v1/catalog/version` 路由。
- 修改：`skill-home-server/internal/api/handlers/skill.go`
  - 在创建、更新、删除 skill 的事务里 bump 目录版本。
- 修改：`skill-home-server/internal/api/handlers/version.go`
  - 在发布版本、删除版本的事务里 bump 目录版本。
- 修改：`skill-home-server/internal/api/handlers/handler_integration_test.go`
  - 增加目录版本接口、成功 bump、失败不 bump 的集成测试。
- 修改：`skill-home-cli/internal/registry/types.go`
  - 新增目录版本响应结构。
- 修改：`skill-home-cli/internal/registry/client.go`
  - 新增 `GetCatalogVersion()`。
- 修改：`skill-home-cli/internal/registry/client_test.go`
  - 覆盖新接口调用与解析测试。
- 修改：`API.md`
  - 记录目录版本接口契约与语义。

### 新文件

- 新建：`skill-home-server/internal/models/catalog_state.go`
  - 定义目录状态模型。
- 新建：`skill-home-server/internal/api/handlers/catalog.go`
  - 提供读取目录版本的 handler。
- 新建：`skill-home-server/internal/api/handlers/catalog_state.go`
  - 放置目录状态初始化、读取和原子递增辅助逻辑。
- 新建：`skill-home-server/internal/api/handlers/catalog_test.go`
  - 覆盖目录状态辅助逻辑的单元测试。

---

### Task 1：先补目录状态模型与读取接口

**Files:**
- Create: `skill-home-server/internal/models/catalog_state.go`
- Create: `skill-home-server/internal/api/handlers/catalog.go`
- Create: `skill-home-server/internal/api/handlers/catalog_state.go`
- Create: `skill-home-server/internal/api/handlers/catalog_test.go`
- Modify: `skill-home-server/internal/storage/database.go`
- Modify: `skill-home-server/cmd/server/main.go`

- [ ] **Step 1: 先写失败测试**
  - 在 `skill-home-server/internal/api/handlers/catalog_test.go` 中增加测试：
    - 状态表为空时，`ensureCatalogState` 会创建初始记录
    - 初始 `catalog_version` 为 `1`
    - `GetCatalogVersion` handler 返回 `catalog_version` 和 `updated_at`

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/api/handlers -run 'TestEnsureCatalogState|TestGetCatalogVersion'`
Expected: FAIL，因为当前没有目录状态模型、辅助逻辑和读取接口。

- [ ] **Step 3: 做最小实现**
  - 在 `skill-home-server/internal/models/catalog_state.go` 中定义单例状态模型，例如：
    - `ID`
    - `CatalogVersion`
    - `CreatedAt`
    - `UpdatedAt`
  - 在 `skill-home-server/internal/storage/database.go` 里把它加入 `AutoMigrate`
  - 在 `skill-home-server/internal/api/handlers/catalog_state.go` 中实现：
    - `ensureCatalogState(tx *gorm.DB) (*models.CatalogState, error)`
    - `getCatalogState(db *storage.Database) (*models.CatalogState, error)`
  - 在 `skill-home-server/internal/api/handlers/catalog.go` 中实现：
    - `GetCatalogVersion(db *storage.Database) gin.HandlerFunc`
  - 在 `skill-home-server/cmd/server/main.go` 注册：
    - `GET /api/v1/catalog/version`

- [ ] **Step 4: 重新运行测试**

Run: `go test ./internal/api/handlers -run 'TestEnsureCatalogState|TestGetCatalogVersion'`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add skill-home-server/internal/models/catalog_state.go skill-home-server/internal/api/handlers/catalog.go skill-home-server/internal/api/handlers/catalog_state.go skill-home-server/internal/api/handlers/catalog_test.go skill-home-server/internal/storage/database.go skill-home-server/cmd/server/main.go
git commit -m "feat(server): add skill catalog version endpoint"
```

### Task 2：把目录版本 bump 挂到所有相关写操作事务里

**Files:**
- Modify: `skill-home-server/internal/api/handlers/skill.go`
- Modify: `skill-home-server/internal/api/handlers/version.go`
- Modify: `skill-home-server/internal/api/handlers/handler_integration_test.go`
- Modify: `skill-home-server/internal/api/handlers/catalog_state.go`

- [ ] **Step 1: 先写失败的集成测试**
  - 在 `skill-home-server/internal/api/handlers/handler_integration_test.go` 中新增测试：
    - 创建 skill 成功后，目录版本从 `1` 变为 `2`
    - 更新 skill 成功后，目录版本递增 `1`
    - 删除 skill 成功后，目录版本递增 `1`
    - 发布新版本成功后，目录版本递增 `1`
    - 删除版本成功后，目录版本递增 `1`
    - 重复创建 skill、重复发布版本等失败场景不会 bump

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/api/handlers -run 'TestCatalogVersion'`
Expected: FAIL，因为当前写操作还没有递增目录版本。

- [ ] **Step 3: 做最小实现**
  - 在 `skill-home-server/internal/api/handlers/catalog_state.go` 中增加：
    - `bumpCatalogVersionTx(tx *gorm.DB) error`
  - 采用数据库原子更新方式递增，例如：
    - `catalog_version = catalog_version + 1`
  - 在以下现有事务中插入 bump：
    - `CreateSkill`
    - `UpdateSkill`
    - `DeleteSkill`
    - `PublishVersion`
    - `DeleteVersion`
  - 保证 bump 和原业务写入在同一个事务里提交

- [ ] **Step 4: 运行完整 handler 测试**

Run: `go test ./internal/api/handlers`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add skill-home-server/internal/api/handlers/skill.go skill-home-server/internal/api/handlers/version.go skill-home-server/internal/api/handlers/handler_integration_test.go skill-home-server/internal/api/handlers/catalog_state.go
git commit -m "feat(server): bump catalog version on skill mutations"
```

### Task 3：补 CLI 侧读取目录版本的能力

**Files:**
- Modify: `skill-home-cli/internal/registry/types.go`
- Modify: `skill-home-cli/internal/registry/client.go`
- Modify: `skill-home-cli/internal/registry/client_test.go`

- [ ] **Step 1: 先写失败测试**
  - 在 `skill-home-cli/internal/registry/client_test.go` 中新增测试：
    - `GetCatalogVersion()` 会请求 `/api/v1/catalog/version`
    - 返回的 `catalog_version` 和 `updated_at` 能正确反序列化
    - 服务端返回错误时，客户端会透传错误

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/registry -run 'TestGetCatalogVersion'`
Expected: FAIL，因为当前 client 没有该接口。

- [ ] **Step 3: 做最小实现**
  - 在 `skill-home-cli/internal/registry/types.go` 中新增：
    - `CatalogVersionResponse`
  - 在 `skill-home-cli/internal/registry/client.go` 中新增：
    - `GetCatalogVersion() (*CatalogVersionResponse, error)`
  - 不在这一阶段引入磁盘缓存、列表缓存文件或命令层行为改动
  - 只把“获取远端目录版本号”的能力补齐，作为后续缓存策略的稳定基础

- [ ] **Step 4: 运行 registry 测试**

Run: `go test ./internal/registry`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add skill-home-cli/internal/registry/types.go skill-home-cli/internal/registry/client.go skill-home-cli/internal/registry/client_test.go
git commit -m "feat(cli): add catalog version client"
```

### Task 4：补文档与端到端验收

**Files:**
- Modify: `API.md`

- [ ] **Step 1: 更新接口文档**
  - 在 `API.md` 中新增 `GET /api/v1/catalog/version`
  - 说明：
    - `catalog_version` 是目录结构缓存判断的唯一权威字段
    - `updated_at` 仅用于观测与排障
    - 哪些公开目录写操作会触发版本号变化
    - `download_count`、`rating`、`rating_count` 等动态统计字段不在该版本号覆盖范围

- [ ] **Step 2: 运行服务端与 CLI 验证**

Run:
- `go test ./...` in `skill-home-server`
- `go test ./internal/registry` in `skill-home-cli`

Expected: PASS

- [ ] **Step 3: 做一次手工端到端验收**

Run on target environment:
- `curl https://soulstore.ciqtek.com/skill-home/api/v1/catalog/version`
- 发布一个测试 skill 或更新一个测试 skill 元数据
- 再次 `curl https://soulstore.ciqtek.com/skill-home/api/v1/catalog/version`
- 用 CLI 调用 `GetCatalogVersion()` 对应路径做一次请求验证

Expected:
- 第一次能拿到当前目录版本
- 成功写操作后 `catalog_version` 递增 `1`
- 返回结构稳定

- [ ] **Step 4: 提交**

```bash
git add API.md
git commit -m "docs: describe skill catalog version api"
```
