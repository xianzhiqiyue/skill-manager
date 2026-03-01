# 贡献指南

感谢您对 Skill-Home 项目的关注！本文档将指导您如何参与项目开发。

## 开发环境

### 必备工具

- Go 1.21+
- Docker & Docker Compose
- PostgreSQL 15+ (或 Docker)
- Redis 7+ (或 Docker)
- MinIO (或 Docker)
- Git

### 快速搭建

```bash
# 克隆项目
git clone <repo-url>
cd skillManage

# 启动基础设施
cd skill-home-server/deployments/docker
docker-compose up -d

# 验证服务
psql -h localhost -p 5432 -U skillhome -c "SELECT 1"
redis-cli -p 6379 ping
curl http://localhost:9000/minio/health/live
```

## 项目结构

```
skillManage/
├── skill-home-server/      # 服务端
│   ├── cmd/server/         # 入口
│   ├── internal/           # 内部代码
│   │   ├── api/            # HTTP 处理器
│   │   │   ├── handlers/   # 业务逻辑
│   │   │   └── middleware/ # 中间件
│   │   ├── models/         # 数据模型
│   │   ├── storage/        # 存储层
│   │   └── config/         # 配置
│   ├── migrations/         # 数据库迁移
│   └── pkg/                # 公共包
│
├── skill-home-cli/         # CLI 工具
│   ├── cmd/skill-home/     # 入口
│   ├── internal/
│   │   ├── cmd/            # 子命令
│   │   ├── registry/       # API 客户端
│   │   ├── ide/            # IDE 适配器
│   │   └── sync/           # 同步引擎
│   └── pkg/
│       └── validator/      # 安全扫描
```

## 开发流程

### 1. 创建分支

```bash
git checkout -b feature/your-feature-name
# 或
git checkout -b fix/issue-description
```

### 2. 服务端开发

```bash
cd skill-home-server

# 安装依赖
go mod download

# 设置环境变量
export SKILL_HOME_SERVER_PORT=8080
export SKILL_HOME_DATABASE_HOST=localhost
export SKILL_HOME_DATABASE_PORT=5432
export SKILL_HOME_DATABASE_USER=skillhome
export SKILL_HOME_DATABASE_PASSWORD=yourpassword
export SKILL_HOME_DATABASE_NAME=skillhome
export SKILL_HOME_STORAGE_TYPE=minio
export SKILL_HOME_STORAGE_ENDPOINT=localhost:9000
export SKILL_HOME_STORAGE_ACCESS_KEY=minioadmin
export SKILL_HOME_STORAGE_SECRET_KEY=minioadmin
export SKILL_HOME_STORAGE_BUCKET=skill-home
export SKILL_HOME_AUTH_JWT_SECRET=your-secret-key

# 运行
go run cmd/server/main.go

# 测试
go test ./...

# 构建
go build -o server cmd/server/main.go
```

### 3. CLI 开发

```bash
cd skill-home-cli

# 安装依赖
go mod download

# 运行
go run cmd/skill-home/main.go --help

# 测试
go test ./...

# 构建
go build -o skill-home cmd/skill-home/main.go
```

### 4. 代码规范

#### Go 代码风格

- 使用 `gofmt` 格式化代码
- 使用 `golint` 检查代码
- 使用 `go vet` 静态分析

```bash
# 格式化
gofmt -w .

# 检查
golint ./...
go vet ./...

# 全部检查
go fmt ./... && go vet ./... && go test ./...
```

#### 提交信息规范

使用 [Conventional Commits](https://www.conventionalcommits.org/):

```
类型(范围): 描述

[可选的正文]

[可选的脚注]
```

类型:
- `feat`: 新功能
- `fix`: 修复
- `docs`: 文档
- `style`: 格式调整
- `refactor`: 重构
- `test`: 测试
- `chore`: 构建/工具

示例:
```
feat(auth): 添加 API Key 认证支持

- 实现 API Key 生成和验证
- 添加中间件保护路由
- 更新文档

Fixes #123
```

## 数据库迁移

### 创建迁移

使用 GORM AutoMigrate:

```go
// internal/storage/migrate.go
func AutoMigrate(db *gorm.DB) error {
    return db.AutoMigrate(
        &models.User{},
        &models.Skill{},
        &models.SkillVersion{},
        // 添加新模型
        &models.NewModel{},
    )
}
```

### 手动迁移

如需复杂迁移，使用 migrations 目录:

```bash
# 创建迁移文件
touch migrations/001_add_new_field.sql

# 执行迁移
psql -h localhost -U skillhome -d skillhome -f migrations/001_add_new_field.sql
```

## 测试

### 单元测试

```bash
cd skill-home-server
go test ./internal/... -v

# 覆盖率
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### 集成测试

```bash
# 启动测试环境
docker-compose -f deployments/docker/docker-compose.test.yml up -d

# 运行测试
go test ./... -tags=integration

# 清理
docker-compose -f deployments/docker/docker-compose.test.yml down
```

### API 测试

使用 curl 或 Postman:

```bash
# 测试脚本
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"test","email":"test@test.com","password":"test123"}'
```

## 提交 PR

### 准备

1. 确保代码通过所有测试
2. 更新相关文档
3. 添加 CHANGELOG 条目

```bash
# 检查
go fmt ./...
go vet ./...
go test ./...

# 提交
git add .
git commit -m "feat(xxx): 描述"
git push origin feature/xxx
```

### PR 模板

```markdown
## 描述
简要描述更改内容

## 类型
- [ ] Bug 修复
- [ ] 新功能
- [ ] 文档更新
- [ ] 重构

## 测试
- [ ] 单元测试通过
- [ ] 集成测试通过
- [ ] 手动测试验证

## 检查清单
- [ ] 代码遵循项目规范
- [ ] 文档已更新
- [ ] 测试已添加/更新

## 相关 Issue
Fixes #123
```

## 代码审查

审查清单:

- [ ] 代码逻辑正确
- [ ] 错误处理完善
- [ ] 性能考虑
- [ ] 安全问题
- [ ] 测试覆盖
- [ ] 文档更新

## 发布流程

1. 更新版本号
2. 更新 CHANGELOG
3. 创建 Release PR
4. 合并后打标签
5. 构建发布包

## 联系方式

- Issue: GitHub Issues
- 邮件: your-email@example.com

## 许可证

贡献即表示同意将代码授权为 MIT 许可证。
