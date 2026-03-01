package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/skill-home/cli/internal/import/types"
)

// CodexImporter Codex 技能导入器
type CodexImporter struct {
	sourcePath string
	skillName  string
}

// NewCodexImporter 创建 Codex 导入器
func NewCodexImporter(sourceURL string) (*CodexImporter, error) {
	// 解析 codex:// 协议
	path := strings.TrimPrefix(sourceURL, "codex://")
	path = expandPath(path)

	// 获取技能名称
	skillName := filepath.Base(path)

	return &CodexImporter{
		sourcePath: path,
		skillName:  skillName,
	}, nil
}

// GetSkillInfo 获取技能信息
func (c *CodexImporter) GetSkillInfo() (*types.SkillInfo, error) {
	info := &types.SkillInfo{
		Name:   c.skillName,
		Source: "codex",
		URL:    c.sourcePath,
		Notes: []string{
			"从 Codex 技能目录导入",
			fmt.Sprintf("源路径: %s", c.sourcePath),
		},
	}

	// Codex 可能使用 agent.json 存储元数据
	agentJSONPath := filepath.Join(c.sourcePath, "agent.json")
	if data, err := os.ReadFile(agentJSONPath); err == nil {
		var agent struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Version     string `json:"version"`
		}
		if err := json.Unmarshal(data, &agent); err == nil {
			info.Name = agent.Name
			info.Description = agent.Description
			info.Version = agent.Version
		}
	}

	return info, nil
}

// Download 下载技能（本地复制）
func (c *CodexImporter) Download(destPath string) error {
	return copyDir(c.sourcePath, destPath)
}

// ConvertToSkill 转换为通用技能格式
func (c *CodexImporter) ConvertToSkill(sourcePath string) (*types.Skill, error) {
	skill := &types.Skill{
		Name:       c.skillName,
		Version:    "0.1.0",
		License:    "MIT",
		References: make(map[string]string),
		Scripts:    make(map[string]string),
	}

	// 尝试不同的源文件格式
	// 1. 首先尝试 SKILL.md
	skillMDPath := filepath.Join(sourcePath, "SKILL.md")
	if content, err := os.ReadFile(skillMDPath); err == nil {
		skill.Content = string(content)
		c.parseFrontmatter(skill, skill.Content)
	} else {
		// 2. 尝试 agent.md
		agentMDPath := filepath.Join(sourcePath, "agent.md")
		if content, err := os.ReadFile(agentMDPath); err == nil {
			skill.Content = c.convertAgentMD(string(content))
		}
	}

	// 3. 尝试从 agent.json 获取元数据
	agentJSONPath := filepath.Join(sourcePath, "agent.json")
	if data, err := os.ReadFile(agentJSONPath); err == nil {
		c.parseAgentJSON(skill, data)
	}

	// 读取 references（Codex 可能使用 knowledge/ 目录）
	knowledgeDir := filepath.Join(sourcePath, "knowledge")
	if entries, err := os.ReadDir(knowledgeDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				content, err := os.ReadFile(filepath.Join(knowledgeDir, entry.Name()))
				if err == nil {
					skill.References[entry.Name()] = string(content)
				}
			}
		}
	}

	// 也尝试 references 目录
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
func (c *CodexImporter) parseFrontmatter(skill *types.Skill, content string) {
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

// parseAgentJSON 解析 agent.json
func (c *CodexImporter) parseAgentJSON(skill *types.Skill, data []byte) {
	var agent struct {
		Name        string            `json:"name"`
		Description string            `json:"description"`
		Version     string            `json:"version"`
		Author      string            `json:"author"`
		System      string            `json:"system"`
		Tags        []string          `json:"tags"`
		Config      map[string]interface{} `json:"config"`
	}

	if err := json.Unmarshal(data, &agent); err != nil {
		return
	}

	if agent.Name != "" {
		skill.Name = agent.Name
	}
	if agent.Description != "" {
		skill.Description = agent.Description
	}
	if agent.Version != "" {
		skill.Version = agent.Version
	}
	if agent.Author != "" {
		skill.Author = agent.Author
	}
	if len(agent.Tags) > 0 {
		skill.Tags = agent.Tags
	}

	// 如果没有 SKILL.md 内容，从 system 字段生成
	if skill.Content == "" && agent.System != "" {
		skill.Content = c.generateSkillMD(agent.System)
	}
}

// convertAgentMD 转换 agent.md 格式
func (c *CodexImporter) convertAgentMD(content string) string {
	// 尝试提取 frontmatter
	re := regexp.MustCompile(`(?s)^---\n(.*?)\n---\n(.*)$`)
	matches := re.FindStringSubmatch(content)

	if len(matches) == 3 {
		// 已有 frontmatter，保持原样但添加我们的标记
		return content
	}

	// 没有 frontmatter，生成一个
	return fmt.Sprintf(`---
name: %s
version: 0.1.0
description: Imported from Codex
tags: [imported, codex]
license: MIT
---

# %s

从 Codex 导入的技能。

%s
`, c.skillName, c.skillName, content)
}

// generateSkillMD 从 system 提示词生成 SKILL.md
func (c *CodexImporter) generateSkillMD(system string) string {
	return fmt.Sprintf(`---
name: %s
version: 0.1.0
description: Imported from Codex agent configuration
tags: [imported, codex]
license: MIT
---

# %s

从 Codex 配置导入的技能。

## 系统提示词

%s
`, c.skillName, c.skillName, system)
}

// addSourceHeader 添加来源标记
func (c *CodexImporter) addSourceHeader(content string) string {
	header := fmt.Sprintf("<!--\n  Source: Codex\n  Original Path: %s\n  Imported: %s\n-->\n\n",
		c.sourcePath, getCurrentDate())
	return header + content
}
