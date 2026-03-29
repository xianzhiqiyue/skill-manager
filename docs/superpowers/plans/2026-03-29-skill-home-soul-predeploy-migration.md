# Skill Home Soul Predeploy Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在不切换对外入口的前提下，把当前运行在旧阿里云服务器上的 Skill Home 服务完整预部署到 soul 正式服务器，并完成数据、静态发布产物和运行环境同步。

**Architecture:** 保持现网运行模式不变：`skill-home` 主服务继续用 `/opt/skill-home/server` + `systemd` 管理，数据库、MinIO、Redis 由 Docker 承载。新机先复刻旧机的目录、依赖端口、数据与静态发布树，通过本机或 server-to-server relay 完成迁移，验收通过后再进入后续反代/切流阶段。

**Tech Stack:** Go server binary, systemd, Docker Compose / Docker, PostgreSQL 16, MinIO, Redis 7, GitHub Actions hosted release assets, server-manager

---

### Task 1: Snapshot Source Deployment

**Files:**
- Read: `/etc/systemd/system/skill-home.service` on `阿里云`
- Read: `/opt/skill-home/.env` on `阿里云`
- Read: `/opt/skill-home/releases/**` on `阿里云`
- Create: `/root/skill-home-migration/` on `阿里云`
- Create: `/root/skill-home-migration/skill-home.sql.gz` on `阿里云`
- Create: `/root/skill-home-migration/minio-data.tar.gz` on `阿里云`
- Create: `/root/skill-home-migration/opt-skill-home.tar.gz` on `阿里云`

- [ ] **Step 1: Verify current source layout and health**

Run:
`python3 "$HOME/.codex/skills/server-manager/main.py" exec "阿里云" 'systemctl show skill-home --property=Environment --no-pager && curl -fsSL http://127.0.0.1:8080/health'`

Expected: 输出当前 `skill-home` 环境变量，健康检查返回 `status=ok`

- [ ] **Step 2: Create migration workspace on source server**

Run:
`python3 "$HOME/.codex/skills/server-manager/main.py" exec "阿里云" 'mkdir -p /root/skill-home-migration && ls -ld /root/skill-home-migration'`

Expected: 目录创建成功

- [ ] **Step 3: Export PostgreSQL data**

Run:
`python3 "$HOME/.codex/skills/server-manager/main.py" exec "阿里云" 'PGPASSWORD="$(python3 - <<'"'"'"'"'"'"'"'"'PY'"'"'"'"'"'"'"'"'\nfrom pathlib import Path\nfor line in Path("/opt/skill-home/.env").read_text().splitlines():\n    if line.startswith("DB_PASSWORD="):\n        print(line.split("=", 1)[1])\n        break\nPY\n)" docker exec -e PGPASSWORD="$PGPASSWORD" skill-home-postgres pg_dump -U skillhome skillhome | gzip -c > /root/skill-home-migration/skill-home.sql.gz && ls -lh /root/skill-home-migration/skill-home.sql.gz'`

Expected: 生成压缩 SQL dump

- [ ] **Step 4: Archive MinIO objects**

Run:
`python3 "$HOME/.codex/skills/server-manager/main.py" exec "阿里云" 'tar -C /var/lib/docker/volumes/skill-home_minio_data -czf /root/skill-home-migration/minio-data.tar.gz . && ls -lh /root/skill-home-migration/minio-data.tar.gz'`

Expected: 生成 MinIO 数据压缩包

- [ ] **Step 5: Archive deploy directory**

Run:
`python3 "$HOME/.codex/skills/server-manager/main.py" exec "阿里云" 'tar -C /opt -czf /root/skill-home-migration/opt-skill-home.tar.gz skill-home && ls -lh /root/skill-home-migration/opt-skill-home.tar.gz'`

Expected: 生成 `/opt/skill-home` 目录压缩包

### Task 2: Prepare Soul Server Runtime

**Files:**
- Create: `/opt/skill-home/` on `soul正式服务器`
- Create: `/opt/skill-home/.env` on `soul正式服务器`
- Create: `/opt/skill-home/docker-compose.infra.yml` on `soul正式服务器`
- Create: `/etc/systemd/system/skill-home.service` on `soul正式服务器`

- [ ] **Step 1: Create destination directory structure**

Run:
`python3 "$HOME/.codex/skills/server-manager/main.py" exec "soul正式服务器" 'mkdir -p /opt/skill-home /opt/skill-home/tmp /opt/skill-home/releases /opt/skill-home/web && ls -ld /opt/skill-home /opt/skill-home/tmp'`

Expected: 目录创建成功

