# Skill-Home API 文档

## 基本信息

- **基础 URL**: `https://soulstore.ciqtek.com/skill-home`
- **API 版本**: `v1`
- **数据格式**: JSON
- **认证方式**: Bearer Token (JWT 或 API Key)

## 认证

### 注册

```http
POST /api/v1/auth/register
Content-Type: application/json

{
  "username": "testuser",
  "display_name_zh": "测试用户",
  "email": "test@example.com",
  "password": "testpass123"
}
```

**响应**:

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "d30d52dd-d3ea-4761-ad9d-c18ecd4431bc",
    "username": "testuser",
    "display_name_zh": "测试用户",
    "email": "test@example.com",
    "is_admin": false,
    "is_super_admin": false
  }
}
```

### 登录

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "test@example.com",
  "password": "testpass123"
}
```

**响应**: 同注册

### 使用 Token

```http
GET /api/v1/user
Authorization: Bearer <token>
```

## 权限模型

当前第一阶段沿用 `is_admin` / `is_super_admin` 两个角色标记，并在管理接口中返回派生 `role` 字段：

| 身份 | 核心权限 |
|------|----------|
| 匿名用户 | 浏览公开 skill、搜索、下载公开版本、记录匿名安装/分享事件 |
| 注册用户 | 发布 skill、管理自己的 skill/版本、评分、点赞、管理自己的 API Key、查看自己的统计和审计日志、修改个人资料和密码 |
| Skill owner | 更新、公开/私有切换、弃用、删除自己拥有的 skill，发布/删除自己的版本 |
| Admin | 管理公开目录推荐状态 |
| Super admin | 管理任意 skill/版本，查看和管理用户，查看全局审计日志 |

安全护栏：

- 停用用户后，该用户的 JWT 与 API Key 都不能继续访问认证接口。
- Super admin 不能停用当前登录账号。
- 系统拒绝移除或停用最后一个有效 super admin。
- 用户权限变更写入 `admin.user.update` 审计日志。

## 错误处理

所有错误响应格式:

```json
{
  "code": "ERROR_CODE",
  "message": "Human readable message"
}
```

常见错误码:

| Code | HTTP Status | 说明 |
|------|-------------|------|
| `INVALID_INPUT` | 400 | 请求参数错误 |
| `UNAUTHORIZED` | 401 | 未认证 |
| `FORBIDDEN` | 403 | 无权限 |
| `NOT_FOUND` | 404 | 资源不存在 |
| `ALREADY_EXISTS` | 409 | 资源已存在 |
| `INTERNAL_ERROR` | 500 | 服务器内部错误 |

## 接口列表

### 健康检查

```http
GET /health
```

**响应**:

```json
{
  "service": "skill-home-registry",
  "status": "ok",
  "version": "1.0.0"
}
```

### 技能管理

#### 获取公开目录版本

```http
GET /api/v1/catalog/version
```

该接口用于判断 Skill Home 公开目录结构是否发生变化。客户端应将返回的 `catalog_version` 作为目录结构缓存失效的唯一权威字段，用来决定是否重新拉取公开 skill 列表与搜索结果。

**响应**:

```json
{
  "catalog_version": 42,
  "updated_at": "2026-03-30T08:00:00Z"
}
```

- `catalog_version` 会随着公开目录结构发生有效变更而递增，客户端应仅以它判断目录结构缓存是否失效。
- `updated_at` 仅用于观测和排障，表示最近一次目录版本变更时间，不应用作缓存对比键。
- 成功的公开目录变更会触发版本递增，至少包括：创建公开 skill、公开 skill 更新、公开 skill 删除、公开 skill 发布版本、公开 skill 删除版本、推荐状态变更。
- 私有 skill 的创建、更新、删除、发布版本、删除版本不会触发公开目录版本变化。
- `download_count`、`like_count`、`install_count`、`rating`、`rating_count` 等动态统计字段不属于该版本号覆盖范围；如果客户端需要这些字段的最新值，应主动重新拉取列表或详情接口。

#### 列出技能

```http
GET /api/v1/skills?page=1&per_page=20&q=keyword&tag=tag1&namespace=testuser
```

**参数**:

