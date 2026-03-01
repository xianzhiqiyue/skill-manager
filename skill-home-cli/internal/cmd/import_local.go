package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/skill-home/cli/internal/import/types"
)

// LocalImporter 本地技能导入器
type LocalImporter struct {
	sourcePath string
	skillName  string
}

// NewLocalImporter 创建本地导入器
func NewLocalImporter(sourceURL string) (*LocalImporter, error) {
	path := expandPath(sourceURL)

	// 验证路径存在
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("路径不存在: %s", path)
	}

	// 获取技能名称
	skillName := filepath.Base(path)

	return &LocalImporter{
		sourcePath: path,
		skillName:  skillName,
	}, nil
}

// GetSkillInfo 获取技能信息
func (l *LocalImporter) GetSkillInfo() (*types.SkillInfo, error) {
	info := &types.SkillInfo{
		Name:   l.skillName,
		Source: "local",
		URL:    l.sourcePath,
		Notes: []string{
			"从本地目录导入",
			fmt.Sprintf("源路径: %s", l.sourcePath),
		},
	}

	// 尝试读取 SKILL.md 获取更多信息
	skillFile := filepath.Join(l.sourcePath, "SKILL.md")
	if content, err := os.ReadFile(skillFile); err == nil {
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
func (l *LocalImporter) Download(destPath string) error {
	return copyDir(l.sourcePath, destPath)
}

// ConvertToSkill 转换为通用技能格式
func (l *LocalImporter) ConvertToSkill(sourcePath string) (*types.Skill, error) {
	skill := &types.Skill{
		Name:       l.skillName,
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
	l.parseFrontmatter(skill, skill.Content)

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
	skill.Content = l.addSourceHeader(skill.Content)

	return skill, nil
}

// parseFrontmatter 解析 frontmatter
func (l *LocalImporter) parseFrontmatter(skill *types.Skill, content string) {
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
func (l *LocalImporter) addSourceHeader(content string) string {
	header := fmt.Sprintf("<!--\n  Source: Local\n  Original Path: %s\n  Imported: %s\n-->\n\n",
		l.sourcePath, getCurrentDate())
	return header + content
}
