# skill-home Server

AI Skill 注册中心后端服务

## 技术栈

- **Web 框架**: Gin
- **数据库**: PostgreSQL + GORM
- **对象存储**: OSS / S3 / MinIO (S3 兼容)
- **缓存**: Redis
- **认证**: JWT + API Key

## 快速开始

### 使用 Docker Compose

```bash
cd deployments/docker
docker-compose up -d
```

服务将启动在:
- API: http://localhost:8080
- MinIO Console: http://localhost:9001 (minioadmin/minioadmin)
- PostgreSQL: localhost:5432

### 本地开发

```bash
# 安装依赖
go mod download

# 设置环境变量
export SKILL_HOME_DATABASE_PASSWORD=your-password
export SKILL_HOME_AUTH_JWT_SECRET=your-secret

# 运行
go run cmd/server/main.go
```

## API 文档

### 公开接口

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /health | 健康检查 |
| GET | /api/v1/skills | 列出技能 |
| GET | /api/v1/skills/:namespace/:name | 获取技能详情 |
| GET | /api/v1/skills/:namespace/:name/versions | 列出版本 |
| GET | /api/v1/search?q=keyword | 搜索技能 |
| GET | /api/v1/download/:namespace/:name/:version | 下载技能 |

### 需要认证

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /api/v1/user | 当前用户 |
| GET | /api/v1/user/skills | 我的技能 |
| GET | /api/v1/user/api-keys | 我的 API Key 列表 |
| POST | /api/v1/user/api-keys | 创建 API Key |
| DELETE | /api/v1/user/api-keys/:id | 撤销 API Key |
| POST | /api/v1/skills | 创建技能 |
| PUT | /api/v1/skills/:namespace/:name | 更新技能 |
| DELETE | /api/v1/skills/:namespace/:name | 删除技能 |
| POST | /api/v1/skills/:namespace/:name/versions | 发布版本 |
| DELETE | /api/v1/skills/:namespace/:name/versions/:version | 删除版本 |

## 环境变量

| 变量名 | 描述 | 默认值 |
|--------|------|--------|
| `SKILL_HOME_SERVER_PORT` | 服务端口 | 8080 |
| `SKILL_HOME_SERVER_BASE_PATH` | 服务挂载前缀，例如 `/skill-home` | 空 |
| `SKILL_HOME_DATABASE_HOST` | 数据库主机 | localhost |
| `SKILL_HOME_DATABASE_PORT` | 数据库端口 | 5432 |
| `SKILL_HOME_DATABASE_USER` | 数据库用户 | skillhome |
| `SKILL_HOME_DATABASE_PASSWORD` | 数据库密码 | - |
| `SKILL_HOME_DATABASE_NAME` | 数据库名 | skillhome |
| `SKILL_HOME_STORAGE_TYPE` | 存储类型 (`minio` / `s3` / `local`) | local |
| `SKILL_HOME_STORAGE_ENDPOINT` | MinIO/S3/OSS endpoint，不带协议 | - |
| `SKILL_HOME_STORAGE_ACCESS_KEY` | 存储访问密钥 | - |
| `SKILL_HOME_STORAGE_SECRET_KEY` | 存储密钥 | - |
| `SKILL_HOME_STORAGE_BUCKET` | 对象存储 bucket | `skill-home` |
| `SKILL_HOME_STORAGE_REGION` | 对象存储 region | 空 |
| `SKILL_HOME_STORAGE_USE_SSL` | 对象存储是否启用 SSL | false |
| `SKILL_HOME_STORAGE_PUBLIC_BASE_URL` | 公共对象根地址，公开 skill 会据此返回绝对 `download_url` | 空 |
| `SKILL_HOME_AUTH_JWT_SECRET` | JWT 密钥 | dev-secret |

### OSS 兼容读取

如果部署侧已经维护了 `SOULSTORE_SKILL_OSS_*`，当前服务端会在 `SKILL_HOME_STORAGE_*` 缺失时，把它们作为 OSS 存储配置 fallback：

- `SOULSTORE_SKILL_OSS_INTERNAL_ENDPOINT`
- `SOULSTORE_SKILL_OSS_PUBLIC_CNAME_ENDPOINT`
- `SOULSTORE_SKILL_OSS_ACCESS_KEY_ID`
- `SOULSTORE_SKILL_OSS_ACCESS_KEY_SECRET`
- `SOULSTORE_SKILL_OSS_BUCKET_RELEASE`
- `SOULSTORE_SKILL_OSS_REGION`
- `SOULSTORE_SKILL_OSS_USE_INTERNAL_ENDPOINT`

建议：
- 代码兼容层可以复用这套变量，减少迁移时重复填写
- 生产部署最终仍建议把标准化后的结果写成 `SKILL_HOME_STORAGE_*`

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
