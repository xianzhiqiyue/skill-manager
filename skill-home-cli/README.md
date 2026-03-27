# skill-home CLI

AI Skill 跨平台管理工具，支持 Claude Code、GitHub Copilot、Cursor、Codex 等多个 IDE，实现技能的"一次编写，到处同步"。

## 功能特性

- 🚀 **技能创建**: 快速生成符合规范的 SKILL.md 模板
- 🎨 **交互式创建**: 丰富的模板选择，支持交互式向导创建技能
- 📥 **技能导入**: 从 GitHub、Claude Code、Codex 等导入并转换技能
- ✅ **格式验证**: 验证 SKILL.md 格式是否符合标准
- 🔒 **安全扫描**: 本地检测恶意命令和提示词注入攻击
- 📦 **技能打包**: 将技能打包为可分发的 .zip 文件
- 🔄 **多 IDE 同步**: 一键同步技能到 Claude Code、GitHub Copilot、Cursor、Codex、OpenClaw
- 🔗 **双模同步**: 支持符号链接(Symlink)和物理镜像两种同步模式
- ☁️ **注册中心**: 推送、删除、拉取、搜索、详情查看与评分
- 📋 **技能列表**: 查看本地和云端已安装的技能
- 🩺 **环境诊断**: 检查 registry、认证、路径和 IDE 配置
- 📦 **生命周期管理**: install / uninstall / update

## 安装

### 使用安装脚本（推荐）

```bash
curl -fsSL http://47.122.112.210:8080/install.sh -o /tmp/skill-home-install.sh
bash /tmp/skill-home-install.sh
```

安装脚本会优先从 Skill Home 自身的 `/releases` 镜像下载 `checksums.txt` 和平台包；如果站内镜像缺失或不可用，才会回退到 GitHub Release。

指定版本：

```bash
bash /tmp/skill-home-install.sh v0.2.0
```

升级已安装 CLI：

```bash
skill-home self-update
skill-home self-update v0.2.3
```

`self-update` 也使用同样的 hosted-first、GitHub-fallback 策略。如需覆盖站内镜像地址，可设置 `SKILL_HOME_RELEASES_BASE_URL`。

### 从源码安装

```bash
git clone https://github.com/xianzhiqiyue/skill-manager.git
cd skill-manager/skill-home-cli
make install
```

## 快速开始

### 1. 创建新技能

#### 方式一：快速创建

```bash
skill-home init my-skill
cd my-skill
```

#### 方式二：交互式创建（推荐）

```bash
# 启动交互式向导
skill-home create

# 快速使用默认模板
skill-home create my-skill --quick

# 使用特定模板
skill-home create my-skill --template code-reviewer
```

支持的模板：
- `basic` - 基础模板
- `code-reviewer` - 代码审查专家
- `api-designer` - API 设计专家
- `refactor-expert` - 代码重构专家
- `test-expert` - 测试专家
- `doc-writer` - 文档编写专家
- `security-auditor` - 安全审计专家
- `performance-optimizer` - 性能优化专家

### 2. 编辑 SKILL.md

```markdown
---
name: my-skill
version: 1.0.0
description: 我的第一个 AI 技能
---

你是一个专业的代码审查助手...
```

### 3. 安全扫描

```bash
skill-home scan
```

### 4. 验证格式

```bash
skill-home validate
```

### 5. 从外部导入技能

```bash
# 从 GitHub 导入
skill-home import github.com/user/repo
skill-home import https://github.com/user/repo

# 从 Claude Code 导入
skill-home import claude://~/.claude/skills/my-skill

# 从 Codex 导入
skill-home import codex://~/.codex/skills/my-skill

# 从 Cursor 导入（自动转换 .mdc 格式）
skill-home import cursor://~/.cursor/rules/my-rule.mdc

# 指定输出目录
skill-home import github.com/user/repo -o ./my-imported-skill
```

### 6. 同步到 IDE

```bash
# 同步到所有启用的 IDE
skill-home sync

# 安装远程技能并同步到 IDE
skill-home install @user/my-skill

# 仅同步到特定 IDE
skill-home sync --ide cursor

# 同步到 OpenClaw
skill-home sync --ide openclaw

# 同步到全局配置
skill-home sync --global
```

## 命令参考

| 命令 | 说明 |
|------|------|
| `skill-home init <name>` | 创建新技能模板（基础版） |
| `skill-home create [name]` | 交互式创建技能（增强版） |
| `skill-home import <source>` | 从外部源导入技能 |
| `skill-home validate [path]` | 验证 SKILL.md 格式 |
| `skill-home scan [path]` | 扫描技能安全 |
| `skill-home pack [path]` | 打包技能 |
| `skill-home sync [path]` | 同步技能到 IDE |
| `skill-home list` | 列出已安装的技能 |
| `skill-home doctor` | 诊断本地环境与 registry 配置 |
| `skill-home self-update [version]` | 更新当前 CLI 到最新或指定版本 |
| `skill-home activity` | 查看最近的账号活动 |

