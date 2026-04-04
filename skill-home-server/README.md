# skill-home Server

`skill-home-server` 是 Skill Home 的注册中心与分发编排服务，负责技能元数据、目录版本、认证、版本管理、下载入口和安装页托管。

## 能力概览

| 能力域 | 说明 |
|--------|------|
| 目录服务 | 公开 skill 列表、详情、版本列表、搜索 |
| 目录版本 | 提供 `catalog/version`，供 CLI 判断是否需要刷新远程目录缓存 |
| 认证与权限 | 用户注册、密码登录、JWT 会话、API Key |
| 技能治理 | 创建技能、更新元数据、发布版本、删除技能、删除版本 |
| 管理员 | 可维护公开目录中的推荐 skill |
| 超级管理员 | 可管理任意 skill/版本，并可修改其他用户的密码、启停状态和超管权限 |
| 下载编排 | 公开 zip skill 优先返回 OSS 直链；兼容保留 `/api/v1/download/...` |
| 运营能力 | 评分、审计日志、用户技能列表 |
| 安装页托管 | `install.sh` 与 `/releases/*` 由服务端统一挂载 |

## 当前入口

- 部署入口：`https://soulstore.ciqtek.com/skill-home`
- 本地开发：`http://localhost:8080`
- API 前缀：`/api/v1`

## 公开接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/health` | 健康检查 |
| GET | `/api/v1/catalog/version` | 获取公开目录版本号 |
| POST | `/api/v1/auth/register` | 用户注册 |
| POST | `/api/v1/auth/login` | 用户登录 |
| GET | `/api/v1/skills` | 列出公开 skill |
| GET | `/api/v1/skills/:namespace/:name` | 获取 skill 详情 |
| GET | `/api/v1/skills/:namespace/:name/versions` | 获取版本列表 |
| GET | `/api/v1/search` | 搜索公开 skill |
| GET | `/api/v1/download/:namespace/:name/:version` | 兼容下载入口 |
| GET | `/install.sh` | CLI 安装脚本 |
| GET | `/releases/*assetPath` | CLI 发布产物托管 |

说明：

- `GET /api/v1/skills/:namespace/:name` 和 `/versions` 在公开 skill 场景下可匿名访问
- 公开 zip skill 的 `download_url` 会优先返回 OSS 绝对地址
- `/api/v1/download/...` 仍保留，兼容旧 CLI 和显式版本下载
- `credentials` 当前只在详情接口 `GET /api/v1/skills/:namespace/:name` 返回；列表和搜索接口首期不返回该字段

## 需要认证的接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/user` | 当前用户 |
| GET | `/api/v1/user/skills` | 当前用户的 skill 列表 |
| GET | `/api/v1/user/audit-logs` | 最近审计日志 |
| GET | `/api/v1/user/api-keys` | API Key 列表 |
| POST | `/api/v1/user/api-keys` | 创建 API Key |
| DELETE | `/api/v1/user/api-keys/:id` | 撤销 API Key |
| POST | `/api/v1/skills` | 创建技能并发布首个版本 |
| PUT | `/api/v1/skills/:namespace/:name` | 更新技能元数据 |
| DELETE | `/api/v1/skills/:namespace/:name` | 删除技能 |
| POST | `/api/v1/skills/:namespace/:name/versions` | 发布新版本 |
| DELETE | `/api/v1/skills/:namespace/:name/versions/:version` | 删除版本 |
| POST | `/api/v1/skills/:namespace/:name/rating` | 为技能评分 |
| GET | `/api/v1/admin/users` | 超级管理员查看用户列表 |
| PUT | `/api/v1/admin/users/:id` | 超级管理员修改用户密码、权限、启停状态 |
| PATCH | `/api/v1/admin/skills/:namespace/:name/recommendation` | 管理员或超级管理员更新推荐状态 |

说明：

