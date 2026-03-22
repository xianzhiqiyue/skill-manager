# AI 编辑器技能系统调研报告

## 概述

本报告调研了四种主流 AI 编辑器/代理的技能系统：Claude Code、Codex、Cursor 和 OpenClaw。这些系统允许开发者为 AI 助手创建自定义指令、规则和技能包，以增强其在特定领域或项目中的表现。

---

## 1. Claude Code 技能系统

### 1.1 核心概念

Claude Code 使用 **Skills** 机制，通过 `SKILL.md` 文件定义 AI 助手的上下文增强指令。技能可以包含项目特定的规范、代码模式、架构决策和工具使用指南。

### 1.2 文件格式

```markdown
---
# 技能元数据
name: skill-name
version: 1.0.0
description: 技能描述

# 可选字段
namespace: "@user"
author: Author Name
tags: [tag1, tag2]
license: MIT
homepage: https://github.com/...

# IDE 配置
ide_config:
  claude:
    globs: ["**/*.{js,ts}"]
    auto_activate: true
    file_context: true
---

# 技能内容（Markdown 格式）

## 核心能力
...

## 工作流程
...
```

### 1.3 关键特性

| 特性 | 说明 |
|------|------|
 **存储位置** | `~/.claude/skills/`（全局）或 `.claude/skills/`（项目级） |
| **激活方式** | 自动激活（基于 globs 匹配）或手动引用 |
| **文件匹配** | 使用 glob 模式决定何时激活技能 |
| **上下文感知** | 可设置 `file_context: true` 获取文件内容 |

### 1.4 目录结构

```
~/.claude/skills/
├── skill-name/
│   ├── SKILL.md           # 技能定义文件
│   └── README.md          # 可选说明文档
```

### 1.5 使用场景

- 项目特定的代码规范
- 框架/库的使用指南
- 团队编码标准
- 复杂工作流的步骤指导
- 领域专业知识（如安全、性能优化）

---

## 2. Codex 技能系统

### 2.1 核心概念

Codex（OpenAI 的代码编辑器）使用 **Agents** 和 **Skills** 机制。技能通过 `.mdc` 文件（Markdown with Config）定义，存储在 `.codex/agents/` 或 `.codex/skills/` 目录。

### 2.2 文件格式

Codex 使用 `.mdc` 扩展名，包含 YAML frontmatter：

```markdown
---
name: skill-name
description: 技能描述
tools: [read, edit, bash, glob, grep]
 glob: "**/*.py"
auto_execute: false
---

# 技能指令

当用户询问 Python 代码时：
1. 先检查代码风格
2. 提供类型提示建议
3. 确保遵循 PEP 8 规范
```

### 2.3 关键特性

| 特性 | 说明 |
|------|------|
| **存储位置** | `.codex/agents/` 或 `.codex/skills/` |
| **文件扩展名** | `.mdc` |
| **工具声明** | 在 frontmatter 中声明可用的工具 |
| **glob 匹配** | 指定文件匹配模式 |
| **自动执行** | 可设置是否自动执行某些操作 |

### 2.4 与 Claude Code 的区别

1. **文件扩展名不同**：Codex 使用 `.mdc`，Claude 使用 `.md`
2. **工具显式声明**：Codex 需要在 frontmatter 声明工具权限
3. **目录结构**：Codex 使用 `agents/` 和 `skills/` 子目录

---

## 3. Cursor Rules 系统

### 3.1 核心概念

Cursor 使用 **Rules** 机制，通过 `.cursor/rules/*.md` 文件定义系统提示。规则是项目本地的，用于指导 Cursor IDE 的 AI 助手在特定代码库中的行为。

### 3.2 文件格式

```markdown
---
name: rule-name
metadata:
  author: Author Name
  version: 1.0.0
---

# 角色定义

你是 Senior software engineer，具有 Java 开发经验。

## 目标

根据以下规则审查和改进代码...

## 规则详情

1. 使用 Optional 替代 null 检查
2. 优先使用 Stream API
3. 遵循 SOLID 原则
```

### 3.3 关键特性

| 特性 | 说明 |
|------|------|
| **存储位置** | `.cursor/rules/*.md`（项目本地） |
| **命名规范** | 使用数字前缀组织（100-, 110-, 120-） |
| **引用方式** | 在聊天中使用 `@rule-name` |
| **分类管理** | 按功能领域分组 |

### 3.4 规则分类示例

