package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// SkillTemplate 技能模板定义
type SkillTemplate struct {
	Name        string
	Description string
	Category    string
	Content     string
}

// 预定义的技能模板
var skillTemplates = []SkillTemplate{
	{
		Name:        "basic",
		Description: "基础模板 - 通用技能",
		Category:    "general",
		Content: `---
name: {{.Name}}
version: 0.1.0
description: {{.Description}}
{{if .Namespace}}namespace: "{{.Namespace}}"{{end}}
{{if .Author}}author: {{.Author}}{{end}}
{{if .Tags}}tags:{{range .Tags}}
  - {{.}}{{end}}{{end}}
license: {{.License}}
{{if .Homepage}}homepage: {{.Homepage}}{{end}}
---

# {{.Name}}

你是 {{.Description}} 专家，帮助用户完成相关任务。

## 职责

1. 分析用户需求
2. 提供专业的建议和解决方案
3. 给出具体的实施步骤

## 输出格式

- 清晰的结构化回复
- 包含具体示例
- 注明注意事项
`,
	},
	{
		Name:        "code-reviewer",
		Description: "代码审查专家",
		Category:    "development",
		Content: `---
name: {{.Name}}
version: 0.1.0
description: {{.Description}}
{{if .Namespace}}namespace: "{{.Namespace}}"{{end}}
{{if .Author}}author: {{.Author}}{{end}}
tags:
  - code-review
  - quality
  - best-practices
license: {{.License}}
ide_config:
  cursor:
    globs: ["**/*.{ts,tsx,js,jsx,py,go,rs}"]
    always_apply: false
  claude:
    auto_activate: true
---

# {{.Description}}

你是资深代码审查专家，专注于发现潜在问题和改进建议。

## 审查维度

### 1. 代码质量
- 代码可读性和可维护性
- 命名规范
- 函数/类设计合理性
- 注释完整性

### 2. 安全性
- SQL 注入风险
- XSS 漏洞
- 敏感信息泄露
- 不安全的依赖

### 3. 性能
- 算法复杂度
- 资源泄漏
- 不必要的循环/递归
- 缓存使用

### 4. 最佳实践
- 设计模式应用
- 错误处理
- 日志记录
- 测试覆盖

## 输出格式

对于每个问题，请提供：

**[严重程度] 问题类型: 简要描述**

- 位置: 文件路径和行号
- 问题: 详细说明
- 建议: 如何修复
- 示例:
  ```语言
  // 修复后的代码示例
  ```

严重程度分级：
- 🔴 Critical: 必须立即修复（安全漏洞、严重错误）
- 🟡 Warning: 建议修复（性能问题、维护性）
- 🔵 Info: 可选改进（风格、优化建议）
`,
	},
	{
		Name:        "api-designer",
		Description: "API 设计专家",
		Category:    "development",
		Content: `---
name: {{.Name}}
version: 0.1.0
description: {{.Description}}
{{if .Namespace}}namespace: "{{.Namespace}}"{{end}}
{{if .Author}}author: {{.Author}}{{end}}
tags:
  - api
  - design
  - rest
  - graphql
license: {{.License}}
---

# {{.Description}}

你是 API 设计专家，帮助设计和评审 RESTful、GraphQL 和 gRPC API。

## 核心能力

### API 设计原则
- RESTful 设计规范
- GraphQL Schema 设计
- 版本控制策略
- 端点命名规范

### 安全性
- 认证与授权 (OAuth 2.0, JWT, API Key)
- 速率限制
- 输入验证
- CORS 配置

### 文档规范
- OpenAPI/Swagger
- API 文档最佳实践
- 示例和用例

## 设计审查清单

- [ ] URL 结构是否清晰
- [ ] HTTP 方法使用是否正确
- [ ] 状态码是否恰当
- [ ] 错误响应格式是否统一
- [ ] 是否支持分页/过滤/排序
- [ ] 是否有适当的缓存策略
- [ ] 是否考虑过向后兼容性
`,
	},
	{
		Name:        "refactor-expert",
		Description: "代码重构专家",
		Category:    "development",
		Content: `---
name: {{.Name}}
version: 0.1.0
description: {{.Description}}
{{if .Namespace}}namespace: "{{.Namespace}}"{{end}}
{{if .Author}}author: {{.Author}}{{end}}
tags:
  - refactoring
  - clean-code
  - architecture
license: {{.License}}
ide_config:
  claude:
    auto_activate: false
---

# {{.Description}}

你是代码重构专家，专注于改善代码结构而不改变外部行为。

## 重构技术

### 代码级别
- 提取函数/变量
- 内联函数/变量
- 重命名
- 移动函数

### 设计级别
- 消除重复 (DRY)
- 简化条件表达式
- 引入设计模式
- 解耦模块

### 架构级别
- 分层架构
- 依赖注入
- 接口抽象
- 事件驱动

## 重构流程

1. **识别坏味道**
   - 长函数
   - 大类
   - 重复代码
   - 过长的参数列表

2. **确保安全**
   - 检查现有测试
   - 识别依赖关系
   - 评估风险

3. **小步前进**
   - 每次只做一件事
   - 频繁测试
   - 提交代码

4. **验证结果**
   - 运行测试
   - 代码审查
   - 性能测试
`,
	},
	{
		Name:        "test-expert",
		Description: "测试专家",
		Category:    "development",
		Content: `---
name: {{.Name}}
version: 0.1.0
description: {{.Description}}
{{if .Namespace}}namespace: "{{.Namespace}}"{{end}}
{{if .Author}}author: {{.Author}}{{end}}
tags:
  - testing
  - tdd
  - quality
license: {{.License}}
---

# {{.Description}}

你是软件测试专家，帮助设计和实现高质量的测试策略。

## 测试类型

### 单元测试
- 测试单一功能
- Mock 外部依赖
- 快速执行
- 高覆盖率

### 集成测试
- 测试组件交互
- 数据库/服务集成
- 端到端流程

### E2E 测试
- 用户场景模拟
- UI 自动化
- 关键路径验证

## 测试原则

- **FIRST**: Fast, Independent, Repeatable, Self-validating, Timely
- **AAA**: Arrange, Act, Assert
- **一个概念一个测试**
- **描述性行为命名**

## 测试模板

` + "```" + `语言
// 测试结构
describe('功能模块', () => {
  beforeEach(() => {
    // 初始化
  });

  it('应该在做某事时产生某结果', () => {
    // Arrange
    const input = ...;

    // Act
    const result = functionUnderTest(input);

    // Assert
    expect(result).toBe(expected);
  });
});
` + "```" + `
`,
	},
	{
		Name:        "doc-writer",
		Description: "文档编写专家",
		Category:    "documentation",
		Content: `---
name: {{.Name}}
version: 0.1.0
description: {{.Description}}
{{if .Namespace}}namespace: "{{.Namespace}}"{{end}}
{{if .Author}}author: {{.Author}}{{end}}
tags:
  - documentation
  - writing
  - technical-writing
license: {{.License}}
---

# {{.Description}}

你是技术文档专家，帮助编写清晰、准确、易读的技术文档。

## 文档类型

### 用户文档
- 快速开始指南
- 使用教程
- API 参考
- 常见问题

### 开发文档
- 架构设计
- 贡献指南
- 代码规范
- 部署文档

## 写作原则

1. **清晰优先**
   - 使用简单语言
   - 短句优先
   - 主动语态

2. **结构化**
   - 标题层级清晰
   - 使用列表和表格
   - 重要内容前置

3. **示例丰富**
   - 代码示例
   - 截图说明
   - 实际用例

## 文档模板

### README 结构
1. 项目简介（一句话）
2. 功能特性
3. 安装说明
4. 快速开始
5. 详细文档链接
6. 贡献指南
7. 许可证
`,
	},
	{
		Name:        "security-auditor",
		Description: "安全审计专家",
		Category:    "security",
		Content: `---
name: {{.Name}}
version: 0.1.0
description: {{.Description}}
{{if .Namespace}}namespace: "{{.Namespace}}"{{end}}
{{if .Author}}author: {{.Author}}{{end}}
tags:
  - security
  - audit
  - vulnerability
license: {{.License}}
permissions:
  - file:read
  - network:fetch
---

# {{.Description}}

你是安全审计专家，专注于发现代码中的安全漏洞和风险。

## 审计范围

### Web 安全
- OWASP Top 10
- 注入攻击 (SQL, NoSQL, Command)
- XSS (存储型、反射型、DOM)
- CSRF 防护
- 认证与会话管理

### 应用安全
- 依赖漏洞扫描
- 敏感信息泄露
- 不安全的反序列化
- 访问控制缺陷

### 基础设施
- 容器安全
- 密钥管理
- 网络安全配置
- 日志审计

## 审计报告模板

**[风险等级] 漏洞类型: 标题**

- **CVSS 评分**: X.X
- **位置**: 文件路径
- **描述**: 漏洞详情
- **影响**: 可能的后果
- **修复建议**: 具体步骤
- **参考**: CWE/SANS 链接

风险等级：Critical | High | Medium | Low | Info
`,
	},
	{
		Name:        "performance-optimizer",
		Description: "性能优化专家",
		Category:    "optimization",
		Content: `---
name: {{.Name}}
version: 0.1.0
description: {{.Description}}
{{if .Namespace}}namespace: "{{.Namespace}}"{{end}}
{{if .Author}}author: {{.Author}}{{end}}
tags:
  - performance
  - optimization
  - scalability
license: {{.License}}
---

# {{.Description}}

你是性能优化专家，帮助识别和解决性能瓶颈。

## 优化领域

### 前端性能
- 资源加载优化
- 渲染性能
- JavaScript 执行
- 缓存策略

### 后端性能
- 数据库查询优化
- API 响应时间
- 并发处理
- 内存管理

### 基础设施
- CDN 配置
- 负载均衡
- 自动扩缩容
- 监控告警

## 优化流程

1. **度量 (Measure)**
   - 建立性能基线
   - 识别瓶颈
   - 用户指标收集

2. **分析 (Analyze)**
   - 火焰图分析
   - 查询执行计划
   - 资源使用监控

3. **优化 (Optimize)**
   - 针对性改进
   - A/B 测试验证

4. **监控 (Monitor)**
   - 持续跟踪
   - 回归检测

## 关键指标

- FCP (First Contentful Paint)
- LCP (Largest Contentful Paint)
- TTFB (Time to First Byte)
- 吞吐量
- 错误率
`,
	},
}