### 注册中心命令

| 命令 | 说明 |
|------|------|
| `skill-home login` | 登录到注册中心 |
| `skill-home logout` | 登出 |
| `skill-home whoami` | 显示当前登录用户 |
| `skill-home push [path]` | 推送技能到注册中心 |
| `skill-home delete <skill-ref>` | 删除已发布的远程技能 |
| `skill-home delete-version <skill-ref>` | 删除已发布的远程技能版本 |
| `skill-home pull <skill-ref>` | 从注册中心拉取技能 |
| `skill-home install <skill-ref>` | 拉取并安装到本地 IDE |
| `skill-home uninstall <skill-ref>` | 从本地 IDE 卸载技能 |
| `skill-home update` | 更新本地缓存技能到最新版 |
| `skill-home info <skill-ref>` | 查看技能详情 |
| `skill-home search <keyword>` | 搜索云端技能 |
| `skill-home list --remote` | 列出云端技能 |
| `skill-home rate <skill-ref> --score 5` | 为技能评分 |
| `skill-home activity --action skill.rate` | 查看最近活动 |

**技能引用格式**: `@namespace/name@version`
- `my-skill` - 使用默认命名空间，最新版本
- `@user/my-skill` - 指定命名空间
- `my-skill@1.0.0` - 指定版本
- `@user/my-skill@1.0.0` - 完整格式

### 注册中心认证边界

- `push`、`delete`、`delete-version`、`rate`、`activity`、`whoami` 需要先执行 `skill-home login`
- `pull`、`install`、`update`、`search`、`info`、`list --remote` 对公开 skill 不需要登录
- 访问私有 skill 时，CLI 会提示先执行 `skill-home login` 并确认你有权限

### 登录工作流

`skill-home login` 现在支持两种方式：

1. 直接运行 `skill-home login`，按提示输入邮箱和密码。
CLI 会先登录 skill-home 服务，再自动创建一把 CLI 专用 API Key 并保存到本地配置。
2. 如果你已经在 Web 端 `/settings/api-keys` 创建好了 Key，也可以执行：

```bash
skill-home login --api-key "sk_xxx"
```

也可以直接通过环境变量注入：

```bash
export SKILL_HOME_API_KEY="sk_xxx"
```

## 配置

配置文件位于 `~/.config/skill-home/config.yaml`：

```yaml
# 注册中心配置
registry:
  endpoint: "https://registry.skill-home.dev"
  api_key: "your-api-key"  # 或设置环境变量 SKILL_HOME_API_KEY

# IDE 配置
ide:
  claude:
    enabled: true
    project_path: ".claude/skills"
    global_path: "~/.claude/skills"
  copilot:
    enabled: false
    project_path: ".github/skills"
    global_path: "~/.copilot/skills"
  cursor:
    enabled: true
    project_path: ".cursor/rules"
  codex:
    enabled: true
    project_path: ".agents/skills"
    global_path: "~/.agents/skills"
  openclaw:
    enabled: true
    project_path: "skills"
    global_path: "~/.openclaw/skills"

# 同步配置
sync:
  mode: "auto"              # auto | symlink | mirror
  conflict_strategy: "project_wins"
```

## 开发

```bash
# 安装依赖
make deps

# 构建
make build

# 运行测试
make test

# 代码格式化
make fmt

# 跨平台构建
make build-all
```

## 环境变量

| 变量名 | 说明 |
|--------|------|
| `SKILL_HOME_API_KEY` | API Key，优先于配置文件 |
| `SKILL_HOME_CONFIG` | 配置文件路径 |

## 调试模式

使用 `--debug` 启用详细日志输出：

```bash
skill-home --debug sync
```

## 项目结构

```
skill-home-cli/
├── cmd/skill-home/         # 程序入口
├── internal/
│   ├── cmd/                # CLI 命令实现
│   ├── config/             # 配置管理
│   ├── errors/             # 错误处理
│   ├── import/             # 技能导入器
│   │   ├── types/          # 导入器接口
│   │   └── github/         # GitHub 导入器
│   ├── logger/             # 日志系统
│   ├── registry/           # 注册中心客户端
│   ├── skill/              # 技能解析
│   ├── ide/                # IDE 适配器
│   └── sync/               # 同步引擎
├── pkg/
│   ├── archive/            # 归档工具
│   └── validator/          # 安全扫描器
└── Makefile
```

## 许可证

MIT License