| 类别 | 编号范围 | 示例 |
|------|----------|------|
| 系统提示 | 100-109 | Java 系统提示列表 |
| 构建系统 | 110-119 | Maven 最佳实践、依赖管理 |
| 设计规则 | 120-129 | 面向对象设计、类型设计 |
| 编码规则 | 130-139 | 异常处理、安全编码、并发 |
| 测试规则 | 140-149 | 单元测试、集成测试 |
| 重构规则 | 150-159 | 现代 Java 特性、函数式编程 |
| 性能规则 | 160-169 | JMH 基准测试、性能分析 |
| 文档规则 | 170-179 | README、ADR、UML 图 |

### 3.5 规则类型

1. **自动规则** - 无需用户交互自动应用
2. **交互式规则** - 生成前询问用户问题
3. **非对话规则** - 生成脚本/命令，不进入对话模式

### 3.6 引用示例

```
用户: 改进这个类的设计 @121-java-object-oriented-design

用户: 生成 ADR 文档 @171-java-adr

用户: 创建性能测试 @151-java-performance-jmeter
```

---

## 4. OpenClaw / ClawHub 技能系统

### 4.1 核心概念

OpenClaw（Clawdbot）使用 **Skills** 机制，通过 `SKILL.md` 文件定义可发布的技能包。ClawHub 是公共技能注册中心，支持版本管理、向量搜索和 CLI 工具。

### 4.2 架构

| 组件 | 技术栈 |
|------|--------|
| 前端 | TanStack Start (React + Vite/Nitro) |
| 后端 | Convex (DB + 文件存储 + HTTP actions) |
| 认证 | GitHub OAuth |
| 搜索 | OpenAI 嵌入 + Convex 向量搜索 |
| CLI | `clawhub` 命令行工具 |

### 4.3 文件格式

```markdown
---
name: my-skill
description: 技能的简短描述
version: 1.0.0
license: MIT
homepage: https://github.com/example/my-skill

metadata:
  openclaw:
    # 运行时要求
    requires:
      env:
        - API_KEY
        - OPTIONAL_VAR
      bins:
        - curl
        - jq
      anyBins:
        - docker
        - podman
      config:
        - ~/.mytoolrc

    # 主要环境变量
    primaryEnv: API_KEY

    # 是否始终激活
    always: false

    # 调用键
    skillKey: my-skill

    # 显示图标
    emoji: "✅"

    # 操作系统限制
    os: ["macos", "linux"]

    # 依赖安装
    install:
      - kind: brew
        formula: jq
        bins: [jq]
      - kind: node
        package: typescript
        bins: [tsc]
      - kind: go
        package: github.com/tool/cmd
        bins: [tool]
---

# 技能文档

## 安装

```bash
clawhub install my-skill
```

## 使用

...
```

### 4.4 CLI 命令

```bash
# 认证
clawhub login
clawhub whoami

# 发现
clawhub search <query>
clawhub explore

# 本地管理
clawhub install <slug>
clawhub uninstall <slug>
clawhub list
clawhub update --all

# 检查
clawhub inspect <slug>

# 发布
clawhub publish <path>
clawhub sync
```

### 4.5 元数据字段详解

| 字段 | 类型 | 说明 |
|------|------|------|
| `requires.env` | `string[]` | 必需的环境变量 |
| `requires.bins` | `string[]` | 必需的二进制文件 |
| `requires.anyBins` | `string[]` | 至少需要一个的二进制文件 |
| `requires.config` | `string[]` | 配置文件路径 |
| `primaryEnv` | `string` | 主要凭证环境变量 |
| `always` | `boolean` | 是否始终激活 |
| `skillKey` | `string` | 技能调用键 |
| `emoji` | `string` | 显示图标 |
| `homepage` | `string` | 主页链接 |
| `os` | `string[]` | 操作系统限制 |
| `install` | `array` | 依赖安装规范 |

### 4.6 安装规范类型

```yaml
install:
  # Homebrew
  - kind: brew
    formula: jq
    bins: [jq]

  # Node.js
  - kind: node
    package: typescript
    bins: [tsc]

  # Go
  - kind: go
    package: github.com/user/tool
    bins: [tool]

  # Python (uv)
  - kind: uv
    package: requests
    bins: []
```

### 4.7 技能存储结构

```
skill-folder/
├── SKILL.md              # 必需 - 主技能文件
├── .clawhubignore        # 可选 - 忽略模式
├── .gitignore           # 可选 - Git 忽略
└── [支持文件]            # 可选 - 仅文本文件
```