- [ ] **Step 2: Write destination `.env` with current source secrets**

Run:
`python3 "$HOME/.codex/skills/server-manager/main.py" exec "soul正式服务器" 'cat > /opt/skill-home/.env <<'"'"'"'"'"'"'"'"'EOF'"'"'"'"'"'"'"'"'\nDB_PASSWORD=<from-source-env>\nJWT_SECRET=<from-source-env>\nEOF\nchmod 600 /opt/skill-home/.env'`

Expected: `.env` 写入成功且权限为 `600`

- [ ] **Step 3: Write infra compose file with source-compatible host ports**

Run:
`python3 "$HOME/.codex/skills/server-manager/main.py" exec "soul正式服务器" 'cat > /opt/skill-home/docker-compose.infra.yml <<'"'"'"'"'"'"'"'"'EOF'"'"'"'"'"'"'"'"'\nservices:\n  postgres:\n    image: docker.m.daocloud.io/postgres:16-alpine\n    container_name: skill-home-postgres\n    environment:\n      POSTGRES_USER: skillhome\n      POSTGRES_PASSWORD: ${DB_PASSWORD}\n      POSTGRES_DB: skillhome\n    env_file:\n      - /opt/skill-home/.env\n    ports:\n      - \"15432:5432\"\n    volumes:\n      - skill-home_postgres_data:/var/lib/postgresql/data\n    restart: unless-stopped\n  minio:\n    image: docker.m.daocloud.io/minio/minio:latest\n    container_name: skill-home-minio\n    command: server /data --console-address \":9001\"\n    ports:\n      - \"19000:9000\"\n      - \"19001:9001\"\n    environment:\n      MINIO_ROOT_USER: minioadmin\n      MINIO_ROOT_PASSWORD: minioadmin\n    volumes:\n      - skill-home_minio_data:/data\n    restart: unless-stopped\n  redis:\n    image: docker.m.daocloud.io/library/redis:7-alpine\n    container_name: skill-home-redis\n    ports:\n      - \"16379:6379\"\n    restart: unless-stopped\nvolumes:\n  skill-home_postgres_data:\n  skill-home_minio_data:\nEOF'`

Expected: compose 文件写入成功

- [ ] **Step 4: Start infra containers**

Run:
`python3 "$HOME/.codex/skills/server-manager/main.py" exec "soul正式服务器" 'cd /opt/skill-home && docker compose -f docker-compose.infra.yml --env-file /opt/skill-home/.env up -d && docker ps --format \"table {{.Names}}\\t{{.Status}}\" | grep skill-home'`

Expected: `skill-home-postgres` / `skill-home-minio` / `skill-home-redis` 全部运行

### Task 3: Transfer Source Artifacts and Data

**Files:**
- Read: `/root/skill-home-migration/*.tar.gz` on `阿里云`
- Create: `/opt/skill-home/tmp/*.tar.gz` on `soul正式服务器`
- Create: `/opt/skill-home/server`, `/opt/skill-home/install.sh`, `/opt/skill-home/releases/**`, `/opt/skill-home/web/**` on `soul正式服务器`

- [ ] **Step 1: Decide transfer path**

Run:
`python3 "$HOME/.codex/skills/server-manager/main.py" exec "阿里云" 'command -v nc || command -v ncat || command -v socat || true'`
`python3 "$HOME/.codex/skills/server-manager/main.py" exec "soul正式服务器" 'command -v nc || command -v ncat || command -v socat || true'`

Expected: 至少两端都具备同一种 relay 工具；否则改用本机下载后上传方案

- [ ] **Step 2: Transfer `/opt/skill-home` archive to destination**

Run:
`在确认 relay 工具后，以 server-to-server 或本机 relay 方式，把 /root/skill-home-migration/opt-skill-home.tar.gz 传到 soul:/opt/skill-home/tmp/`

Expected: 新机上出现压缩包

- [ ] **Step 3: Transfer PostgreSQL dump and MinIO archive**

Run:
`在确认 relay 工具后，把 /root/skill-home-migration/skill-home.sql.gz 和 /root/skill-home-migration/minio-data.tar.gz 传到 soul:/opt/skill-home/tmp/`

Expected: 新机上三个压缩包齐全

- [ ] **Step 4: Extract deploy directory on destination**

Run:
`python3 "$HOME/.codex/skills/server-manager/main.py" exec "soul正式服务器" 'tar -C /opt -xzf /opt/skill-home/tmp/opt-skill-home.tar.gz && ls -la /opt/skill-home | sed -n \"1,40p\"'`

