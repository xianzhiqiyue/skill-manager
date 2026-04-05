package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"

	"github.com/skill-home/cli/internal/registry"
)

func printJSON(v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func printSkillResults(result *registry.SearchResult) {
	fmt.Printf("找到 %d 个结果 (第 %d 页)\n\n", result.Total, result.Page)

	for _, skill := range result.Results {
		printSkillSummary(skill)
		fmt.Println()
	}

	if result.Total > result.PerPage*result.Page {
		fmt.Printf("使用 --page %d 查看更多结果\n", result.Page+1)
	}
}

func printSkillSummary(skill registry.Skill) {
	fullName := fmt.Sprintf("@%s/%s", strings.TrimPrefix(skill.Namespace, "@"), skill.Name)
	fmt.Printf("%s %s\n", color.GreenString("•"), color.CyanString(fullName))
	if description := preferredSkillDescription(skill.DescriptionZh, skill.Description); description != "" {
		fmt.Printf("  %s\n", description)
	}

	meta := []string{}
	if skill.LatestVersion != "" {
		meta = append(meta, fmt.Sprintf("v%s", skill.LatestVersion))
	}
	if skill.DownloadCount > 0 {
		meta = append(meta, fmt.Sprintf("%d 下载", skill.DownloadCount))
	}
	if skill.RatingCount > 0 {
		meta = append(meta, fmt.Sprintf("%.1f★ (%d)", skill.Rating, skill.RatingCount))
	}
	if len(skill.Tags) > 0 {
		meta = append(meta, strings.Join(skill.Tags, ", "))
	}
	if len(meta) > 0 {
		fmt.Printf("  %s\n", color.YellowString(strings.Join(meta, " • ")))
	}
}

func formatTimeShort(ts time.Time) string {
	if ts.IsZero() {
		return "-"
	}
	return ts.Format(time.RFC3339)
}
