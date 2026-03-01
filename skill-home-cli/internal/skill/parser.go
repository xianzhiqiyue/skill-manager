package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

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
	IDEConfig     map[string]interface{} `yaml:"ide_config,omitempty"`
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
	if ideConfig, ok := s.Manifest.IDEConfig["cursor"].(map[string]interface{}); ok {
		if g, ok := ideConfig["globs"].([]interface{}); ok && len(g) > 0 {
			globsList := make([]string, len(g))
			for i, v := range g {
				globsList[i] = fmt.Sprintf("%v", v)
			}
			globs = strings.Join(globsList, ", ")
		}
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
