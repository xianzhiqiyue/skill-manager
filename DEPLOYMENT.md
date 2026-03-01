# Skill-Home Server 部署记录

## 部署时间
2026-03-01

## 服务器信息

| 项目 | 值 |
|------|-----|
| **IP 地址** | 47.122.112.210 |
| **用户名** | root |
| **部署目录** | /opt/skill-home |
| **API 端口** | 8080 |

---

## 已部署服务

### 基础设施服务（Docker）

| 服务 | 内部端口 | 外部端口 | 状态 |
|------|---------|---------|------|
| PostgreSQL | 5432 | 15432 | ✅ 运行中 |
| MinIO | 9000/9001 | 19000/19001 | ✅ 运行中 |
| Redis | 6379 | 16379 | ✅ 运行中 |

### 应用服务

| 服务 | 端口 | 状态 |
|------|------|------|
| Skill-Home API | 8080 | ✅ 运行中 |
| Skill-Home CLI | - | ✅ 已部署 |

### CLI 下载

```bash
# 下载 CLI
curl -fsSL -o skill-home http://47.122.112.210:8080/download/cli
chmod +x skill-home
sudo mv skill-home /usr/local/bin/

# 或使用 scp
scp root@47.122.112.210:/opt/skill-home/skill-home /usr/local/bin/
```

### CLI 命令

```bash
skill-home --help              # 查看帮助
skill-home login               # 登录到注册中心
skill-home search <关键词>     # 搜索技能
skill-home pull <技能名>       # 下载技能
skill-home push                # 推送技能到注册中心
skill-home list                # 列出已安装技能
skill-home sync                # 同步技能到 IDE
```

### 服务地址

| 服务 | 访问地址 | 凭证 |
|------|---------|------|
| MinIO Console | http://47.122.112.210:19001 | minioadmin / vbPgYPA/BMs7Lr7xNVu4ouCEdQkgEA4z |
| PostgreSQL | localhost:15432 (仅本地) | skillhome / 9ogpRaTDYn8H8UmpdCocGDbehjSR9k1b |
| Redis | localhost:16379 (仅本地) | - |

---

## 安全配置

### 数据库密码
```
9ogpRaTDYn8H8UmpdCocGDbehjSR9k1b
```

### JWT 密钥
```
wCvPQQBsITss8vZM37rGzUSZuTLeVNwxRuNGAtuPFpl7NJEWngnDeW9IHiwcV
```

---

## 完整功能测试

### API 使用示例

```bash
# 1. 注册用户
curl -X POST http://47.122.112.210:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","email":"test@example.com","password":"testpass123"}'

# 2. 登录获取 Token
curl -X POST http://47.122.112.210:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"testpass123"}'

# 3. 创建 API Key (使用 Token)
curl -X POST http://47.122.112.210:8080/api/v1/user/api-keys \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"cli"}'

# 4. 发布技能
curl -X POST http://47.122.112.210:8080/api/v1/skills \
  -H "Authorization: Bearer <token>" \
  -F "namespace=testuser" \
  -F "name=my-skill" \
  -F "version=0.1.0" \
  -F "skill=@skill.tar.gz"

# 5. 搜索技能
curl "http://47.122.112.210:8080/api/v1/search?q=my-skill"

# 6. 下载技能
curl -o skill.tar.gz \
  "http://47.122.112.210:8080/api/v1/download/testuser/my-skill/0.1.0"
```

### 已修复问题

- [x] JSON 字段类型扫描问题
- [x] 版本查找下载问题
- [x] API Key 生成安全问题
- [x] CLI URL 编码问题

---

## 开发说明

如需重新编译部署：

### 1. 本地编译（在有 Go 环境的机器上）

```bash
# 进入服务端代码目录
cd skill-home-server

# 构建 Linux 二进制文件
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server cmd/server/main.go

# 上传到服务器
scp server root@47.122.112.210:/opt/skill-home/
```

### 2. 服务器上启动 API 服务

```bash
ssh root@47.122.112.210
cd /opt/skill-home

# 设置环境变量
export SKILL_HOME_SERVER_PORT=8080
export SKILL_HOME_SERVER_MODE=production
export SKILL_HOME_DATABASE_HOST=localhost
export SKILL_HOME_DATABASE_PORT=15432
export SKILL_HOME_DATABASE_USER=skillhome
export SKILL_HOME_DATABASE_PASSWORD=9ogpRaTDYn8H8UmpdCocGDbehjSR9k1b
export SKILL_HOME_DATABASE_NAME=skillhome
export SKILL_HOME_DATABASE_SSL_MODE=disable
export SKILL_HOME_STORAGE_TYPE=minio
export SKILL_HOME_STORAGE_ENDPOINT=localhost:19000
export SKILL_HOME_STORAGE_ACCESS_KEY=minioadmin
export SKILL_HOME_STORAGE_SECRET_KEY=minioadmin
export SKILL_HOME_STORAGE_BUCKET=skill-home
export SKILL_HOME_STORAGE_USE_SSL=false
export SKILL_HOME_AUTH_JWT_SECRET=wCvPQQBsITss8vZM37rGzUSZuTLeVNwxRuNGAtuPFpl7NJEWngnDeW9IHiwcV

# 运行
./server
```

