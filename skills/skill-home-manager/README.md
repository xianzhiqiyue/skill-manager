# skill-home-manager

`skill-home-manager` 是面向 Codex / OpenClaw / Xigua 等 AI 助手的工作流 skill，用来把 `skill-home` CLI 的常用操作封装成一组可复用脚本和约束，让“创建、校验、发布、安装 skill”不需要每次从头拼命令。

仓库级统一发布口径见 [../../docs/release-process.md](../../docs/release-process.md)。如果本文件与统一发布流程冲突，以统一发布流程为准。

## 能力概览

| 能力域 | 说明 |
|--------|------|
| CLI 补齐 | 检查本机是否已有 `skill-home`，必要时自动安装或刷新到已发布版本 |
| 本地 skill 工作流 | 新建、校验、安全扫描、打包、预览、导出 |
| 发布前元数据整理 | 在 `validate/pack/push` 前补齐发布命名空间、`category + official tags` |
| Codex / Xigua 安装 | 将本地 skill 镜像安装到 Codex 或 Xigua 全局目录 |
| 远程 registry 工作流 | 查看公开目录、搜索、拉取、安装、发布、删除版本 |
| 环境排障 | 通过 `doctor`、`--debug`、路径检查定位配置问题 |

## 适用场景

- 想在当前仓库里快速新建一个 skill
- 想把本地 skill 同步安装到 Codex 或 Xigua
- 想确认本机 `skill-home` CLI 是否过旧
- 想排查为什么 skill 没生效
- 想发布、删除或拉取远程 skill

## Bundled Scripts

| 脚本 | 作用 | 典型用法 |
|------|------|----------|
| `scripts/bootstrap-cli.sh` | 安装或刷新已发布的 `skill-home` CLI | `bash scripts/bootstrap-cli.sh --version v0.2.16` |
| `scripts/ensure-oauth-login.sh` | 自动复用登录态或发起 OAuth，等待授权并验证账号 | `bash scripts/ensure-oauth-login.sh` |
| `scripts/create-local-skill.sh` | 新建本地 skill 骨架并提示后续补齐分类元数据 | `bash scripts/create-local-skill.sh my-skill "描述"` |
| `scripts/install-to-codex.sh` | 把本地 skill 安装到 Codex 全局目录 | `bash scripts/install-to-codex.sh ./skills/my-skill` |
| `scripts/install-to-xigua.sh` | 把本地 skill 安装到 Xigua 全局目录 | `bash scripts/install-to-xigua.sh ./skills/my-skill` |
| `scripts/inject-platform-context.sh` | 写入或刷新已安装 skill 的平台上下文 | `bash scripts/inject-platform-context.sh auto ./skills/skill-home-manager` |
| `scripts/rebuild-cli.sh` | 重新安装最新已发布 CLI | `bash scripts/rebuild-cli.sh` |

被调用时，不要把脚本路径、Codex 安装目录或打包输出目录写死成开发机绝对路径。优先先解析 `skill_home_manager_root`，再通过 `bash "$skill_home_manager_root/scripts/<script>.sh"` 调用 bundled scripts；涉及安装或打包时，再由 agent 根据配置、环境变量和当前目录判断真实目标路径。

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
skill-home pack ./my-skill --output ./dist/my-skill.zip
```

### 3. 安装到 Codex

```bash
bash scripts/install-to-codex.sh ./my-skill
```

安装校验不要假设固定目录。优先读取 `skill-home` 配置里的 `ide.codex.global_path`，其次看 `$CODEX_HOME/skills`，再回退到宿主环境里真实存在的 `~/.agents/skills` 或 `~/.codex/skills`。
安装脚本会在安装后的 skill 根目录写入 `.skill-home/platform-context.json`，用于后续调用识别当前平台和安装模式。

### 4. 安装到 Xigua

```bash
bash scripts/install-to-xigua.sh ./my-skill
```

安装校验不要假设固定目录。优先读取 `SKILL_HOME_XIGUA_SKILLS_DIR`，其次读取 `skill-home` 配置里的 `ide.xigua.global_path`，再回退到 `~/.xigua-agent/skills`。成功后目标包内应同时存在 `SKILL.md` 和 `skill.json`。
安装脚本会在安装后的 skill 根目录写入 `.skill-home/platform-context.json`，用于后续调用识别当前平台和安装模式。

### 5. 查看远程公开目录

```bash
skill-home list --remote
skill-home search codex
skill-home info @skill-home/skill-home-manager
```

### 6. 发布或删除远程 skill

```bash
bash scripts/ensure-oauth-login.sh  # Agent 执行并等待；用户只在授权页点击允许
skill-home validate ./my-skill
skill-home scan ./my-skill
skill-home pack ./my-skill --output ./dist/my-skill-<version>.zip
skill-home push ./my-skill  # 默认发布到当前用户的拼音发布作用域
skill-home delete @<发布作用域>/my-skill --yes
skill-home delete-version @<发布作用域>/my-skill@1.0.0 --yes
```

统一发布口径下，skill 的默认顺序是：先递增 `SKILL.md` 里的版本号，再执行 `validate -> scan -> pack -> push`。发布命名空间默认使用当前用户的发布作用域；不要沿用登录用户名、`@user`、示例命名空间、历史 manifest 命名空间或旧 `default_namespace`。交互终端里的 `skill-home push` 会在缺少 `category/tags` 时尝试补齐，但这个 skill 的默认要求仍然是先整理好 `SKILL.md`，再进入发布动作。如果用户没有明确指定打包输出位置，agent 需要先判断当前工作目录和后续发布动作是否要求显式 `--output`，不要默认把产物丢到某个固定路径。

## 与 CLI / Registry 的关系

- 这个 skill 不是替代 `skill-home` CLI，而是把常用 CLI 工作流整理成可直接复用的 Codex 能力层
- 公开 skill 的读取命令默认可匿名执行，例如 `list --remote`、`search`、`pull`、`install`
- 写操作如 `push`、`delete`、`delete-version` 需要先具备有效登录态；Agent 运行 `ensure-oauth-login.sh` 自动完成检查、OAuth、配置和验证，用户只点击授权页；无人值守环境继续支持 `SKILL_HOME_API_KEY`
- `push` 默认使用当前用户的发布作用域；OAuth 与兼容登录都会把 `local.default_namespace` 保存为 `@<发布作用域>`；旧版 CLI 需要刷新或显式传 `--namespace @<发布作用域>`
- `list --remote` 与 `search` 已接入目录版本缓存；目录版本未变化时会优先复用本地缓存
- 发布前的 `category + official tags` 词表来自同一个 taxonomy 参考，不应由代理或用户自由发明新官方标签

## 相关文件

- 主定义：[SKILL.md](./SKILL.md)
- 命令模板：[references/cli-workflows.md](./references/cli-workflows.md)
- 分类词表：[references/publish-taxonomy.md](./references/publish-taxonomy.md)
- 安装脚本：[scripts/bootstrap-cli.sh](./scripts/bootstrap-cli.sh)
- OAuth 登录脚本：[scripts/ensure-oauth-login.sh](./scripts/ensure-oauth-login.sh)
- Codex 安装脚本：[scripts/install-to-codex.sh](./scripts/install-to-codex.sh)
- Xigua 安装脚本：[scripts/install-to-xigua.sh](./scripts/install-to-xigua.sh)
