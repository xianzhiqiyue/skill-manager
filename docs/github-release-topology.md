# Skill-Home 发布拓扑

## 目标

把 `CLI` 和 `skill` 的发布职责拆清楚：

- GitHub：负责源码托管、CLI Release、安装脚本入口、变更记录
- Skill-Home Registry：负责 skill 的发现、版本、下载、弃用、评分和安装
- Web：负责宣传、搜索、安装引导和控制台管理

## 当前建议拓扑

### 公开地址

- GitHub 仓库：`https://github.com/xianzhiqiyue/skill-manager`
- GitHub Releases：`https://github.com/xianzhiqiyue/skill-manager/releases`
- Web 门户：`https://skill-home.dev`
- Registry API：`https://registry.skill-home.dev`
- CLI 安装脚本：`https://get.skill-home.dev/install.sh`

### 发布职责

| 对象 | 主发布面 | 说明 |
|------|----------|------|
| `skill-home` CLI | GitHub Releases | 发布预编译二进制和 `checksums.txt` |
| skill 源码 | GitHub 仓库 | 便于协作、审查、PR 和 issue 管理 |
| skill 可安装版本 | Skill-Home Registry | 通过 `skill-home push` 发布，供 `pull/install/search` 使用 |

## CLI 发布规范

### 版本约定

- Git tag：`vX.Y.Z`
- GitHub Release 名称：`skill-home vX.Y.Z`
- CLI `version` 子命令直接显示 `vX.Y.Z`

### Release 资产命名

- `skill-home-darwin-amd64.tar.gz`
- `skill-home-darwin-arm64.tar.gz`
- `skill-home-linux-amd64.tar.gz`
- `skill-home-linux-arm64.tar.gz`
- `skill-home-windows-amd64.zip`
- `checksums.txt`

### 安装入口

推荐入口：

```bash
curl -fsSL https://get.skill-home.dev/install.sh -o /tmp/skill-home-install.sh
bash /tmp/skill-home-install.sh
```

指定版本：

```bash
bash /tmp/skill-home-install.sh v0.2.0
```

默认安装到 `~/.local/bin/skill-home`，避免默认写入 `/usr/local/bin`。

## Skill 发布规范

### 推荐路径

1. skill 源码放在仓库内，例如 `skills/skill-home-manager`
2. 本地或 CI 先跑：
   - `skill-home validate`
   - `skill-home scan`
3. 再执行：
   - `skill-home push <skill-dir>`

### 为什么不把 skill 直接发 GitHub Release

- GitHub Release 不适合做 skill 搜索、评分、弃用、版本详情和在线安装
- Registry 才是 `pull/install/search/info` 的真实数据源
- GitHub 更适合放 skill 源码和文档，而不是取代 registry

## GitHub Actions 设计

### 1. CLI Release

文件：`.github/workflows/release-cli.yml`

触发方式：

- push tag `v*`
- 手动 `workflow_dispatch`

职责：

- 运行 `skill-home-cli` 测试
- 构建多平台二进制
- 打包 release 资产
- 上传到 GitHub Release

### 2. Skill Publish

文件：`.github/workflows/publish-skill.yml`

触发方式：

- 手动 `workflow_dispatch`

输入：

- `skill_path`
- `version`（可选）

依赖：

- GitHub Secret：`SKILL_HOME_API_KEY`
- GitHub Variable：`SKILL_HOME_REGISTRY_ENDPOINT`

职责：

- 构建当前仓库里的 `skill-home` CLI
- 验证和扫描目标 skill
- 推送到 registry

## 建议的域名落地

### DNS 与托管

- `skill-home.dev`
  - 指向现有 Web 门户
- `registry.skill-home.dev`
  - 反向代理到现有 registry 服务
- `get.skill-home.dev`
  - 返回固定安装脚本
  - 可托管在 GitHub Pages、Cloudflare Pages 或现有 Web 服务静态目录

## 维护约束

- GitHub Releases 只用于 CLI，不用于 skill 包分发
- skill 是否弃用由 `is_deprecated` 显式控制，不靠“旧版本”或“旧命名空间”推断
- 历史版本必须保持可 `pull`，只要 registry 中仍有对应版本记录和文件
