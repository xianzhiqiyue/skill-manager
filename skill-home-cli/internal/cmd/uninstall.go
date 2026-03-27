package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/skill-home/cli/internal/config"
	"github.com/skill-home/cli/internal/ide"
)

type uninstallOptions struct {
	ide       string
	global    bool
	keepCache bool
}

func newUninstallCmd() *cobra.Command {
	opts := &uninstallOptions{}

	cmd := &cobra.Command{
		Use:   "uninstall <skill-ref>",
		Short: "从本地 IDE 卸载技能",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUninstall(args[0], opts)
		},
	}

	cmd.Flags().StringVar(&opts.ide, "ide", "", targetIDEUsageText())
	cmd.Flags().BoolVar(&opts.global, "global", false, "从全局配置卸载而非项目配置")
	cmd.Flags().BoolVar(&opts.keepCache, "keep-cache", false, "保留本地缓存")

	return cmd
}

func runUninstall(skillRef string, opts *uninstallOptions) error {
	namespace, name, _, err := config.ParseSkillRef(skillRef)
	if err != nil {
		return err
	}
	namespace = strings.TrimPrefix(namespace, "@")

	resolver, err := config.NewPathResolver()
	if err != nil {
		return err
	}

	ides := getTargetIDEs(&syncOptions{ide: opts.ide})
	removed := 0
	for _, ideType := range ides {
		var targetPath string
		if opts.global {
			targetPath, err = resolver.GetIDEGlobalPath(ideType)
		} else {
			targetPath, err = resolver.GetIDEProjectPath(ideType)
		}
		if err != nil {
			fmt.Printf("%s %s: %v\n", color.RedString("✗"), ideType, err)
			continue
		}

		adapter, err := ide.NewAdapter(ideType, targetPath)
		if err != nil {
			fmt.Printf("%s %s: %v\n", color.RedString("✗"), ideType, err)
			continue
		}

		if err := adapter.UninstallSkill(name); err != nil && !os.IsNotExist(err) {
			fmt.Printf("%s %s: %v\n", color.RedString("✗"), ideType, err)
			continue
		}
		fmt.Printf("%s 已从 %s 卸载\n", color.GreenString("✓"), ideType)
		removed++
	}

	if !opts.keepCache {
		cachePath := config.GetSkillSourcePath(namespace, name)
		if _, err := os.Stat(cachePath); err == nil {
			if err := removeExistingOutput(cachePath); err != nil {
				return err
			}
			fmt.Printf("%s 已删除缓存: %s\n", color.GreenString("✓"), cachePath)
		}
	}

	if removed == 0 {
		return fmt.Errorf("未找到可卸载的 IDE 安装")
	}
	return nil
}
