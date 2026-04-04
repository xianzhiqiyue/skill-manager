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
- `download_count`、`rating`、`rating_count` 等动态统计字段不属于该版本号覆盖范围；如果客户端需要这些字段的最新值，应主动重新拉取列表或详情接口。

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
      "description": "技能描述",
      "category": "automation",
      "author": "testuser",
      "tags": ["workflow", "automation"],
      "download_count": 42,
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
  "description": "技能描述",
  "category": "automation",
  "tags": ["workflow"],
  "license": "MIT",
  "download_count": 42,
  "rating": 4.8,
  "rating_count": 5,
  "is_recommended": true,
  "is_public": true,
  "latest_version": "1.0.0",
  "download_url": "https://oss-example.aliyuncs.com/public-skills/testuser/my-skill/1.0.0.zip",
  "created_at": "2026-03-01T10:00:00Z",
  "updated_at": "2026-03-01T12:00:00Z",
  "owner": {
    "id": "...",
    "username": "testuser",
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
| description | string | 描述 |
| category | string | 官方一级分类 |
| tags | []string | 官方标签，1-4 个 |
| license | string | 许可证 |
| is_public | bool | 是否公开 |
| is_recommended | bool | 是否推荐 |
| download_count | int | 下载次数 |
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
| email | string | 邮箱 |
| avatar_url | string | 头像 URL |
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
