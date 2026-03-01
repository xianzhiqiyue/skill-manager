package validator

import (
	"fmt"
	"regexp"
	"strings"
)

// Severity 问题严重级别
type Severity string

const (
	SeverityCritical Severity = "critical" // 严重，阻止发布
	SeverityHigh     Severity = "high"     // 高，阻止发布（可强制覆盖）
	SeverityMedium   Severity = "medium"   // 中，警告
	SeverityLow      Severity = "low"      // 低，提示
)

// Category 问题类别
type Category string

const (
	CategoryDangerousCommand Category = "dangerous_command" // 危险命令
	CategoryInjection        Category = "prompt_injection"  // 提示词注入
	CategorySuspiciousCode   Category = "suspicious_code"   // 可疑代码
)

// ScanIssue 扫描发现的问题
type ScanIssue struct {
	Severity   Severity `json:"severity"`
	Category   Category `json:"category"`
	File       string   `json:"file"`
	Line       int      `json:"line"`
	Column     int      `json:"column"`
	Match      string   `json:"match"`
	Message    string   `json:"message"`
	Suggestion string   `json:"suggestion,omitempty"`
}

// ScanResult 扫描结果
type ScanResult struct {
	Status   string      `json:"status"` // pass / warn / fail
	Summary  string      `json:"summary"`
	Issues   []ScanIssue `json:"issues"`
	Scanned  int         `json:"scanned_files"`
	Duration int64       `json:"duration_ms"`
}

// Scanner 安全扫描器
type Scanner struct {
	rules []Rule
}

// Rule 扫描规则
type Rule struct {
	Name        string
	Severity    Severity
	Category    Category
	Pattern     *regexp.Regexp
	Message     string
	Suggestion  string
	FilePattern *regexp.Regexp // 匹配的文件类型，nil 表示所有文件
}

// NewScanner 创建扫描器
func NewScanner() *Scanner {
	return &Scanner{
		rules: defaultRules(),
	}
}

// AddRule 添加自定义规则
func (s *Scanner) AddRule(rule Rule) {
	s.rules = append(s.rules, rule)
}

// ScanContent 扫描文件内容
func (s *Scanner) ScanContent(filename, content string) []ScanIssue {
	issues := []ScanIssue{}
	lines := strings.Split(content, "\n")

	for _, rule := range s.rules {
		// 检查文件类型匹配
		if rule.FilePattern != nil && !rule.FilePattern.MatchString(filename) {
			continue
		}

		// 按行扫描
		for lineNum, line := range lines {
			if matches := rule.Pattern.FindAllStringIndex(line, -1); matches != nil {
				for _, match := range matches {
					issue := ScanIssue{
						Severity:   rule.Severity,
						Category:   rule.Category,
						File:       filename,
						Line:       lineNum + 1,
						Column:     match[0] + 1,
						Match:      line[match[0]:match[1]],
						Message:    rule.Message,
						Suggestion: rule.Suggestion,
					}
					issues = append(issues, issue)
				}
			}
		}
	}

	return issues
}

// ScanSkill 扫描整个技能
func (s *Scanner) ScanSkill(skillPath string, files map[string]string) *ScanResult {
	result := &ScanResult{
		Status:  "pass",
		Issues:  []ScanIssue{},
		Scanned: len(files),
	}

	for filename, content := range files {
		issues := s.ScanContent(filename, content)
		result.Issues = append(result.Issues, issues...)
	}

	// 根据问题级别确定状态
	for _, issue := range result.Issues {
		switch issue.Severity {
		case SeverityCritical:
			result.Status = "fail"
		case SeverityHigh:
			if result.Status != "fail" {
				result.Status = "fail"
			}
		case SeverityMedium, SeverityLow:
			if result.Status == "pass" {
				result.Status = "warn"
			}
		}
	}

	// 生成摘要
	criticalCount := 0
	highCount := 0
	mediumCount := 0
	lowCount := 0

	for _, issue := range result.Issues {
		switch issue.Severity {
		case SeverityCritical:
			criticalCount++
		case SeverityHigh:
			highCount++
		case SeverityMedium:
			mediumCount++
		case SeverityLow:
			lowCount++
		}
	}

	result.Summary = fmt.Sprintf(
		"发现 %d 个问题 (严重: %d, 高: %d, 中: %d, 低: %d)",
		len(result.Issues), criticalCount, highCount, mediumCount, lowCount,
	)

	return result
}

