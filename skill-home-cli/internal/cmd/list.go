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

type listOptions struct {
	remote    bool
	namespace string
	format    string
}

func newListCmd() *cobra.Command {
	opts := &listOptions{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "列出已安装的技能",
		Long:  "列出本地已安装或已缓存的技能",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.remote, "remote", "r", false, "列出云端已发布的技能（需登录）")
	cmd.Flags().StringVarP(&opts.namespace, "namespace", "n", "", "按命名空间筛选")
	cmd.Flags().StringVarP(&opts.format, "format", "f", "table", "输出格式 (table/json)")

	return cmd
}

func runList(opts *listOptions) error {
	if opts.remote {
		return listRemoteSkills(opts)
	}
	return listLocalSkills(opts)
}

func listLocalSkills(opts *listOptions) error {
	skillsDir := config.C.Local.SkillsDir

	// 检查目录是否存在
	if _, err := os.Stat(skillsDir); os.IsNotExist(err) {
		fmt.Println("本地技能目录为空")
		fmt.Printf("运行 '%s' 创建你的第一个技能\n", color.YellowString("skill-home init <name>"))
		return nil
	}

	// 扫描命名空间
	nsEntries, err := os.ReadDir(skillsDir)
	if err != nil {
		return fmt.Errorf("读取技能目录失败: %w", err)
	}

	fmt.Printf("本地技能缓存目录: %s\n\n", color.CyanString(skillsDir))

	found := false
	for _, nsEntry := range nsEntries {
		if !nsEntry.IsDir() || strings.HasPrefix(nsEntry.Name(), ".") {
			continue
		}

		ns := nsEntry.Name()
		if opts.namespace != "" && ns != opts.namespace {
			continue
		}

		// 扫描技能
		nsPath := filepath.Join(skillsDir, ns)
		skillEntries, err := os.ReadDir(nsPath)
		if err != nil {
			continue
		}

		for _, skillEntry := range skillEntries {
			if !skillEntry.IsDir() {
				continue
			}

			skillName := skillEntry.Name()
			skillPath := filepath.Join(nsPath, skillName)

			// 尝试解析技能获取版本
			version := "unknown"
			if s, err := skill.Parse(skillPath); err == nil {
				version = s.Manifest.Version
			}

			fmt.Printf("  %s/%s@%s\n", color.GreenString("@"+ns), skillName, color.YellowString(version))
			found = true
		}
	}

	if !found {
		fmt.Println("没有找到本地技能")
		fmt.Printf("运行 '%s' 创建你的第一个技能\n", color.YellowString("skill-home init <name>"))
	}

	return nil
}

func listRemoteSkills(opts *listOptions) error {
	// TODO: 实现远程技能列表
	return fmt.Errorf("远程技能列表功能尚未实现")
}

func stringsHasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
