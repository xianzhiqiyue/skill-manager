# 统一发布流程

这份文档定义 `skill-manager` 仓库的统一发布口径。后续涉及 `server`、`web`、`cli`、`skill` 的发布，一律以这里为准。

## 发布原则

- 先判断本次改动影响哪个交付物，不要默认四个组件一起发
- 先完成代码、测试和文档，再做版本变更与发布
- 先推送 Git 仓库，再发布对外制品
- 所有线上发布都要能回滚，并保留最小必要的备份
- `skill` 发布不要覆盖已有版本；默认递增版本号后再发布

## 交付物判定

### 只发 `skill-home-server`

适用于：

- 注册中心接口、权限、鉴权、对象存储、下载链路变更
- 数据模型、服务端配置、systemd 或生产部署变更

不自动触发：

- `skill-home-cli`
- `skill-home-web`
- `skills/skill-home-manager`

### 只发 `skill-home-web`

适用于：

- 前端页面、交互、展示、安装引导变更

### 只发 `skill-home-cli`

适用于：

- CLI 子命令、配置解析、打包/同步/安装逻辑变更

### 只发 `skills/skill-home-manager`

适用于：

- `skills/skill-home-manager` 自身说明、脚本、工作流约束变更
- 面向 Codex 的安装、打包、发布、路径探测逻辑变更

## 标准顺序

一次完整发布默认按下面顺序执行：

1. 判断影响面，确认本次要发布哪些交付物
2. 补测试、跑验证、更新文档
3. 更新对应版本号
4. 提交代码
5. 推送到远端仓库
6. 发布对应制品
7. 做线上验收
8. 记录结果和回滚点

## Git 口径

所有发布动作之前，先保证：

- 本地工作区没有无关脏改动
- 需要发布的版本号已经写入仓库文件
- 发布相关文档已经同步更新

发布前统一执行：

```bash
git status --short
git add <changed-files>
git commit -m "<type>: <summary>"
git push origin main
```

如果某个交付物已经发出，但仓库中的版本号还没提交，视为流程不完整，需要立刻补提交并推送。

## Skill 发布口径

`skill` 发布默认通过 `skill-home-manager` 工作流执行。

统一顺序：

1. 修改目标 skill 的 `SKILL.md`，递增版本号
2. 确认 `category` 与 `tags` 合法
3. 执行校验和扫描
4. 如有需要，显式打包到 `dist/`
5. 推送到 registry

推荐命令：

```bash
skill-home validate skills/skill-home-manager
skill-home scan skills/skill-home-manager
skill-home pack skills/skill-home-manager --output dist/skill-home-manager-<version>.zip
skill-home push skills/skill-home-manager
```

约束：

- 不要默认使用 `--force` 覆盖已存在版本
- 如果远端版本已存在，默认做法是递增版本号，例如 `0.2.8 -> 0.2.9`
- 发布成功后，要把版本号改动提交到 Git

## Server 发布口径

`server` 发布只针对当前生产入口：

- 主机：`soul正式服务器`
- 地址：`121.40.85.95`
- 对外入口：`https://soulstore.ciqtek.com/skill-home`

不要把生产发布到历史错误机器 `47.122.112.210`。

统一顺序：

1. 本地构建带版本信息的 Linux 二进制
2. 备份远端 `/opt/skill-home/server`
3. 如有需要，更新 `/etc/systemd/system/skill-home.service`
4. 上传并替换新二进制
5. `systemctl restart skill-home`
6. 检查本机和公网健康状态

最低验收：

```bash
curl -fsS http://127.0.0.1:8080/skill-home/health
curl -fsS https://soulstore.ciqtek.com/skill-home/health
```

如果服务支持版本注入，发布产物不能保留 `dev` 版本字符串。

## CLI / Web 发布口径

`cli` 和 `web` 只有在对应目录存在真实改动时才发布，不跟随 `server` 或 `skill` 自动联动发版。

判断原则：

- 改了 `skill-home-cli/` 才考虑发 `cli`
- 改了 `skill-home-web/` 才考虑发 `web`
- 仅因 `server` 发布成功，不自动补发 `cli` 或 `web`

## 发布后验收口径

### Skill

- `skill-home push` 返回成功
- 记录远端版本号和下载地址

### Server

- 本机健康检查成功
- 公网健康检查成功
- 新接口或权限点可用

### Git

- `origin/main` 已包含本次发布对应提交

## 回滚口径

### Skill

- 不删除已成功发布的版本，除非确认要撤回
- 如需修复，优先发布下一个版本

### Server

- 发布前备份远端二进制和 service 文件
- 回滚时恢复备份并重启服务

## 当前仓库约定

- `server` 部署细节以 [DEPLOYMENT.md](../DEPLOYMENT.md) 和 [skill-home-server/deployments/docker/deploy-notes.md](../skill-home-server/deployments/docker/deploy-notes.md) 为准
- `skill-home-manager` 的 skill 发布细节以 [skills/skill-home-manager/README.md](../skills/skill-home-manager/README.md) 为准
- 如果其他文档与本文冲突，以本文为准，并应尽快回收旧口径