Expected: `server`、`install.sh`、`releases`、`web` 等文件落到 `/opt/skill-home`

- [ ] **Step 5: Restore PostgreSQL and MinIO data**

Run:
`python3 "$HOME/.codex/skills/server-manager/main.py" exec "soul正式服务器" 'gunzip -c /opt/skill-home/tmp/skill-home.sql.gz | docker exec -i skill-home-postgres psql -U skillhome -d skillhome'`
`python3 "$HOME/.codex/skills/server-manager/main.py" exec "soul正式服务器" 'docker compose -f /opt/skill-home/docker-compose.infra.yml --env-file /opt/skill-home/.env down minio && tar -C /var/lib/docker/volumes/skill-home_minio_data -xzf /opt/skill-home/tmp/minio-data.tar.gz && docker compose -f /opt/skill-home/docker-compose.infra.yml --env-file /opt/skill-home/.env up -d minio'`

Expected: 数据恢复完成

### Task 4: Start Skill Home on Soul Server

**Files:**
- Create: `/etc/systemd/system/skill-home.service` on `soul正式服务器`
- Read: `/opt/skill-home/server` on `soul正式服务器`

- [ ] **Step 1: Write source-compatible systemd unit**

Run:
`python3 "$HOME/.codex/skills/server-manager/main.py" exec "soul正式服务器" 'cat > /etc/systemd/system/skill-home.service <<'"'"'"'"'"'"'"'"'EOF'"'"'"'"'"'"'"'"'\n[Unit]\nDescription=Skill-Home Registry Server\nAfter=network.target docker.service\n\n[Service]\nType=simple\nUser=root\nWorkingDirectory=/opt/skill-home\nEnvironment=SKILL_HOME_SERVER_PORT=8080\nEnvironment=SKILL_HOME_DATABASE_HOST=localhost\nEnvironment=SKILL_HOME_DATABASE_PORT=15432\nEnvironment=SKILL_HOME_DATABASE_USER=skillhome\nEnvironmentFile=/opt/skill-home/.env\nEnvironment=SKILL_HOME_DATABASE_NAME=skillhome\nEnvironment=SKILL_HOME_DATABASE_SSL_MODE=disable\nEnvironment=SKILL_HOME_STORAGE_TYPE=minio\nEnvironment=SKILL_HOME_STORAGE_ENDPOINT=localhost:19000\nEnvironment=SKILL_HOME_STORAGE_ACCESS_KEY=minioadmin\nEnvironment=SKILL_HOME_STORAGE_SECRET_KEY=minioadmin\nEnvironment=SKILL_HOME_STORAGE_BUCKET=skill-home\nEnvironment=SKILL_HOME_STORAGE_USE_SSL=false\nEnvironment=SKILL_HOME_AUTH_JWT_SECRET=${JWT_SECRET}\nExecStart=/opt/skill-home/server\nRestart=always\nRestartSec=5\n\n[Install]\nWantedBy=multi-user.target\nEOF\nsystemctl daemon-reload && systemctl enable skill-home'`

Expected: systemd 单元创建并 enable 成功

- [ ] **Step 2: Start destination service**

Run:
`python3 "$HOME/.codex/skills/server-manager/main.py" exec "soul正式服务器" 'systemctl restart skill-home && systemctl is-active skill-home'`

Expected: 返回 `active`

### Task 5: Predeploy Verification Without Cutover

**Files:**
- Read: `/opt/skill-home/releases/latest.json` on `soul正式服务器`
- Read: `/opt/skill-home/web/**` on `soul正式服务器`
- Read: remote health/search/install endpoints on `soul正式服务器`

- [ ] **Step 1: Verify service health on destination**

Run:
`python3 "$HOME/.codex/skills/server-manager/main.py" exec "soul正式服务器" 'curl -fsSL http://127.0.0.1:8080/health && curl -fsSL http://127.0.0.1:8080/releases/latest.json'`

Expected: 健康检查返回 `status=ok`，`latest.json` 返回当前版本

- [ ] **Step 2: Verify registry data presence**

Run:
`python3 "$HOME/.codex/skills/server-manager/main.py" exec "soul正式服务器" 'curl -fsSL http://127.0.0.1:8080/api/v1/skills | sed -n \"1,20p\"'`

Expected: 能看到现网 skill 数据，而不是空库

- [ ] **Step 3: Record predeploy completion and stop before cutover**

Run:
`在本地更新迁移记录，明确标注：GitHub Actions vars、代码里的公开地址、反向代理和 DNS 暂不变更。`

Expected: 新机可独立跑通，但旧机继续对外服务