| 参数 | 类型 | 说明 |
|------|------|------|
| page | int | 页码，默认 1 |
| per_page | int | 每页数量，默认 20 |
| q | string | 搜索关键词 |
| tag | string | 标签筛选 |
| namespace | string | 命名空间筛选 |

**响应**:

```json
{
  "total": 100,
  "page": 1,
  "per_page": 20,
  "results": [
    {
      "id": "...",
      "namespace": "testuser",
      "name": "my-skill",
      "owner_username": "testuser",
      "owner_display_name_zh": "测试用户",
      "description": "技能描述",
      "category": "automation",
      "author": "testuser",
      "tags": ["workflow", "automation"],
      "download_count": 42,
      "like_count": 8,
      "install_count": 13,
      "rating": 4.8,
      "rating_count": 5,
      "is_recommended": true,
      "latest_version": "1.0.0",
      "download_url": "https://oss-example.aliyuncs.com/public-skills/testuser/my-skill/1.0.0.zip",
      "created_at": "2026-03-01T10:00:00Z",
      "updated_at": "2026-03-01T12:00:00Z"
    }
  ]
}
```

公开 skill 的 `download_url` 可能直接是 OSS 公网绝对地址；旧服务或未配置公共对象地址时，也可能继续返回 `/api/v1/download/...` 兼容路径。

#### 获取技能详情

```http
GET /api/v1/skills/:namespace/:name
Authorization: Bearer <token>  // 可选，访问私有技能或返回 user_rating 时需要
```

**响应**:

```json
{
  "id": "...",
  "namespace": "testuser",
  "name": "my-skill",
  "owner_id": "...",
  "owner_username": "testuser",
  "owner_display_name_zh": "测试用户",
  "description": "技能描述",
  "category": "automation",
  "tags": ["workflow"],
  "license": "MIT",
  "download_count": 42,
  "like_count": 8,
  "install_count": 13,
  "rating": 4.8,
  "rating_count": 5,
  "viewer_liked": true,
  "is_recommended": true,
  "is_public": true,
  "latest_version": "1.0.0",
  "download_url": "https://oss-example.aliyuncs.com/public-skills/testuser/my-skill/1.0.0.zip",
  "created_at": "2026-03-01T10:00:00Z",
  "updated_at": "2026-03-01T12:00:00Z",
  "owner": {
    "id": "...",
    "username": "testuser",
    "display_name_zh": "测试用户",
    "email": "test@example.com"
  },
  "user_rating": {
    "id": "...",
    "skill_id": "...",
    "user_id": "...",
    "rating": 5,
    "comment": "很实用"
  },
  "versions": [
    {
      "id": "...",
      "version": "1.0.0",
      "download_url": "https://oss-example.aliyuncs.com/public-skills/testuser/my-skill/1.0.0.zip",
      "size_bytes": 1024,
      "scan_status": "pass",
      "published_at": "2026-03-01T10:00:00Z"
    }
  ]
}
```

#### 发布技能

```http
POST /api/v1/skills
Authorization: Bearer <token>
Content-Type: multipart/form-data

namespace: testuser
name: my-skill
description: 技能描述
category: automation
version: 1.0.0
license: MIT
tags: workflow, automation
is_public: true
skill: <文件>
```

`category` 和 `tags` 都是必填字段，其中 `tags` 必须是 1 到 4 个官方标签。

**响应**:

```json
{
  "namespace": "testuser",
  "name": "my-skill",
  "version": "1.0.0",
  "download_url": "https://oss-example.aliyuncs.com/public-skills/testuser/my-skill/1.0.0.zip"
}
```

当 skill 是公开包且对象存储已配置公共地址时，`download_url` 会返回可直接下载的 OSS 绝对地址；否则仍会返回 `/api/v1/download/...` 兼容路径。

#### 更新技能

```http
PUT /api/v1/skills/:namespace/:name
Authorization: Bearer <token>
Content-Type: application/json

{
  "description": "新描述",
  "category": "docs",
  "tags": ["docs", "workflow"],
  "license": "Apache-2.0",
  "is_public": false
}
```

#### 设置 skill 推荐状态

```http
PATCH /api/v1/admin/skills/:namespace/:name/recommendation
Authorization: Bearer <token>
Content-Type: application/json

{
  "is_recommended": true
}
```

说明：

