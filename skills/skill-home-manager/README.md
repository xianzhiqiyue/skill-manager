# skill-home-manager

`skill-home-manager` 是面向 Codex / OpenClaw 等 AI 助手的工作流 skill，用来把 `skill-home` CLI 的常用操作封装成一组可复用脚本和约束，让“创建、校验、发布、安装 skill”不需要每次从头拼命令。

## 能力概览

| 能力域 | 说明 |
|--------|------|
| CLI 补齐 | 检查本机是否已有 `skill-home`，必要时自动安装或刷新到已发布版本 |
| 本地 skill 工作流 | 新建、校验、安全扫描、打包、预览、导出 |
| 发布前元数据整理 | 在 `validate/pack/push` 前补齐 `category + official tags` |
| Codex 安装 | 将本地 skill 镜像安装到 Codex 全局目录 |
| 远程 registry 工作流 | 查看公开目录、搜索、拉取、安装、发布、删除版本 |
| 环境排障 | 通过 `doctor`、`--debug`、路径检查定位配置问题 |

## 适用场景

- 想在当前仓库里快速新建一个 skill
- 想把本地 skill 同步安装到 Codex
- 想确认本机 `skill-home` CLI 是否过旧
- 想排查为什么 skill 没生效
- 想发布、删除或拉取远程 skill

## Bundled Scripts

| 脚本 | 作用 | 典型用法 |
|------|------|----------|
| `scripts/bootstrap-cli.sh` | 安装或刷新已发布的 `skill-home` CLI | `bash scripts/bootstrap-cli.sh --version v0.2.16` |
| `scripts/create-local-skill.sh` | 新建本地 skill 骨架并提示后续补齐分类元数据 | `bash scripts/create-local-skill.sh my-skill "描述"` |
| `scripts/install-to-codex.sh` | 把本地 skill 安装到 Codex 全局目录 | `bash scripts/install-to-codex.sh ./skills/my-skill` |
| `scripts/rebuild-cli.sh` | 重新安装最新已发布 CLI | `bash scripts/rebuild-cli.sh` |

被调用时，不要把脚本路径写死成开发机仓库绝对路径。优先先解析 `skill_home_manager_root`，再通过 `bash "$skill_home_manager_root/scripts/<script>.sh"` 调用 bundled scripts。

## 常见工作流

### 1. 补齐本机 CLI

```bash
bash scripts/bootstrap-cli.sh
skill-home version
```

### 2. 新建并整理一个本地 skill

```bash
bash scripts/create-local-skill.sh my-skill "我的 skill 描述"

# 按 references/publish-taxonomy.md 补齐 category 和 official tags
skill-home validate ./my-skill
skill-home scan ./my-skill
```

### 3. 安装到 Codex

```bash
bash scripts/install-to-codex.sh ./my-skill
```

### 4. 查看远程公开目录

```bash
skill-home list --remote
skill-home search codex
skill-home info @skill-home/skill-home-manager
```

### 5. 发布或删除远程 skill

```bash
skill-home login
skill-home push ./my-skill
skill-home delete @team/my-skill --yes
skill-home delete-version @team/my-skill@1.0.0 --yes
```

交互终端里的 `skill-home push` 会在缺少 `category/tags` 时尝试补齐，但这个 skill 的默认要求仍然是先整理好 `SKILL.md`，再进入 `validate -> pack -> push`。

## 与 CLI / Registry 的关系

- 这个 skill 不是替代 `skill-home` CLI，而是把常用 CLI 工作流整理成可直接复用的 Codex 能力层
- 公开 skill 的读取命令默认可匿名执行，例如 `list --remote`、`search`、`pull`、`install`
- 写操作如 `push`、`delete`、`delete-version` 需要先 `skill-home login`
- `list --remote` 与 `search` 已接入目录版本缓存；目录版本未变化时会优先复用本地缓存
- 发布前的 `category + official tags` 词表来自同一个 taxonomy 参考，不应由代理或用户自由发明新官方标签

## 相关文件

- 主定义：[SKILL.md](./SKILL.md)
- 命令模板：[references/cli-workflows.md](./references/cli-workflows.md)
- 分类词表：[references/publish-taxonomy.md](./references/publish-taxonomy.md)
- 安装脚本：[scripts/bootstrap-cli.sh](./scripts/bootstrap-cli.sh)
- Codex 安装脚本：[scripts/install-to-codex.sh](./scripts/install-to-codex.sh)
