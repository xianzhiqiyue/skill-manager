package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/skill-home/cli/internal/registry"
)

type activityOptions struct {
	page    int
	perPage int
	action  string
	format  string
}

func newActivityCmd() *cobra.Command {
	opts := &activityOptions{}

	cmd := &cobra.Command{
		Use:   "activity",
		Short: "查看最近的账号活动",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runActivity(opts)
		},
	}

	cmd.Flags().IntVarP(&opts.page, "page", "p", 1, "页码")
	cmd.Flags().IntVar(&opts.perPage, "per-page", 20, "每页数量")
	cmd.Flags().StringVar(&opts.action, "action", "", "按动作筛选")
	cmd.Flags().StringVarP(&opts.format, "format", "f", "table", "输出格式 (table/json)")

	return cmd
}

func runActivity(opts *activityOptions) error {
	if err := requireRegistryLogin(); err != nil {
		return err
	}

	result, err := newRegistryClient().ListAuditLogs(opts.page, opts.perPage, opts.action)
	if err != nil {
		return fmt.Errorf("获取活动日志失败: %w", err)
	}

	if opts.format == "json" {
		return printJSON(result)
	}

	if len(result.Results) == 0 {
		fmt.Println("没有活动记录")
		return nil
	}

	fmt.Printf("最近活动 (%d)\n\n", result.Total)
	for _, item := range result.Results {
		fmt.Printf("%s  %s  %s\n", formatTimeShort(item.CreatedAt), item.Action, formatAuditResource(item))
		if meta := formatAuditMetadata(item.Metadata); meta != "" {
			fmt.Printf("  %s\n", meta)
		}
	}

	if result.Total > result.PerPage*result.Page {
		fmt.Printf("\n使用 --page %d 查看更多活动\n", result.Page+1)
	}
	return nil
}

func formatAuditResource(item registry.AuditLog) string {
	if item.ResourceType == "" {
		return "-"
	}
	if item.ResourceID != nil && *item.ResourceID != "" {
		return fmt.Sprintf("%s/%s", item.ResourceType, *item.ResourceID)
	}
	return item.ResourceType
}

func formatAuditMetadata(metadata map[string]interface{}) string {
	if len(metadata) == 0 {
		return ""
	}
	keys := make([]string, 0, len(metadata))
	for key := range metadata {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", key, metadata[key]))
	}
	return strings.Join(parts, "  ")
}
