# skill-home Server

AI Skill 注册中心后端服务

## 技术栈

- **Web 框架**: Gin
- **数据库**: PostgreSQL + GORM
- **对象存储**: MinIO (S3 兼容)
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
| `SKILL_HOME_DATABASE_HOST` | 数据库主机 | localhost |
| `SKILL_HOME_DATABASE_PORT` | 数据库端口 | 5432 |
| `SKILL_HOME_DATABASE_USER` | 数据库用户 | skillhome |
| `SKILL_HOME_DATABASE_PASSWORD` | 数据库密码 | - |
| `SKILL_HOME_DATABASE_NAME` | 数据库名 | skillhome |
| `SKILL_HOME_STORAGE_TYPE` | 存储类型 (minio/s3/local) | local |
| `SKILL_HOME_STORAGE_ENDPOINT` | MinIO/S3 地址 | - |
| `SKILL_HOME_STORAGE_ACCESS_KEY` | 存储访问密钥 | - |
| `SKILL_HOME_STORAGE_SECRET_KEY` | 存储密钥 | - |
| `SKILL_HOME_AUTH_JWT_SECRET` | JWT 密钥 | dev-secret |

## 许可证

MIT License
