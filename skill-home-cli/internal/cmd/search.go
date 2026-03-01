package cmd

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/skill-home/cli/internal/registry"
)

type searchOptions struct {
	namespace string
	tags      []string
	page      int
	perPage   int
	format    string
}

func newSearchCmd() *cobra.Command {
	opts := &searchOptions{}

	cmd := &cobra.Command{
		Use:   "search <keyword>",
		Short: "搜索技能",
		Long:  "在注册中心搜索技能",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := ""
			if len(args) > 0 {
				query = args[0]
			}
			return runSearch(query, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.namespace, "namespace", "n", "", "按命名空间筛选")
	cmd.Flags().StringArrayVarP(&opts.tags, "tag", "t", nil, "按标签筛选")
	cmd.Flags().IntVarP(&opts.page, "page", "p", 1, "页码")
	cmd.Flags().IntVar(&opts.perPage, "per-page", 20, "每页数量")
	cmd.Flags().StringVarP(&opts.format, "format", "f", "table", "输出格式 (table/json)")

	return cmd
}

func runSearch(query string, opts *searchOptions) error {
	// API Key 可选
	apiKey := viper.GetString("registry.api_key")
	server := viper.GetString("registry.endpoint")
	if server == "" {
		server = "https://registry.skill-home.dev"
	}

	fmt.Printf("搜索: %s\n", color.CyanString(query))
	if len(opts.tags) > 0 {
		fmt.Printf("标签: %s\n", color.CyanString(strings.Join(opts.tags, ", ")))
	}
	fmt.Println()

	// 创建客户端
	client := registry.NewClient(server, apiKey)

	// 搜索
	result, err := client.Search(query, opts.tags, opts.page, opts.perPage)
	if err != nil {
		return fmt.Errorf("搜索失败: %w", err)
	}

	if len(result.Results) == 0 {
		fmt.Println("没有找到匹配的技能")
		return nil
	}

	// 输出结果
	if opts.format == "json" {
		// TODO: JSON 输出
		return fmt.Errorf("JSON 格式尚未实现")
	}

	// 表格输出
	fmt.Printf("找到 %d 个结果 (第 %d 页)\n\n", result.Total, result.Page)

	for _, skill := range result.Results {
		fullName := fmt.Sprintf("@%s/%s", skill.Namespace, skill.Name)
		fmt.Printf("%s %s\n", color.GreenString("•"), color.CyanString(fullName))
		fmt.Printf("  %s\n", skill.Description)

		// 显示元数据
		meta := []string{}
		if skill.LatestVersion != "" {
			meta = append(meta, fmt.Sprintf("v%s", skill.LatestVersion))
		}
		if skill.DownloadCount > 0 {
			meta = append(meta, fmt.Sprintf("%d 下载", skill.DownloadCount))
		}
		if skill.RatingCount > 0 {
			meta = append(meta, fmt.Sprintf("%.1f★", skill.Rating))
		}
		if len(skill.Tags) > 0 {
			meta = append(meta, strings.Join(skill.Tags, ", "))
		}

		if len(meta) > 0 {
			fmt.Printf("  %s\n", color.YellowString(strings.Join(meta, " • ")))
		}
		fmt.Println()
	}

	// 分页提示
	if result.Total > result.PerPage*result.Page {
		fmt.Printf("使用 --page %d 查看更多结果\n", result.Page+1)
	}

	fmt.Printf("\n运行 '%s' 安装技能\n", color.YellowString("skill-home pull <name>"))

	return nil
}