- 仅 `is_admin=true` 或 `is_super_admin=true` 的用户可调用。
- 只有公开 skill 可以被设置为推荐。
- 推荐 skill 会在技能中心和搜索结果中优先排序。

#### 删除技能

```http
DELETE /api/v1/skills/:namespace/:name
Authorization: Bearer <token>
```

#### 为技能评分

```http
POST /api/v1/skills/:namespace/:name/rating
Authorization: Bearer <token>
Content-Type: application/json

{
  "rating": 5,
  "comment": "很实用"
}
```

**响应**:

```json
{
  "skill": {
    "id": "...",
    "namespace": "testuser",
    "name": "my-skill",
    "rating": 4.8,
    "rating_count": 5
  },
  "user_rating": {
    "id": "...",
    "skill_id": "...",
    "user_id": "...",
    "rating": 5,
    "comment": "很实用"
  }
}
```

#### 点赞 / 取消点赞

```http
POST /api/v1/skills/:namespace/:name/like
Authorization: Bearer <token>

DELETE /api/v1/skills/:namespace/:name/like
Authorization: Bearer <token>
```

**响应**: 更新后的 skill 详情，包含 `like_count` 和当前用户视角的 `viewer_liked`。

#### 记录安装事件

```http
POST /api/v1/skills/:namespace/:name/install-events
Authorization: Bearer <token>  // 可选
Content-Type: application/json

{
  "version": "1.0.0",
  "target": "codex",
  "install_mode": "mirror",
  "client_version": "0.1.0"
}
```

**响应**:

```json
{
  "install_count": 14,
  "skill": {
    "namespace": "testuser",
    "name": "my-skill",
    "install_count": 14
  }
}
```

#### 记录分享事件

```http
POST /api/v1/skills/:namespace/:name/share-events
Authorization: Bearer <token>  // 可选
Content-Type: application/json

{
  "channel": "copy-link"
}
```

**响应**:

```json
{
  "message": "Share event recorded"
}
```

### 版本管理

#### 列出版本

```http
GET /api/v1/skills/:namespace/:name/versions
```

**响应**:

```json
[
  {
    "id": "...",
    "skill_id": "...",
    "version": "1.0.0",
    "download_url": "https://oss-example.aliyuncs.com/public-skills/testuser/my-skill/1.0.0.zip",
    "manifest": null,
    "size_bytes": 1024,
    "scan_status": "pass",
    "scan_result": {"issues": []},
    "published_by": "...",
    "published_at": "2026-03-01T10:00:00Z",
    "created_at": "2026-03-01T10:00:00Z"
  }
]
```

#### 发布新版本

```http
POST /api/v1/skills/:namespace/:name/versions
Authorization: Bearer <token>
Content-Type: multipart/form-data

version: 1.1.0
skill: <文件>
```

**响应**:

```json
{
  "namespace": "testuser",
  "name": "my-skill",
  "version": "1.1.0",
  "download_url": "https://oss-example.aliyuncs.com/public-skills/testuser/my-skill/1.1.0.zip"
}
```

#### 删除版本

```http
DELETE /api/v1/skills/:namespace/:name/versions/:version
Authorization: Bearer <token>
```

### 搜索

```http
GET /api/v1/search?q=keyword&tag=tag1&namespace=testuser&page=1&per_page=20
```

**响应**: 同列出技能

### 下载

```http
GET /api/v1/download/:namespace/:name/:version
```

该接口仍然保留为兼容下载入口：

- 对公开 skill，服务端可能直接返回 `302`，`Location` 指向对象存储里的现成包文件，例如 `.zip` 直链。
- 对未配置公共对象地址的部署，或需要兼容旧客户端的场景，服务端会继续走 Skill Home 下载入口，并按当前下载参数返回包文件；默认不带 `format` 时仍会保持 ZIP 兼容语义，必要时会在服务端做压缩格式转换。

**响应**: `302` 跳转到对象存储直链，或直接返回下载文件流；具体 `Content-Type` / 文件后缀取决于请求的下载格式与对象原始格式。

### 用户管理

#### 获取当前用户

```http
GET /api/v1/user
Authorization: Bearer <token>
```

**响应**:

