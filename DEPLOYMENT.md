# Skill Home 生产部署说明

## 当前生产入口

| 项目 | 值 |
|------|-----|
| 公开入口 | `https://soulstore.ciqtek.com/skill-home` |
| 生产主机 | `121.40.85.95` |
| 部署目录 | `/opt/skill-home` |
| systemd 服务 | `skill-home.service` |
| 反向代理 | 系统 Nginx，挂载在 `soulstore.ciqtek.com/skill-home/` |

## 当前架构

- Skill Home 服务负责管理、搜索、聚合、鉴权和下载地址编排。
- PostgreSQL 负责元数据。
- Redis 负责缓存。
- 公共 skill 包当前由阿里云 OSS 承载。
- soul 上的 MinIO 容器仍保留，用于历史对象核对、回填或回滚，不再作为当前公开 skill 包的运行时主存储。

## 关键文件

- systemd unit：`/etc/systemd/system/skill-home.service`
- 运行时环境：`/opt/skill-home/.env`
- 基础设施编排：`/opt/skill-home/docker-compose.infra.yml`
- 服务二进制：`/opt/skill-home/server`

## 当前配置原则

### 1. 存储配置以 `.env` 为准

生产环境当前使用 `/opt/skill-home/.env` 中的 `SKILL_HOME_STORAGE_*`：

```env
SKILL_HOME_STORAGE_TYPE=s3
SKILL_HOME_STORAGE_ENDPOINT=<oss internal endpoint，不带协议>
SKILL_HOME_STORAGE_ACCESS_KEY=<oss access key id>
SKILL_HOME_STORAGE_SECRET_KEY=<oss access key secret>
SKILL_HOME_STORAGE_BUCKET=<release bucket>
SKILL_HOME_STORAGE_REGION=<oss region>
SKILL_HOME_STORAGE_USE_SSL=true
SKILL_HOME_STORAGE_PUBLIC_BASE_URL=https://<public object root>
```

说明：
- `SKILL_HOME_STORAGE_ENDPOINT` 不要带 `http://` 或 `https://`
- `SKILL_HOME_STORAGE_PUBLIC_BASE_URL` 需要带协议，并且必须直接指向对象根路径

### 2. 不要在 unit 里硬编码存储覆盖项

`skill-home.service` 可以继续保留服务端口、数据库主机、数据库端口等固定项，但不要再写这些：

```ini
Environment=SKILL_HOME_STORAGE_TYPE=...
Environment=SKILL_HOME_STORAGE_ENDPOINT=...
Environment=SKILL_HOME_STORAGE_ACCESS_KEY=...
Environment=SKILL_HOME_STORAGE_SECRET_KEY=...
Environment=SKILL_HOME_STORAGE_BUCKET=...
Environment=SKILL_HOME_STORAGE_REGION=...
Environment=SKILL_HOME_STORAGE_USE_SSL=...
Environment=SKILL_HOME_STORAGE_PUBLIC_BASE_URL=...
```

原因：
- systemd 的 `Environment=` 会覆盖 `EnvironmentFile=/opt/skill-home/.env`
- 存储切换到 OSS 后，如果 unit 里还保留旧 MinIO 值，服务会继续连旧存储

### 3. `SOULSTORE_SKILL_OSS_*` 只是兼容 fallback

当前服务端代码已支持：当 `SKILL_HOME_STORAGE_*` 缺失时，可从 `SOULSTORE_SKILL_OSS_*` 推导 OSS 配置。

但生产部署仍建议：
- 最终落盘到 `/opt/skill-home/.env` 的，统一使用 `SKILL_HOME_STORAGE_*`
- `SOULSTORE_SKILL_OSS_*` 更适合作为本地迁移脚本或上层系统的来源变量

## 当前 unit 示例

这是 soul 当前推荐的 `skill-home.service` 形态：

