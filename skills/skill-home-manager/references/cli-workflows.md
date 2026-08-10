# Skill Home CLI 工作流

## 常用路径

- 当前工作目录: `pwd`
- Codex 全局 skills 目录: 不要写死，优先探测 `SKILL_HOME_CODEX_SKILLS_DIR`、`skill-home` 配置里的 `ide.codex.global_path`、`$CODEX_HOME/skills`，再回退到宿主环境里已存在的 `~/.agents/skills` 或 `~/.codex/skills`
- Xigua 全局 skills 目录: 不要写死，优先探测 `SKILL_HOME_XIGUA_SKILLS_DIR`、`skill-home` 配置里的 `ide.xigua.global_path`，再回退到 `~/.xigua-agent/skills`
- 平台上下文文件: 已安装 skill 根目录下的 `.skill-home/platform-context.json`。每次使用本 skill 前先读取；缺失时用 `scripts/inject-platform-context.sh auto "$skill_home_manager_root"` 写入。
- skill-home 配置文件: `~/.config/skill-home/config.yaml`
- CLI API Key 环境变量: `SKILL_HOME_API_KEY`，优先于配置文件里的 `registry.api_key`
- Linux/macOS 公共 CLI 安装脚本: `https://soulstore.ciqtek.com/skill-home/install.sh`
- Windows x64 公共 CLI 安装脚本: `https://soulstore.ciqtek.com/skill-home/install.ps1`
- GitHub Releases: `https://github.com/xianzhiqiyue/skill-manager/releases`

除非用户明确在维护 `skill-manager` 仓库，否则不要假设本机存在任何 `skill-home` 源码目录。
执行 bundled scripts 前，先把当前宿主上的已安装 skill 根目录记成 `skill_home_manager_root`。优先顺序:

- `$SKILL_HOME_MANAGER_ROOT`
- `skill-home` 配置文件里的 `ide.codex.global_path/skill-home-manager`
- `skill-home` 配置文件里的 `ide.xigua.global_path/skill-home-manager`
- `$CODEX_HOME/skills/skill-home-manager`
- `~/.agents/skills/skill-home-manager`
- `~/.codex/skills/skill-home-manager`
- `~/.xigua-agent/skills/skill-home-manager`

不要把这个路径写死成某台开发机的仓库绝对路径。

## 平台上下文注入

安装引导必须把平台信息固化到已安装 skill 根目录:

```bash
bash "$skill_home_manager_root/scripts/inject-platform-context.sh" auto "$skill_home_manager_root"
```

如果刚完成安装且已知道目标平台和路径，使用显式参数:

```bash
bash "$skill_home_manager_root/scripts/inject-platform-context.sh" codex "$installed_skill_dir" global "$target_path" mirror
bash "$skill_home_manager_root/scripts/inject-platform-context.sh" xigua "$installed_skill_dir" global "$target_path" mirror
```

后续操作先读取 `.skill-home/platform-context.json`：

- `platform=codex`: 优先使用 Codex 路径和 mirror 模式排障。
- `platform=xigua`: 优先检查 Xigua package 布局，确认 `SKILL.md` 和 `skill.json` 同时存在。
- 如果文件缺失或与用户本次指定平台冲突，完成操作后刷新这个文件。

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

## Agent 自主完成 OAuth 登录和配置

涉及 `push`、`delete`、`delete-version`、`activity`、`whoami` 或私有 skill 读取时，Agent 直接运行 bundled script：

```bash
bash "$skill_home_manager_root/scripts/ensure-oauth-login.sh"
```

这个脚本负责检查或安装 CLI、复用已有凭证、刷新不支持 OAuth 的旧 CLI、发起授权、等待结果、保存配置并执行 `whoami` 验证。脚本成功后，Agent 不再等待用户回复，直接继续原始任务。

首次登录时，CLI 会打开一次性授权页。Agent 保持该命令会话运行并持续轮询，只提醒用户在页面点击“允许”；不要让用户运行命令、生成或复制 API Key、编辑配置，或授权后回来确认。

浏览器没有自动打开时，CLI 会打印完整授权链接。Agent 把这个链接明确交给用户并继续等待。只有已经证实当前环境无法尝试打开浏览器时，才使用：

```bash
bash "$skill_home_manager_root/scripts/ensure-oauth-login.sh" --no-browser
```

指定注册中心或延长等待时间时，由 Agent 直接传参：

```bash
bash "$skill_home_manager_root/scripts/ensure-oauth-login.sh" --server https://registry.example.com --oauth-timeout 15m
```

