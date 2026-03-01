# skill-home CLI

AI Skill 跨平台管理工具，支持 Claude Code、Cursor、Codex 等多个 IDE，实现技能的"一次编写，到处同步"。

## 功能特性

- 🚀 **技能创建**: 快速生成符合规范的 SKILL.md 模板
- 🎨 **交互式创建**: 丰富的模板选择，支持交互式向导创建技能
- 📥 **技能导入**: 从 GitHub、Claude Code、Codex 等导入并转换技能
- ✅ **格式验证**: 验证 SKILL.md 格式是否符合标准
- 🔒 **安全扫描**: 本地检测恶意命令和提示词注入攻击
- 📦 **技能打包**: 将技能打包为可分发的 .tar.gz 文件
- 🔄 **多 IDE 同步**: 一键同步技能到 Claude Code、Cursor、Codex
- 🔗 **双模同步**: 支持符号链接(Symlink)和物理镜像两种同步模式
- ☁️ **注册中心**: 推送、拉取和搜索云端技能
- 📋 **技能列表**: 查看本地和云端已安装的技能

## 安装

### 从源码安装

```bash
git clone https://github.com/skill-home/cli.git
cd skill-home-cli
make install
```

### 使用安装脚本

```bash
curl -sSL https://get.skill-home.dev | sh
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

# 仅同步到特定 IDE
skill-home sync --ide cursor

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

### 注册中心命令

| 命令 | 说明 |
|------|------|
| `skill-home login` | 登录到注册中心 |
| `skill-home logout` | 登出 |
| `skill-home whoami` | 显示当前登录用户 |
| `skill-home push [path]` | 推送技能到注册中心 |
| `skill-home pull <skill-ref>` | 从注册中心拉取技能 |
| `skill-home search <keyword>` | 搜索云端技能 |

**技能引用格式**: `@namespace/name@version`
- `my-skill` - 使用默认命名空间，最新版本
- `@user/my-skill` - 指定命名空间
- `my-skill@1.0.0` - 指定版本
- `@user/my-skill@1.0.0` - 完整格式

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
  cursor:
    enabled: true
    project_path: ".cursor/rules"
  codex:
    enabled: true
    project_path: ".agents/skills"
    global_path: "~/.agents/skills"

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