- 管理员和超级管理员可以设置公开 skill 的推荐状态
- 超级管理员可以发布、修改、删除任意用户名下的 skill 和版本
- 超级管理员可以重置其他用户密码，调整 `is_admin`、`is_super_admin` 与 `is_active`
- 当前用户接口 `/api/v1/user`、登录响应、注册响应都会返回 `is_admin` 与 `is_super_admin`

## 超级管理员引导

服务端支持启动时按用户名引导一个超级管理员：

```bash
export SKILL_HOME_AUTH_BOOTSTRAP_SUPER_ADMIN=zhuyuxiao314
```

服务启动并完成数据库迁移后，会把该用户名对应的用户记录写成 `is_super_admin=true`。这个操作是幂等的，适合首次开通或灾备恢复。

## 接口示例

### 1. 健康检查

```bash
curl -fsS https://soulstore.ciqtek.com/skill-home/health
```

示例响应：

```json
{"service":"skill-home-registry","status":"ok","version":"main-b523062"}
```

### 2. 获取目录版本

```bash
curl -fsS https://soulstore.ciqtek.com/skill-home/api/v1/catalog/version
```

示例响应：

```json
{"catalog_version":3,"updated_at":"2026-03-30T18:55:34.57511+08:00"}
```

### 3. 列出公开 skill

```bash
curl -fsS "https://soulstore.ciqtek.com/skill-home/api/v1/skills?page=1&per_page=20"
```

按标签或关键字筛选：

```bash
curl -fsS "https://soulstore.ciqtek.com/skill-home/api/v1/skills?q=codex&tag=skill-home"
```

### 4. 获取 skill 详情

```bash
curl -fsS https://soulstore.ciqtek.com/skill-home/api/v1/skills/skill-home/skill-home-manager
```

你会拿到：

- `latest_version`
- 顶层 `download_url`
- 各版本 `versions[].download_url`
- 顶层 `credentials`，它来自最新版本 `manifest.metadata.openclaw.credentials`
- `is_public`、`is_recommended`、`download_count`、`rating`

示例片段：

```json
{
  "namespace": "team",
  "name": "demo-skill",
  "latest_version": "1.0.1",
  "download_url": "https://skills-static.example.com/skills/team/demo-skill/latest.zip",
  "credentials": [
    {
      "id": "openai_api_key",
      "env": "OPENAI_API_KEY",
      "label": "OpenAI API Key",
      "description": "用于访问 OpenAI 接口",
      "secret": true,
      "required": true,
      "input": "password",
      "help_url": "https://platform.openai.com/api-keys",
      "group": "llm_provider"
    }
  ],
  "versions": [
    {
      "version": "1.0.1",
      "manifest": {
        "name": "demo-skill",
        "version": "1.0.1",
        "description": "Demo skill",
        "requires": ["OPENAI_API_KEY"],
        "metadata": {
          "openclaw": {
            "credentials": [
              {
                "id": "openai_api_key",
                "env": "OPENAI_API_KEY"
              }
            ]
          }
        }
      }
    }
  ]
}
```

### 5. 搜索公开 skill

```bash
curl -fsS "https://soulstore.ciqtek.com/skill-home/api/v1/search?q=skill&tag=codex"
```

### 6. 通过兼容入口下载指定版本

```bash
curl -fL "https://soulstore.ciqtek.com/skill-home/api/v1/download/skill-home/skill-home-manager/0.2.5?format=zip" -o skill-home-manager.zip
```

### 7. 注册和登录

```bash
curl -X POST https://soulstore.ciqtek.com/skill-home/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"yourname","email":"you@example.com","password":"yourpass"}'

curl -X POST https://soulstore.ciqtek.com/skill-home/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"you@example.com","password":"yourpass"}'
```

### 8. 用 Bearer Token 获取当前用户

```bash
export TOKEN="your-jwt-token"

curl -fsS https://soulstore.ciqtek.com/skill-home/api/v1/user \
  -H "Authorization: Bearer ${TOKEN}"
```

### 9. 创建 API Key

```bash
curl -X POST https://soulstore.ciqtek.com/skill-home/api/v1/user/api-keys \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"name":"cli"}'
```

