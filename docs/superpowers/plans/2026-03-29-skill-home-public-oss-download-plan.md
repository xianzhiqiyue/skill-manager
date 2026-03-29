# Skill Home 公共 Skill 包 OSS 直链分发实施计划

> **给代理执行者的要求：** 必须使用 `superpowers:subagent-driven-development`（推荐）或 `superpowers:executing-plans` 按任务逐项执行。所有步骤使用 `- [ ]` 复选框跟踪。

**Goal:** 让公共 skill 包由阿里云 OSS 直接分发，同时保留旧 `/api/v1/download/...` 兼容入口，并让 Web 与 CLI 逐步切换到服务端返回的 `download_url`。

**Architecture:** 服务端继续维护 skill 元数据、发布流程和搜索能力，公共 skill 包通过对象存储抽象映射到 OSS 公网地址。新接口语义以绝对 `download_url` 为准，旧下载路由保留为兼容层，在无需格式转换时返回 `302` 到 OSS。Web 与 CLI 统一改成优先消费服务端返回的下载地址，只有面对旧服务端时才回退到历史路径。

**Tech Stack:** Go 1.21、Gin、GORM、TypeScript、Vite、Vitest、现有 CLI registry client、阿里云 OSS（S3 兼容或原生接入）

---

## 文件结构映射

### 现有文件

- 修改：`skill-home-server/internal/config/config.go`
  - 增加 OSS 公共下载相关配置项，并保证环境变量能覆盖。
- 修改：`skill-home-server/internal/storage/object.go`
  - 为对象 key 增加公共下载 URL 解析能力。
- 修改：`skill-home-server/internal/api/handlers/helpers.go`
  - 放置公共下载 URL 计算辅助逻辑，避免在多个 handler 中重复拼接。
- 修改：`skill-home-server/internal/api/handlers/skill.go`
  - 创建 skill 时返回公共 `download_url`。
- 修改：`skill-home-server/internal/api/handlers/version.go`
  - 发布版本时返回公共 `download_url`，并让兼容下载路由在可直链时 `302` 到 OSS。
- 修改：`skill-home-server/internal/api/handlers/handler_integration_test.go`
  - 增加公共下载 URL、重定向和回退行为的集成测试。
- 修改：`skill-home-server/internal/models/skill.go`
  - 为 API 输出补充非持久化的 `download_url` 字段，必要时也给版本结构补充对应字段。
- 修改：`skill-home-web/src/api.ts`
  - 类型定义补充 `download_url`，下载链接生成改为优先使用接口返回值。
- 修改：`skill-home-web/src/pages/PublishNewPage.tsx`
  - 发布成功卡片不再强行拼接 `API_BASE + download_url`。
- 修改：`skill-home-web/src/pages/skill/SkillOverviewPage.tsx`
  - 详情页下载按钮直接使用 skill 元数据里的 `download_url`。
- 修改：`skill-home-web/src/hooks/useRegistryApp.test.tsx`
  - 覆盖新老下载地址兼容逻辑。
- 修改：`skill-home-cli/internal/registry/types.go`
  - 为 `Skill`、必要的版本结构补充 `download_url` 字段。
- 修改：`skill-home-cli/internal/registry/client.go`
  - 下载流程优先使用服务端返回的 `download_url`，再回退到旧路由。
- 修改：`skill-home-cli/internal/registry/client_test.go`
  - 增加绝对 URL 和回退路径测试。
- 修改：`API.md`
  - 更新 `download_url` 语义与示例。
- 修改：`README.md`
  - 更新公共 skill 下载语义说明。

### 新文件

- 新建：`skill-home-server/internal/storage/object_test.go`
  - 为公共下载 URL 生成逻辑补齐单元测试。
- 新建：`skill-home-server/cmd/public-oss-backfill/main.go`
  - 提供可重复执行的历史公共 skill 包回填工具。

---

### Task 1：补齐服务端 OSS 公共 URL 能力与失败测试

**Files:**
- Create: `skill-home-server/internal/storage/object_test.go`
- Modify: `skill-home-server/internal/config/config.go`
- Modify: `skill-home-server/internal/storage/object.go`
- Modify: `skill-home-server/internal/api/handlers/helpers.go`

