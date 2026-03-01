# Skill-Home

AI Skill 跨平台管理工具链，实现 AI 技能在 Claude Code、GitHub Copilot、Codex、Cursor 之间的统一管理与同步。

## 项目概述

Skill-Home 是一个完整的 AI Skill 生态系统，包含：

- **注册中心 (Registry)**: 云端技能存储与分发
- **CLI 工具**: 本地技能管理与 IDE 同步
- **多 IDE 支持**: Claude Code、Cursor、Codex 等

## 快速开始

### 使用已部署的服务

```bash
# 1. 下载 CLI
scp root@47.122.112.210:/opt/skill-home/skill-home /usr/local/bin/

# 2. 注册账号
curl -X POST http://47.122.112.210:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"yourname","email":"you@example.com","password":"yourpass"}'

# 3. 创建技能
skill-home init my-first-skill
cd my-first-skill

# 4. 编辑 SKILL.md，然后打包上传
skill-home pack -o skill.tar.gz .
curl -X POST http://47.122.112.210:8080/api/v1/skills \
  -H "Authorization: Bearer <your-token>" \
  -F "skill=@skill.tar.gz"
```

## 项目结构

```
skillManage/
├── skill-home-server/      # 注册中心服务端 (Go + Gin)
│   ├── cmd/server/         # 程序入口
│   ├── internal/           # 内部实现
│   │   ├── api/            # HTTP 接口
│   │   ├── models/         # 数据模型
│   │   ├── storage/        # 存储层 (PostgreSQL + MinIO)
│   │   └── config/         # 配置管理
│   └── deployments/docker/ # Docker 部署
│
├── skill-home-cli/         # 命令行工具 (Go + Cobra)
│   ├── cmd/skill-home/     # 程序入口
│   ├── internal/           # 内部实现
│   │   ├── cmd/            # 子命令
│   │   ├── registry/       # 注册中心客户端
│   │   ├── ide/            # IDE 适配器
│   │   └── sync/           # 同步引擎
│   └── pkg/validator/      # 安全扫描
│
├── 技术规格文档.md         # 详细技术设计
├── 需求梳理.md             # 需求文档
└── DEPLOYMENT.md           # 部署记录
```

## 核心功能

### 服务端 (skill-home-server)

| 功能 | 状态 | 说明 |
|------|------|------|
| 用户认证 | ✅ | JWT + 密码登录 |
| API Key | ✅ | 命令行工具认证 |
| 技能管理 | ✅ | CRUD + 版本控制 |
| 对象存储 | ✅ | MinIO 集成 |
| 安全扫描 | ✅ | 基础规则检测 |

### 客户端 (skill-home-cli)

| 功能 | 状态 | 说明 |
|------|------|------|
| 技能初始化 | ✅ | SKILL.md 模板生成 |
| 格式验证 | ✅ | YAML 前端验证 |
| 安全扫描 | ✅ | 本地规则检测 |
| 技能打包 | ✅ | tar.gz 压缩 |
| IDE 同步 | 🚧 | Claude/Cursor/Codex |
| 注册中心 | ✅ | 推送/拉取/搜索 |

## API 端点

### 公开接口

```
GET    /health                          健康检查
POST   /api/v1/auth/register            用户注册
POST   /api/v1/auth/login               用户登录
GET    /api/v1/skills                   技能列表
GET    /api/v1/skills/:ns/:name         技能详情
GET    /api/v1/skills/:ns/:name/versions 版本列表
GET    /api/v1/search?q=keyword         搜索技能
GET    /api/v1/download/:ns/:name/:ver  下载技能
```

### 认证接口

```
GET    /api/v1/user                     当前用户
GET    /api/v1/user/skills              我的技能
POST   /api/v1/user/api-keys            创建 API Key
DELETE /api/v1/user/api-keys/:id        撤销 API Key
POST   /api/v1/skills                   发布技能
POST   /api/v1/skills/:ns/:name/versions 发布版本
```

完整 API 文档见 [API.md](API.md)

## 部署架构

```
┌─────────────────────────────────────────┐
│              用户层 (User Layer)         │
│         skill-home CLI 工具              │
└──────────────┬──────────────────────────┘
               │ HTTPS/JSON API
               ▼
┌─────────────────────────────────────────┐
│           注册中心 (Registry)            │
│        Go + Gin (Port 8080)              │
│  • Auth API    • Skill API              │
│  • Search API  • Storage API            │
└──────┬────────────┬────────────┬────────┘
       │            │            │
       ▼            ▼            ▼
┌──────────┐ ┌──────────┐ ┌──────────────┐
│PostgreSQL│ │  Redis   │ │    MinIO     │
│  15432   │ │  16379   │ │ 19000/19001  │
└──────────┘ └──────────┘ └──────────────┘
```

## 环境要求

- **服务端**: Go 1.21+, PostgreSQL 15+, Redis 7+, MinIO
- **客户端**: Go 1.21+ 或预编译二进制
- **IDE**: Claude Code / Cursor / Codex

## 开发指南

### 服务端开发

```bash
cd skill-home-server

# 安装依赖
go mod download

# 设置环境变量
export SKILL_HOME_DATABASE_PASSWORD=xxx
export SKILL_HOME_AUTH_JWT_SECRET=xxx
export SKILL_HOME_STORAGE_SECRET_KEY=xxx

# 运行
go run cmd/server/main.go

# 构建
go build -o server cmd/server/main.go
```

### CLI 开发

```bash
cd skill-home-cli

# 安装依赖
go mod download

# 运行
go run cmd/skill-home/main.go --help

# 构建
go build -o skill-home cmd/skill-home/main.go
```

## 部署

详见 [DEPLOYMENT.md](DEPLOYMENT.md)

快速部署：

```bash
# 服务器上
cd /opt/skill-home
docker-compose up -d  # 启动基础设施
systemctl start skill-home  # 启动 API 服务
```

## 文档

| 文档 | 说明 |
|------|------|
| [DEPLOYMENT.md](DEPLOYMENT.md) | 部署记录与配置 |
| [API.md](API.md) | API 详细文档 |
| [技术规格文档.md](技术规格文档.md) | 技术设计文档 |
| [需求梳理.md](需求梳理.md) | 需求分析 |
| skill-home-server/README.md | 服务端说明 |
| skill-home-cli/README.md | CLI 说明 |

## 贡献

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/xxx`)
3. 提交更改 (`git commit -am 'Add xxx'`)
4. 推送分支 (`git push origin feature/xxx`)
5. 创建 Pull Request

## 许可证

MIT License

## 联系

- 项目主页: http://47.122.112.210:8080
- MinIO 控制台: http://47.122.112.210:19001
