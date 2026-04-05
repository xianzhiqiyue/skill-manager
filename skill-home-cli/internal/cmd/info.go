package cmd

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/skill-home/cli/internal/config"
)

type infoOptions struct {
	format string
}

func newInfoCmd() *cobra.Command {
	opts := &infoOptions{}

	cmd := &cobra.Command{
		Use:   "info <skill-ref>",
		Short: "查看技能详情",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInfo(args[0], opts)
		},
	}

	cmd.Flags().StringVarP(&opts.format, "format", "f", "table", "输出格式 (table/json)")
	return cmd
}

func runInfo(skillRef string, opts *infoOptions) error {
	namespace, name, _, err := config.ParseSkillRef(skillRef)
	if err != nil {
		return err
	}

	client := newRegistryClient()
	skill, err := client.GetSkill(namespace, name)
	if err != nil {
		return fmt.Errorf("获取技能详情失败: %w", wrapRegistryReadError(err))
	}

	if len(skill.Versions) == 0 {
		if versions, err := client.ListVersions(namespace, name); err == nil {
			skill.Versions = versions
		}
	}

	if opts.format == "json" {
		return printJSON(skill)
	}

	fullName := fmt.Sprintf("@%s/%s", strings.TrimPrefix(skill.Namespace, "@"), skill.Name)
	fmt.Printf("%s\n", color.CyanString(fullName))
	if description := preferredSkillDescription(skill.DescriptionZh, skill.Description); description != "" {
		fmt.Printf("%s\n", description)
	}
	fmt.Println()
	fmt.Printf("最新版本: %s\n", color.YellowString(skill.LatestVersion))
	fmt.Printf("下载量: %d\n", skill.DownloadCount)
	if skill.RatingCount > 0 {
		fmt.Printf("评分: %.1f (%d)\n", skill.Rating, skill.RatingCount)
	}
	fmt.Printf("可见性: %s\n", visibilityLabel(skill.IsPublic))
	if skill.License != "" {
		fmt.Printf("许可证: %s\n", skill.License)
	}
	if len(skill.Tags) > 0 {
		fmt.Printf("标签: %s\n", strings.Join(skill.Tags, ", "))
	}
	if skill.Homepage != "" {
		fmt.Printf("主页: %s\n", skill.Homepage)
	}
	if skill.Owner != nil && skill.Owner.Username != "" {
		fmt.Printf("所有者: %s\n", skill.Owner.Username)
	}
	fmt.Printf("更新时间: %s\n", formatTimeShort(skill.UpdatedAt))

	if skill.UserRating != nil {
		fmt.Printf("我的评分: %d", skill.UserRating.Rating)
		if skill.UserRating.Comment != "" {
			fmt.Printf(" (%s)", skill.UserRating.Comment)
		}
		fmt.Println()
	}

	if len(skill.Versions) > 0 {
		fmt.Println()
		fmt.Println("版本:")
		for _, version := range skill.Versions {
			line := fmt.Sprintf("  - v%s", version.Version)
			if version.ScanStatus != "" {
				line += fmt.Sprintf("  [%s]", version.ScanStatus)
			}
			if !version.PublishedAt.IsZero() {
				line += fmt.Sprintf("  %s", formatTimeShort(version.PublishedAt))
			}
			fmt.Println(line)
		}
	}

	return nil
}

func visibilityLabel(public bool) string {
	if public {
		return "public"
	}
	return "private"
}
