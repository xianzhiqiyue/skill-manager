# Skill Creator 示例

本目录包含使用 Skill-Creator 创建的示例技能，展示如何编写支持多平台的统一技能格式。

## 示例技能

### 1. code-reviewer (代码审查专家)

**描述**: 智能代码审查专家 - 自动化代码质量、安全性和最佳实践检查

**特性**:
- 代码质量检查（可读性、可维护性、正确性）
- 安全性审查（OWASP Top 10、注入攻击、密钥管理）
- 性能优化建议
- 最佳实践检查

**平台配置**:
```yaml
ide_config:
  claude:
    globs: ["**/*.{ts,tsx,js,jsx,py,go,rs,java,rb,php}"]
    auto_activate: false
  codex:
    globs: ["**/*.{ts,tsx,js,jsx,py,go,rs}"]
    tools: [read, edit, bash, glob, grep]
  cursor:
    globs: ["**/*"]
    always_apply: false
```

**使用方法**:
```bash
# 验证
skill-home validate ./code-reviewer-skill

# 预览导出
skill-home preview ./code-reviewer-skill -p all

# 导出并安装
skill-home export ./code-reviewer-skill -p claude --install
skill-home export ./code-reviewer-skill -p codex --install
skill-home export ./code-reviewer-skill -p cursor --install
```

---

### 2. security-auditor (安全审计专家)

**描述**: 安全审计专家 - 自动化漏洞扫描、合规检查和安全加固建议

**特性**:
- OWASP Top 10 完整检查清单
- 代码安全审计（输入验证、输出编码、会话管理）
- 基础设施安全（容器、云配置）
- 合规检查（GDPR、PCI DSS、SOC 2）
- CVSS 风险评级

**平台配置**:
```yaml
ide_config:
  claude:
    globs: ["**/*"]
    auto_activate: false
  codex:
    globs: ["**/*"]
    tools: [read, edit, bash, glob, grep]
  cursor:
    globs: ["**/*"]
    always_apply: false
```

**使用方法**:
```bash
# 验证
skill-home validate ./security-auditor-skill

# 导出到所有平台
skill-home export ./security-auditor-skill -p all

# 导出并安装到 Claude
skill-home export ./security-auditor-skill -p claude --install
```

---

## 技能结构

每个技能目录包含:

```
skill-name/
├── SKILL.md          # 主要技能定义文件
└── (可选)
    ├── references/   # 参考资料
    └── scripts/      # 辅助脚本
```

### SKILL.md 格式

```markdown
---
name: skill-name
version: 1.0.0
description: 技能描述
namespace: "@skill-home"
author: Author Name
license: MIT
tags: [tag1, tag2]
homepage: https://github.com/...
ide_config:
  claude:
    globs: ["**/*"]
    auto_activate: false
    file_context: true
  codex:
    globs: ["**/*"]
    auto_activate: false
    tools: [read, edit, bash]
  cursor:
    globs: ["**/*"]
    always_apply: false
---

# 技能标题

技能内容（Markdown 格式）...
```

---

## 导出格式对照

| 统一格式 | Claude 导出 | Codex 导出 | Cursor 导出 |
|----------|-------------|------------|-------------|
| `name` | `name` | `name` | `title` |
| `description` | `description` | `description` | `description` |
| `ide_config.claude.globs` | `globs` | - | - |
| `ide_config.codex.globs` | - | `glob` | - |
| `ide_config.cursor.globs` | - | - | `globs` |
| `ide_config.codex.tools` | - | `tools` | - |
| 文件扩展名 | `.md` | `.mdc` | `.mdc` |
| 输出路径 | `~/.claude/skills/{name}/` | `.codex/agents/{name}/` | `.cursor/rules/{name}.mdc` |

---

## 参考资源

这些示例技能参考了以下开源资源:

- [everything-claude-code](https://github.com/affaan-m/everything-claude-code) - Claude Code 配置集合
- [awesome-cursorrules](https://github.com/PatrickJS/awesome-cursorrules) - Cursor 规则集合
- [OWASP Top 10](https://owasp.org/www-project-top-ten/) - Web 安全标准

---

## 创建新技能

使用 Skill-Creator CLI 创建新技能:

```bash
# 交互式创建
skill-home create my-skill

# 快速创建
skill-home create my-skill --platforms claude,codex -q

# 使用模板
skill-home create my-skill -t security-auditor
```