// createOptions 创建选项
type createOptions struct {
	outputDir string
	template  string
	quick     bool
}

func newCreateCmd() *cobra.Command {
	opts := &createOptions{}

	cmd := &cobra.Command{
		Use:   "create [skill-name]",
		Short: "交互式创建技能（增强版）",
		Long: `通过交互式向导创建新的技能。

支持多种预定义模板，也可以从基础模板开始自定义。

示例:
  skill-home create                    # 启动交互式向导
  skill-home create my-skill           # 使用默认模板创建
  skill-home create my-skill -t code-reviewer  # 指定模板`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			skillName := ""
			if len(args) > 0 {
				skillName = args[0]
			}
			return runCreate(skillName, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.outputDir, "output", "o", ".", "输出目录")
	cmd.Flags().StringVarP(&opts.template, "template", "t", "", "使用指定模板 (basic|code-reviewer|api-designer|refactor-expert|test-expert|doc-writer|security-auditor|performance-optimizer)")
	cmd.Flags().BoolVarP(&opts.quick, "quick", "q", false, "快速模式，使用默认值")

	return cmd
}

// SkillAnswers 用户回答
type SkillAnswers struct {
	Name        string
	Description string
	Namespace   string
	Author      string
	Tags        []string
	License     string
	Homepage    string
	Template    string
}

func runCreate(skillName string, opts *createOptions) error {
	answers := &SkillAnswers{}

	// 快速模式
	if opts.quick {
		if skillName == "" {
			return fmt.Errorf("快速模式需要提供技能名称")
		}
		answers.Name = skillName
		answers.Description = strings.Title(strings.ReplaceAll(skillName, "-", " "))
		answers.License = "MIT"
		if opts.template != "" {
			answers.Template = opts.template
		} else {
			answers.Template = "basic"
		}
	} else {
		// 交互式向导
		if err := runInteractiveWizard(skillName, opts.template, answers); err != nil {
			return err
		}
	}

	// 验证技能名称
	if err := validateSkillName(answers.Name); err != nil {
		return err
	}

	// 创建技能目录
	skillDir := filepath.Join(opts.outputDir, answers.Name)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 检查文件是否已存在
	skillFile := filepath.Join(skillDir, "SKILL.md")
	if _, err := os.Stat(skillFile); err == nil {
		overwrite := false
		prompt := &survey.Confirm{
			Message: fmt.Sprintf("技能文件已存在: %s，是否覆盖?", skillFile),
			Default: false,
		}
		if err := survey.AskOne(prompt, &overwrite); err != nil {
			return err
		}
		if !overwrite {
			fmt.Println(color.YellowString("已取消"))
			return nil
		}
	}

	// 获取模板内容
	template := getTemplate(answers.Template)
	if template == nil {
		return fmt.Errorf("未知模板: %s", answers.Template)
	}

	// 渲染模板
	content, err := renderTemplate(template.Content, answers)
	if err != nil {
		return fmt.Errorf("渲染模板失败: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(skillFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	// 创建子目录
	os.MkdirAll(filepath.Join(skillDir, "references"), 0755)
	os.MkdirAll(filepath.Join(skillDir, "scripts"), 0755)

	// 输出成功信息
	fmt.Println()
	fmt.Println(color.GreenString("✓"), "技能创建成功!")
	fmt.Printf("  名称: %s\n", color.CyanString(answers.Name))
	fmt.Printf("  模板: %s\n", color.CyanString(template.Name))
	fmt.Printf("  位置: %s\n", color.CyanString(skillDir))
	fmt.Println()
	fmt.Println("下一步:")
	fmt.Printf("  1. 编辑 %s 完善技能内容\n", color.YellowString("SKILL.md"))
	if len(answers.Tags) > 0 {
		fmt.Printf("  2. 运行 %s 验证格式\n", color.YellowString("skill-home validate"))
		fmt.Printf("  3. 运行 %s 测试同步\n", color.YellowString("skill-home sync"))
	} else {
		fmt.Printf("  2. 添加标签以便更容易被发现\n")
		fmt.Printf("  3. 运行 %s 验证格式\n", color.YellowString("skill-home validate"))
	}

	return nil
}

func runInteractiveWizard(skillName string, templateName string, answers *SkillAnswers) error {
	fmt.Println(color.CyanString("\n🚀 Skill Creator 交互式向导"))
	fmt.Println(strings.Repeat("─", 50))

	// 1. 技能名称
	if skillName == "" {
		prompt := &survey.Input{
			Message: "技能名称 (kebab-case):",
			Help:    "使用小写字母、数字和连字符，如: code-reviewer",
		}
		if err := survey.AskOne(prompt, &answers.Name, survey.WithValidator(survey.Required)); err != nil {
			return err
		}
	} else {
		answers.Name = skillName
		fmt.Printf("技能名称: %s\n", color.GreenString(answers.Name))
	}

	// 2. 选择模板
	if templateName == "" {
		templateOptions := getTemplateOptions()
		prompt := &survey.Select{
			Message: "选择技能模板:",
			Options: templateOptions,
			Description: func(value string, index int) string {
				// 从选项中提取模板名并查找描述
				name := extractTemplateName(value)
				t := getTemplate(name)
				if t != nil {
					return fmt.Sprintf("[%s] %s", t.Category, t.Description)
				}
				return ""
			},
		}
		var selected string
		if err := survey.AskOne(prompt, &selected); err != nil {
			return err
		}
		answers.Template = extractTemplateName(selected)
	} else {
		answers.Template = templateName
	}

	// 3. 描述
	prompt := &survey.Input{
		Message: "简短描述:",
		Default: strings.Title(strings.ReplaceAll(answers.Name, "-", " ")),
		Help:    "一句话描述这个技能的用途",
	}
	if err := survey.AskOne(prompt, &answers.Description, survey.WithValidator(survey.Required)); err != nil {
		return err
	}

	// 4. 命名空间（可选）
	prompt2 := &survey.Input{
		Message: "命名空间 (可选, 如 @username):",
		Help:    "用于组织和区分不同来源的技能",
	}
	if err := survey.AskOne(prompt2, &answers.Namespace); err != nil {
		return err
	}

	// 5. 作者（可选）
	prompt3 := &survey.Input{
		Message: "作者 (可选, 格式: Name <email>):",
	}
	if err := survey.AskOne(prompt3, &answers.Author); err != nil {
		return err
	}

	// 6. 标签（可选）
	prompt4 := &survey.Input{
		Message: "标签 (可选, 逗号分隔):",
		Help:    "例如: go,backend,api",
	}
	var tagsStr string
	if err := survey.AskOne(prompt4, &tagsStr); err != nil {
		return err
	}
	if tagsStr != "" {
		answers.Tags = parseTags(tagsStr)
	}

	// 7. 许可证
	licenses := []string{"MIT", "Apache-2.0", "BSD-3-Clause", "GPL-3.0", "Proprietary", "Other"}
	prompt5 := &survey.Select{
		Message: "选择许可证:",
		Options: licenses,
		Default: "MIT",
	}
	if err := survey.AskOne(prompt5, &answers.License); err != nil {
		return err
	}

	// 8. 主页（可选）
	prompt6 := &survey.Input{
		Message: "主页 URL (可选):",
		Help:    "项目主页或文档链接",
	}
	if err := survey.AskOne(prompt6, &answers.Homepage); err != nil {
		return err
	}

	fmt.Println(strings.Repeat("─", 50))
	return nil
}

func getTemplateOptions() []string {
	options := make([]string, len(skillTemplates))
	for i, t := range skillTemplates {
		options[i] = fmt.Sprintf("%s (%s)", t.Name, t.Description)
	}
	return options
}

func extractTemplateName(option string) string {
	// 从 "name (description)" 中提取 name
	idx := strings.Index(option, " (")
	if idx > 0 {
		return option[:idx]
	}
	return option
}

func getTemplate(name string) *SkillTemplate {
	for i := range skillTemplates {
		if skillTemplates[i].Name == name {
			return &skillTemplates[i]
		}
	}
	return nil
}

func parseTags(tagsStr string) []string {
	parts := strings.Split(tagsStr, ",")
	tags := make([]string, 0, len(parts))
	for _, p := range parts {
		tag := strings.TrimSpace(p)
		if tag != "" {
			tags = append(tags, tag)
		}
	}
	return tags
}

// renderTemplate 使用简单的字符串替换渲染模板
func renderTemplate(template string, data *SkillAnswers) (string, error) {
	result := template

	// 基本替换
	result = strings.ReplaceAll(result, "{{.Name}}", data.Name)
	result = strings.ReplaceAll(result, "{{.Description}}", data.Description)
	result = strings.ReplaceAll(result, "{{.License}}", data.License)

	// 条件替换
	if data.Namespace != "" {
		result = strings.ReplaceAll(result, "{{if .Namespace}}namespace: \"{{.Namespace}}\"{{end}}", fmt.Sprintf("namespace: \"%s\"", data.Namespace))
	} else {
		result = strings.ReplaceAll(result, "{{if .Namespace}}namespace: \"{{.Namespace}}\"{{end}}\n", "")
	}

	if data.Author != "" {
		result = strings.ReplaceAll(result, "{{if .Author}}author: {{.Author}}{{end}}", fmt.Sprintf("author: %s", data.Author))
	} else {
		result = strings.ReplaceAll(result, "{{if .Author}}author: {{.Author}}{{end}}\n", "")
	}

	if data.Homepage != "" {
		result = strings.ReplaceAll(result, "{{if .Homepage}}homepage: {{.Homepage}}{{end}}", fmt.Sprintf("homepage: %s", data.Homepage))
	} else {
		result = strings.ReplaceAll(result, "{{if .Homepage}}homepage: {{.Homepage}}{{end}}\n", "")
	}

	// 标签替换
	if len(data.Tags) > 0 {
		tagsYaml := "tags:"
		for _, tag := range data.Tags {
			tagsYaml += fmt.Sprintf("\n  - %s", tag)
		}
		result = strings.ReplaceAll(result, "{{if .Tags}}tags:{{range .Tags}}\n  - {{.}}{{end}}{{end}}", tagsYaml)
	} else {
		result = strings.ReplaceAll(result, "{{if .Tags}}tags:{{range .Tags}}\n  - {{.}}{{end}}{{end}}\n", "")
	}

	return result, nil
}

// 为了兼容性，保留 init 命令
func newLegacyInitCmd() *cobra.Command {
	return newInitCmd()
}

// readLine 读取一行输入
func readLine(reader *bufio.Reader, prompt string) (string, error) {
	fmt.Print(prompt)
	return reader.ReadString('\n')
}