### 3. 使用 Systemd 管理（推荐）

创建服务文件：

```bash
cat > /etc/systemd/system/skill-home.service << 'EOF'
[Unit]
Description=Skill-Home Registry Server
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/skill-home
Environment="SKILL_HOME_SERVER_PORT=8080"
Environment="SKILL_HOME_DATABASE_HOST=localhost"
Environment="SKILL_HOME_DATABASE_PORT=15432"
Environment="SKILL_HOME_DATABASE_USER=skillhome"
Environment="SKILL_HOME_DATABASE_PASSWORD=9ogpRaTDYn8H8UmpdCocGDbehjSR9k1b"
Environment="SKILL_HOME_DATABASE_NAME=skillhome"
Environment="SKILL_HOME_DATABASE_SSL_MODE=disable"
Environment="SKILL_HOME_STORAGE_TYPE=minio"
Environment="SKILL_HOME_STORAGE_ENDPOINT=localhost:19000"
Environment="SKILL_HOME_STORAGE_ACCESS_KEY=minioadmin"
Environment="SKILL_HOME_STORAGE_SECRET_KEY=minioadmin"
Environment="SKILL_HOME_STORAGE_BUCKET=skill-home"
Environment="SKILL_HOME_STORAGE_USE_SSL=false"
Environment="SKILL_HOME_AUTH_JWT_SECRET=wCvPQQBsITss8vZM37rGzUSZuTLeVNwxRuNGAtuPFpl7NJEWngnDeW9IHiwcV"
ExecStart=/opt/skill-home/server
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# 启动服务
systemctl daemon-reload
systemctl enable skill-home
systemctl start skill-home
systemctl status skill-home
```

---

## 验证部署

### 健康检查
```bash
curl http://47.122.112.210:8080/health
```

### API 端点

| 方法 | 路径 | 说明 | 认证 |
|------|------|------|------|
| GET | /health | 健康检查 | 否 |
| POST | /api/v1/auth/register | 用户注册 | 否 |
| POST | /api/v1/auth/login | 用户登录 | 否 |
| GET | /api/v1/skills | 列出技能 | 否 |
| GET | /api/v1/skills/:namespace/:name | 技能详情 | 否 |
| GET | /api/v1/search?q= | 搜索技能 | 否 |
| GET | /api/v1/download/:ns/:name/:version | 下载技能 | 否 |
| GET | /api/v1/user | 当前用户信息 | 是 |
| GET | /api/v1/user/skills | 用户技能列表 | 是 |
| POST | /api/v1/skills | 创建技能 | 是 |
| POST | /api/v1/user/api-keys | 创建 API Key | 是 |

---

## 管理命令

### 查看基础设施日志
```bash
cd /opt/skill-home
docker compose logs -f
```

### 重启基础设施
```bash
cd /opt/skill-home
docker compose restart
```

### 查看 API 服务日志
```bash
journalctl -u skill-home -f
```

### 重启 API 服务
```bash
systemctl restart skill-home
```

---

## 防火墙配置

确保开放以下端口：

```bash
ufw allow 8080/tcp    # API 服务
ufw allow 19000/tcp   # MinIO API
ufw allow 19001/tcp   # MinIO Console
```

---

## 备份

### 备份数据库
```bash
docker exec skill-home-postgres pg_dump -U skillhome skillhome > backup.sql
```

### 备份存储
```bash
docker exec skill-home-minio mc mirror /data /backup
```

---

## 安全建议

1. ⚠️ **修改 MinIO 默认密码** - 当前为 minioadmin/minioadmin
2. ⚠️ **限制端口访问** - 使用防火墙限制 15432/16379 仅本地访问
3. ⚠️ **启用 SSL** - 生产环境应使用 HTTPS
4. ⚠️ **定期备份** - 设置自动备份任务

---

## 文件清单

服务器上的文件：

```
/opt/skill-home/
├── docker-compose.yml      # Docker 编排文件
├── .env                    # 环境变量
├── server                  # API 服务二进制
├── skill-home              # CLI 工具二进制
├── config.yaml             # CLI 默认配置
└── deployments/
    └── docker/
        └── deploy-notes.md # 详细部署说明
```

---

## 附录 A：CLI 打包与安装详解

### A.1 本地构建安装

```bash
# 进入项目目录
cd skill-home-cli

# 安装依赖
make deps

# 构建（输出到 bin/ 目录）
make build

# 安装到系统（需要权限）
sudo make install          # 安装到 /usr/local/bin
```

### A.2 开发模式运行

```bash
# 直接运行，不构建
make dev

# 或
go run ./cmd/skill-home
```

### A.3 跨平台构建

```bash
# 构建所有平台版本（输出到 dist/）
make build-all
```

生成的文件：
- `skill-home-darwin-amd64` - macOS Intel
- `skill-home-darwin-arm64` - macOS Apple Silicon
- `skill-home-linux-amd64` - Linux x64
- `skill-home-linux-arm64` - Linux ARM64
- `skill-home-windows-amd64.exe` - Windows

