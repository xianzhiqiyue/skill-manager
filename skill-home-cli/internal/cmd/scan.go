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

const maxScannedFileBytes = 1024 * 1024

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
	files, err := collectSkillFiles(path)
	if err != nil {
		return err
	}

	// 执行扫描
	fmt.Printf("正在扫描 %d 个文件...\n\n", len(files))

	result := validator.NewScanner().ScanSkill(path, files)

	// 输出结果
	if opts.json {
		if err := printJSON(result); err != nil {
			return err
		}
		return evaluateScanResult(result, opts.strict, true)
	}

	// 显示问题
	if len(result.Issues) == 0 {
		fmt.Println(color.GreenString("✓"), "未检测到安全问题")
		return nil
	}

	printScanResult(result)
	return evaluateScanResult(result, opts.strict, false)
}

func printScanResult(result *validator.ScanResult) {
	if result == nil || len(result.Issues) == 0 {
		return
	}

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

func collectSkillFiles(path string) (map[string]string, error) {
	files := make(map[string]string)

	root, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("解析技能路径失败: %w", err)
	}

	err = filepath.Walk(root, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Name() == "." {
			return nil
		}
		if strings.HasPrefix(info.Name(), ".") || shouldSkipFile(info.Name()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		if info.Size() > maxScannedFileBytes {
			return nil
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("读取文件失败 %s: %w", filePath, err)
		}
		if strings.ContainsRune(string(content), '\x00') {
			return nil
		}

		rel, err := filepath.Rel(root, filePath)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = string(content)
		return nil
	})
	if err != nil {
		return nil, err
	}

	if _, ok := files["SKILL.md"]; !ok {
		return nil, fmt.Errorf("读取 SKILL.md 失败: 文件不存在")
	}

	return files, nil
}

func scanSkillPath(path string) (*validator.ScanResult, error) {
	files, err := collectSkillFiles(path)
	if err != nil {
		return nil, err
	}
	return validator.NewScanner().ScanSkill(path, files), nil
}

func evaluateScanResult(result *validator.ScanResult, strict bool, silent bool) error {
	if strict && result.Status != "pass" {
		return fmt.Errorf("严格模式：发现安全问题")
	}

	if result.HasCritical() {
		if !silent {
			fmt.Println()
			fmt.Println(color.RedString("✗"), "发现严重问题，阻止发布")
		}
		return fmt.Errorf("安全扫描未通过")
	}

	if result.HasHighSeverity() {
		if !silent {
			fmt.Println()
			fmt.Println(color.YellowString("!"), "发现高级别问题，请修复后发布")
			fmt.Printf("或使用 %s 强制继续（不推荐）\n", color.YellowString("--force"))
		}
		return fmt.Errorf("安全扫描未通过")
	}

	return nil
}