### 10. 通过 multipart 创建 skill

```bash
curl -X POST https://soulstore.ciqtek.com/skill-home/api/v1/skills \
  -H "Authorization: Bearer ${TOKEN}" \
  -F "namespace=team" \
  -F "name=demo-skill" \
  -F "version=1.0.0" \
  -F "is_public=true" \
  -F "skill=@./demo-skill.zip;type=application/zip"
```

### 11. 发布 skill 新版本

```bash
curl -X POST https://soulstore.ciqtek.com/skill-home/api/v1/skills/team/demo-skill/versions \
  -H "Authorization: Bearer ${TOKEN}" \
  -F "version=1.0.1" \
  -F "skill=@./demo-skill-v1.0.1.zip;type=application/zip"
```

## Skill Manifest 与凭证约定

### 归档与解析范围

- `POST /api/v1/skills` 和 `POST /api/v1/skills/:namespace/:name/versions` 接收 `.zip` 或 `.tar.gz`
- 服务端会在上传时 best-effort 读取归档中的 `SKILL.md` frontmatter，并把解析结果写入 `skill_versions.manifest`
- 如果 `SKILL.md` 缺失、没有 frontmatter，或 frontmatter YAML 解析失败，发布不会因此被阻塞；只是 `manifest` / `credentials` 不会被自动补齐

### `metadata.openclaw.credentials`

如果 `SKILL.md` 包含下面这段：

```yaml
---
name: demo-skill
version: 1.0.1
description: Demo skill
metadata:
  openclaw:
    credentials:
      - id: openai_api_key
        env: OPENAI_API_KEY
        label: OpenAI API Key
        description: 用于访问 OpenAI 接口
        secret: true
        required: true
        input: password
        help_url: https://platform.openai.com/api-keys
        group: llm_provider
---
```

则服务端会：

1. 把整段 frontmatter 持久化到对应版本的 `manifest`
2. 在详情接口顶层额外返回 `credentials`
3. 保持各版本里的 `manifest.metadata.openclaw.credentials` 原样可读

### `credentials` 字段约定

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 凭证项稳定标识 |
| `env` | string | 运行时注入的环境变量名 |
| `label` | string | 面向用户的显示名称 |
| `description` | string | 简短说明 |
| `secret` | boolean | 是否按敏感值处理 |
| `required` | boolean | 是否必填 |
| `input` | string | 建议的输入类型，例如 `password` / `text` / `url` |
| `help_url` | string | 获取凭证的帮助链接 |
| `group` | string | 分组标识，用于表达同一类 provider/凭证组 |

### `requires` 兼容语义

- 新客户端应优先消费详情接口顶层的 `credentials`
- 旧客户端如果只认识 `manifest.requires`，仍然可以继续工作
- 当 `SKILL.md` 已显式声明 `requires` 时，服务端按原值保存
- 当 `SKILL.md` 没有 `requires`，但存在 `metadata.openclaw.credentials` 时，服务端会自动从 `credentials[].env` 派生 `manifest.requires`

兼容派生示例：

```yaml
metadata:
  openclaw:
    credentials:
      - id: openai_api_key
        env: OPENAI_API_KEY
      - id: anthropic_api_key
        env: ANTHROPIC_API_KEY
```

会生成：

```json
{
  "requires": ["OPENAI_API_KEY", "ANTHROPIC_API_KEY"]
}
```

## 运行与开发

### Docker Compose

```bash
cd deployments/docker
docker-compose up -d
```

默认端口：

- API：`http://localhost:8080`
- MinIO Console：`http://localhost:9001`
- PostgreSQL：`localhost:5432`

### 本地开发

```bash
go mod download

export SKILL_HOME_DATABASE_PASSWORD=your-password
export SKILL_HOME_AUTH_JWT_SECRET=your-secret

go run cmd/server/main.go
```

查看版本信息：

```bash
go run cmd/server/main.go version
```

## 环境变量

