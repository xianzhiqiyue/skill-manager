package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/skill-home/cli/internal/config"
	"github.com/skill-home/cli/internal/skill"
)

type updateOptions struct {
	ide      string
	mode     string
	global   bool
	force    bool
	skipScan bool
}

func newUpdateCmd() *cobra.Command {
	opts := &updateOptions{}

	cmd := &cobra.Command{
		Use:   "update",
		Short: "将本地缓存技能更新到远程最新版",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(opts)
		},
	}

	cmd.Flags().StringVar(&opts.ide, "ide", "", "指定 IDE (claude/copilot/cursor/codex)")
	cmd.Flags().StringVar(&opts.mode, "mode", "", "同步模式 (auto/symlink/mirror)")
	cmd.Flags().BoolVar(&opts.global, "global", false, "同步到全局配置而非项目配置")
	cmd.Flags().BoolVarP(&opts.force, "force", "f", false, "即使本地已存在最新版也重新下载")
	cmd.Flags().BoolVar(&opts.skipScan, "skip-scan", false, "跳过安装前安全扫描")

	return cmd
}

func runUpdate(opts *updateOptions) error {
	entries, err := discoverCachedSkills()
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		fmt.Println("没有可更新的本地技能")
		return nil
	}

	client := newRegistryClient()
	updated := 0
	for _, entry := range entries {
		skillInfo, err := client.GetSkill(entry.Namespace, entry.Name)
		if err != nil {
			fmt.Printf("%s 获取 @%s/%s 失败: %v\n", color.RedString("✗"), entry.Namespace, entry.Name, wrapRegistryReadError(err))
			continue
		}
		if skillInfo.LatestVersion == "" {
			continue
		}

		latestPath := config.GetSkillCacheDir(entry.Namespace, entry.Name, skillInfo.LatestVersion)
		if _, err := os.Stat(latestPath); err == nil && !opts.force {
			fmt.Printf("%s @%s/%s 已是最新版 (%s)\n", color.BlueString("→"), entry.Namespace, entry.Name, skillInfo.LatestVersion)
			continue
		}

		fmt.Printf("更新 @%s/%s -> %s\n", entry.Namespace, entry.Name, skillInfo.LatestVersion)
		pulled, err := pullSkillRef(fmt.Sprintf("@%s/%s@%s", entry.Namespace, entry.Name, skillInfo.LatestVersion), &pullOptions{
			extract: true,
			force:   opts.force,
		})
		if err != nil {
			fmt.Printf("%s 更新失败: %v\n", color.RedString("✗"), err)
			continue
		}

		if !opts.skipScan && config.C.Security.ScanOnInstall {
			if err := scanInstallTarget(pulled.OutputDir, opts.force); err != nil {
				fmt.Printf("%s 扫描失败: %v\n", color.RedString("✗"), err)
				continue
			}
		}

		s, err := skill.Parse(pulled.OutputDir)
		if err != nil {
			fmt.Printf("%s 解析失败: %v\n", color.RedString("✗"), err)
			continue
		}

		if err := syncParsedSkill(s, &syncOptions{
			ide:    opts.ide,
			mode:   opts.mode,
			global: opts.global,
		}); err != nil {
			fmt.Printf("%s 同步失败: %v\n", color.RedString("✗"), err)
			continue
		}
		updated++
	}

	if updated == 0 {
		fmt.Println("没有技能被更新")
		return nil
	}

	fmt.Printf("%s 已更新 %d 个技能\n", color.GreenString("✓"), updated)
	return nil
}

type cachedSkillEntry struct {
	Namespace string
	Name      string
}

func discoverCachedSkills() ([]cachedSkillEntry, error) {
	results := []cachedSkillEntry{}
	nsEntries, err := os.ReadDir(config.C.Local.SkillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return results, nil
		}
		return nil, err
	}

	for _, nsEntry := range nsEntries {
		if !nsEntry.IsDir() || strings.HasPrefix(nsEntry.Name(), ".") {
			continue
		}
		nsPath := filepath.Join(config.C.Local.SkillsDir, nsEntry.Name())
		skillEntries, err := os.ReadDir(nsPath)
		if err != nil {
			continue
		}
		for _, skillEntry := range skillEntries {
			if !skillEntry.IsDir() || strings.HasPrefix(skillEntry.Name(), ".") {
				continue
			}
			results = append(results, cachedSkillEntry{
				Namespace: nsEntry.Name(),
				Name:      skillEntry.Name(),
			})
		}
	}

	return results, nil
}
