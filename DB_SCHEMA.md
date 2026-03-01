# 数据库设计文档

## 概述

- **数据库**: PostgreSQL 15+
- **ORM**: GORM v2
- **UUID 生成**: gen_random_uuid()

## 表结构

### users (用户表)

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | UUID | PK, default: gen_random_uuid() | 用户 ID |
| username | VARCHAR(32) | NOT NULL, UNIQUE | 用户名 |
| email | VARCHAR(255) | NOT NULL, UNIQUE | 邮箱 |
| password | VARCHAR(255) | NOT NULL | 密码哈希 |
| avatar_url | VARCHAR(500) | | 头像 URL |
| is_active | BOOLEAN | DEFAULT: true | 是否激活 |
| created_at | TIMESTAMP | NOT NULL | 创建时间 |
| updated_at | TIMESTAMP | NOT NULL | 更新时间 |
| deleted_at | TIMESTAMP | INDEX | 软删除 |

**索引**:
- PRIMARY KEY (id)
- UNIQUE INDEX (username)
- UNIQUE INDEX (email)
- INDEX (deleted_at)

### skills (技能表)

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | UUID | PK, default: gen_random_uuid() | 技能 ID |
| namespace | VARCHAR(64) | NOT NULL, part of UK | 命名空间 |
| name | VARCHAR(64) | NOT NULL, part of UK | 技能名称 |
| owner_id | UUID | NOT NULL, FK -> users.id | 所有者 |
| description | VARCHAR(500) | | 描述 |
| description_zh | VARCHAR(500) | | 中文描述 |
| author | VARCHAR(255) | | 作者名 |
| tags | TEXT[] | | 标签数组 |
| license | VARCHAR(50) | | 许可证 |
| homepage | VARCHAR(500) | | 主页 |
| repository | VARCHAR(500) | | 仓库地址 |
| download_count | BIGINT | DEFAULT: 0 | 下载次数 |
| rating_sum | BIGINT | DEFAULT: 0 | 评分总和 |
| rating_count | BIGINT | DEFAULT: 0 | 评分次数 |
| is_public | BOOLEAN | DEFAULT: true | 是否公开 |
| is_deprecated | BOOLEAN | DEFAULT: false | 是否弃用 |
| latest_version | VARCHAR(20) | | 最新版本 |
| created_at | TIMESTAMP | NOT NULL | 创建时间 |
| updated_at | TIMESTAMP | NOT NULL | 更新时间 |
| deleted_at | TIMESTAMP | INDEX | 软删除 |

**索引**:
- PRIMARY KEY (id)
- UNIQUE INDEX idx_namespace_name (namespace, name)
- INDEX (owner_id)
- INDEX (deleted_at)

### skill_versions (技能版本表)

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | UUID | PK, default: gen_random_uuid() | 版本 ID |
| skill_id | UUID | NOT NULL, FK -> skills.id, part of UK | 所属技能 |
| version | VARCHAR(20) | NOT NULL, part of UK | 版本号 |
| manifest | JSONB | | 技能清单 |
| dependencies | TEXT[] | | 依赖数组 |
| storage_path | VARCHAR(500) | NOT NULL | 存储路径 |
| size_bytes | BIGINT | NOT NULL | 文件大小 |
| checksum | VARCHAR(64) | | 校验和 |
| scan_status | VARCHAR(20) | DEFAULT: 'pending' | 扫描状态 |
| scan_result | JSONB | | 扫描结果 |
| published_by | UUID | NOT NULL, FK -> users.id | 发布者 |
| published_at | TIMESTAMP | NOT NULL | 发布时间 |
| created_at | TIMESTAMP | NOT NULL | 创建时间 |
| deleted_at | TIMESTAMP | INDEX | 软删除 |

**索引**:
- PRIMARY KEY (id)
- UNIQUE INDEX idx_skill_version (skill_id, version)
- INDEX (skill_id)
- INDEX (deleted_at)

### api_keys (API Key 表)

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | UUID | PK, default: gen_random_uuid() | Key ID |
| user_id | UUID | NOT NULL, FK -> users.id | 所属用户 |
| key_hash | VARCHAR(255) | NOT NULL | Key 哈希 |
| name | VARCHAR(100) | | 名称 |
| prefix | VARCHAR(16) | | Key 前缀 |
| last_used_at | TIMESTAMPTZ | | 最后使用时间 |
| expires_at | TIMESTAMPTZ | | 过期时间 |
| created_at | TIMESTAMPTZ | NOT NULL | 创建时间 |
| deleted_at | TIMESTAMPTZ | INDEX | 软删除 |

**索引**:
- PRIMARY KEY (id)
- INDEX (user_id)
- INDEX (deleted_at)

### audit_logs (审计日志表)

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | UUID | PK, default: gen_random_uuid() | 日志 ID |
| user_id | UUID | NULLABLE, FK -> users.id | 操作用户 |
| action | VARCHAR(50) | NOT NULL | 操作类型 |
| resource_type | VARCHAR(50) | NOT NULL | 资源类型 |
| resource_id | UUID | | 资源 ID |
| metadata | JSONB | | 元数据 |
| ip_address | VARCHAR(45) | | IP 地址 |
| user_agent | VARCHAR(500) | | User Agent |
| created_at | TIMESTAMP | NOT NULL | 创建时间 |

**索引**:
- PRIMARY KEY (id)
- INDEX (user_id)
- INDEX (resource_type, resource_id)
- INDEX (created_at)

