package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/skill-home/cli/pkg/validator"
)

type scanOptions struct {
	strict bool
	json   bool
}

func newScanCmd() *cobra.Command {
	opts := &scanOptions{}

	cmd := &cobra.Command{
		Use:   "scan [path]",
		Short: "扫描技能安全",
		Long:  "扫描技能文件中的安全风险（恶意命令、提示词注入等）",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) > 0 {
				path = args[0]
			}
			return runScan(path, opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.strict, "strict", "s", false, "严格模式（将中级别问题视为错误）")
	cmd.Flags().BoolVar(&opts.json, "json", false, "JSON 格式输出")

	return cmd
}

func runScan(path string, opts *scanOptions) error {
	// 收集需要扫描的文件
	files := make(map[string]string)

	// 读取 SKILL.md
	skillFile := filepath.Join(path, "SKILL.md")
	if content, err := os.ReadFile(skillFile); err == nil {
		files["SKILL.md"] = string(content)
	} else {
		return fmt.Errorf("读取 SKILL.md 失败: %w", err)
	}

	// 读取 scripts 目录
	scriptsDir := filepath.Join(path, "scripts")
	if entries, err := os.ReadDir(scriptsDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			content, err := os.ReadFile(filepath.Join(scriptsDir, entry.Name()))
			if err == nil {
				files["scripts/"+entry.Name()] = string(content)
			}
		}
	}

	// 读取 references 目录
	refsDir := filepath.Join(path, "references")
	if entries, err := os.ReadDir(refsDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			content, err := os.ReadFile(filepath.Join(refsDir, entry.Name()))
			if err == nil {
				files["references/"+entry.Name()] = string(content)
			}
		}
	}

	// 执行扫描
	fmt.Printf("正在扫描 %d 个文件...\n\n", len(files))

	scanner := validator.NewScanner()
	result := scanner.ScanSkill(path, files)

	// 输出结果
	if opts.json {
		// TODO: JSON 输出
		return fmt.Errorf("JSON 格式尚未实现")
	}

	// 显示问题
	if len(result.Issues) == 0 {
		fmt.Println(color.GreenString("✓"), "未检测到安全问题")
		return nil
	}

	// 按文件分组显示
	fileIssues := make(map[string][]validator.ScanIssue)
	for _, issue := range result.Issues {
		fileIssues[issue.File] = append(fileIssues[issue.File], issue)
	}

	for filename, issues := range fileIssues {
		fmt.Printf("%s %s\n", color.CyanString("📄"), filename)
		for _, issue := range issues {
			printIssue(issue)
		}
		fmt.Println()
	}

	// 摘要
	fmt.Println(result.Summary)

	// 严格模式检查
	if opts.strict && result.Status != "pass" {
		return fmt.Errorf("严格模式：发现安全问题")
	}

	// 阻止发布的问题
	if result.HasCritical() {
		fmt.Println()
		fmt.Println(color.RedString("✗"), "发现严重问题，阻止发布")
		return fmt.Errorf("安全扫描未通过")
	}

	if result.HasHighSeverity() {
		fmt.Println()
		fmt.Println(color.YellowString("!"), "发现高级别问题，请修复后发布")
		fmt.Printf("或使用 %s 强制推送（不推荐）\n", color.YellowString("--force"))
		return fmt.Errorf("安全扫描未通过")
	}

	return nil
}

func printIssue(issue validator.ScanIssue) {
	var severityIcon string
	var severityStr string

	switch issue.Severity {
	case validator.SeverityCritical:
		severityIcon = "🔴"
		severityStr = color.RedString(string(issue.Severity))
	case validator.SeverityHigh:
		severityIcon = "🟠"
		severityStr = color.YellowString(string(issue.Severity))
	case validator.SeverityMedium:
		severityIcon = "🟡"
		severityStr = color.YellowString(string(issue.Severity))
	case validator.SeverityLow:
		severityIcon = "🔵"
		severityStr = color.BlueString(string(issue.Severity))
	}

	fmt.Printf("  %s %s [%s:%d:%d]\n",
		severityIcon,
		severityStr,
		issue.File,
		issue.Line,
		issue.Column,
	)
	fmt.Printf("     %s\n", issue.Message)
	fmt.Printf("     匹配: %s\n", color.MagentaString(issue.Match))
	if issue.Suggestion != "" {
		fmt.Printf("     建议: %s\n", issue.Suggestion)
	}
}
