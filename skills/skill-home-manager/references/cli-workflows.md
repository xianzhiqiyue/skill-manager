# Skill Home CLI 工作流

## 常用路径

- 当前工作目录: `pwd`
- Codex 全局 skills 目录: 优先使用 `$CODEX_HOME/skills`，常见值是 `~/.codex/skills`
- skill-home 配置文件: `~/.config/skill-home/config.yaml`
- 公共 CLI 安装脚本: `https://soulstore.ciqtek.com/skill-home/install.sh`
- GitHub Releases: `https://github.com/xianzhiqiyue/skill-manager/releases`

除非用户明确在维护 `skill-manager` 仓库，否则不要假设本机存在任何 `skill-home` 源码目录。
执行 bundled scripts 前，先把当前宿主上的已安装 skill 根目录记成 `skill_home_manager_root`。优先顺序:

- `$SKILL_HOME_MANAGER_ROOT`
- `skill-home` 配置文件里的 `ide.codex.global_path/skill-home-manager`
- `$CODEX_HOME/skills/skill-home-manager`
- `~/.codex/skills/skill-home-manager`
- `~/.agents/skills/skill-home-manager`

不要把这个路径写死成某台开发机的仓库绝对路径。

## 先确保 CLI 可用

优先跑 bundled script，让 skill 自己从公开安装入口补齐 CLI:

```bash
bash "$skill_home_manager_root/scripts/bootstrap-cli.sh"
```

这个脚本会优先下载部署页的 `install.sh`，部署页失败时回退到 GitHub 上的安装脚本。  
默认安装到 `~/.local/bin/skill-home`；如果系统级安装，使用 `--system`。

如果你明确需要系统级安装:

```bash
bash "$skill_home_manager_root/scripts/bootstrap-cli.sh" --system
```

如果你明确要刷新到最新发布版本:

```bash
bash "$skill_home_manager_root/scripts/rebuild-cli.sh"
```

如果你明确要安装指定版本:

```bash
bash "$skill_home_manager_root/scripts/bootstrap-cli.sh" --version v0.2.4
```

## 本地 skill 工作流

### 1. 新建一个 skill 子项目

优先用 bundled script:

```bash
bash "$skill_home_manager_root/scripts/create-local-skill.sh" my-skill "我的 skill 描述"
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

### 2. 补齐发布前元数据

无论是 `init`、`create` 还是导入后的 skill，都要在 `validate` 前确认 `SKILL.md` 里有：

- 1 个合法的 `category`
- 1 到 4 个合法的 `official tags`

分类和标签选择参考：

```bash
cat references/publish-taxonomy.md
```

代理默认应该根据 skill 名称、描述和正文内容先推断并写回；只有存在歧义时再问用户一个短问题。

### 3. 校验与扫描

```bash
skill-home validate ./my-skill
skill-home scan ./my-skill
```

### 4. 预览、导出、打包

```bash
skill-home preview ./my-skill -p codex
skill-home export ./my-skill -p codex
skill-home pack ./my-skill
```

### 5. 安装到 Codex

优先使用镜像模式，避免 WSL 符号链接问题:

```bash
bash "$skill_home_manager_root/scripts/install-to-codex.sh" ./my-skill

skill-home sync ./my-skill --ide codex --global --mode mirror
find "${CODEX_HOME:-$HOME/.codex}/skills/my-skill" -maxdepth 2 -type f
```

## 刷新本地 CLI

当 `skill-home --help`、`doctor` 或 `sync` 的行为看起来不像最新 release 时:

```bash
bash "$skill_home_manager_root/scripts/bootstrap-cli.sh" --force-reinstall

bash "$skill_home_manager_root/scripts/rebuild-cli.sh"
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
- 本地 skill 是否已经补齐 `category/tags` 并通过了 `validate`
- 是否把“本地目录”误当成“远程 skill 引用”去执行 `install`

## registry 相关命令

写操作先确认已登录；公开 skill 的读操作可匿名执行:

```bash
skill-home login
skill-home list --remote
skill-home search keyword
skill-home pull @namespace/name
skill-home install @namespace/name --ide codex --global --mode mirror
skill-home push /path/to/skill
skill-home delete @namespace/name --yes
skill-home delete-version @namespace/name@1.2.3 --yes
```

规则:

- `push`、`delete`、`delete-version` 必须先 `skill-home login`
- `push` 在交互终端里会尝试补齐缺失的 `category/tags`，但代理默认仍应先整理好 `SKILL.md`
- `pull`、`install`、`update`、`search`、`info`、`list --remote` 对公开 skill 不要求登录
- 匿名读取私有 skill 时，如果看到“该 skill 可能是私有的”，先登录再重试

远程目录缓存说明:

- `skill-home list --remote` 与 `skill-home search` 会先检查远程目录版本；版本未变化时优先复用本地目录缓存
- 注册中心临时失败但本地已有对应缓存时，CLI 会自动回退，并提示“结果可能过期”
- `--format json` 下，结果主体仍保持纯 JSON；过期提示只会输出到 `stderr`
- 这层缓存只覆盖公开目录结构，不保证下载量、评分等动态统计字段实时

如果 registry 健康检查失败，回退到本地 `validate/sync/export/pack` 工作流。
