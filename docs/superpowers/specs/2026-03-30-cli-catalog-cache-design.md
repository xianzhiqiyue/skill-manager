# Skill Home CLI 目录缓存设计

## 1. 目标

为 `skill-home` CLI 增加一层轻量的“远程目录缓存”，让以下两个命令在常规场景下减少全量拉取，在远端临时失败时也能优雅回退：

- `skill-home list --remote`
- `skill-home search`

期望用户可见结果是：

- 当远端公开目录结构没有变化时，CLI 直接复用本地缓存结果
- 当远端公开目录结构发生变化时，CLI 自动刷新缓存
- 当远端临时失败但本地已有缓存时，CLI 自动回退到旧缓存，并明确提示“结果可能过期”
- 当本地没有缓存且远端失败时，CLI 仍然明确报错

本次设计同时更新 `skill-home-manager` 这个 Codex skill 的说明，让它能正确描述新的 CLI 行为。

## 2. 背景

当前 `skill-home` CLI 已经具备：

- `GetCatalogVersion()`：读取远端 `GET /api/v1/catalog/version`
- `list --remote`：直接请求远端公开技能列表
- `search`：直接请求远端搜索接口

但当前命令层还没有真正利用 `catalog_version`：

- 每次 `list --remote` 都重新请求远端列表
- 每次 `search` 都重新请求远端搜索结果
- 远端短暂失败时，CLI 没有旧缓存可回退

与此同时，服务端已经收窄了 `catalog_version` 的语义：

- 它只覆盖公开目录结构相关变更
- 不覆盖 `download_count`、`rating`、`rating_count` 等动态统计字段

因此，CLI 的缓存语义也必须保持一致，不能把它包装成“所有列表字段都一定最新”的强保证。

## 3. 核心设计

### 3.1 缓存范围

本阶段只给以下两个命令接入目录缓存：

- `skill-home list --remote`
- `skill-home search`

本阶段不扩展到：

- `info`
- `list-versions`
- `pull/install/update`
- 任何写操作命令

### 3.2 语义边界

CLI 只把 `catalog_version` 视为“公开目录结构缓存”的权威字段。

这意味着：

- 当 `catalog_version` 未变化时，CLI 可以复用本地缓存的列表和搜索结果
- CLI 不承诺缓存里的 `download_count`、`rating`、`rating_count` 是最新值
- 如果用户关心这些动态统计字段的新鲜度，应重新请求远端列表或详情接口

### 3.3 失败回退策略

当远端失败时，行为固定为：

1. 如果本地存在对应缓存，则自动回退
2. CLI 输出明确提示，例如“已使用本地缓存，结果可能过期”
3. 如果本地不存在对应缓存，则保持当前错误行为，直接报错

这个策略同时适用于：

- 目录版本检查失败
- 刷新列表请求失败
- 刷新搜索请求失败

## 4. 缓存结构

缓存文件放在 CLI 配置目录下，不放进本地 skill 安装目录。

建议目录：

- `~/.config/skill-home/cache/remote-catalog/`

包含两类文件：

### 4.1 全局状态文件

文件：

- `state.json`

字段建议：

- `registry_endpoint`
- `catalog_version`
- `checked_at`
- `updated_at`

说明：

- `registry_endpoint` 用于区分不同注册中心，避免切换 endpoint 后误用旧缓存
- `catalog_version` 表示当前缓存对应的远端目录版本
- `checked_at` 表示最近一次成功校验目录版本的时间
- `updated_at` 对应远端接口返回的目录更新时间，仅用于观测

### 4.2 查询结果文件

目录：

- `queries/`

每个查询结果一个文件，key 至少包含：

- 命令类型：`list` 或 `search`
- `namespace`
- `query`
- `tags`
- `page`
- `per_page`

建议做法：

- 先构造稳定的 JSON key
- 再取 hash 作为文件名

查询结果文件中保存：

- 原始结果对象
- 对应 `catalog_version`
- 缓存写入时间

## 5. CLI 行为

### 5.1 `list --remote`

读取流程：

1. 构造查询 key
2. 读取 `state.json` 和当前查询缓存
3. 请求 `GetCatalogVersion()`
4. 如果远端 `catalog_version` 与本地一致，且当前查询缓存存在，则直接返回缓存结果
5. 如果远端 `catalog_version` 不一致，或当前查询缓存不存在，则请求真实 `ListSkills`
6. 请求成功后刷新 `state.json` 与当前查询缓存
7. 如果任一步远端请求失败但本地已有当前查询缓存，则回退并提示“结果可能过期”
8. 如果没有缓存，则报错

### 5.2 `search`

行为与 `list --remote` 一致，只是实际拉取使用 `Search`。

差异点仅在 query key 的组成中包含：

- `query`
- `tags`

### 5.3 输出提示

默认 table 输出下：

- 命中缓存且目录版本未变化时，不强制额外提示
- 只有在“远端失败 -> 回退旧缓存”时，输出明确提示

JSON 输出下：

- 不修改原结果结构
- 回退提示仍走 stderr 或普通提示输出，不污染 JSON 主体

## 6. 代码结构建议

建议在 CLI 内新增一层很薄的缓存模块，而不是把逻辑直接散落在 `list.go` 和 `search.go`。

建议新增职责：

- 目录缓存状态读写
- 查询缓存 key 构造
- 目录版本检查
- 带回退的读取流程封装

命令层只负责：

- 解析参数
- 调缓存层获取结果
- 渲染输出

这样后续如果要扩展：

- `info`
- 缓存清理命令
- 更细粒度的 TTL / 强制刷新

都可以在现有结构上继续演进。

## 7. `skill-home-manager` 更新范围

本次不新增专门脚本，只更新 skill 说明与工作流文案。

至少更新：

- `/mnt/c/Users/zhuyu/.codex/skills/skill-home-manager/SKILL.md`

建议同步更新：

- `references/cli-workflows.md`（如果该文件里已经描述远程发现或排障流程）

需要体现的行为：

- `list --remote` / `search` 现在会优先利用目录缓存
- registry 临时失败时，CLI 会自动回退到旧缓存并提示结果可能过期
- 该缓存只覆盖目录结构，不保证下载量和评分等动态统计字段实时

## 8. 测试策略

### 8.1 CLI 测试

至少覆盖以下场景：

- `catalog_version` 未变化时直接命中缓存
- `catalog_version` 变化时刷新缓存
- 远端失败但本地有缓存时自动回退
- 本地无缓存且远端失败时仍然报错
- 切换 `registry_endpoint` 后不会误用旧缓存

### 8.2 文档/skill 验收

至少确认：

- `skill-home-manager` 说明已更新
- 行为描述与 CLI 实现一致
- 中文表述与现有风格一致

## 9. 非目标

本阶段不做：

- 全局离线模式
- 手动 `cache clear` 命令
- 对 `info` / `list-versions` 的缓存
- 动态统计字段的一致性保证
- 将目录缓存下沉到共享 daemon 或后台服务

## 10. 验收标准

满足以下条件即算完成：

- `list --remote` 和 `search` 已接入目录缓存
- 远端目录版本不变时能直接复用缓存
- 远端失败时若本地有缓存会自动回退并提示
- 本地无缓存时仍然清晰报错
- `skill-home-manager` 说明已同步更新
- 全部新增/受影响测试通过
