package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/skill-home/cli/internal/export"
	"github.com/skill-home/cli/internal/skill"
)

// exportOptions 导出选项
type exportOptions struct {
	platform string
	output   string
	dryRun   bool
	install  bool
}

func newExportCmd() *cobra.Command {
	opts := &exportOptions{}

	cmd := &cobra.Command{
		Use:   "export \u003cskill-path\u003e",
		Short: "导出技能到指定 IDE 平台",
		Long: `将统一格式的技能导出到指定的 IDE 平台格式。

支持的平台:
    - claude: Claude Code 格式 (~/.claude/skills/)
  - copilot: GitHub Copilot 格式 (~/.copilot/skills/)
  - codex:  OpenAI Codex 格式 (.codex/agents/)
  - cursor: Cursor 格式 (.cursor/rules/)
  - all:    导出到所有平台

示例:
  # 导出到 Claude Code
  skill-home export ./my-skill -p claude

  # 导出并安装到 Claude Code
  skill-home export ./my-skill -p claude --install

  # 导出到所有平台
  skill-home export ./my-skill -p all

  # 导出到指定目录（预览）
  skill-home export ./my-skill -p claude -o ./output`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			skillPath := args[0]
			return runExport(skillPath, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.platform, "platform", "p", "", "目标平台 (claude|copilot|codex|cursor|all)")
	cmd.Flags().StringVarP(&opts.output, "output", "o", "", "输出路径（默认使用平台默认路径）")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "只显示将要执行的操作，不实际导出")
	cmd.Flags().BoolVar(&opts.install, "install", false, "导出并安装到平台")

	cmd.MarkFlagRequired("platform")

	return cmd
}

func runExport(skillPath string, opts *exportOptions) error {
	// 解析技能
	s, err := skill.Parse(skillPath)
	if err != nil {
		return fmt.Errorf("解析技能失败: %w", err)
	}

	fmt.Printf("\n%s %s\n", color.CyanString("导出技能:"), color.YellowString(s.Manifest.Name))
	fmt.Printf("  版本: %s\n", s.Manifest.Version)
	fmt.Printf("  描述: %s\n", s.Manifest.Description)
	fmt.Println()

	// 确定要导出的平台
	platforms := []string{}
	if opts.platform == "all" {
		platforms = []string{"claude", "copilot", "codex", "cursor"}
	} else {
		platforms = strings.Split(opts.platform, ",")
	}

	engine := export.NewEngine("")

	// 导出到每个平台
	for _, platform := range platforms {
		platform = strings.TrimSpace(platform)

		fmt.Printf("%s 导出到 %s...\n", color.BlueString("→"), color.CyanString(platform))

		// 执行导出
		result, err := engine.Export(s, platform)
		if err != nil {
			fmt.Printf("  %s 导出失败: %v\n", color.RedString("✗"), err)
			continue
		}

		// 确定目标路径
		targetPath := opts.output
		if targetPath == "" {
			if opts.install {
				defaultPath, err := export.GetDefaultTargetPath(platform)
				if err != nil {
					fmt.Printf("  %s 获取默认路径失败: %v\n", color.RedString("✗"), err)
					continue
				}
				targetPath = defaultPath
			} else {
				targetPath = filepath.Join("./export", platform)
			}
		}

		// 显示导出信息
		fmt.Printf("  目标路径: %s\n", color.YellowString(targetPath))
		fmt.Printf("  文件数量: %d\n", len(result.Files))
		for fileName := range result.Files {
			fmt.Printf("    - %s\n", fileName)
		}

		// 如果不是 dry-run，执行实际导出
		if !opts.dryRun {
			skillDir := filepath.Join(targetPath, s.Manifest.Name)
			if err := engine.Install(result, skillDir); err != nil {
				fmt.Printf("  %s 安装失败: %v\n", color.RedString("✗"), err)
				continue
			}
			fmt.Printf("  %s 导出成功\n", color.GreenString("✓"))
		} else {
			fmt.Printf("  %s (dry-run 模式，未实际写入)\n", color.YellowString("○"))
		}

		fmt.Println()
	}

	return nil
}
