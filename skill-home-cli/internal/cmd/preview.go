package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/skill-home/cli/internal/export"
	"github.com/skill-home/cli/internal/skill"
)

// previewOptions 预览选项
type previewOptions struct {
	platform string
	output   string
}

func newPreviewCmd() *cobra.Command {
	opts := &previewOptions{}

	cmd := &cobra.Command{
		Use:   "preview <skill-path>",
		Short: "预览技能导出效果",
		Long: `预览技能导出到指定平台的效果，不实际写入文件。

支持预览多个平台的导出结果，方便在正式导出前检查转换是否正确。

示例:
  # 预览 Claude 导出效果
  skill-home preview ./my-skill -p claude

  # 预览所有平台
  skill-home preview ./my-skill -p all

  # 将预览输出到文件
  skill-home preview ./my-skill -p cursor -o ./preview.md`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			skillPath := args[0]
			return runPreview(skillPath, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.platform, "platform", "p", "", "预览指定平台 (claude|copilot|codex|cursor|all)")
	cmd.Flags().StringVarP(&opts.output, "output", "o", "", "将预览输出到文件")

	cmd.MarkFlagRequired("platform")

	return cmd
}

func runPreview(skillPath string, opts *previewOptions) error {
	// 解析技能
	s, err := skill.Parse(skillPath)
	if err != nil {
		return fmt.Errorf("解析技能失败: %w", err)
	}

	// 确定要预览的平台
	platforms := []string{}
	if opts.platform == "all" {
		platforms = []string{"claude", "copilot", "codex", "cursor"}
	} else {
		platforms = strings.Split(opts.platform, ",")
	}

	engine := export.NewEngine("")

	var output strings.Builder

	// 标题
	title := fmt.Sprintf("技能导出预览: %s\n", s.Manifest.Name)
	output.WriteString(title)
	output.WriteString(strings.Repeat("=", len(title)) + "\n\n")

	output.WriteString(fmt.Sprintf("版本: %s\n", s.Manifest.Version))
	output.WriteString(fmt.Sprintf("描述: %s\n\n", s.Manifest.Description))

	// 预览每个平台
	for _, platform := range platforms {
		platform = strings.TrimSpace(platform)

		result, err := engine.Export(s, platform)
		if err != nil {
			output.WriteString(fmt.Sprintf("❌ %s 平台导出失败: %v\n\n", platform, err))
			continue
		}

		// 平台标题
		platformTitle := fmt.Sprintf("[%s] 平台", strings.ToUpper(platform))
		output.WriteString(platformTitle + "\n")
		output.WriteString(strings.Repeat("-", len(platformTitle)) + "\n\n")

		// 显示文件列表
		output.WriteString(fmt.Sprintf("生成文件数: %d\n", len(result.Files)))
		for fileName := range result.Files {
			output.WriteString(fmt.Sprintf("  📄 %s\n", fileName))
		}
		output.WriteString("\n")

		// 显示主文件内容预览
		mainFile := getMainFile(result.Platform, s.Manifest.Name)
		if content, ok := result.Files[mainFile]; ok {
			output.WriteString(fmt.Sprintf("--- %s 内容预览 ---\n", mainFile))
			preview := truncateString(string(content), 2000)
			output.WriteString(preview)
			if len(content) > 2000 {
				output.WriteString("\n... (内容已截断)")
			}
			output.WriteString("\n\n")
		}

		output.WriteString("\n")
	}

	// 输出结果
	if opts.output != "" {
		if err := os.WriteFile(opts.output, []byte(output.String()), 0644); err != nil {
			return fmt.Errorf("写入预览文件失败: %w", err)
		}
		fmt.Printf("%s 预览已保存到: %s\n", color.GreenString("✓"), color.CyanString(opts.output))
	} else {
		fmt.Println(output.String())
	}

	return nil
}

func getMainFile(platform, skillName string) string {
	switch platform {
	case "claude":
		return "SKILL.md"
	case "copilot":
		return "SKILL.md"
	case "codex", "cursor":
		return skillName + ".mdc"
	default:
		return "SKILL.md"
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
