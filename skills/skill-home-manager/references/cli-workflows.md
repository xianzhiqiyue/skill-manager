# Skill Home CLI Workflows

## 常用路径

- 仓库根目录: `/home/zhuyue/code/skill-manager`
- CLI 源码目录: `/home/zhuyue/code/skill-manager/skill-home-cli`
- Codex 全局 skills 目录: `/mnt/c/Users/zhuyu/.codex/skills`
- skill-home 配置文件: `/home/zhuyue/.config/skill-home/config.yaml`

## 先确保 CLI 可用

优先跑 bundled script，让 skill 自己把 CLI 补齐:

```bash
/home/zhuyue/code/skill-manager/skills/skill-home-manager/scripts/bootstrap-cli.sh
```

如果 `skill-home` 已经存在且命中 `~/.local/bin/skill-home`、`/usr/local/bin/skill-home` 或当前仓库本地构建产物，这个脚本会直接打印版本并退出。  
如果 `skill-home` 不存在，或者当前命中的是别的旧二进制，它会在 apt 环境下自动安装 Go、从当前仓库源码构建 CLI，并默认覆盖 `~/.local/bin/skill-home`。

如果你明确需要系统级安装:

```bash
/home/zhuyue/code/skill-manager/skills/skill-home-manager/scripts/bootstrap-cli.sh --system
```

如果你明确要强制按当前源码重建:

```bash
/home/zhuyue/code/skill-manager/skills/skill-home-manager/scripts/rebuild-cli.sh
```

## 本地 skill 工作流

### 1. 新建一个 skill 子项目

优先用 bundled script:

```bash
/home/zhuyue/code/skill-manager/skills/skill-home-manager/scripts/create-local-skill.sh my-skill "我的 skill 描述"
```

如果要手动执行 CLI，先切到目标父目录再执行:

```bash
cd /home/zhuyue/code/skill-manager/skills
skill-home init my-skill
```

如果需要交互式模板:

```bash
cd /home/zhuyue/code/skill-manager/skills
skill-home create my-skill
```

### 2. 校验与扫描

```bash
skill-home validate /home/zhuyue/code/skill-manager/skills/my-skill
skill-home scan /home/zhuyue/code/skill-manager/skills/my-skill
```

### 3. 预览、导出、打包

```bash
skill-home preview /home/zhuyue/code/skill-manager/skills/my-skill -p codex
skill-home export /home/zhuyue/code/skill-manager/skills/my-skill -p codex
skill-home pack /home/zhuyue/code/skill-manager/skills/my-skill
```

### 4. 安装到 Codex

优先使用镜像模式，避免 WSL 符号链接问题:

```bash
/home/zhuyue/code/skill-manager/skills/skill-home-manager/scripts/install-to-codex.sh /home/zhuyue/code/skill-manager/skills/my-skill

skill-home sync /home/zhuyue/code/skill-manager/skills/my-skill --ide codex --global --mode mirror
find /mnt/c/Users/zhuyu/.codex/skills/my-skill -maxdepth 2 -type f
```

## 重新构建本地 CLI

当 `skill-home --help`、`doctor` 或 `sync` 的行为和源码不一致时:

```bash
/home/zhuyue/code/skill-manager/skills/skill-home-manager/scripts/bootstrap-cli.sh --force-rebuild

/home/zhuyue/code/skill-manager/skills/skill-home-manager/scripts/rebuild-cli.sh

cd /home/zhuyue/code/skill-manager/skill-home-cli
go test ./...
make build
sudo install -m 0755 /home/zhuyue/code/skill-manager/skill-home-cli/bin/skill-home /usr/local/bin/skill-home
skill-home version
```

## 环境排障

```bash
skill-home doctor
skill-home --debug sync /path/to/skill --ide codex --global --mode mirror
```

优先检查:

- 配置文件是否是 `/home/zhuyue/.config/skill-home/config.yaml`
- Codex 全局路径是否是 `/mnt/c/Users/zhuyu/.codex/skills`
- 本地 skill 是否通过了 `validate`
- 是否把“本地目录”误当成“远程 skill 引用”去执行 `install`

## registry 相关命令

只有在 API Key、endpoint、健康检查都正常时才继续:

```bash
skill-home login
skill-home search keyword
skill-home pull @namespace/name
skill-home install @namespace/name --ide codex --global --mode mirror
skill-home push /path/to/skill
```

如果 registry 健康检查失败，回退到本地 `validate/sync/export/pack` 工作流。