- [ ] **Step 1: 先写失败测试**
  - 在 `skill-home-server/internal/storage/object_test.go` 中新增测试：
    - 已配置 `PublicBaseURL` 时，`skills/team/reviewer/pkg.zip` 能生成 `https://skills-static.example.com/skills/team/reviewer/pkg.zip`
    - `PublicBaseURL` 缺失时，公共 URL 生成函数返回“不可直链”的空结果，而不是拼出坏地址
    - 含有前后斜杠的 base URL 会被规整，避免出现双斜杠

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/storage -run 'TestObjectStoragePublicURL'`
Expected: FAIL，因为对象存储层当前还没有公共 URL 能力。

- [ ] **Step 3: 做最小实现**
  - 在 `skill-home-server/internal/config/config.go` 的 `StorageConfig` 中新增公共下载 URL 配置，例如 `PublicBaseURL`
  - 让 `loadFromEnv()` 支持如 `SKILL_HOME_STORAGE_PUBLIC_BASE_URL`
  - 在 `skill-home-server/internal/storage/object.go` 中为 `ObjectStorage` 增加公共 URL 相关字段和方法，例如 `PublicURL(key string) (string, bool)`
  - 把 URL 拼接逻辑封装到 `helpers.go` 或存储层自身，避免 handler 手工字符串拼接

- [ ] **Step 4: 重新运行测试**

Run: `go test ./internal/storage`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add skill-home-server/internal/config/config.go skill-home-server/internal/storage/object.go skill-home-server/internal/storage/object_test.go skill-home-server/internal/api/handlers/helpers.go
git commit -m "feat(server): add public object url support"
```

### Task 2：让服务端 API 返回 OSS `download_url`，并保留旧下载兼容入口

**Files:**
- Modify: `skill-home-server/internal/models/skill.go`
- Modify: `skill-home-server/internal/api/handlers/skill.go`
- Modify: `skill-home-server/internal/api/handlers/version.go`
- Modify: `skill-home-server/internal/api/handlers/handler_integration_test.go`
- Modify: `skill-home-server/internal/api/handlers/helpers.go`

- [ ] **Step 1: 先写失败的集成测试**
  - 在 `handler_integration_test.go` 中新增以下测试：
    - 创建公共 skill 成功后，响应中的 `download_url` 为绝对 OSS 地址
    - 发布公共 skill 新版本后，响应中的 `download_url` 为绝对 OSS 地址
    - 请求 `GET /api/v1/download/:namespace/:name/:version?format=zip` 且对象本身就是 zip 时，返回 `302`，`Location` 指向 OSS
    - 当未配置 `PublicBaseURL` 时，同一个下载请求仍走旧的流式下载逻辑

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/api/handlers -run 'TestCreateSkillReturnsPublicDownloadURL|TestPublishVersionReturnsPublicDownloadURL|TestDownloadSkillRedirectsPublicZipToOSS|TestDownloadSkillFallsBackWithoutPublicBaseURL'`
Expected: FAIL，因为当前响应仍然返回相对 `/api/v1/download/...`，旧下载接口也不会做重定向。

- [ ] **Step 3: 做最小实现**
  - 在 `skill-home-server/internal/models/skill.go` 中为 `Skill` 增加非持久化 `DownloadURL string 'gorm:"-" json:"download_url,omitempty"'`
  - 如有必要，也为 `SkillVersion` 增加非持久化 `DownloadURL`
  - 在 `skill-home-server/internal/api/handlers/helpers.go` 中新增统一的 `resolvePublicDownloadURL(...)` 辅助函数
  - 修改 `skill-home-server/internal/api/handlers/skill.go` 和 `skill-home-server/internal/api/handlers/version.go`：
    - 创建 / 发布响应优先返回绝对 OSS 地址
    - 详情和列表返回的 skill 结构也补齐 `download_url`
  - 在 `DownloadSkill` 中判断：
    - 公共 skill
    - 请求格式与源归档格式一致
    - 且存在可用公共 URL
    - 满足以上条件时直接 `302` 到 OSS
    - 否则继续沿用当前流式下载与格式转换逻辑

- [ ] **Step 4: 运行完整 handler 测试**

Run: `go test ./internal/api/handlers`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add skill-home-server/internal/models/skill.go skill-home-server/internal/api/handlers/helpers.go skill-home-server/internal/api/handlers/skill.go skill-home-server/internal/api/handlers/version.go skill-home-server/internal/api/handlers/handler_integration_test.go
git commit -m "feat(server): return public oss download urls"
```

### Task 3：让 Web 和 CLI 都优先消费 `download_url`

**Files:**
- Modify: `skill-home-web/src/api.ts`
- Modify: `skill-home-web/src/pages/PublishNewPage.tsx`
- Modify: `skill-home-web/src/pages/skill/SkillOverviewPage.tsx`
- Modify: `skill-home-web/src/hooks/useRegistryApp.test.tsx`
- Modify: `skill-home-cli/internal/registry/types.go`
- Modify: `skill-home-cli/internal/registry/client.go`
- Modify: `skill-home-cli/internal/registry/client_test.go`

- [ ] **Step 1: 先写前端和 CLI 的失败测试**
  - Web：
    - 在 `useRegistryApp.test.tsx` 或相邻测试里新增用例，验证 skill 详情含绝对 `download_url` 时，下载按钮直接使用该地址
    - 新增发布成功场景用例，验证复制的链接不会被错误拼成 `API_BASE + https://...`
  - CLI：
    - 在 `client_test.go` 中新增测试，验证 `Download()` 会先请求 skill 详情，拿到绝对 `download_url` 后直接下载
    - 再新增一个测试，模拟旧服务端没有 `download_url`，验证 CLI 回退到 `/api/v1/download/...`

