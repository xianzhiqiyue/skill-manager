# Skill-Home

AI Skill 跨平台管理工具链，实现 AI 技能在 Claude Code、GitHub Copilot、Codex、Cursor 之间的统一管理与同步。

## 项目概述

Skill-Home 是一个完整的 AI Skill 生态系统，包含：

- **注册中心 (Registry)**: 负责技能元数据管理、搜索发现、发布流程与兼容下载入口
- **CLI 工具**: 本地技能管理与 IDE 同步
- **多 IDE 支持**: Claude Code、GitHub Copilot、Cursor、Codex 等

其中公开 skill 包本体由对象存储/OSS 承载，Skill Home 服务端主要负责元数据、权限、搜索和分发地址编排。

## 快速开始

### 使用已部署的服务

```bash
# 1. 安装 CLI
curl -fsSL https://soulstore.ciqtek.com/skill-home/install.sh -o /tmp/skill-home-install.sh
bash /tmp/skill-home-install.sh

# 升级已安装 CLI
skill-home self-update

# 2. 注册账号
curl -X POST https://soulstore.ciqtek.com/skill-home/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"yourname","email":"you@example.com","password":"yourpass"}'

# 3. 创建技能
skill-home init my-first-skill
cd my-first-skill

# 4. 编辑 SKILL.md，然后打包上传
skill-home pack -o my-first-skill.zip .
skill-home push .
```

CLI 安装与自更新统一从当前 Skill Home 服务托管的 `/releases` 目录下载版本元数据、校验文件和各平台二进制。

发布和删除远程 skill 需要先登录；公开 skill 的 `pull / install / update / search / info` 不需要登录。

对于公开 skill，CLI / Web 会优先使用服务端返回的 `download_url` 直连 OSS；`/api/v1/download/...` 仍作为兼容入口保留，便于旧客户端和平滑迁移。

## 发布规范

仓库内所有 `server / web / cli / skill` 的发布顺序、影响面判断和验收口径，统一以 [docs/release-process.md](docs/release-process.md) 为准。

最小要求：

- 先判断本次改动影响哪个交付物，不默认四个组件一起发布
- 先完成版本变更、提交和 `git push origin main`，再发布对外制品
- `skill` 默认递增版本号后发布，不覆盖已有版本
- `server` 生产部署只面向 `soul正式服务器 121.40.85.95`

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
├── skills/
│   └── skill-home-manager/ # 面向 Codex / OpenClaw 等 AI 助手的 skill，封装 skill-home CLI 工作流
│
├── skill-home-web/         # Web 前端，负责宣传、搜索发现、安装引导
├── docs/                   # 设计与补充文档
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
| 对象存储 | ✅ | Skill Home 管理元数据，公开 skill 包由 OSS/对象存储承载 |
| 安全扫描 | ✅ | 基础规则检测 |
| 技能评分 | ✅ | 1-5 星评分与评论 |
| 搜索排序 | ✅ | PostgreSQL 全文搜索 + 下载量排序 |

### 客户端 (skill-home-cli)

| 功能 | 状态 | 说明 |
|------|------|------|
| 技能初始化 | ✅ | SKILL.md 模板生成 |
| 格式验证 | ✅ | YAML 前端验证 |
| 安全扫描 | ✅ | 本地规则检测 |
| 技能打包 | ✅ | ZIP 压缩 |
| IDE 同步 | ✅ | Claude/Copilot/Cursor/Codex |
| 注册中心 | ✅ | 推送/拉取/搜索/详情 |
| 生命周期管理 | ✅ | install / uninstall / update |
| 环境诊断 | ✅ | doctor 检查配置与连通性 |

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
GET    /api/v1/user/audit-logs          最近活动
POST   /api/v1/user/api-keys            创建 API Key
DELETE /api/v1/user/api-keys/:id        撤销 API Key
POST   /api/v1/skills                   发布技能
POST   /api/v1/skills/:ns/:name/versions 发布版本
DELETE /api/v1/skills/:ns/:name         删除技能
DELETE /api/v1/skills/:ns/:name/versions/:version 删除版本
POST   /api/v1/skills/:ns/:name/rating  为技能评分
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
- **IDE**: Claude Code / GitHub Copilot / Cursor / Codex

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
| [docs/release-process.md](docs/release-process.md) | 统一发布流程与验收口径 |
| [API.md](API.md) | API 详细文档 |
| [技术规格文档.md](技术规格文档.md) | 技术设计文档 |
| [需求梳理.md](需求梳理.md) | 需求分析 |
| [docs/skill-home-web-redesign-spec.md](docs/skill-home-web-redesign-spec.md) | Web 重构规格文档 |
| [docs/github-release-topology.md](docs/github-release-topology.md) | GitHub / Registry 发布拓扑 |
| [skill-home-server/README.md](skill-home-server/README.md) | 服务端说明 |
| [skill-home-cli/README.md](skill-home-cli/README.md) | CLI 说明 |
| [skills/skill-home-manager/README.md](skills/skill-home-manager/README.md) | AI 助手 skill 能力与脚本说明 |

## 贡献

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/xxx`)
3. 提交更改 (`git commit -am 'Add xxx'`)
4. 推送分支 (`git push origin feature/xxx`)
5. 创建 Pull Request

## 许可证

MIT License

## 联系

- 项目主页: https://soulstore.ciqtek.com/skill-home/
- MinIO 控制台: http://121.40.85.95:19001
