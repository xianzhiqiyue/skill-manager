package validator

import (
	"regexp"
)

// ScanIssue 扫描问题
type ScanIssue struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Line     int    `json:"line"`
	Match    string `json:"match"`
}

// ScanResult 扫描结果
type ScanResult struct {
	Status  string      `json:"status"`
	Issues  []ScanIssue `json:"issues"`
}

// Scanner 安全扫描器
type Scanner struct {
	rules []Rule
}

// Rule 扫描规则
type Rule struct {
	Severity string
	Pattern  *regexp.Regexp
	Message  string
}

// NewScanner 创建扫描器
func NewScanner() *Scanner {
	return &Scanner{
		rules: []Rule{
			{
				Severity: "critical",
				Pattern:  regexp.MustCompile(`(?i)rm\s+-rf\s+/($|\s|;|&&|\|\||#)`),
				Message:  "检测到删除根目录的危险命令",
			},
			{
				Severity: "high",
				Pattern:  regexp.MustCompile(`(?i)ignore\s+(all\s+)?(previous|above|earlier)\s+(instructions?|commands?)`),
				Message:  "检测到提示词注入模式",
			},
		},
	}
}

// ScanContent 扫描内容
func (s *Scanner) ScanContent(content string) *ScanResult {
	result := &ScanResult{Status: "pass", Issues: []ScanIssue{}}

	for _, rule := range s.rules {
		if matches := rule.Pattern.FindAllStringIndex(content, -1); matches != nil {
			for _, match := range matches {
				issue := ScanIssue{
					Severity: rule.Severity,
					Message:  rule.Message,
					Line:     1,
					Match:    content[match[0]:match[1]],
				}
				result.Issues = append(result.Issues, issue)
				if rule.Severity == "critical" {
					result.Status = "fail"
				}
			}
		}
	}

	return result
}