- [ ] **Step 2: 运行测试确认失败**

Run:
- `npm test -- --runInBand useRegistryApp`
- `go test ./internal/registry -run 'TestDownloadUsesAbsoluteDownloadURLWhenPresent|TestDownloadFallsBackToLegacyEndpointWhenDownloadURLMissing'`

Expected: FAIL，因为前端仍在自行拼 `/api/v1/download/...`，CLI 也没有读取 skill 详情里的 `download_url`。

- [ ] **Step 3: 做最小实现**
  - 在 `skill-home-web/src/api.ts` 的 `SkillSummary` / `SkillDetail` 类型中增加 `download_url?: string`
  - 修改 `getDownloadUrl()`：
    - 优先返回 skill 自带的 `download_url`
    - 只有缺失时才回退到旧 `/api/v1/download/...`
  - 修改 `PublishNewPage.tsx`，发布成功页复制链接时：
    - 若 `download_url` 已是绝对地址，直接使用
    - 若仍是相对地址，再按 `API_BASE` 补齐
  - 在 CLI 的 `types.go` 里为 `Skill` 增加 `DownloadURL string`
  - 修改 `client.go` 的 `Download()`：
    - 先调用 `GetSkill(namespace, name)`
    - 如果拿到 `download_url` 且目标版本匹配 `latest_version` 或可安全复用，则直接发起该 URL 下载
    - 否则回退到原有 `/api/v1/download/...?...`
    - 绝对 URL 下载时不能再与 registry base URL 拼接
  - 如发现 `GetSkill()` 还不能覆盖指定版本下载，则补一个轻量级版本查询路径，但不要引入不必要的新接口

- [ ] **Step 4: 运行验证**

Run:
- `npm test`
- `go test ./internal/registry`

Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add skill-home-web/src/api.ts skill-home-web/src/pages/PublishNewPage.tsx skill-home-web/src/pages/skill/SkillOverviewPage.tsx skill-home-web/src/hooks/useRegistryApp.test.tsx skill-home-cli/internal/registry/types.go skill-home-cli/internal/registry/client.go skill-home-cli/internal/registry/client_test.go
git commit -m "feat(clients): prefer server download urls"
```

### Task 4：补齐历史公共包回填工具、文档和端到端验收

**Files:**
- Create: `skill-home-server/cmd/public-oss-backfill/main.go`
- Modify: `API.md`
- Modify: `README.md`

- [ ] **Step 1: 先为回填工具定义最小行为**
  - 读取数据库中的公共 skill 版本及其 `storage_path`
  - 校验 OSS 上对象是否存在
  - 若不存在，则从现有存储源复制到 OSS
  - 输出成功、跳过和失败统计
  - 支持 dry-run，避免第一次执行就改动对象存储

- [ ] **Step 2: 实现回填工具**
  - 在 `skill-home-server/cmd/public-oss-backfill/main.go` 中复用现有配置加载和存储抽象
  - 优先保证工具能幂等执行，不要求第一版做并发优化
  - 如当前抽象不足以支持“检查对象是否存在”或“跨后端复制”，补最小接口，但不要把生产 handler 复杂化

- [ ] **Step 3: 更新文档**
  - `API.md`：
    - 说明公共 skill 的 `download_url` 现在可能是 OSS 绝对地址
    - 保留 `/api/v1/download/...` 作为兼容入口说明
  - `README.md`：
    - 说明 Skill Home 负责管理和搜索，公共 skill 包由 OSS 承载

- [ ] **Step 4: 运行验收**

Run:
- `go test ./...`
- `npm test`
- `go run ./cmd/public-oss-backfill --dry-run`

Expected: PASS

- [ ] **Step 5: 做一次手工联调验收**

Run on target environment:
- 发布一个新的公共 skill
- `curl -i 'https://<registry-host>/api/v1/download/<namespace>/<name>/<version>?format=zip'`
- `curl -s 'https://<registry-host>/api/v1/skills/<namespace>/<name>'`
- 用最新 CLI 执行 `skill-home pull <namespace>/<name>@<version>`
- 用一个旧 CLI 再验证一次兼容下载

Expected:
- skill 详情里的 `download_url` 是 OSS 绝对地址
- 旧下载接口返回 `302` 到 OSS
- 新 CLI 与 Web 都直接走 OSS
- 旧 CLI 仍可通过兼容入口成功下载

- [ ] **Step 6: 提交**

```bash
git add skill-home-server/cmd/public-oss-backfill/main.go API.md README.md
git commit -m "docs: describe public oss skill delivery"
```