```ini
[Unit]
Description=Skill-Home Registry Server
After=network.target docker.service
Requires=docker.service

[Service]
Type=simple
User=root
WorkingDirectory=/opt/skill-home
EnvironmentFile=/opt/skill-home/.env
Environment=SKILL_HOME_SERVER_PORT=8080
Environment=SKILL_HOME_DATABASE_HOST=localhost
Environment=SKILL_HOME_DATABASE_PORT=15432
Environment=SKILL_HOME_DATABASE_USER=skillhome
Environment=SKILL_HOME_DATABASE_NAME=skillhome
Environment=SKILL_HOME_DATABASE_SSL_MODE=disable
ExecStart=/opt/skill-home/server
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

如果数据库密码、JWT 密钥也放进 `.env`，可以继续复用 `EnvironmentFile=/opt/skill-home/.env`，不要在文档里写真实密钥。

## OSS 切换顺序

把运行时存储从 MinIO 切到 OSS 时，顺序必须是：

1. 准备好目标 OSS 的 `SKILL_HOME_STORAGE_*`
2. 准备好源 MinIO 的 `SKILL_HOME_BACKFILL_SOURCE_*`
3. 先跑 `public-oss-backfill --dry-run`
4. 再正式执行回填
5. 回填成功后，才把生产运行时存储切到 OSS
6. 删除 unit 里的旧 `Environment=SKILL_HOME_STORAGE_*`
7. `systemctl daemon-reload && systemctl restart skill-home`

如果先切运行时，再回填历史公开包，旧对象会直接下载失败。

## 历史公共包回填

回填工具在：

- `skill-home-server/cmd/public-oss-backfill`

典型执行方式：

```bash
cd /opt/skill-home

set -a
source /opt/skill-home/.env
source /tmp/public-oss-backfill.env
set +a

/tmp/public-oss-backfill --dry-run
/tmp/public-oss-backfill
```

最常见的临时源配置是：

```env
SKILL_HOME_BACKFILL_SOURCE_TYPE=minio
SKILL_HOME_BACKFILL_SOURCE_ENDPOINT=localhost:19000
SKILL_HOME_BACKFILL_SOURCE_ACCESS_KEY=minioadmin
SKILL_HOME_BACKFILL_SOURCE_SECRET_KEY=minioadmin
SKILL_HOME_BACKFILL_SOURCE_BUCKET=skill-home
SKILL_HOME_BACKFILL_SOURCE_USE_SSL=false
```

## 验收命令

### 健康检查

```bash
curl -fsS http://127.0.0.1:8080/skill-home/health
curl -fsS https://soulstore.ciqtek.com/skill-home/health
```

### 真实下载检查

不要只看 `HEAD`，应使用真实 `GET`：

```bash
curl -fsS -o /tmp/skill-home-manager.zip -w 'status=%{http_code} size=%{size_download}\n' \
  https://soulstore.ciqtek.com/skill-home/api/v1/download/skill-home/skill-home-manager/0.2.3
```

### 回填复验

```bash
/tmp/public-oss-backfill --dry-run
```

如果已经全部回填成功，结果应以 `skip ... 目标对象已存在` 为主。

## 回滚

修改前建议先备份：

```bash
cp /etc/systemd/system/skill-home.service /etc/systemd/system/skill-home.service.bak.$(date +%Y%m%d%H%M%S)
cp /opt/skill-home/.env /opt/skill-home/.env.bak.$(date +%Y%m%d%H%M%S)
```

如果需要回滚：

1. 恢复最近一份 `skill-home.service.bak.*`
2. 恢复最近一份 `.env.bak.*`
3. 执行：

```bash
systemctl daemon-reload
systemctl restart skill-home.service
```

## 备注

- 当前公开 skill 下载已经从 soul 实测走通 OSS。
- soul 本机仍保留 `skill-home-minio` 容器，便于历史核对和回滚。
- 这份文档描述的是当前生产事实，不是最早的 `47.122.112.210` 历史部署记录。