| 变量名 | 描述 | 默认值 |
|--------|------|--------|
| `SKILL_HOME_SERVER_PORT` | 服务端口 | 8080 |
| `SKILL_HOME_SERVER_MODE` | 运行模式（`development` / `production`） | `development` |
| `SKILL_HOME_SERVER_BASE_PATH` | 服务挂载前缀，例如 `/skill-home` | 空 |
| `SKILL_HOME_DATABASE_HOST` | 数据库主机 | localhost |
| `SKILL_HOME_DATABASE_PORT` | 数据库端口 | 5432 |
| `SKILL_HOME_DATABASE_USER` | 数据库用户 | skillhome |
| `SKILL_HOME_DATABASE_PASSWORD` | 数据库密码 | - |
| `SKILL_HOME_DATABASE_NAME` | 数据库名 | skillhome |
| `SKILL_HOME_DATABASE_SSL_MODE` | PostgreSQL SSL 模式 | `disable` |
| `SKILL_HOME_STORAGE_TYPE` | 存储类型（`minio` / `s3` / `local`） | local |
| `SKILL_HOME_STORAGE_ENDPOINT` | MinIO/S3/OSS endpoint，不带协议 | - |
| `SKILL_HOME_STORAGE_ACCESS_KEY` | 存储访问密钥 | - |
| `SKILL_HOME_STORAGE_SECRET_KEY` | 存储密钥 | - |
| `SKILL_HOME_STORAGE_BUCKET` | 对象存储 bucket | `skill-home` |
| `SKILL_HOME_STORAGE_REGION` | 对象存储 region | 空 |
| `SKILL_HOME_STORAGE_USE_SSL` | 对象存储是否启用 SSL | false |
| `SKILL_HOME_STORAGE_PUBLIC_BASE_URL` | 公共对象根地址，公开 zip skill 会返回绝对 `download_url` | 空 |
| `SKILL_HOME_AUTH_JWT_SECRET` | JWT 密钥 | `dev-secret` |

## OSS 兼容读取

如果部署侧已经维护了 `SOULSTORE_SKILL_OSS_*`，当前服务端会在 `SKILL_HOME_STORAGE_*` 缺失时把它们作为 fallback：

- `SOULSTORE_SKILL_OSS_INTERNAL_ENDPOINT`
- `SOULSTORE_SKILL_OSS_PUBLIC_CNAME_ENDPOINT`
- `SOULSTORE_SKILL_OSS_ACCESS_KEY_ID`
- `SOULSTORE_SKILL_OSS_ACCESS_KEY_SECRET`
- `SOULSTORE_SKILL_OSS_BUCKET_RELEASE`
- `SOULSTORE_SKILL_OSS_REGION`
- `SOULSTORE_SKILL_OSS_USE_INTERNAL_ENDPOINT`

建议生产环境最终仍落到标准化的 `SKILL_HOME_STORAGE_*`，兼容层主要用于迁移期。

## 历史公共包回填

如果公开 skill 已从 MinIO 迁到 OSS，但历史对象还留在旧存储里，可以使用：

```bash
go run ./cmd/public-oss-backfill --dry-run
go run ./cmd/public-oss-backfill
```

典型源配置：

```bash
export SKILL_HOME_BACKFILL_SOURCE_TYPE=minio
export SKILL_HOME_BACKFILL_SOURCE_ENDPOINT=localhost:19000
export SKILL_HOME_BACKFILL_SOURCE_ACCESS_KEY=minioadmin
export SKILL_HOME_BACKFILL_SOURCE_SECRET_KEY=minioadmin
export SKILL_HOME_BACKFILL_SOURCE_BUCKET=skill-home
export SKILL_HOME_BACKFILL_SOURCE_USE_SSL=false
```

建议顺序：

1. 先执行 `--dry-run`
2. 确认源对象和目标 OSS 都可访问
3. 正式执行回填
4. 回填完成后，再把运行时存储切到 OSS

## 许可证

MIT License