已有的配置凭证或 `SKILL_HOME_API_KEY` 会被自动复用，适用于无人值守环境。环境变量凭证失效时，脚本只在自身进程内忽略它并切换到 OAuth，不要求用户清理环境。不要在日志、提交信息或回复中展示完整 API Key。

如果刷新后的 CLI 仍没有 `--no-browser`，或服务端没有 OAuth device 接口，说明已发布版本尚未具备 OAuth 能力。此时停止需要登录的动作并报告部署阻塞，不得退回“让用户去网页生成 Key”。

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

- 发布命名空间使用当前 API Key 对应用户的发布作用域（`skill-home whoami` 输出里的“发布作用域”，引用形态为 `@<发布作用域>/<skill-name>`）
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
skill-home pack ./my-skill --output ./dist/my-skill.zip
```

如果用户没有明确要求输出位置，先根据当前 skill 所在目录和后续用途决定是否显式传 `--output`；不要默认把压缩包落到固定路径。

### 5. 安装到 Codex 或 Xigua

优先使用镜像模式，避免 WSL 符号链接问题:

```bash
bash "$skill_home_manager_root/scripts/install-to-codex.sh" ./my-skill

skill-home sync ./my-skill --ide codex --global --mode mirror
skill-home doctor
```

安装后的校验路径也不要写死。优先查看 `skill-home doctor` 输出的 `codex 全局路径`，再到该目录下确认 `my-skill/SKILL.md` 是否存在。
`install-to-codex.sh` 会自动写入 `.skill-home/platform-context.json`。如果手动执行 `skill-home sync`，同步成功后补跑 `inject-platform-context.sh codex`。

安装到 Xigua:

```bash
bash "$skill_home_manager_root/scripts/install-to-xigua.sh" ./my-skill

skill-home sync ./my-skill --ide xigua --global --mode mirror
skill-home doctor
```

安装后的校验路径同样不要写死。优先查看 `skill-home doctor` 输出的 `xigua 全局路径`，再到该目录下确认 `my-skill/SKILL.md` 和 `my-skill/skill.json` 是否存在。
`install-to-xigua.sh` 会自动写入 `.skill-home/platform-context.json`。如果手动执行 `skill-home sync`，同步成功后补跑 `inject-platform-context.sh xigua`。

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
skill-home --debug sync /path/to/skill --ide xigua --global --mode mirror
```

优先检查:

- 配置文件是否是 `~/.config/skill-home/config.yaml`
- Codex 全局路径是否与 `skill-home doctor` 输出一致
- Xigua 全局路径是否与 `skill-home doctor` 输出一致
- 本地 skill 是否已经补齐 `category/tags` 并通过了 `validate`
- 是否把“本地目录”误当成“远程 skill 引用”去执行 `install`

## registry 相关命令

写操作先由 Agent 执行 OAuth 登录脚本；公开 skill 的读操作可匿名执行:

```bash
bash "$skill_home_manager_root/scripts/ensure-oauth-login.sh"
skill-home list --remote
skill-home search keyword
skill-home pull @namespace/name
skill-home install @namespace/name --ide codex --global --mode mirror
skill-home install @namespace/name --ide xigua --global --mode mirror
skill-home push /path/to/skill
skill-home delete @namespace/name --yes
skill-home delete-version @namespace/name@1.2.3 --yes
```

规则:

- `push`、`delete`、`delete-version` 必须先具备有效登录态；Agent 用 bundled script 自动复用凭证或完成 OAuth，用户只点击授权页
- `push` 默认发布到当前用户的发布作用域；OAuth 与兼容登录都会把 `local.default_namespace` 保存为 `@<发布作用域>`；不要默认使用登录用户名、`@user`、示例命名空间或历史 manifest 命名空间
- `push` 在交互终端里会尝试补齐缺失的 `category/tags`，但代理默认仍应先整理好 `SKILL.md`
- `pull`、`install`、`update`、`search`、`info`、`list --remote` 对公开 skill 不要求登录
- 匿名读取私有 skill 时，如果看到“该 skill 可能是私有的”，Agent 先执行 OAuth 登录脚本并自动重试

远程目录缓存说明:

- `skill-home list --remote` 与 `skill-home search` 会先检查远程目录版本；版本未变化时优先复用本地目录缓存
- 注册中心临时失败但本地已有对应缓存时，CLI 会自动回退，并提示“结果可能过期”
- `--format json` 下，结果主体仍保持纯 JSON；过期提示只会输出到 `stderr`
- 这层缓存只覆盖公开目录结构，不保证下载量、评分等动态统计字段实时

如果 registry 健康检查失败，回退到本地 `validate/sync/export/pack` 工作流。
