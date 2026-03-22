# Skill-Creator 设计文档

## 1. 概述

### 1.1 目标
构建一个通用 Skill-Creator，实现 AI 技能的"一次编写，多平台使用"。支持将统一格式的技能导出到 Claude Code、Codex、Cursor 等多个 IDE 平台。

### 1.2 核心原则
- **单一数据源**: 使用统一的 SKILL.md 作为技能源文件
- **平台无关**: 技能内容不依赖特定 IDE 的语法
- **按需导出**: 根据目标平台自动转换格式
- **可扩展**: 易于添加对新 IDE 平台的支持

---

## 2. 架构设计

### 2.1 系统架构

```
┌─────────────────────────────────────────────────────────────┐
│                    Skill-Creator CLI                         │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐         │
│  │  create │  │ export  │  │ preview │  │ validate│         │
│  └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘         │
└───────┼────────────┼────────────┼────────────┼──────────────┘
        │            │            │            │
        └────────────┴────────────┴────────────┘
                           │
              ┌────────────▼────────────┐
              │   Unified Skill Format   │
              │     (SKILL.md)           │
              └────────────┬────────────┘
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
   ┌────▼────┐        ┌────▼────┐        ┌────▼────┐
   │  Claude │        │  Codex  │        │  Cursor │
   │ Adapter │        │ Adapter │        │ Adapter │
   └────┬────┘        └────┬────┘        └────┬────┘
        │                  │                  │
   ~/.claude/          .codex/           .cursor/
    skills/            agents/            rules/
```

### 2.2 核心组件

| 组件 | 职责 | 关键文件 |
|------|------|----------|
| **统一格式解析器** | 解析 SKILL.md 的 frontmatter 和 body | `internal/skill/parser.go` |
| **平台适配器** | 将统一格式转换为各平台特定格式 | `internal/ide/*.go` |
| **导出引擎** | 协调导出流程，管理文件输出 | `internal/export/engine.go` |
| **预览器** | 生成导出预览，不实际写入文件 | `internal/export/preview.go` |
| **模板系统** | 提供预定义技能模板 | `internal/cmd/create.go` |

---

## 3. 统一技能格式

### 3.1 SKILL.md 结构

```markdown
---
# 基础元数据（所有平台通用）
name: skill-name
version: 1.0.0
description: 技能描述
namespace: "@username"
author: Author Name <email@example.com>
license: MIT
homepage: https://github.com/...
tags: [ai-assistant, code-review]

# 多平台配置（按需填写）
ide_config:
  # Claude Code 配置
  claude:
    globs: ["**/*.{js,ts}"]
    auto_activate: true
    file_context: true

  # Codex 配置
  codex:
    globs: ["**/*"]
    auto_activate: true
    tools: [read, edit, bash]

  # Cursor 配置
  cursor:
    globs: ["**/*.{js,ts}"]
    always_apply: false
---

# 技能内容（Markdown 格式）

## 角色定义
你是专业的 AI 编程助手...

## 核心能力
...

## 工作流程
...
```

### 3.2 数据模型

```go
// Manifest 技能元数据
type Manifest struct {
    Name          string                 `yaml:"name"`
    Version       string                 `yaml:"version"`
    Description   string                 `yaml:"description"`
    Namespace     string                 `yaml:"namespace,omitempty"`
    Author        string                 `yaml:"author,omitempty"`
    Tags          []string               `yaml:"tags,omitempty"`
    License       string                 `yaml:"license,omitempty"`
    Homepage      string                 `yaml:"homepage,omitempty"`
    IDEConfig     IDEConfig              `yaml:"ide_config,omitempty"`
}

// IDEConfig 多平台 IDE 配置
type IDEConfig struct {
    Claude *ClaudeConfig `yaml:"claude,omitempty"`
    Codex  *CodexConfig  `yaml:"codex,omitempty"`
    Cursor *CursorConfig `yaml:"cursor,omitempty"`
}

// ClaudeConfig Claude Code 特定配置
type ClaudeConfig struct {
    Globs        []string `yaml:"globs,omitempty"`
    AutoActivate bool     `yaml:"auto_activate,omitempty"`
    FileContext  bool     `yaml:"file_context,omitempty"`
}

// CodexConfig Codex 特定配置
type CodexConfig struct {
    Globs        []string `yaml:"globs,omitempty"`
    AutoActivate bool     `yaml:"auto_activate,omitempty"`
    Tools        []string `yaml:"tools,omitempty"`
}

// CursorConfig Cursor 特定配置
type CursorConfig struct {
    Globs       []string `yaml:"globs,omitempty"`
    AlwaysApply bool     `yaml:"always_apply,omitempty"`
}
```

---

## 4. 平台适配器设计

### 4.1 适配器接口

