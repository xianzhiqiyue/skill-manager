package validator

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
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
	CategorySensitiveData    Category = "sensitive_data"    // 敏感信息
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
					rawMatch := line[match[0]:match[1]]
					issue := ScanIssue{
						Severity:   rule.Severity,
						Category:   rule.Category,
						File:       filename,
						Line:       lineNum + 1,
						Column:     match[0] + 1,
						Match:      sanitizeMatch(rule.Category, rawMatch),
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
	start := time.Now()
	result := &ScanResult{
		Status:  "pass",
		Issues:  []ScanIssue{},
		Scanned: len(files),
	}

	filenames := make([]string, 0, len(files))
	for filename := range files {
		filenames = append(filenames, filename)
	}
	sort.Strings(filenames)

	for _, filename := range filenames {
		content := files[filename]
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
	result.Duration = time.Since(start).Milliseconds()

	return result
}

func sanitizeMatch(category Category, match string) string {
	if category == CategorySensitiveData {
		return "<redacted:sensitive-data>"
	}
	if len(match) > 160 {
		return match[:157] + "..."
	}
	return match
}

// defaultRules 返回默认扫描规则
func defaultRules() []Rule {
	return []Rule{
		// ========== 危险命令 ==========
		{
			Name:       "rm-rf-root",
			Severity:   SeverityCritical,
			Category:   CategoryDangerousCommand,
			Pattern:    regexp.MustCompile(`(?i)rm\s+-rf\s+/($|\s|;|&&|\|\||#)`),
			Message:    "检测到删除根目录的危险命令",
			Suggestion: "这会导致系统所有数据被删除，请检查命令安全性",
		},
		{
			Name:       "rm-rf-all",
			Severity:   SeverityCritical,
			Category:   CategoryDangerousCommand,
			Pattern:    regexp.MustCompile(`(?i)rm\s+-rf\s+/\s*\*`),
			Message:    "检测到删除所有文件的危险命令",
			Suggestion: "这会删除系统所有文件，请检查命令安全性",
		},
		{
			Name:       "mkfs",
			Severity:   SeverityCritical,
			Category:   CategoryDangerousCommand,
			Pattern:    regexp.MustCompile(`(?i)\bmkfs\.\w+\s+/dev/`),
			Message:    "检测到格式化文件系统命令",
			Suggestion: "这会格式化磁盘，导致数据丢失",
		},
		{
			Name:       "dd-disk",
			Severity:   SeverityCritical,
			Category:   CategoryDangerousCommand,
			Pattern:    regexp.MustCompile(`(?i)\bdd\s+.*of=/dev/[sh]d[a-z]`),
			Message:    "检测到直接写入磁盘的命令",
			Suggestion: "这可能会覆盖磁盘数据",
		},
		{
			Name:       "fork-bomb",
			Severity:   SeverityCritical,
			Category:   CategoryDangerousCommand,
			Pattern:    regexp.MustCompile(`:\(\)\{\s*:\|:\s*&\s*\};:`),
			Message:    "检测到 Fork Bomb",
			Suggestion: "这会导致系统资源耗尽崩溃",
		},
		{
			Name:       "chmod-root",
			Severity:   SeverityCritical,
			Category:   CategoryDangerousCommand,
			Pattern:    regexp.MustCompile(`(?i)\bchmod\s+-R\s+777\s+/($|\s|;|&&|\|\||#)`),
			Message:    "检测到递归放开根目录权限的危险命令",
			Suggestion: "这会破坏系统权限边界，请限定到明确的临时目录",
		},
		{
			Name:       "reverse-shell-dev-tcp",
			Severity:   SeverityCritical,
			Category:   CategoryDangerousCommand,
			Pattern:    regexp.MustCompile(`(?i)(bash\s+-i|sh\s+-i|/bin/(ba)?sh).*?/dev/tcp/`),
			Message:    "检测到反向 shell 命令",
			Suggestion: "技能中不应包含建立远程 shell 的命令",
		},
		{
			Name:       "netcat-exec",
			Severity:   SeverityCritical,
			Category:   CategoryDangerousCommand,
			Pattern:    regexp.MustCompile(`(?i)\b(nc|ncat|netcat)\b[^\n]*(\s-e\s|\s-c\s)`),
			Message:    "检测到 netcat 执行 shell 的危险命令",
			Suggestion: "技能中不应包含远程执行 shell 的网络监听或连接命令",
		},
		{
			Name:       "curl-pipe-shell",
			Severity:   SeverityHigh,
			Category:   CategoryDangerousCommand,
			Pattern:    regexp.MustCompile(`(?i)curl\s+[^|]*\|\s*(ba)?sh`),
			Message:    "检测到远程代码执行 (curl | sh)",
			Suggestion: "从网络下载并直接执行代码存在安全风险，建议先下载验证再执行",
		},
		{
			Name:       "wget-pipe-shell",
			Severity:   SeverityHigh,
			Category:   CategoryDangerousCommand,
			Pattern:    regexp.MustCompile(`(?i)wget\s+[^|]*(-O\s*-\s*)?\|\s*(ba)?sh`),
			Message:    "检测到远程代码执行 (wget | sh)",
			Suggestion: "从网络下载并直接执行代码存在安全风险",
		},
		{
			Name:       "base64-pipe-shell",
			Severity:   SeverityHigh,
			Category:   CategoryDangerousCommand,
			Pattern:    regexp.MustCompile(`(?i)\bbase64\s+(-d|--decode)\b[^|]*\|\s*(ba)?sh`),
			Message:    "检测到 base64 解码后直接执行 shell",
			Suggestion: "隐藏脚本内容后直接执行难以审核，请展开脚本并显式说明用途",
		},
		{
			Name:       "powershell-iex-download",
			Severity:   SeverityHigh,
			Category:   CategoryDangerousCommand,
			Pattern:    regexp.MustCompile(`(?i)\b(iwr|irm|invoke-webrequest|invoke-restmethod)\b[^\n|]*\|\s*(iex|invoke-expression)`),
			Message:    "检测到 PowerShell 下载后直接执行",
			Suggestion: "从网络下载并直接执行代码存在安全风险",
		},
		{
			Name:       "powershell-encoded-command",
			Severity:   SeverityHigh,
			Category:   CategoryDangerousCommand,
			Pattern:    regexp.MustCompile(`(?i)\bpowershell(\.exe)?\b[^\n]*(\-enc|\-encodedcommand)\b`),
			Message:    "检测到 PowerShell 编码命令",
			Suggestion: "编码命令会隐藏真实行为，请改为可审核的明文脚本",
		},
		{
			Name:       "eval-js",
			Severity:   SeverityMedium,
			Category:   CategorySuspiciousCode,
			Pattern:    regexp.MustCompile(`(?i)\beval\s*\(`),
			Message:    "检测到使用 eval()",
			Suggestion: "eval() 可能导致代码注入攻击，建议使用更安全的方式",
		},
		{
			Name:       "exec-js",
			Severity:   SeverityMedium,
			Category:   CategorySuspiciousCode,
			Pattern:    regexp.MustCompile(`(?i)\bexec\s*\(`),
			Message:    "检测到使用 exec()",
			Suggestion: "确保输入已正确转义和验证",
		},
		{
			Name:        "subprocess-shell-true",
			Severity:    SeverityHigh,
			Category:    CategorySuspiciousCode,
			Pattern:     regexp.MustCompile(`(?i)subprocess\.(call|run|check_output)\s*\([^)]*shell\s*=\s*True`),
			Message:     "检测到 Python subprocess 使用 shell=True",
			Suggestion:  "shell=True 存在命令注入风险，建议直接使用列表传参",
			FilePattern: regexp.MustCompile(`\.py$`),
		},
		{
			Name:        "os-system",
			Severity:    SeverityMedium,
			Category:    CategorySuspiciousCode,
			Pattern:     regexp.MustCompile(`(?i)\bos\.system\s*\(`),
			Message:     "检测到使用 os.system()",
			Suggestion:  "os.system() 存在安全风险，建议使用 subprocess 模块",
			FilePattern: regexp.MustCompile(`\.py$`),
		},
		{
			Name:       "ssh-private-key-read",
			Severity:   SeverityHigh,
			Category:   CategorySensitiveData,
			Pattern:    regexp.MustCompile(`(?i)(cat|type|more|less|cp|scp)\s+[^#\n]*(~/.ssh/id_(rsa|ed25519)|\.ssh/id_(rsa|ed25519))`),
			Message:    "检测到读取 SSH 私钥的操作",
			Suggestion: "不要在技能中读取或上传用户私钥",
		},

		// ========== 敏感信息 ==========
		{
			Name:       "private-key-block",
			Severity:   SeverityCritical,
			Category:   CategorySensitiveData,
			Pattern:    regexp.MustCompile(`-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----`),
			Message:    "检测到私钥内容",
			Suggestion: "请删除真实私钥，改用环境变量或凭证描述声明所需秘钥",
		},
		{
			Name:       "openai-api-key",
			Severity:   SeverityCritical,
			Category:   CategorySensitiveData,
			Pattern:    regexp.MustCompile(`\bsk-(proj-)?[A-Za-z0-9_-]{20,}\b`),
			Message:    "检测到疑似 OpenAI API Key",
			Suggestion: "请删除真实 API Key，改用 OPENAI_API_KEY 等环境变量声明",
		},
		{
			Name:       "github-token",
			Severity:   SeverityCritical,
			Category:   CategorySensitiveData,
			Pattern:    regexp.MustCompile(`\b(ghp|gho|ghu|ghs|ghr)_[A-Za-z0-9_]{20,}\b|\bgithub_pat_[A-Za-z0-9_]{20,}\b`),
			Message:    "检测到疑似 GitHub Token",
			Suggestion: "请删除真实 Token，改用凭证描述或运行时环境变量",
		},
		{
			Name:       "aws-access-key",
			Severity:   SeverityCritical,
			Category:   CategorySensitiveData,
			Pattern:    regexp.MustCompile(`\b(AKIA|ASIA)[0-9A-Z]{16}\b`),
			Message:    "检测到疑似 AWS Access Key",
			Suggestion: "请删除真实 Access Key，改用运行时凭证配置",
		},
		{
			Name:       "generic-secret-assignment",
			Severity:   SeverityCritical,
			Category:   CategorySensitiveData,
			Pattern:    regexp.MustCompile(`(?i)\b(api[_-]?key|access[_-]?key|secret|token|password|passwd|private[_-]?key)\b\s*[:=]\s*['"]?[^'"\s#]{12,}`),
			Message:    "检测到疑似秘钥或敏感值",
			Suggestion: "请移除真实敏感值，改用环境变量、requires 或 metadata.openclaw.credentials 描述",
		},

		// ========== 提示词注入 ==========
		{
			Name:       "ignore-instructions",
			Severity:   SeverityHigh,
			Category:   CategoryInjection,
			Pattern:    regexp.MustCompile(`(?i)ignore\s+(all\s+)?(previous|above|earlier|prior)\s+(instructions?|commands?|prompts?)`),
			Message:    "检测到提示词注入模式 (ignore instructions)",
			Suggestion: "这可能是试图覆盖系统指令的攻击",
		},
		{
			Name:       "disregard-prompt",
			Severity:   SeverityHigh,
			Category:   CategoryInjection,
			Pattern:    regexp.MustCompile(`(?i)disregard\s+(your\s+)?(system\s+)?(prompt|instructions?)`),
			Message:    "检测到提示词注入模式 (disregard prompt)",
			Suggestion: "这可能是试图覆盖系统指令的攻击",
		},
		{
			Name:       "dan-mode",
			Severity:   SeverityHigh,
			Category:   CategoryInjection,
			Pattern:    regexp.MustCompile(`(?i)\bDAN\s+(mode|do\s+anything\s+now)\b`),
			Message:    "检测到 DAN 模式提示",
			Suggestion: "这是已知的提示词注入技术",
		},
		{
			Name:       "jailbreak",
			Severity:   SeverityMedium,
			Category:   CategoryInjection,
			Pattern:    regexp.MustCompile(`(?i)\bjailbreak\b`),
			Message:    "检测到越狱相关提示",
			Suggestion: "这可能是试图绕过安全限制",
		},
		{
			Name:       "system-prompt-override",
			Severity:   SeverityHigh,
			Category:   CategoryInjection,
			Pattern:    regexp.MustCompile(`(?i)system\s*prompt\s*:\s*`),
			Message:    "检测到系统提示覆盖尝试",
			Suggestion: "这可能是试图伪装系统提示",
		},
		{
			Name:       "role-override",
			Severity:   SeverityLow,
			Category:   CategoryInjection,
			Pattern:    regexp.MustCompile(`(?i)you\s+are\s+now\s+\w+`),
			Message:    "检测到角色覆盖尝试",
			Suggestion: "这可能是试图改变 AI 角色定义",
		},
		{
			Name:       "developer-mode",
			Severity:   SeverityMedium,
			Category:   CategoryInjection,
			Pattern:    regexp.MustCompile(`(?i)developer\s+mode\s+(enabled?|on)`),
			Message:    "检测到开发者模式提示",
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