// defaultRules 返回默认扫描规则
func defaultRules() []Rule {
	return []Rule{
		// ========== 危险命令 ==========
		{
			Name:     "rm-rf-root",
			Severity: SeverityCritical,
			Category: CategoryDangerousCommand,
			Pattern:  regexp.MustCompile(`(?i)rm\s+-rf\s+/($|\s|;|&&|\|\||#)`),
			Message:  "检测到删除根目录的危险命令",
			Suggestion: "这会导致系统所有数据被删除，请检查命令安全性",
		},
		{
			Name:     "rm-rf-all",
			Severity: SeverityCritical,
			Category: CategoryDangerousCommand,
			Pattern:  regexp.MustCompile(`(?i)rm\s+-rf\s+/\s*\*`),
			Message:  "检测到删除所有文件的危险命令",
			Suggestion: "这会删除系统所有文件，请检查命令安全性",
		},
		{
			Name:     "mkfs",
			Severity: SeverityCritical,
			Category: CategoryDangerousCommand,
			Pattern:  regexp.MustCompile(`(?i)\bmkfs\.\w+\s+/dev/`),
			Message:  "检测到格式化文件系统命令",
			Suggestion: "这会格式化磁盘，导致数据丢失",
		},
		{
			Name:     "dd-disk",
			Severity: SeverityCritical,
			Category: CategoryDangerousCommand,
			Pattern:  regexp.MustCompile(`(?i)\bdd\s+.*of=/dev/[sh]d[a-z]`),
			Message:  "检测到直接写入磁盘的命令",
			Suggestion: "这可能会覆盖磁盘数据",
		},
		{
			Name:     "fork-bomb",
			Severity: SeverityCritical,
			Category: CategoryDangerousCommand,
			Pattern:  regexp.MustCompile(`:\(\)\{\s*:\|:\s*&\s*\};:`),
			Message:  "检测到 Fork Bomb",
			Suggestion: "这会导致系统资源耗尽崩溃",
		},
		{
			Name:     "curl-pipe-shell",
			Severity: SeverityHigh,
			Category: CategoryDangerousCommand,
			Pattern:  regexp.MustCompile(`(?i)curl\s+[^|]*\|\s*(ba)?sh`),
			Message:  "检测到远程代码执行 (curl | sh)",
			Suggestion: "从网络下载并直接执行代码存在安全风险，建议先下载验证再执行",
		},
		{
			Name:     "wget-pipe-shell",
			Severity: SeverityHigh,
			Category: CategoryDangerousCommand,
			Pattern:  regexp.MustCompile(`(?i)wget\s+[^|]*(-O\s*-\s*)?\|\s*(ba)?sh`),
			Message:  "检测到远程代码执行 (wget | sh)",
			Suggestion: "从网络下载并直接执行代码存在安全风险",
		},
		{
			Name:     "eval-js",
			Severity: SeverityMedium,
			Category: CategorySuspiciousCode,
			Pattern:  regexp.MustCompile(`(?i)\beval\s*\(`),
			Message:  "检测到使用 eval()",
			Suggestion: "eval() 可能导致代码注入攻击，建议使用更安全的方式",
		},
		{
			Name:     "exec-js",
			Severity: SeverityMedium,
			Category: CategorySuspiciousCode,
			Pattern:  regexp.MustCompile(`(?i)\bexec\s*\(`),
			Message:  "检测到使用 exec()",
			Suggestion: "确保输入已正确转义和验证",
		},
		{
			Name:     "subprocess-shell-true",
			Severity: SeverityHigh,
			Category: CategorySuspiciousCode,
			Pattern:  regexp.MustCompile(`(?i)subprocess\.(call|run|check_output)\s*\([^)]*shell\s*=\s*True`),
			Message:  "检测到 Python subprocess 使用 shell=True",
			Suggestion: "shell=True 存在命令注入风险，建议直接使用列表传参",
			FilePattern: regexp.MustCompile(`\.py$`),
		},
		{
			Name:     "os-system",
			Severity: SeverityMedium,
			Category: CategorySuspiciousCode,
			Pattern:  regexp.MustCompile(`(?i)\bos\.system\s*\(`),
			Message:  "检测到使用 os.system()",
			Suggestion: "os.system() 存在安全风险，建议使用 subprocess 模块",
			FilePattern: regexp.MustCompile(`\.py$`),
		},

		// ========== 提示词注入 ==========
		{
			Name:     "ignore-instructions",
			Severity: SeverityHigh,
			Category: CategoryInjection,
			Pattern:  regexp.MustCompile(`(?i)ignore\s+(all\s+)?(previous|above|earlier|prior)\s+(instructions?|commands?|prompts?)`),
			Message:  "检测到提示词注入模式 (ignore instructions)",
			Suggestion: "这可能是试图覆盖系统指令的攻击",
		},
		{
			Name:     "disregard-prompt",
			Severity: SeverityHigh,
			Category: CategoryInjection,
			Pattern:  regexp.MustCompile(`(?i)disregard\s+(your\s+)?(system\s+)?(prompt|instructions?)`),
			Message:  "检测到提示词注入模式 (disregard prompt)",
			Suggestion: "这可能是试图覆盖系统指令的攻击",
		},
		{
			Name:     "dan-mode",
			Severity: SeverityHigh,
			Category: CategoryInjection,
			Pattern:  regexp.MustCompile(`(?i)\bDAN\s+(mode|do\s+anything\s+now)\b`),
			Message:  "检测到 DAN 模式提示",
			Suggestion: "这是已知的提示词注入技术",
		},
		{
			Name:     "jailbreak",
			Severity: SeverityMedium,
			Category: CategoryInjection,
			Pattern:  regexp.MustCompile(`(?i)\bjailbreak\b`),
			Message:  "检测到越狱相关提示",
			Suggestion: "这可能是试图绕过安全限制",
		},
		{
			Name:     "system-prompt-override",
			Severity: SeverityHigh,
			Category: CategoryInjection,
			Pattern:  regexp.MustCompile(`(?i)system\s*prompt\s*:\s*`),
			Message:  "检测到系统提示覆盖尝试",
			Suggestion: "这可能是试图伪装系统提示",
		},
		{
			Name:     "role-override",
			Severity: SeverityLow,
			Category: CategoryInjection,
			Pattern:  regexp.MustCompile(`(?i)you\s+are\s+now\s+\w+`),
			Message:  "检测到角色覆盖尝试",
			Suggestion: "这可能是试图改变 AI 角色定义",
		},
		{
			Name:     "developer-mode",
			Severity: SeverityMedium,
			Category: CategoryInjection,
			Pattern:  regexp.MustCompile(`(?i)developer\s+mode\s+(enabled?|on)`),
			Message:  "检测到开发者模式提示",
			Suggestion: "这可能是试图启用受限功能",
		},
	}
}

// ShouldBlock 是否阻止发布
func (r *ScanResult) ShouldBlock(force bool) bool {
	if r.Status == "pass" {
		return false
	}

	for _, issue := range r.Issues {
		if issue.Severity == SeverityCritical {
			return true
		}
		if issue.Severity == SeverityHigh && !force {
			return true
		}
	}

	return false
}

// HasCritical 是否有严重问题
func (r *ScanResult) HasCritical() bool {
	for _, issue := range r.Issues {
		if issue.Severity == SeverityCritical {
			return true
		}
	}
	return false
}

// HasHighSeverity 是否有高级别问题
func (r *ScanResult) HasHighSeverity() bool {
	for _, issue := range r.Issues {
		if issue.Severity == SeverityHigh || issue.Severity == SeverityCritical {
			return true
		}
	}
	return false
}