```go
// PlatformAdapter 平台适配器接口
type PlatformAdapter interface {
    // GetType 返回平台类型
    GetType() string

    // GetTargetPath 返回技能的目标路径
    GetTargetPath(skillName string) string

    // Convert 将统一技能格式转换为平台特定格式
    Convert(skill *skill.Skill) (*ConversionResult, error)

    // Install 安装技能到目标平台
    Install(skill *skill.Skill, targetPath string) error

    // Uninstall 从目标平台卸载技能
    Uninstall(skillName string, targetPath string) error
}

// ConversionResult 转换结果
type ConversionResult struct {
    Files map[string][]byte  // 文件名 -> 内容
    Manifest map[string]interface{}  // 转换后的元数据
}
```

### 4.2 各平台转换规则

#### Claude Code
| 统一格式 | Claude 格式 | 说明 |
|----------|-------------|------|
| `name` | `name` | 直接使用 |
| `ide_config.claude.globs` | `globs` | 数组转为 YAML |
| `ide_config.claude.auto_activate` | `auto_activate` | 布尔值 |
| `ide_config.claude.file_context` | `file_context` | 布尔值 |
| `SKILL.md` | `SKILL.md` | 保持原样 |

#### Codex
| 统一格式 | Codex 格式 | 说明 |
|----------|------------|------|
| `name` | `name` | 直接使用 |
| `ide_config.codex.globs` | `glob` | 单字符串（逗号分隔） |
| `ide_config.codex.tools` | `tools` | 数组 |
| `SKILL.md` | `{name}.mdc` | 扩展名改为 .mdc |

#### Cursor
| 统一格式 | Cursor 格式 | 说明 |
|----------|-------------|------|
| `name` | `title` | 字段名变更 |
| `description` | `description` | 直接使用 |
| `ide_config.cursor.globs` | `globs` | 单字符串（逗号分隔） |
| `SKILL.md` | `{name}.mdc` | 单文件格式 |

---

## 5. 命令设计

### 5.1 命令列表

```bash
# 创建新技能
skill-home create [skill-name] [flags]
  -t, --template string    使用模板 (basic|code-reviewer|api-designer|...)
  -o, --output string      输出目录 (默认 ".")
  -q, --quick              快速模式，使用默认值
  --platforms string       目标平台，逗号分隔 (claude,codex,cursor)

# 导出技能到指定平台
skill-home export <skill-path> [flags]
  -p, --platform string    目标平台 (claude|codex|cursor|all)
  -o, --output string      输出路径（默认使用平台默认路径）
  --dry-run                只显示将要执行的操作
  --install                导出并安装到平台

# 预览导出效果
skill-home preview <skill-path> [flags]
  -p, --platform string    预览指定平台输出 (claude|codex|cursor)
  -o, --output string      将预览输出到文件

# 验证技能格式
skill-home validate <skill-path> [flags]
  --strict                 严格模式，检查所有平台配置
```

### 5.2 使用示例

```bash
# 创建一个支持多平台的技能
skill-home create my-skill --platforms claude,codex

# 导出技能到 Claude Code
skill-home export ./my-skill -p claude --install

# 导出技能到所有平台
skill-home export ./my-skill -p all

# 预览 Cursor 导出效果
skill-home preview ./my-skill -p cursor

# 验证技能格式
skill-home validate ./my-skill --strict
```

---

## 6. 实现计划

### Phase 1: 基础架构
- [x] 定义统一技能格式（扩展 Manifest）
- [ ] 实现 IDEConfig 数据结构
- [ ] 增强现有适配器以支持转换功能

### Phase 2: 导出功能
- [ ] 实现 export 命令
- [ ] 实现各平台转换逻辑
- [ ] 支持 --install 标志直接安装

### Phase 3: 预览功能
- [ ] 实现 preview 命令
- [ ] 生成格式化的预览输出
- [ ] 支持差异对比

### Phase 4: 增强功能
- [ ] 增强 create 命令支持多平台配置
- [ ] 添加更多技能模板
- [ ] 支持条件内容块（平台特定内容）

---

## 7. 扩展性设计

### 7.1 添加新平台支持

要添加对新 IDE 平台的支持，只需：

1. 在 `IDEConfig` 中添加新平台配置结构
2. 实现 `PlatformAdapter` 接口
3. 在 `NewAdapter` 工厂函数中注册

```go
// 示例：添加 Windsurf 平台支持

// 1. 定义配置
type WindsurfConfig struct {
    Globs []string `yaml:"globs,omitempty"`
}

// 2. 实现适配器
type WindsurfAdapter struct{}

func (a *WindsurfAdapter) GetType() string { return "windsurf" }
func (a *WindsurfAdapter) Convert(skill *skill.Skill) (*ConversionResult, error) {
    // 转换逻辑
}

// 3. 注册适配器
func NewAdapter(ideType string) (PlatformAdapter, error) {
    switch ideType {
    // ... 其他平台
    case "windsurf":
        return &WindsurfAdapter{}, nil
    }
}
```

---

## 8. 参考

- [Claude Code Skills 文档](https://docs.anthropic.com/en/docs/claude-code/skills)
- [Codex Agents 文档](https://github.com/openai/codex)
- [Cursor Rules 文档](https://cursor.com)
- [OpenClaw/ClawHub 项目](https://github.com/openclaw/clawhub)
