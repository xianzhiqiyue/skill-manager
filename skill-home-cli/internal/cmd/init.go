package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

const skillTemplate = `---
# 技能元数据
name: %s
version: 0.1.0
description: %s 的简短描述

# 可选字段
# namespace: "@username"
# description_zh: 中文描述
# author: Your Name <email@example.com>
# tags: [tag1, tag2]
# license: MIT
# homepage: https://github.com/username/%s

# IDE 配置
# ide_config:
#   cursor:
#     globs: ["**/*.{ts,tsx}"]
#   claude:
#     auto_activate: true
---

# %s

你是 %s 专家，帮助用户完成相关任务。

## 工作流程

1. 分析用户需求
2. 提供专业的建议和解决方案
3. 给出具体的实施步骤

## 注意事项

- 保持专业和友好的态度
- 在不确定时询问更多上下文
- 提供可操作的反馈
`

type initOptions struct {
	outputDir string
	force     bool
}

func newInitCmd() *cobra.Command {
	opts := &initOptions{}

	cmd := &cobra.Command{
		Use:   "init <skill-name>",
		Short: "创建新的技能模板",
		Long:  "根据指定的名称创建一个新的 SKILL.md 模板文件",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(args[0], opts)
		},
	}

	cmd.Flags().StringVarP(&opts.outputDir, "output", "o", ".", "输出目录")
	cmd.Flags().BoolVarP(&opts.force, "force", "f", false, "强制覆盖已存在的文件")

	return cmd
}

func runInit(name string, opts *initOptions) error {
	// 验证技能名称
	if err := validateSkillName(name); err != nil {
		return err
	}

	// 创建输出目录
	skillDir := filepath.Join(opts.outputDir, name)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 检查文件是否已存在
	skillFile := filepath.Join(skillDir, "SKILL.md")
	if _, err := os.Stat(skillFile); err == nil && !opts.force {
		return fmt.Errorf("技能文件已存在: %s (使用 --force 覆盖)", skillFile)
	}

	// 生成描述
	description := strings.ReplaceAll(name, "-", " ")
	description = strings.Title(description)

	// 写入模板
	content := fmt.Sprintf(skillTemplate,
		name,
		description,
		name,
		description,
		description,
	)

	if err := os.WriteFile(skillFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	// 创建示例目录
	os.MkdirAll(filepath.Join(skillDir, "references"), 0755)
	os.MkdirAll(filepath.Join(skillDir, "scripts"), 0755)

	fmt.Println(color.GreenString("✓"), "创建技能模板成功!")
	fmt.Printf("  位置: %s\n", color.CyanString(skillDir))
	fmt.Printf("  文件: %s\n", color.CyanString(skillFile))
	fmt.Println()
	fmt.Println("下一步:")
	fmt.Printf("  1. 编辑 %s 完善技能定义\n", color.YellowString("SKILL.md"))
	fmt.Printf("  2. 运行 %s 验证格式\n", color.YellowString("skill-home validate"))
	fmt.Printf("  3. 运行 %s 测试同步\n", color.YellowString("skill-home sync"))

	return nil
}

func validateSkillName(name string) error {
	if name == "" {
		return fmt.Errorf("技能名称不能为空")
	}

	// 检查是否只包含允许的字符
	for _, c := range name {
		if !isValidSkillNameChar(c) {
			return fmt.Errorf("技能名称只能包含小写字母、数字和连字符: %s", name)
		}
	}

	// 不能以连字符开头或结尾
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		return fmt.Errorf("技能名称不能以连字符开头或结尾: %s", name)
	}

	// 不能包含连续连字符
	if strings.Contains(name, "--") {
		return fmt.Errorf("技能名称不能包含连续连字符: %s", name)
	}

	return nil
}

func isValidSkillNameChar(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-'
}

// SkillManifest 表示 SKILL.md 的元数据
type SkillManifest struct {
	Name            string                 `yaml:"name"`
	Version         string                 `yaml:"version"`
	Description     string                 `yaml:"description"`
	Namespace       string                 `yaml:"namespace,omitempty"`
	DescriptionZh   string                 `yaml:"description_zh,omitempty"`
	Author          string                 `yaml:"author,omitempty"`
	Tags            []string               `yaml:"tags,omitempty"`
	License         string                 `yaml:"license,omitempty"`
	Homepage        string                 `yaml:"homepage,omitempty"`
	Repository      string                 `yaml:"repository,omitempty"`
	Requires        []string               `yaml:"requires,omitempty"`
	IDEConfig       map[string]interface{} `yaml:"ide_config,omitempty"`
	Permissions     []string               `yaml:"permissions,omitempty"`
	Engines         map[string]string      `yaml:"engines,omitempty"`
}

// GetCreatedAt returns the creation time for the skill
func GetCreatedAt() string {
	return time.Now().Format("2006-01-02")
}
