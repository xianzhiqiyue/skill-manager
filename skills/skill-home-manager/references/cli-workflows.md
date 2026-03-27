# Skill Home CLI Workflows

## 常用路径

- 当前工作目录: `pwd`
- Codex 全局 skills 目录: 优先使用 `$CODEX_HOME/skills`，常见值是 `~/.codex/skills`
- skill-home 配置文件: `~/.config/skill-home/config.yaml`
- 公共 CLI 安装脚本: `http://47.122.112.210:8080/install.sh`
- GitHub Releases: `https://github.com/xianzhiqiyue/skill-manager/releases`

除非用户明确在维护 `skill-manager` 仓库，否则不要假设本机存在任何 `skill-home` 源码目录。

## 先确保 CLI 可用

优先跑 bundled script，让 skill 自己从公开安装入口补齐 CLI:

```bash
scripts/bootstrap-cli.sh
```

这个脚本会从 Skill Home 服务下载 `install.sh`，再由脚本继续从同一服务的 `/releases` 拉取对应平台 CLI。
默认安装到 `~/.local/bin/skill-home`；如果系统级安装，使用 `--system`。

如果你明确需要系统级安装:

```bash
scripts/bootstrap-cli.sh --system
```

如果你明确要刷新到最新发布版本:

```bash
scripts/rebuild-cli.sh
```

如果你明确要安装指定版本:

```bash
scripts/bootstrap-cli.sh --version v0.2.4
```

## 本地 skill 工作流

### 1. 新建一个 skill 子项目

优先用 bundled script:

```bash
scripts/create-local-skill.sh my-skill "我的 skill 描述"
```

默认情况下：

- 如果当前目录下有 `skills/` 子目录，就优先创建到 `./skills`
- 否则直接创建到当前目录

如果要手动执行 CLI，先切到目标父目录再执行：

```bash
mkdir -p ./skills
cd ./skills
skill-home init my-skill
```

如果需要交互式模板:

```bash
skill-home create my-skill
```

### 2. 校验与扫描

```bash
skill-home validate ./my-skill
skill-home scan ./my-skill
```

### 3. 预览、导出、打包

```bash
skill-home preview ./my-skill -p codex
skill-home export ./my-skill -p codex
skill-home pack ./my-skill
```

### 4. 安装到 Codex

优先使用镜像模式，避免 WSL 符号链接问题:

```bash
scripts/install-to-codex.sh ./my-skill

skill-home sync ./my-skill --ide codex --global --mode mirror
find "${CODEX_HOME:-$HOME/.codex}/skills/my-skill" -maxdepth 2 -type f
```

## 刷新本地 CLI

当 `skill-home --help`、`doctor` 或 `sync` 的行为看起来不像最新 release 时:

```bash
scripts/bootstrap-cli.sh --force-reinstall

scripts/rebuild-cli.sh
skill-home self-update
```

## 环境排障

```bash
skill-home doctor
skill-home --debug sync /path/to/skill --ide codex --global --mode mirror
```

优先检查:

- 配置文件是否是 `~/.config/skill-home/config.yaml`
- Codex 全局路径是否与 `$CODEX_HOME/skills` 或 `~/.codex/skills` 一致
- `/releases/latest.json` 和对应版本产物是否已经同步到当前 Skill Home 服务
- 本地 skill 是否通过了 `validate`
- 是否把“本地目录”误当成“远程 skill 引用”去执行 `install`

## registry 相关命令

写操作先确认已登录；公开 skill 的读操作可匿名执行:

```bash
skill-home login
skill-home search keyword
skill-home pull @namespace/name
skill-home install @namespace/name --ide codex --global --mode mirror
skill-home push /path/to/skill
skill-home delete @namespace/name --yes
skill-home delete-version @namespace/name@1.2.3 --yes
```

登录补充:

- 直接运行 `skill-home login` 时，CLI 会提示输入邮箱和密码，并自动创建一把本机可复用的 CLI API Key
- 如果已经在 Web 端 `/settings/api-keys` 创建好了 Key，也可以执行 `skill-home login --api-key "sk_xxx"`
- 在 CI 或受控环境里，也可以直接注入 `SKILL_HOME_API_KEY`

规则:

- `push`、`delete`、`delete-version` 必须先 `skill-home login`
- `pull`、`install`、`update`、`search`、`info` 对公开 skill 不要求登录
- 匿名读取私有 skill 时，如果看到“该 skill 可能是私有的”，先登录再重试

如果 registry 健康检查失败，回退到本地 `validate/sync/export/pack` 工作流。
