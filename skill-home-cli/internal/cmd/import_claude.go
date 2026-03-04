package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/skill-home/cli/internal/import/types"
)

// ClaudeImporter Claude Code 技能导入器
type ClaudeImporter struct {
	sourcePath string
	skillName  string
}

// NewClaudeImporter 创建 Claude 导入器
func NewClaudeImporter(sourceURL string) (*ClaudeImporter, error) {
	// 解析 claude:// 协议
	path := strings.TrimPrefix(sourceURL, "claude://")
	path = expandPath(path)

	// 获取技能名称
	skillName := filepath.Base(path)

	return &ClaudeImporter{
		sourcePath: path,
		skillName:  skillName,
	}, nil
}

// GetSkillInfo 获取技能信息
func (c *ClaudeImporter) GetSkillInfo() (*types.SkillInfo, error) {
	info := &types.SkillInfo{
		Name:   c.skillName,
		Source: "claude",
		URL:    c.sourcePath,
		Notes: []string{
			"从 Claude Code 技能目录导入",
			fmt.Sprintf("源路径: %s", c.sourcePath),
		},
	}

	// 尝试读取 SKILL.md 获取更多信息
	skillFile := filepath.Join(c.sourcePath, "SKILL.md")
	if content, err := os.ReadFile(skillFile); err == nil {
		// 简单解析 frontmatter
		contentStr := string(content)
		if idx := strings.Index(contentStr, "description:"); idx > 0 {
			line := contentStr[idx:]
			if endIdx := strings.Index(line, "\n"); endIdx > 0 {
				line = line[:endIdx]
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					info.Description = strings.TrimSpace(parts[1])
				}
			}
		}
	}

	return info, nil
}

// Download 下载技能（本地复制）
func (c *ClaudeImporter) Download(destPath string) error {
	return copyDir(c.sourcePath, destPath)
}

// ConvertToSkill 转换为通用技能格式
func (c *ClaudeImporter) ConvertToSkill(sourcePath string) (*types.Skill, error) {
	skill := &types.Skill{
		Name:       c.skillName,
		Version:    "0.1.0",
		License:    "MIT",
		References: make(map[string]string),
		Scripts:    make(map[string]string),
	}

	// 读取 SKILL.md
	skillMDPath := filepath.Join(sourcePath, "SKILL.md")
	content, err := os.ReadFile(skillMDPath)
	if err != nil {
		return nil, fmt.Errorf("读取 SKILL.md 失败: %w", err)
	}

	skill.Content = string(content)

	// 解析元数据
	c.parseFrontmatter(skill, skill.Content)

	// 读取 references
	refsDir := filepath.Join(sourcePath, "references")
	if entries, err := os.ReadDir(refsDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				content, err := os.ReadFile(filepath.Join(refsDir, entry.Name()))
				if err == nil {
					skill.References[entry.Name()] = string(content)
				}
			}
		}
	}

	// 读取 scripts
	scriptsDir := filepath.Join(sourcePath, "scripts")
	if entries, err := os.ReadDir(scriptsDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				content, err := os.ReadFile(filepath.Join(scriptsDir, entry.Name()))
				if err == nil {
					skill.Scripts[entry.Name()] = string(content)
				}
			}
		}
	}

	// 添加来源标记
	skill.Content = c.addSourceHeader(skill.Content)

	return skill, nil
}

// parseFrontmatter 解析 frontmatter
func (c *ClaudeImporter) parseFrontmatter(skill *types.Skill, content string) {
	lines := strings.Split(content, "\n")
	inFrontmatter := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				continue
			} else {
				break
			}
		}

		if inFrontmatter {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				switch key {
				case "name":
					skill.Name = value
				case "version":
					skill.Version = value
				case "description":
					skill.Description = value
				case "author":
					skill.Author = value
				case "license":
					skill.License = value
				}
			}
		}
	}
}

// addSourceHeader 添加来源标记
func (c *ClaudeImporter) addSourceHeader(content string) string {
	header := fmt.Sprintf("<!--\n  Source: Claude Code\n  Original Path: %s\n  Imported: %s\n-->\n\n",
		c.sourcePath, getCurrentDate())
	return header + content
}

// copyDir 复制目录
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 计算相对路径
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// 复制文件
		return copyFile(path, dstPath)
	})
}

// copyFile 复制文件
func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	return os.WriteFile(dst, input, 0644)
}

// getCurrentDate 获取当前日期
func getCurrentDate() string {
	return time.Now().Format("2006-01-02")
}
