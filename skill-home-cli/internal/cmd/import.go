package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/skill-home/cli/internal/import/github"
	"github.com/skill-home/cli/internal/import/types"
)

// importOptions 导入选项
type importOptions struct {
	outputDir   string
	source      string
	convertOnly bool
	force       bool
}

func newImportCmd() *cobra.Command {
	opts := &importOptions{}

	cmd := &cobra.Command{
		Use:   "import <source-url>",
		Short: "从外部源导入技能",
		Long: `从 GitHub、Claude Code、Codex 等外部源导入技能并转换为通用 SKILL.md 格式。

支持的源:
  GitHub:    skill-home import github.com/user/repo
  GitHub:    skill-home import https://github.com/user/repo
  Claude:    skill-home import claude://~/.claude/skills/skill-name
  Codex:     skill-home import codex://~/.codex/skills/skill-name
  Local:     skill-home import /path/to/local/skill

示例:
  # 从 GitHub 导入
  skill-home import github.com/example/code-review-skill

  # 从 Claude Code 技能目录导入
  skill-home import claude://~/.claude/skills/my-skill

  # 仅转换不导入
  skill-home import ./local-skill --convert-only`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runImport(args[0], opts)
		},
	}

	cmd.Flags().StringVarP(&opts.outputDir, "output", "o", ".", "输出目录")
	cmd.Flags().StringVarP(&opts.source, "source", "s", "", "强制指定源类型 (github|claude|codex|cursor|local)")
	cmd.Flags().BoolVar(&opts.convertOnly, "convert-only", false, "仅转换格式，不保存到注册中心")
	cmd.Flags().BoolVarP(&opts.force, "force", "f", false, "强制覆盖已存在的文件")

	return cmd
}

func runImport(sourceURL string, opts *importOptions) error {
	fmt.Println(color.CyanString("📥 开始导入技能"))
	fmt.Printf("源地址: %s\n\n", color.YellowString(sourceURL))

	// 自动检测源类型
	sourceType := opts.source
	if sourceType == "" {
		sourceType = detectSourceType(sourceURL)
	}

	fmt.Printf("检测到源类型: %s\n", color.GreenString(sourceType))

	// 创建对应的 importer
	importer, err := createImporter(sourceType, sourceURL)
	if err != nil {
		return err
	}

	// 获取技能信息
	fmt.Println("正在获取技能信息...")
	info, err := importer.GetSkillInfo()
	if err != nil {
		return fmt.Errorf("获取技能信息失败: %w", err)
	}

	fmt.Printf("\n技能信息:\n")
	fmt.Printf("  名称: %s\n", color.CyanString(info.Name))
	if info.Description != "" {
		fmt.Printf("  描述: %s\n", info.Description)
	}
	if info.Author != "" {
		fmt.Printf("  作者: %s\n", info.Author)
	}
	if info.Version != "" {
		fmt.Printf("  版本: %s\n", info.Version)
	}

	// 下载技能
	fmt.Println("\n正在下载技能...")
	tempDir, err := os.MkdirTemp("", "skill-import-*")
	if err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tempDir)

	downloadPath := filepath.Join(tempDir, "source")
	if err := importer.Download(downloadPath); err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	fmt.Println(color.GreenString("✓"), "下载完成")

	// 转换为通用格式
	fmt.Println("正在转换为通用 SKILL.md 格式...")
	skill, err := importer.ConvertToSkill(downloadPath)
	if err != nil {
		return fmt.Errorf("转换失败: %w", err)
	}
	fmt.Println(color.GreenString("✓"), "转换完成")

	// 确定输出路径
	outputDir := opts.outputDir
	if outputDir == "" {
		outputDir = filepath.Join(".", skill.Name)
	} else {
		outputDir = filepath.Join(outputDir, skill.Name)
	}

	// 检查是否已存在
	if _, err := os.Stat(outputDir); err == nil && !opts.force {
		return fmt.Errorf("目录已存在: %s (使用 --force 覆盖)", outputDir)
	}

	// 创建输出目录
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}

	// 保存转换后的技能
	fmt.Println("正在保存技能...")
	if err := saveSkill(skill, outputDir); err != nil {
		return fmt.Errorf("保存失败: %w", err)
	}

	// 输出结果
	fmt.Println()
	fmt.Println(color.GreenString("✓"), "技能导入成功!")
	fmt.Printf("  位置: %s\n", color.CyanString(outputDir))
	fmt.Printf("  文件: %s\n", color.CyanString(filepath.Join(outputDir, "SKILL.md")))

	// 显示转换报告
	if len(info.Warnings) > 0 {
		fmt.Println()
		fmt.Println(color.YellowString("⚠ 转换警告:"))
		for _, warning := range info.Warnings {
			fmt.Printf("  - %s\n", warning)
		}
	}

	if len(info.Notes) > 0 {
		fmt.Println()
		fmt.Println(color.BlueString("ℹ 转换说明:"))
		for _, note := range info.Notes {
			fmt.Printf("  - %s\n", note)
		}
	}

	fmt.Println()
	fmt.Println("下一步:")
	fmt.Printf("  1. 检查 %s 内容\n", color.YellowString("SKILL.md"))
	fmt.Printf("  2. 运行 %s 验证格式\n", color.YellowString("skill-home validate"))
	fmt.Printf("  3. 运行 %s 同步到 IDE\n", color.YellowString("skill-home sync"))

	return nil
}