### skill_ratings (技能评分表)

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | UUID | PK, default: gen_random_uuid() | 评分 ID |
| skill_id | UUID | NOT NULL, FK -> skills.id | 技能 ID |
| user_id | UUID | NOT NULL, FK -> users.id | 用户 ID |
| rating | INTEGER | NOT NULL | 评分 (1-5) |
| comment | VARCHAR(1000) | | 评论 |
| created_at | TIMESTAMP | NOT NULL | 创建时间 |
| updated_at | TIMESTAMP | NOT NULL | 更新时间 |

**索引**:
- PRIMARY KEY (id)
- UNIQUE INDEX (skill_id, user_id)
- INDEX (skill_id)
- INDEX (user_id)

## 关系图

```
┌─────────────┐       ┌─────────────┐       ┌─────────────────┐
│    users    │       │   skills    │       │ skill_versions  │
├─────────────┤       ├─────────────┤       ├─────────────────┤
│ id (PK)     │◄──────┤ owner_id    │◄──────┤ skill_id        │
│ username    │       │ id (PK)     │       │ id (PK)         │
│ email       │       │ namespace   │       │ version         │
│ password    │       │ name        │       │ storage_path    │
└─────────────┘       │ latest_ver  │       │ published_by    │
         │            └─────────────┘       └─────────────────┘
         │                   │                      │
         │                   │                      │
         ▼                   ▼                      ▼
┌─────────────┐       ┌─────────────┐       ┌─────────────────┐
│  api_keys   │       │skill_ratings│       │   audit_logs    │
├─────────────┤       ├─────────────┤       ├─────────────────┤
│ id (PK)     │       │ id (PK)     │       │ id (PK)         │
│ user_id     │       │ skill_id    │       │ user_id         │
│ key_hash    │       │ user_id     │       │ action          │
└─────────────┘       └─────────────┘       └─────────────────┘
```

## 自定义类型

### JSON 类型

```go
type JSON map[string]interface{}

// Scan 实现 sql.Scanner
func (j *JSON) Scan(value interface{}) error

// Value 实现 driver.Valuer
func (j JSON) Value() (driver.Value, error)
```

### StringArray 类型

```go
type StringArray []string

// Scan 实现 sql.Scanner (解析 PostgreSQL text[])
func (a *StringArray) Scan(value interface{}) error

// Value 实现 driver.Valuer (生成 PostgreSQL text[])
func (a StringArray) Value() (driver.Value, error)
```

## 迁移脚本

### 初始迁移

```sql
-- 启用 UUID 扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 用户表
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(32) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    avatar_url VARCHAR(500),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_users_deleted_at ON users(deleted_at);

-- 技能表
CREATE TABLE skills (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    namespace VARCHAR(64) NOT NULL,
    name VARCHAR(64) NOT NULL,
    owner_id UUID NOT NULL REFERENCES users(id),
    description VARCHAR(500),
    description_zh VARCHAR(500),
    author VARCHAR(255),
    tags TEXT[],
    license VARCHAR(50),
    homepage VARCHAR(500),
    repository VARCHAR(500),
    download_count BIGINT DEFAULT 0,
    rating_sum BIGINT DEFAULT 0,
    rating_count BIGINT DEFAULT 0,
    is_public BOOLEAN DEFAULT true,
    is_deprecated BOOLEAN DEFAULT false,
    latest_version VARCHAR(20),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP,
    UNIQUE(namespace, name)
);

CREATE INDEX idx_skills_owner ON skills(owner_id);
CREATE INDEX idx_skills_deleted_at ON skills(deleted_at);

-- 版本表
CREATE TABLE skill_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    skill_id UUID NOT NULL REFERENCES skills(id),
    version VARCHAR(20) NOT NULL,
    manifest JSONB,
    dependencies TEXT[],
    storage_path VARCHAR(500) NOT NULL,
    size_bytes BIGINT NOT NULL,
    checksum VARCHAR(64),
    scan_status VARCHAR(20) DEFAULT 'pending',
    scan_result JSONB,
    published_by UUID NOT NULL REFERENCES users(id),
    published_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP,
    UNIQUE(skill_id, version)
);

CREATE INDEX idx_versions_skill ON skill_versions(skill_id);
CREATE INDEX idx_versions_deleted_at ON skill_versions(deleted_at);

-- API Key 表
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    key_hash VARCHAR(255) NOT NULL,
    name VARCHAR(100),
    prefix VARCHAR(16),
    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_api_keys_user ON api_keys(user_id);
```

## 性能优化

### 查询优化

1. **搜索技能**:
```sql
-- 使用 ILIKE 进行不区分大小写的搜索
CREATE INDEX idx_skills_search ON skills USING gin(to_tsvector('simple', name || ' ' || COALESCE(description, '')));
```

2. **热门技能**:
```sql
-- 预计算下载次数索引
CREATE INDEX idx_skills_popular ON skills(download_count DESC) WHERE is_public = true;
```

### 分区策略

对于大型部署，考虑以下分区:

1. **audit_logs 按时间分区**:
```sql
CREATE TABLE audit_logs_2026_03 PARTITION OF audit_logs
    FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');
```

2. **skill_versions 按 skill_id 哈希分区**:
```sql
CREATE TABLE skill_versions_hash PARTITION OF skill_versions
    FOR VALUES WITH (MODULUS 4, REMAINDER 0);
```

## 备份策略

### 自动备份

```bash
# 每日备份
pg_dump -h localhost -U skillhome skillhome > backup_$(date +%Y%m%d).sql

# 保留最近 7 天
find . -name "backup_*.sql" -mtime +7 -delete
```

### 恢复

```bash
# 恢复数据
psql -h localhost -U skillhome -d skillhome < backup_20260301.sql
```
</invoke>