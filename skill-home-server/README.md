# skill-home Server

`skill-home-server` 是 Skill Home 的注册中心与分发编排服务，负责技能元数据、目录版本、认证、版本管理、下载入口和安装页托管。

## 能力概览

| 能力域 | 说明 |
|--------|------|
| 目录服务 | 公开 skill 列表、详情、版本列表、搜索 |
| 目录版本 | 提供 `catalog/version`，供 CLI 判断是否需要刷新远程目录缓存 |
| 认证与权限 | 用户注册、密码登录、JWT 会话、API Key |
| 技能治理 | 创建技能、更新元数据、发布版本、删除技能、删除版本 |
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
- `is_public`、`download_count`、`rating`

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