// detectSourceType 自动检测源类型
func detectSourceType(url string) string {
	// 检查 URL 模式
	if strings.HasPrefix(url, "github.com/") ||
		strings.HasPrefix(url, "https://github.com/") ||
		strings.HasPrefix(url, "gh://") {
		return "github"
	}

	if strings.HasPrefix(url, "claude://") {
		return "claude"
	}

	if strings.HasPrefix(url, "codex://") {
		return "codex"
	}

	if strings.HasPrefix(url, "cursor://") {
		return "cursor"
	}

	// 检查本地路径
	if strings.HasPrefix(url, "/") || strings.HasPrefix(url, "~") || strings.HasPrefix(url, ".") {
		// 进一步检测是哪种 IDE 的技能
		expanded := expandPath(url)
		if strings.Contains(expanded, ".claude") {
			return "claude"
		}
		if strings.Contains(expanded, ".codex") || strings.Contains(expanded, ".agents") {
			return "codex"
		}
		if strings.Contains(expanded, ".cursor") {
			return "cursor"
		}
		return "local"
	}

	// 默认为 GitHub
	return "github"
}

// createImporter 创建对应的 importer
func createImporter(sourceType, sourceURL string) (types.SkillImporter, error) {
	switch sourceType {
	case "github":
		return github.NewImporter(sourceURL)
	case "claude":
		return NewClaudeImporter(sourceURL)
	case "codex":
		return NewCodexImporter(sourceURL)
	case "cursor":
		return NewCursorImporter(sourceURL)
	case "local":
		return NewLocalImporter(sourceURL)
	default:
		return nil, fmt.Errorf("不支持的源类型: %s", sourceType)
	}
}

// saveSkill 保存技能到目录
func saveSkill(skill *types.Skill, outputDir string) error {
	// 创建目录结构
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(outputDir, "references"), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(outputDir, "scripts"), 0755); err != nil {
		return err
	}

	// 保存 SKILL.md
	skillPath := filepath.Join(outputDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte(skill.Content), 0644); err != nil {
		return err
	}

	// 保存 references
	for name, content := range skill.References {
		refPath := filepath.Join(outputDir, "references", name)
		if err := os.WriteFile(refPath, []byte(content), 0644); err != nil {
			return err
		}
	}

	// 保存 scripts
	for name, content := range skill.Scripts {
		scriptPath := filepath.Join(outputDir, "scripts", name)
		if err := os.WriteFile(scriptPath, []byte(content), 0755); err != nil {
			return err
		}
	}

	return nil
}

// expandPath 展开路径中的 ~
func expandPath(path string) string {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(home, path[1:])
		}
	}
	return path
}