### 4.8 限制

| 限制项 | 值 |
|--------|-----|
| 总包大小 | 50MB |
| 文件类型 | 仅文本文件 |
| 嵌入文件 | SKILL.md + 最多约40个非 .md 文件 |

### 4.9 Nix 插件支持

特殊技能类型，捆绑 CLI 二进制文件 + 技能包：

```yaml
---
name: peekaboo
description: Capture and automate macOS UI
metadata:
  clawdbot:
    nix:
      plugin: "github:clawdbot/nix-tools?dir=tools/peekaboo"
      systems: ["aarch64-darwin"]
---
```

---

## 5. 四者对比

| 特性 | Claude Code | Codex | Cursor | OpenClaw |
|------|-------------|-------|--------|----------|
| **文件格式** | Markdown + YAML | Markdown + YAML | Markdown + YAML | Markdown + YAML |
| **文件扩展名** | `.md` | `.mdc` | `.md` | `.md` |
| **主文件名** | `SKILL.md` | `*.mdc` | `*.md` | `SKILL.md` |
| **存储位置** | `~/.claude/skills/` | `.codex/agents/` | `.cursor/rules/` | 本地 + 注册中心 |
| **作用域** | 全局/项目 | 项目 | 项目 | 全局（通过注册中心） |
| **注册中心** | 无 | 无 | 无 | ClawHub |
| **CLI 工具** | 无 | 部分 | 无 | `clawhub` |
| **版本控制** | 无 | 无 | 无 | 是（semver） |
| **搜索功能** | 无 | 无 | 无 | 向量搜索 |
| **依赖声明** | 无 | 无 | 无 | 完整支持 |
| **自动激活** | 是（globs） | 是（glob） | 否（手动@） | 可配置 |
| **分发方式** | 复制/ git | 复制/ git | 复制/ git | CLI / 注册中心 |

---

## 6. 建议的统一技能格式

基于以上调研，建议采用以下统一格式，兼容多种 AI 编辑器：

```markdown
---
# 基础元数据
name: skill-name
version: 1.0.0
description: 技能描述
namespace: "@user"
author: Author Name
license: MIT
homepage: https://github.com/...
tags: [ai-assistant, code-review]

# IDE 配置（多编辑器支持）
ide_config:
  # Claude Code
  claude:
    globs: ["**/*.{js,ts}"]
    auto_activate: true
    file_context: true

  # Codex
  codex:
    globs: ["**/*"]
    auto_activate: true
    tools: [read, edit, bash]

  # Cursor
  cursor:
    globs: ["**/*.{js,ts}"]
    auto_activate: false

  # OpenClaw
  openclaw:
    globs: ["**/*"]
    auto_activate: true
    requires:
      env: [API_KEY]
      bins: [curl]
    primaryEnv: API_KEY
    install:
      - kind: brew
        formula: jq
        bins: [jq]
---

# 技能内容

## 角色
你是专业的 AI 编程助手...

## 核心能力
...

## 工作流程
...

## 示例
...
```

---

## 7. 参考资源

### Claude Code
- 文档：https://docs.anthropic.com/en/docs/claude-code/skills
- 格式：全局 `~/.claude/skills/`，项目 `.claude/skills/`

### Codex
- 文档：https://github.com/openai/codex
- 格式：`.codex/agents/*.mdc`

### Cursor
- 参考项目：https://github.com/jabrena/cursor-rules-java
- 格式：`.cursor/rules/*.md`
- 网站：https://cursor.com

### OpenClaw / ClawHub
- 项目地址：https://github.com/openclaw/clawhub
- 注册中心：https://clawhub.ai
- Soul 注册中心：https://onlycrabs.ai

---

## 8. 结论

四种技能系统各有特点：

1. **Claude Code**：简洁的 Markdown 格式，支持全局和项目级技能，适合团队和个人使用
2. **Codex**：类似 Claude，使用 `.mdc` 扩展名，工具权限显式声明
3. **Cursor**：项目本地的规则系统，使用 `@引用` 方式激活，适合大型项目的编码规范
4. **OpenClaw**：最完整的生态系统，包含注册中心、CLI、版本管理和依赖声明，适合构建可分享的技能包

对于 Skill-Home 项目，建议参考 OpenClaw 的设计，提供：
- 完整的元数据声明（依赖、环境变量、二进制要求）
- 注册中心支持（版本管理、搜索）
- CLI 工具（安装、发布、同步）
- 多 IDE 配置（一个技能，多处使用）