```json
{
  "id": "...",
  "username": "testuser",
  "display_name_zh": "测试用户",
  "email": "test@example.com",
  "avatar_url": "",
  "created_at": "2026-03-01T10:00:00Z"
}
```

#### 获取用户技能

```http
GET /api/v1/user/skills
Authorization: Bearer <token>
```

#### 获取当前用户统计

```http
GET /api/v1/user/stats
Authorization: Bearer <token>
```

#### 获取公开用户统计

```http
GET /api/v1/users/:username/stats
```

**响应**:

```json
{
  "user_id": "...",
  "username": "testuser",
  "display_name_zh": "测试用户",
  "skill_count": 3,
  "public_skill_count": 2,
  "total_like_count": 11,
  "total_install_count": 7,
  "total_download_count": 19,
  "average_rating": 4.5,
  "total_rating_count": 4
}
```

#### 更新当前用户资料

```http
PUT /api/v1/user/profile
Authorization: Bearer <token>
Content-Type: application/json

{
  "display_name_zh": "测试用户",
  "avatar_url": "https://example.com/avatar.png"
}
```

说明：

- `display_name_zh` 必填，注册后仍可由本人修改。
- `avatar_url` 可选。

**响应**：更新后的当前用户对象。

#### 修改当前用户密码

```http
PUT /api/v1/user/password
Authorization: Bearer <token>
Content-Type: application/json

{
  "current_password": "old-password",
  "new_password": "new-password"
}
```

说明：

- 必须校验当前密码。
- `new_password` 至少 6 个字符。

**响应**:

```json
{
  "message": "Password updated"
}
```

#### 获取最近活动

```http
GET /api/v1/user/audit-logs?page=1&per_page=20&action=skill.rate
Authorization: Bearer <token>
```

**响应**:

```json
{
  "total": 2,
  "page": 1,
  "per_page": 20,
  "results": [
    {
      "id": "...",
      "action": "skill.rate",
      "resource_type": "skill",
      "resource_id": "...",
      "metadata": {
        "namespace": "testuser",
        "name": "my-skill",
        "rating": 5
      },
      "ip_address": "127.0.0.1",
      "user_agent": "skill-home-cli",
      "created_at": "2026-03-09T12:00:00Z"
    }
  ]
}
```

### 用户权限管理

以下接口仅 super admin 可访问。

#### 用户列表

```http
GET /api/v1/admin/users?page=1&per_page=20&q=zhang&role=admin&status=active
Authorization: Bearer <token>
```

**参数**:

| 参数 | 类型 | 说明 |
|------|------|------|
| page | int | 页码，默认 1 |
| per_page | int | 每页数量，默认 20 |
| q | string | 搜索用户名、邮箱或中文名 |
| role | string | `all`、`member`、`admin`、`super_admin` |
| status | string | `all`、`active`、`inactive` |

**响应**:

```json
{
  "total": 2,
  "page": 1,
  "per_page": 20,
  "results": [
    {
      "id": "...",
      "username": "testuser",
      "display_name_zh": "测试用户",
      "email": "test@example.com",
      "avatar_url": "",
      "role": "member",
      "is_active": true,
      "is_admin": false,
      "is_super_admin": false,
      "created_at": "2026-03-01T10:00:00Z",
      "updated_at": "2026-03-01T10:00:00Z"
    }
  ]
}
```

#### 更新用户权限

```http
PUT /api/v1/admin/users/:id
Authorization: Bearer <token>
Content-Type: application/json

{
  "is_active": true,
  "is_admin": false,
  "is_super_admin": false,
  "password": "new-password"
}
```

说明：

- 字段均可选，但至少提供一个字段。
- `password` 至少 6 个字符，用于 super admin 重置用户密码。
- 禁止停用当前登录账号。
- 禁止移除或停用最后一个有效 super admin。

**响应**：更新后的用户对象，包含派生 `role` 字段。

#### 全局审计日志

```http
GET /api/v1/admin/audit-logs?page=1&per_page=20&action=admin.user.update&user_id=:id
Authorization: Bearer <token>
```

**响应**:

```json
{
  "total": 1,
  "page": 1,
  "per_page": 20,
  "results": [
    {
      "id": "...",
      "user_id": "...",
      "action": "admin.user.update",
      "resource_type": "user",
      "resource_id": "...",
      "metadata": {
        "target_user_id": "...",
        "username": "testuser",
        "is_admin": true
      },
      "created_at": "2026-03-09T12:00:00Z"
    }
  ]
}
```

### API Key 管理

#### 创建 API Key

```http
POST /api/v1/user/api-keys
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "cli-key",
  "expires_at": "2026-12-31T23:59:59Z"  // 可选
}
```

**响应**:

```json
{
  "id": "...",
  "name": "cli-key",
  "key": "sk_ik0uga3c3zTe9MBk_...",
  "prefix": "sk_ik0ug",
  "expires_at": "2026-12-31T23:59:59Z",
  "created_at": "2026-03-01T10:00:00Z"
}
```

**注意**: `key` 只在创建时返回，请妥善保存

#### 撤销 API Key

```http
DELETE /api/v1/user/api-keys/:id
Authorization: Bearer <token>
```

## 数据模型

### Skill

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 唯一标识 |
| namespace | string | 命名空间 |
| name | string | 技能名称 |
| owner_username | string | 所有者用户名 |
| owner_display_name_zh | string | 所有者中文名 |
| description | string | 描述 |
| category | string | 官方一级分类 |
| tags | []string | 官方标签，1-4 个 |
| license | string | 许可证 |
| is_public | bool | 是否公开 |
| is_recommended | bool | 是否推荐 |
| download_count | int | 下载次数 |
| like_count | int | 点赞数 |
| install_count | int | 安装事件数 |
| viewer_liked | bool | 当前用户是否已点赞，详情接口返回 |
| rating | float64 | 平均评分 |
| rating_count | int | 评分次数 |
| latest_version | string | 最新版本 |

### SkillVersion

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 唯一标识 |
| skill_id | UUID | 所属技能 |
| version | string | 版本号 |
| size_bytes | int64 | 文件大小 |
| scan_status | string | 扫描状态 (pass/warn/fail) |
| scan_result | object | 扫描结果 |
| published_at | datetime | 发布时间 |

### User

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 唯一标识 |
| username | string | 用户名 |
| display_name_zh | string | 中文名 |
| email | string | 邮箱 |
| avatar_url | string | 头像 URL |
| role | string | 管理接口返回的派生角色：member/admin/super_admin |
| is_active | bool | 是否启用 |
| is_admin | bool | 是否管理员 |
| is_super_admin | bool | 是否超级管理员 |
| created_at | datetime | 创建时间 |

## 示例

### 完整发布流程

```bash
# 1. 登录获取 Token
TOKEN=$(curl -s -X POST https://soulstore.ciqtek.com/skill-home/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"testpass123"}' \
  | jq -r '.token')

# 2. 创建技能包
cat > SKILL.md << 'EOF'
---
name: my-skill
version: 1.0.0
description: 我的技能
category: productivity
tags:
  - workflow
---

# 我的技能

这是一个示例技能。
EOF
tar -czf skill.tar.gz SKILL.md

# 3. 发布技能
curl -X POST https://soulstore.ciqtek.com/skill-home/api/v1/skills \
  -H "Authorization: Bearer $TOKEN" \
  -F "namespace=testuser" \
  -F "name=my-skill" \
  -F "description=我的技能" \
  -F "category=productivity" \
  -F "version=1.0.0" \
  -F "tags=workflow" \
  -F "skill=@skill.tar.gz"

# 4. 搜索技能
curl "https://soulstore.ciqtek.com/skill-home/api/v1/search?q=my-skill"

# 5. 下载技能
curl -L -o my-skill-1.0.0.zip \
  "https://oss-example.aliyuncs.com/public-skills/testuser/my-skill/1.0.0.zip"

# 兼容旧客户端时，仍可继续使用服务端下载入口
curl -L -o my-skill-1.0.0.zip \
  "https://soulstore.ciqtek.com/skill-home/api/v1/download/testuser/my-skill/1.0.0"
```

## 限流

- 公开接口: 100 请求/分钟
- 认证接口: 1000 请求/分钟
- 下载接口: 60 请求/分钟

## 版本历史

| 版本 | 日期 | 说明 |
|------|------|------|
| 1.0.0 | 2026-03-01 | 初始版本，基础功能完成 |
