package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type validateOptions struct {
	strict bool
}

func newValidateCmd() *cobra.Command {
	opts := &validateOptions{}

	cmd := &cobra.Command{
		Use:   "validate [path]",
		Short: "验证 SKILL.md 格式",
		Long:  "验证指定的 SKILL.md 文件是否符合规范",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) > 0 {
				path = args[0]
			}
			return runValidate(path, opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.strict, "strict", "s", false, "启用严格模式")

	return cmd
}

func runValidate(path string, opts *validateOptions) error {
	// 查找 SKILL.md 文件
	skillFile := findSkillFile(path)
	if skillFile == "" {
		return fmt.Errorf("未找到 SKILL.md 文件: %s", path)
	}

	fmt.Printf("验证文件: %s\n", color.CyanString(skillFile))
	fmt.Println()

	// 读取文件
	content, err := os.ReadFile(skillFile)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}

	// 验证
	result := validateSkill(content, opts.strict)

	// 输出结果
	if len(result.Errors) == 0 && len(result.Warnings) == 0 {
		fmt.Println(color.GreenString("✓"), "验证通过!")
		return nil
	}

	// 显示错误
	for _, err := range result.Errors {
		fmt.Printf("%s %s: %s\n", color.RedString("✗"), err.Field, err.Message)
	}

	// 显示警告
	for _, warn := range result.Warnings {
		fmt.Printf("%s %s: %s\n", color.YellowString("!"), warn.Field, warn.Message)
	}

	fmt.Println()
	if len(result.Errors) > 0 {
		fmt.Printf("发现 %d 个错误, %d 个警告\n", len(result.Errors), len(result.Warnings))
		return fmt.Errorf("验证失败")
	}

	fmt.Printf("%s 验证通过, 有 %d 个警告\n", color.GreenString("✓"), len(result.Warnings))
	return nil
}

func findSkillFile(path string) string {
	// 如果路径已经是文件，直接返回
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		if filepath.Base(path) == "SKILL.md" {
			return path
		}
	}

	// 在目录中查找 SKILL.md
	skillFile := filepath.Join(path, "SKILL.md")
	if _, err := os.Stat(skillFile); err == nil {
		return skillFile
	}

	return ""
}

// ValidationResult 验证结果
type ValidationResult struct {
	Errors   []ValidationIssue
	Warnings []ValidationIssue
}

// ValidationIssue 验证问题
type ValidationIssue struct {
	Field   string
	Message string
}

func validateSkill(content []byte, strict bool) *ValidationResult {
	result := &ValidationResult{
		Errors:   []ValidationIssue{},
		Warnings: []ValidationIssue{},
	}

	// 解析 frontmatter
	frontmatter, body, err := parseFrontmatter(string(content))
	if err != nil {
		result.Errors = append(result.Errors, ValidationIssue{
			Field:   "frontmatter",
			Message: fmt.Sprintf("解析失败: %v", err),
		})
		return result
	}

	// 验证必填字段
	var manifest SkillManifest
	if err := yaml.Unmarshal([]byte(frontmatter), &manifest); err != nil {
		result.Errors = append(result.Errors, ValidationIssue{
			Field:   "frontmatter",
			Message: fmt.Sprintf("YAML 解析失败: %v", err),
		})
		return result
	}

	// 验证 name
	if manifest.Name == "" {
		result.Errors = append(result.Errors, ValidationIssue{
			Field:   "name",
			Message: "技能名称不能为空",
		})
	} else if err := validateSkillName(manifest.Name); err != nil {
		result.Errors = append(result.Errors, ValidationIssue{
			Field:   "name",
			Message: err.Error(),
		})
	}

	// 验证 version
	if manifest.Version == "" {
		result.Errors = append(result.Errors, ValidationIssue{
			Field:   "version",
			Message: "版本号不能为空",
		})
	} else if !isValidSemver(manifest.Version) {
		result.Errors = append(result.Errors, ValidationIssue{
			Field:   "version",
			Message: "版本号必须是有效的 SemVer 格式 (如: 1.0.0)",
		})
	}

	// 验证 description
	if manifest.Description == "" {
		result.Errors = append(result.Errors, ValidationIssue{
			Field:   "description",
			Message: "描述不能为空",
		})
	} else if len(manifest.Description) > 200 {
		result.Warnings = append(result.Warnings, ValidationIssue{
			Field:   "description",
			Message: "描述建议不超过 200 个字符",
		})
	}

	// 验证 license
	if manifest.License != "" && !isValidSPDX(manifest.License) {
		result.Warnings = append(result.Warnings, ValidationIssue{
			Field:   "license",
			Message: fmt.Sprintf("许可证 '%s' 不是标准 SPDX 标识符", manifest.License),
		})
	}

	// 验证正文内容
	if strings.TrimSpace(body) == "" {
		result.Errors = append(result.Errors, ValidationIssue{
			Field:   "content",
			Message: "技能正文不能为空",
		})
	}

	// 严格模式检查
	if strict {
		if manifest.Author == "" {
			result.Warnings = append(result.Warnings, ValidationIssue{
				Field:   "author",
				Message: "建议填写作者信息",
			})
		}
		if len(manifest.Tags) == 0 {
			result.Warnings = append(result.Warnings, ValidationIssue{
				Field:   "tags",
				Message: "建议添加标签以提高可搜索性",
			})
		}
	}

	return result
}

// parseFrontmatter 解析 frontmatter
func parseFrontmatter(content string) (string, string, error) {
	content = strings.TrimSpace(content)

	// 检查是否以 --- 开头
	if !strings.HasPrefix(content, "---") {
		return "", "", fmt.Errorf("SKILL.md 必须以 --- 开头")
	}

	// 找到第二个 ---
	endIdx := strings.Index(content[3:], "---")
	if endIdx == -1 {
		return "", "", fmt.Errorf("未找到 frontmatter 结束标记 ---")
	}

	frontmatter := strings.TrimSpace(content[3 : 3+endIdx])
	body := strings.TrimSpace(content[3+endIdx+3:])

	return frontmatter, body, nil
}

// isValidSemver 验证 SemVer 格式
func isValidSemver(version string) bool {
	// 简化验证: 主版本号.次版本号.修订号
	pattern := `^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`
	re := regexp.MustCompile(pattern)
	return re.MatchString(version)
}

// isValidSPDX 验证 SPDX 许可证标识符
func isValidSPDX(license string) bool {
	// 常见的 SPDX 标识符
	validLicenses := []string{
		"MIT", "Apache-2.0", "GPL-3.0", "GPL-2.0", "BSD-2-Clause",
		"BSD-3-Clause", "ISC", "MPL-2.0", "LGPL-3.0", "LGPL-2.1",
		"Unlicense", "CC0-1.0", "Proprietary",
	}

	for _, valid := range validLicenses {
		if strings.EqualFold(license, valid) {
			return true
		}
	}
	return false
}
