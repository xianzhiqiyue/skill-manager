# Skill-Home API 文档

## 基本信息

- **基础 URL**: `http://47.122.112.210:8080`
- **API 版本**: `v1`
- **数据格式**: JSON
- **认证方式**: Bearer Token (JWT)

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
    "email": "test@example.com"
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

#### 列出技能

```http
GET /api/v1/skills?page=1&per_page=20&q=keyword&tag=tag1
```

**参数**:

| 参数 | 类型 | 说明 |
|------|------|------|
| page | int | 页码，默认 1 |
| per_page | int | 每页数量，默认 20 |
| q | string | 搜索关键词 |
| tag | string | 标签筛选 |

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
      "author": "testuser",
      "tags": ["example", "tutorial"],
      "download_count": 42,
      "rating_count": 5,
      "latest_version": "1.0.0",
      "created_at": "2026-03-01T10:00:00Z",
      "updated_at": "2026-03-01T12:00:00Z"
    }
  ]
}
```

#### 获取技能详情

```http
GET /api/v1/skills/:namespace/:name
```

**响应**:

```json
{
  "id": "...",
  "namespace": "testuser",
  "name": "my-skill",
  "owner_id": "...",
  "description": "技能描述",
  "tags": ["example"],
  "license": "MIT",
  "download_count": 42,
  "is_public": true,
  "latest_version": "1.0.0",
  "created_at": "2026-03-01T10:00:00Z",
  "owner": {
    "id": "...",
    "username": "testuser",
    "email": "test@example.com"
  },
  "versions": [
    {
      "id": "...",
      "version": "1.0.0",
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
version: 1.0.0
license: MIT
is_public: true
skill: <文件>
```

**响应**:

```json
{
  "namespace": "testuser",
  "name": "my-skill",
  "version": "1.0.0",
  "download_url": "/api/v1/download/testuser/my-skill/1.0.0"
}
```

#### 更新技能

```http
PUT /api/v1/skills/:namespace/:name
Authorization: Bearer <token>
Content-Type: application/json

{
  "description": "新描述",
  "tags": ["new-tag"],
  "license": "Apache-2.0",
  "is_public": false
}
```

#### 删除技能

```http
DELETE /api/v1/skills/:namespace/:name
Authorization: Bearer <token>
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

#### 删除版本

```http
DELETE /api/v1/skills/:namespace/:name/versions/:version
Authorization: Bearer <token>
```

### 搜索

```http
GET /api/v1/search?q=keyword&tag=tag1&page=1&per_page=20
```

**响应**: 同列出技能

### 下载

```http
GET /api/v1/download/:namespace/:name/:version
```

**响应**: 文件流 (application/gzip)

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
| tags | []string | 标签 |
| license | string | 许可证 |
| is_public | bool | 是否公开 |
| download_count | int | 下载次数 |
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
| created_at | datetime | 创建时间 |

## 示例

### 完整发布流程

```bash
# 1. 登录获取 Token
TOKEN=$(curl -s -X POST http://47.122.112.210:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"testpass123"}' \
  | jq -r '.token')

# 2. 创建技能包
cat > SKILL.md << 'EOF'
---
name: my-skill
version: 1.0.0
description: 我的技能
tags:
  - example
---

# 我的技能

这是一个示例技能。
EOF
tar -czf skill.tar.gz SKILL.md

# 3. 发布技能
curl -X POST http://47.122.112.210:8080/api/v1/skills \
  -H "Authorization: Bearer $TOKEN" \
  -F "namespace=testuser" \
  -F "name=my-skill" \
  -F "description=我的技能" \
  -F "version=1.0.0" \
  -F "skill=@skill.tar.gz"

# 4. 搜索技能
curl "http://47.122.112.210:8080/api/v1/search?q=my-skill"

# 5. 下载技能
curl -o my-skill-1.0.0.tar.gz \
  "http://47.122.112.210:8080/api/v1/download/testuser/my-skill/1.0.0"
```

## 限流

- 公开接口: 100 请求/分钟
- 认证接口: 1000 请求/分钟
- 下载接口: 60 请求/分钟

## 版本历史

| 版本 | 日期 | 说明 |
|------|------|------|
| 1.0.0 | 2026-03-01 | 初始版本，基础功能完成 |
