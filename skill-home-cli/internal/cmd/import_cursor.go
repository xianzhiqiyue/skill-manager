package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/skill-home/cli/internal/import/types"
)

// CursorImporter Cursor Rules 导入器
type CursorImporter struct {
	sourcePath string
	skillName  string
}

// NewCursorImporter 创建 Cursor 导入器
func NewCursorImporter(sourceURL string) (*CursorImporter, error) {
	// 解析 cursor:// 协议
	path := strings.TrimPrefix(sourceURL, "cursor://")
	path = expandPath(path)

	// 获取技能名称（移除 .mdc 后缀）
	skillName := filepath.Base(path)
	skillName = strings.TrimSuffix(skillName, ".mdc")
	skillName = strings.TrimSuffix(skillName, ".md")

	return &CursorImporter{
		sourcePath: path,
		skillName:  skillName,
	}, nil
}

// GetSkillInfo 获取技能信息
func (c *CursorImporter) GetSkillInfo() (*types.SkillInfo, error) {
	info := &types.SkillInfo{
		Name:   c.skillName,
		Source: "cursor",
		URL:    c.sourcePath,
		Notes: []string{
			"从 Cursor Rules 导入",
			fmt.Sprintf("源路径: %s", c.sourcePath),
			"Cursor .mdc 格式将转换为通用 SKILL.md",
		},
	}

	// 尝试读取文件获取更多信息
	if content, err := os.ReadFile(c.sourcePath); err == nil {
		contentStr := string(content)

		// 尝试从 frontmatter 提取描述
		if desc := extractFrontmatterField(contentStr, "description"); desc != "" {
			info.Description = desc
		}

		// 尝试提取 globs
		if globs := extractFrontmatterField(contentStr, "globs"); globs != "" {
			info.Notes = append(info.Notes, fmt.Sprintf("适用文件: %s", globs))
		}
	}

	return info, nil
}

// Download 下载技能（本地复制）
func (c *CursorImporter) Download(destPath string) error {
	// 检查源路径是文件还是目录
	info, err := os.Stat(c.sourcePath)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return copyDir(c.sourcePath, destPath)
	}

	// 单文件，创建目录并复制
	if err := os.MkdirAll(destPath, 0755); err != nil {
		return err
	}

	return copyFile(c.sourcePath, filepath.Join(destPath, info.Name()))
}

// ConvertToSkill 转换为通用技能格式
func (c *CursorImporter) ConvertToSkill(sourcePath string) (*types.Skill, error) {
	skill := &types.Skill{
		Name:       c.skillName,
		Version:    "0.1.0",
		License:    "MIT",
		References: make(map[string]string),
		Scripts:    make(map[string]string),
	}

	// 查找 .mdc 文件
	mdcFiles := findMdcFiles(sourcePath)

	if len(mdcFiles) == 0 {
		return nil, fmt.Errorf("未找到 .mdc 文件")
	}

	// 读取并转换第一个 .mdc 文件
	mdcContent, err := os.ReadFile(mdcFiles[0])
	if err != nil {
		return nil, fmt.Errorf("读取 .mdc 文件失败: %w", err)
	}

	skill.Content = c.convertMdcToSkill(string(mdcContent))
	c.parseFrontmatter(skill, skill.Content)

	// 如果有多个 .mdc 文件，将额外的作为 references
	if len(mdcFiles) > 1 {
		for i, file := range mdcFiles[1:] {
			content, err := os.ReadFile(file)
			if err == nil {
				name := fmt.Sprintf("cursor-rule-%d.md", i+1)
				skill.References[name] = string(content)
			}
		}
	}

	// 读取其他 references
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

	// 添加转换说明
	skill.Content = c.addConversionNote(skill.Content)

	return skill, nil
}

// convertMdcToSkill 将 Cursor .mdc 转换为 SKILL.md
func (c *CursorImporter) convertMdcToSkill(mdcContent string) string {
	// 解析 .mdc frontmatter
	title := extractFrontmatterField(mdcContent, "title")
	description := extractFrontmatterField(mdcContent, "description")
	globs := extractFrontmatterField(mdcContent, "globs")
	alwaysApply := extractFrontmatterField(mdcContent, "alwaysApply")

	// 提取正文（frontmatter 之后的内容）
	body := extractBody(mdcContent)

	// 构建 ide_config
	ideConfig := ""
	if globs != "" || alwaysApply != "" {
		ideConfig = "ide_config:\n  cursor:\n"
		if globs != "" {
			// 将逗号分隔的 globs 转换为 YAML 数组
			globsList := parseGlobs(globs)
			ideConfig += "    globs:\n"
			for _, g := range globsList {
				ideConfig += fmt.Sprintf("      - \"%s\"\n", g)
			}
		}
		if alwaysApply != "" {
			ideConfig += fmt.Sprintf("    always_apply: %s\n", alwaysApply)
		}
	}

	// 生成 SKILL.md
	name := c.skillName
	if title != "" {
		name = title
	}

	skillMD := fmt.Sprintf(`---
name: %s
version: 0.1.0
description: %s
tags: [imported, cursor]
license: MIT
`, name, description)

	if ideConfig != "" {
		skillMD += ideConfig
	}

	skillMD += fmt.Sprintf(`---

# %s

%s
`, name, body)

	return skillMD
}

// parseFrontmatter 解析 frontmatter
func (c *CursorImporter) parseFrontmatter(skill *types.Skill, content string) {
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
				}
			}
		}
	}
}

// addConversionNote 添加转换说明
func (c *CursorImporter) addConversionNote(content string) string {
	note := fmt.Sprintf(`<!--
  Source: Cursor
  Original Path: %s
  Converted: %s
  Note: This skill was converted from Cursor .mdc format to SKILL.md
-->

`, c.sourcePath, getCurrentDate())
	return note + content
}

// findMdcFiles 查找所有 .mdc 文件
func findMdcFiles(dir string) []string {
	var files []string

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".mdc") {
			files = append(files, path)
		}
		return nil
	})

	return files
}

// extractFrontmatterField 从 frontmatter 提取字段
func extractFrontmatterField(content, field string) string {
	pattern := fmt.Sprintf(`(?m)^%s:\s*(.+?)$`, regexp.QuoteMeta(field))
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(content)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// extractBody 提取正文内容
func extractBody(content string) string {
	re := regexp.MustCompile(`(?s)^---\n.*?\n---\n(.*)$`)
	matches := re.FindStringSubmatch(content)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return strings.TrimSpace(content)
}

// parseGlobs 解析 globs 字符串
func parseGlobs(globs string) []string {
	// 处理逗号分隔和数组格式
	globs = strings.Trim(globs, "[]")
	parts := strings.Split(globs, ",")
	var result []string
	for _, p := range parts {
		g := strings.TrimSpace(p)
		g = strings.Trim(g, `"'`)
		if g != "" {
			result = append(result, g)
		}
	}
	return result
}