### A.4 发布包制作

```bash
# 创建发布压缩包（tar.gz / zip）
make release

# 输出到 dist/:
# - skill-home-darwin-amd64.tar.gz
# - skill-home-linux-amd64.tar.gz
# - skill-home-windows-amd64.zip
```

### A.5 远程安装脚本

```bash
# 从 GitHub Releases 自动下载安装最新版本
curl -sSL https://get.skill-home.dev | sh

# 或指定版本
curl -sSL https://get.skill-home.dev | sh -s v1.0.0
```

安装脚本流程：
1. 自动检测操作系统和架构
2. 从 GitHub API 获取最新版本
3. 下载对应平台的二进制文件
4. 安装到 `/usr/local/bin`

### A.6 手动下载安装

```bash
# 从 GitHub Releases 下载对应平台的二进制文件
wget https://github.com/skill-home/cli/releases/download/v1.0.0/skill-home-linux-amd64.tar.gz

# 解压并安装
tar -xzf skill-home-linux-amd64.tar.gz
chmod +x skill-home
sudo mv skill-home /usr/local/bin/
```

### A.7 Makefile 命令速查

| 命令 | 说明 |
|------|------|
| `make build` | 构建当前平台二进制文件 |
| `make install` | 安装到 `/usr/local/bin` |
| `make build-all` | 跨平台构建 |
| `make release` | 创建发布包 |
| `make test` | 运行测试 |
| `make fmt` | 代码格式化 |
| `make lint` | 静态检查 |
| `make clean` | 清理构建产物 |

---

## 附录 B：API Key 生成机制详解

### B.1 生成流程

```
┌─────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   用户       │────▶│  注册/登录账号   │────▶│  创建 API Key   │
└─────────────┘     └─────────────────┘     └─────────────────┘
                                                    │
                           ┌────────────────────────┘
                           ▼
                    ┌─────────────────┐
                    │  服务端生成 Key  │
                    │  - 32字节随机数  │
                    │  - Base64编码    │
                    └─────────────────┘
                           │
                           ▼
                    ┌─────────────────┐
                    │  仅显示一次      │
                    │  保存 Key Hash   │
                    └─────────────────┘
```

### B.2 服务端生成代码

**文件**: `skill-home-server/internal/api/handlers/user.go`

```go
func generateAPIKey() (string, error) {
    // 生成 32 字节随机数据
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    // 使用 base64 URL 编码，前缀为 sk_
    return "sk_" + base64.URLEncoding.EncodeToString(b), nil
}
```

**生成的 Key 格式**:
```
sk_XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
```

### B.3 CLI 登录使用流程

```bash
# 1. 登录（交互式输入 API Key）
skill-home login
# 请输入 API Key: sk_xxxxxxxxxx

# 2. 或直接在命令行指定
skill-home login --api-key sk_xxxxxxxxx

# 3. 验证登录状态
skill-home whoami

# 4. 登出
skill-home logout
```

### B.4 服务端 API 端点

| 端点 | 方法 | 说明 | 认证 |
|------|------|------|------|
| `/api/v1/auth/register` | POST | 用户注册 | 否 |
| `/api/v1/auth/login` | POST | 用户登录（返回 JWT） | 否 |
| `/api/v1/user/api-keys` | POST | 创建 API Key | JWT |
| `/api/v1/user/api-keys/:id` | DELETE | 撤销 API Key | JWT |
| `/api/v1/user` | GET | 获取当前用户信息 | API Key |

### B.5 创建 API Key 的请求示例

```bash
# 1. 先登录获取 JWT
curl -X POST https://registry.skill-home.dev/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"your-password"}'

# 2. 使用 JWT 创建 API Key
curl -X POST https://registry.skill-home.dev/api/v1/user/api-keys \
  -H "Authorization: Bearer <JWT_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"name":"my-cli-key","expires_at":"2025-12-31T00:00:00Z"}'

# 响应（仅返回一次）
{
  "id": "uuid",
  "name": "my-cli-key",
  "key": "sk_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
  "prefix": "sk_xxxxxx",
  "expires_at": "2025-12-31T00:00:00Z",
  "created_at": "2024-01-01T00:00:00Z"
}
```

### B.6 安全设计

| 特性 | 说明 |
|------|------|
| Key 存储 | 服务端只存储 Key 的 bcrypt hash，不存明文 |
| 前缀显示 | 存储前 8 个字符用于列表展示（如 `sk_xxxxxx`） |
| 一次性显示 | Key 生成后仅返回一次，丢失需重新创建 |
| 过期时间 | 支持设置可选的过期时间 |
| 权限控制 | 每个用户可以创建多个 API Key，可独立撤销 |

### B.7 配置文件

登录后 API Key 保存在 `~/.config/skill-home/config.yaml`：

```yaml
registry:
  endpoint: "https://registry.skill-home.dev"
  api_key: "sk_xxxxxxxxxxxxxxxx"
```

或通过环境变量设置：

```bash
export SKILL_HOME_API_KEY="sk_xxxxxxxxxxxxxxxx"
```
