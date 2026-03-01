# Skill-Home Server 部署记录

## 服务器信息

| 项目 | 值 |
|------|-----|
| **IP 地址** | 47.122.112.210 |
| **用户名** | root |
| **部署目录** | /opt/skill-home |
| **服务端口** | 8080 |

## 环境变量配置

创建 `/opt/skill-home/.env`：

```bash
DB_PASSWORD=9ogpRaTDYn8H8UmpdCocGDbehjSR9k1b
JWT_SECRET=wCvPQQBsITss8vZM37rGzUSZuTLeVNwxRuNGAtuPFpl7NJEWngnDeW9IHiwcV
```

## 部署步骤

### 1. 连接服务器

```bash
ssh root@47.122.112.210
```

### 2. 进入部署目录

```bash
cd /opt/skill-home
```

### 3. 启动数据库和存储服务

```bash
docker compose up -d postgres minio redis
```

### 4. 本地构建并上传（推荐）

由于服务器网络限制，建议在本地构建后上传：

**本地执行：**
```bash
# 进入服务端代码目录
cd skill-home-server

# 构建 Linux 二进制文件
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server cmd/server/main.go

# 上传到服务器
scp server root@47.122.112.210:/opt/skill-home/
```

**服务器执行：**
```bash
cd /opt/skill-home

# 直接运行
export SKILL_HOME_SERVER_PORT=8080
export SKILL_HOME_SERVER_MODE=production
export SKILL_HOME_DATABASE_HOST=localhost
export SKILL_HOME_DATABASE_PORT=5432
export SKILL_HOME_DATABASE_USER=skillhome
export SKILL_HOME_DATABASE_PASSWORD=9ogpRaTDYn8H8UmpdCocGDbehjSR9k1b
export SKILL_HOME_DATABASE_NAME=skillhome
export SKILL_HOME_STORAGE_TYPE=minio
export SKILL_HOME_STORAGE_ENDPOINT=localhost:9000
export SKILL_HOME_STORAGE_ACCESS_KEY=minioadmin
export SKILL_HOME_STORAGE_SECRET_KEY=minioadmin
export SKILL_HOME_STORAGE_BUCKET=skill-home
export SKILL_HOME_STORAGE_USE_SSL=false
export SKILL_HOME_AUTH_JWT_SECRET=wCvPQQBsITss8vZM37rGzUSZuTLeVNwxRuNGAtuPFpl7NJEWngnDeW9IHiwcV

./server
```

### 5. 使用 systemd 管理（推荐）

创建 `/etc/systemd/system/skill-home.service`：

```ini
[Unit]
Description=Skill-Home Registry Server
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/skill-home
Environment=SKILL_HOME_SERVER_PORT=8080
Environment=SKILL_HOME_SERVER_MODE=production
Environment=SKILL_HOME_DATABASE_HOST=localhost
Environment=SKILL_HOME_DATABASE_PORT=5432
Environment=SKILL_HOME_DATABASE_USER=skillhome
Environment=SKILL_HOME_DATABASE_PASSWORD=9ogpRaTDYn8H8UmpdCocGDbehjSR9k1b
Environment=SKILL_HOME_DATABASE_NAME=skillhome
Environment=SKILL_HOME_STORAGE_TYPE=minio
Environment=SKILL_HOME_STORAGE_ENDPOINT=localhost:9000
Environment=SKILL_HOME_STORAGE_ACCESS_KEY=minioadmin
Environment=SKILL_HOME_STORAGE_SECRET_KEY=minioadmin
Environment=SKILL_HOME_STORAGE_BUCKET=skill-home
Environment=SKILL_HOME_STORAGE_USE_SSL=false
Environment=SKILL_HOME_AUTH_JWT_SECRET=wCvPQQBsITss8vZM37rGzUSZuTLeVNwxRuNGAtuPFpl7NJEWngnDeW9IHiwcV
ExecStart=/opt/skill-home/server
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

启动服务：
```bash
systemctl daemon-reload
systemctl enable skill-home
systemctl start skill-home
systemctl status skill-home
```

## 验证部署

### 健康检查
```bash
curl http://47.122.112.210:8080/health
```

预期响应：
```json
{
  "status": "ok",
  "service": "skill-home-registry",
  "version": "1.0.0"
}
```

## 服务地址

| 服务 | 地址 | 说明 |
|------|------|------|
| API | http://47.122.112.210:8080 | 主服务 |
| MinIO Console | http://47.122.112.210:9001 | 对象存储管理 (minioadmin/minioadmin) |

## API 端点

### 公开接口
- `GET /health` - 健康检查
- `GET /api/v1/skills` - 列出技能
- `GET /api/v1/skills/:namespace/:name` - 技能详情
- `GET /api/v1/search?q=` - 搜索技能
- `GET /api/v1/download/:namespace/:name/:version` - 下载技能

### 需要认证
- `GET /api/v1/user` - 当前用户
- `POST /api/v1/skills` - 创建技能
- `POST /api/v1/skills/:namespace/:name/versions` - 发布版本

## 安全信息

⚠️ **重要**：生产环境请修改以下默认值
- JWT Secret（当前已使用随机生成的密钥）
- 数据库密码（当前已使用随机生成的密码）
- MinIO 管理员密码（当前为 minioadmin/minioadmin）

## 防火墙配置

确保开放以下端口：
```bash
ufw allow 8080/tcp
ufw allow 9000/tcp
ufw allow 9001/tcp
```

## 日志查看

```bash
# Docker 方式
docker compose logs -f api

# Systemd 方式
journalctl -u skill-home -f
```

## 备份

```bash
# 备份数据库
docker exec skill-home-postgres-1 pg_dump -U skillhome skillhome > backup.sql

# 备份存储
tar -czf minio-backup.tar.gz /var/lib/docker/volumes/skill-home_minio_data/_data
```
