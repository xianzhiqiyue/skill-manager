package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/skill-home/cli/internal/config"
	"github.com/skill-home/cli/internal/ide"
	"github.com/skill-home/cli/internal/skill"
	"github.com/skill-home/cli/internal/sync"
)

type syncOptions struct {
	all      bool
	ide      string
	dryRun   bool
	mode     string
	global   bool
}

func newSyncCmd() *cobra.Command {
	opts := &syncOptions{}

	cmd := &cobra.Command{
		Use:   "sync [skill-path]",
		Short: "同步技能到 IDE",
		Long:  "将技能同步到本地 IDE 配置目录（Claude、Cursor、Codex）",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) > 0 {
				path = args[0]
			}
			return runSync(path, opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.all, "all", "a", false, "同步到所有启用的 IDE")
	cmd.Flags().StringVar(&opts.ide, "ide", "", "指定 IDE (claude/cursor/codex)")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "预览同步结果，不实际执行")
	cmd.Flags().StringVar(&opts.mode, "mode", "", "同步模式 (auto/symlink/mirror)")
	cmd.Flags().BoolVar(&opts.global, "global", false, "同步到全局配置而非项目配置")

	return cmd
}

func runSync(path string, opts *syncOptions) error {
	// 解析技能
	s, err := skill.Parse(path)
	if err != nil {
		// 可能是技能名称，尝试从缓存加载
		return fmt.Errorf("解析技能失败: %w", err)
	}

	fmt.Printf("同步技能: %s\n", color.CyanString(s.GetFullName()))
	fmt.Printf("版本: %s\n", color.CyanString(s.Manifest.Version))
	fmt.Println()

	// 创建路径解析器
	resolver, err := config.NewPathResolver()
	if err != nil {
		return err
	}

	// 确定要同步的 IDE
	ides := getTargetIDEs(opts)
	if len(ides) == 0 {
		return fmt.Errorf("没有启用的 IDE，请检查配置")
	}

	// 创建同步引擎
	mode := sync.ModeAuto
	if opts.mode != "" {
		mode = sync.SyncMode(opts.mode)
	}
	engine := sync.NewEngine(mode)

	// 执行同步
	successCount := 0
	for _, ideType := range ides {
		fmt.Printf("同步到 %s...\n", color.YellowString(ideType))

		// 获取目标路径
		var targetPath string
		if opts.global {
			targetPath, err = resolver.GetIDEGlobalPath(ideType)
			if err != nil {
				fmt.Printf("  %s %v\n", color.RedString("✗"), err)
				continue
			}
		} else {
			targetPath, err = resolver.GetIDEProjectPath(ideType)
			if err != nil {
				fmt.Printf("  %s %v\n", color.RedString("✗"), err)
				continue
			}
		}

		// 创建 IDE 适配器
		adapter, err := ide.NewAdapter(ideType, targetPath)
		if err != nil {
			fmt.Printf("  %s %v\n", color.RedString("✗"), err)
			continue
		}

		if opts.dryRun {
			fmt.Printf("  %s 将同步到: %s\n", color.BlueString("→"), targetPath)
			successCount++
			continue
		}

		// 执行同步
		if err := engine.Sync(s, adapter); err != nil {
			fmt.Printf("  %s %v\n", color.RedString("✗"), err)
			continue
		}

		fmt.Printf("  %s 同步成功\n", color.GreenString("✓"))
		successCount++
	}

	fmt.Println()
	if successCount == len(ides) {
		fmt.Println(color.GreenString("✓"), "所有同步完成!")
	} else {
		fmt.Printf("%s 部分同步失败 (%d/%d)\n", color.YellowString("!"), successCount, len(ides))
	}

	return nil
}

func getTargetIDEs(opts *syncOptions) []string {
	ides := []string{}

	if opts.ide != "" {
		return []string{opts.ide}
	}

	if config.C.IDE.Claude.Enabled {
		ides = append(ides, "claude")
	}
	if config.C.IDE.Cursor.Enabled {
		ides = append(ides, "cursor")
	}
	if config.C.IDE.Codex.Enabled {
		ides = append(ides, "codex")
	}

	return ides
}

// listInstalledSkills 列出已安装的技能
func listInstalledSkills(namespace string) ([]string, error) {
	skillsDir := config.C.Local.SkillsDir

	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return nil, err
	}

	skills := []string{}
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			skills = append(skills, entry.Name())
		}
	}

	return skills, nil
}
