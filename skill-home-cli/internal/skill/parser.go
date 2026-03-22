package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// IDEConfig 多平台 IDE 配置
type IDEConfig struct {
	Claude *ClaudeConfig `yaml:"claude,omitempty"`
	Copilot *CopilotConfig `yaml:"copilot,omitempty"`
	Codex  *CodexConfig  `yaml:"codex,omitempty"`
	Cursor *CursorConfig `yaml:"cursor,omitempty"`
}

// ClaudeConfig Claude Code 特定配置
type ClaudeConfig struct {
	Globs        []string `yaml:"globs,omitempty"`
	AutoActivate bool     `yaml:"auto_activate,omitempty"`
	FileContext  bool     `yaml:"file_context,omitempty"`
}

// CopilotConfig GitHub Copilot 特定配置
type CopilotConfig struct {
	Globs        []string `yaml:"globs,omitempty"`
	AutoActivate bool     `yaml:"auto_activate,omitempty"`
	FileContext  bool     `yaml:"file_context,omitempty"`
}

// CodexConfig Codex 特定配置
type CodexConfig struct {
	Globs        []string `yaml:"globs,omitempty"`
	AutoActivate bool     `yaml:"auto_activate,omitempty"`
	Tools        []string `yaml:"tools,omitempty"`
}

// CursorConfig Cursor 特定配置
type CursorConfig struct {
	Globs       []string `yaml:"globs,omitempty"`
	AlwaysApply bool     `yaml:"always_apply,omitempty"`
}

// Manifest 技能元数据
type Manifest struct {
	Name          string                 `yaml:"name"`
	Version       string                 `yaml:"version"`
	Description   string                 `yaml:"description"`
	Namespace     string                 `yaml:"namespace,omitempty"`
	DescriptionZh string                 `yaml:"description_zh,omitempty"`
	Author        string                 `yaml:"author,omitempty"`
	Tags          []string               `yaml:"tags,omitempty"`
	License       string                 `yaml:"license,omitempty"`
	Homepage      string                 `yaml:"homepage,omitempty"`
	Repository    string                 `yaml:"repository,omitempty"`
	Requires      []string               `yaml:"requires,omitempty"`
	IDEConfig     IDEConfig              `yaml:"ide_config,omitempty"`
	Permissions   []string               `yaml:"permissions,omitempty"`
	Engines       map[string]string      `yaml:"engines,omitempty"`
}

// Skill 技能对象
type Skill struct {
	Manifest    Manifest
	Body        string
	Path        string
	References  []string
	Scripts     []string
}

// Parse 从路径解析技能
func Parse(path string) (*Skill, error) {
	skillFile := filepath.Join(path, "SKILL.md")
	content, err := os.ReadFile(skillFile)
	if err != nil {
		return nil, fmt.Errorf("读取 SKILL.md 失败: %w", err)
	}

	skill := &Skill{
		Path: path,
	}

	// 解析 frontmatter 和正文
	frontmatter, body, err := ParseFrontmatter(string(content))
	if err != nil {
		return nil, err
	}

	// 解析 YAML
	if err := yaml.Unmarshal([]byte(frontmatter), &skill.Manifest); err != nil {
		return nil, fmt.Errorf("解析 YAML 失败: %w", err)
	}

	skill.Body = body

	// 扫描 references 和 scripts
	skill.References = scanDir(filepath.Join(path, "references"))
	skill.Scripts = scanDir(filepath.Join(path, "scripts"))

	return skill, nil
}

// ParseFrontmatter 解析 frontmatter
func ParseFrontmatter(content string) (string, string, error) {
	content = strings.TrimSpace(content)

	if !strings.HasPrefix(content, "---") {
		return "", "", fmt.Errorf("SKILL.md 必须以 --- 开头")
	}

	// 找到第二个 ---
	contentWithoutFirst := content[3:]
	endIdx := strings.Index(contentWithoutFirst, "---")
	if endIdx == -1 {
		return "", "", fmt.Errorf("未找到 frontmatter 结束标记 ---")
	}

	frontmatter := strings.TrimSpace(contentWithoutFirst[:endIdx])
	body := strings.TrimSpace(contentWithoutFirst[endIdx+3:])

	return frontmatter, body, nil
}

// scanDir 扫描目录中的文件
func scanDir(dir string) []string {
	files := []string{}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return files
	}

	for _, entry := range entries {
		if !entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			files = append(files, entry.Name())
		}
	}

	return files
}

// GetFullName 获取完整技能名称
func (s *Skill) GetFullName() string {
	ns := s.Manifest.Namespace
	if ns == "" {
		ns = "@user"
	}
	return fmt.Sprintf("%s/%s", ns, s.Manifest.Name)
}

// ToCursorMdc 转换为 Cursor .mdc 格式
func (s *Skill) ToCursorMdc() string {
	// 提取 globs
	globs := "**/*"
	if s.Manifest.IDEConfig.Cursor != nil && len(s.Manifest.IDEConfig.Cursor.Globs) > 0 {
		globs = strings.Join(s.Manifest.IDEConfig.Cursor.Globs, ", ")
	}

	return fmt.Sprintf(`---
title: %s
description: %s
globs: %s
---

%s`, s.Manifest.Name, s.Manifest.Description, globs, s.Body)
}

// SaveAsCursorMdc 保存为 .mdc 文件
func (s *Skill) SaveAsCursorMdc(outputPath string) error {
	content := s.ToCursorMdc()
	return os.WriteFile(outputPath, []byte(content), 0644)
}
