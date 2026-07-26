package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/skill-home/cli/internal/import/types"
)

// XiguaImporter Xigua Agent 技能导入器
type XiguaImporter struct {
	sourcePath string
	skillName  string
}

// NewXiguaImporter 创建 Xigua 导入器
func NewXiguaImporter(sourceURL string) (*XiguaImporter, error) {
	path := strings.TrimPrefix(sourceURL, "xigua://")
	path = expandPath(path)

	skillName := filepath.Base(path)
	return &XiguaImporter{
		sourcePath: path,
		skillName:  skillName,
	}, nil
}

// GetSkillInfo 获取技能信息
func (x *XiguaImporter) GetSkillInfo() (*types.SkillInfo, error) {
	info := &types.SkillInfo{
		Name:   x.skillName,
		Source: "xigua",
		URL:    x.sourcePath,
		Notes: []string{
			"从 Xigua Agent 技能目录导入",
			fmt.Sprintf("源路径: %s", x.sourcePath),
		},
	}

	manifestPath := filepath.Join(x.sourcePath, "skill.json")
	if data, err := os.ReadFile(manifestPath); err == nil {
		var manifest struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Version     string `json:"version"`
		}
		if err := json.Unmarshal(data, &manifest); err == nil {
			if manifest.Name != "" {
				info.Name = manifest.Name
			}
			info.Description = manifest.Description
			info.Version = manifest.Version
		}
	}

	if info.Description == "" {
		skillFile := filepath.Join(x.sourcePath, "SKILL.md")
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
	}

	return info, nil
}

// Download 下载技能（本地复制）
func (x *XiguaImporter) Download(destPath string) error {
	return copyDir(x.sourcePath, destPath)
}

// ConvertToSkill 转换为通用技能格式
func (x *XiguaImporter) ConvertToSkill(sourcePath string) (*types.Skill, error) {
	skill := &types.Skill{
		Name:       x.skillName,
		Version:    "0.1.0",
		License:    "MIT",
		References: make(map[string]string),
		Scripts:    make(map[string]string),
	}

	skillMDPath := filepath.Join(sourcePath, "SKILL.md")
	content, err := os.ReadFile(skillMDPath)
	if err != nil {
		return nil, fmt.Errorf("读取 SKILL.md 失败: %w", err)
	}
	skill.Content = string(content)
	x.parseFrontmatter(skill, skill.Content)

	manifestPath := filepath.Join(sourcePath, "skill.json")
	if data, err := os.ReadFile(manifestPath); err == nil {
		x.parseSkillJSON(skill, data)
	}

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

	skill.Content = x.addSourceHeader(skill.Content)
	return skill, nil
}

func (x *XiguaImporter) parseFrontmatter(skill *types.Skill, content string) {
	lines := strings.Split(content, "\n")
	inFrontmatter := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				continue
			}
			break
		}

		if inFrontmatter {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) != 2 {
				continue
			}
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

func (x *XiguaImporter) parseSkillJSON(skill *types.Skill, data []byte) {
	var manifest struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Version     string `json:"version"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return
	}
	if manifest.Name != "" {
		skill.Name = manifest.Name
	}
	if manifest.Description != "" {
		skill.Description = manifest.Description
	}
	if manifest.Version != "" {
		skill.Version = manifest.Version
	}
}

func (x *XiguaImporter) addSourceHeader(content string) string {
	header := fmt.Sprintf("<!--\n  Source: Xigua Agent\n  Original Path: %s\n  Imported: %s\n-->\n\n",
		x.sourcePath, getCurrentDate())
	return header + content
}
