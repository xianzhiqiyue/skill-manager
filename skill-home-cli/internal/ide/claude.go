package ide

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/skill-home/cli/internal/skill"
)

// ClaudeAdapter Claude IDE 适配器
type ClaudeAdapter struct {
	targetPath string
}

// NewClaudeAdapter 创建 Claude 适配器
func NewClaudeAdapter(targetPath string) *ClaudeAdapter {
	return &ClaudeAdapter{
		targetPath: targetPath,
	}
}

// GetType 返回 IDE 类型
func (a *ClaudeAdapter) GetType() string {
	return "claude"
}

// GetTargetPath 返回技能的目标路径
func (a *ClaudeAdapter) GetTargetPath(skillName string) string {
	return filepath.Join(a.targetPath, skillName)
}

// InstallSkill 安装技能到 Claude
func (a *ClaudeAdapter) InstallSkill(data SkillData) error {
	skillPath := a.GetTargetPath(data.Name)

	// 创建技能目录
	if err := ensureDir(skillPath); err != nil {
		return fmt.Errorf("创建技能目录失败: %w", err)
	}

	// 写入 SKILL.md
	skillFile := filepath.Join(skillPath, "SKILL.md")
	content := append(data.Manifest, '\n', '-', '-', '-', '\n', '\n')
	content = append(content, []byte(data.Body)...)

	if err := writeFile(skillFile, content); err != nil {
		return fmt.Errorf("写入 SKILL.md 失败: %w", err)
	}

	// 写入 references
	if len(data.References) > 0 {
		refDir := filepath.Join(skillPath, "references")
		for name, content := range data.References {
			refPath := filepath.Join(refDir, name)
			if err := writeFile(refPath, content); err != nil {
				return fmt.Errorf("写入 reference 失败: %w", err)
			}
		}
	}

	// 写入 scripts
	if len(data.Scripts) > 0 {
		scriptDir := filepath.Join(skillPath, "scripts")
		for name, content := range data.Scripts {
			scriptPath := filepath.Join(scriptDir, name)
			if err := writeFile(scriptPath, content); err != nil {
				return fmt.Errorf("写入 script 失败: %w", err)
			}
		}
	}

	return nil
}

// UninstallSkill 从 Claude 卸载技能
func (a *ClaudeAdapter) UninstallSkill(skillName string) error {
	skillPath := a.GetTargetPath(skillName)
	return os.RemoveAll(skillPath)
}

// ListSkills 列出已安装的技能
func (a *ClaudeAdapter) ListSkills() ([]string, error) {
	entries, err := os.ReadDir(a.targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	skills := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			// 验证是否是有效的技能目录
			skillFile := filepath.Join(a.targetPath, entry.Name(), "SKILL.md")
			if _, err := os.Stat(skillFile); err == nil {
				skills = append(skills, entry.Name())
			}
		}
	}

	return skills, nil
}

// SupportsSymlink 返回是否支持符号链接
func (a *ClaudeAdapter) SupportsSymlink() bool {
	return true
}

// ConvertToClaudeFormat 将通用技能转换为 Claude 格式
func ConvertToClaudeFormat(s *skill.Skill) SkillData {
	// 读取 references
	refs := make(map[string][]byte)
	refDir := filepath.Join(s.Path, "references")
	if entries, err := os.ReadDir(refDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				content, _ := os.ReadFile(filepath.Join(refDir, entry.Name()))
				refs[entry.Name()] = content
			}
		}
	}

	// 读取 scripts
	scripts := make(map[string][]byte)
	scriptDir := filepath.Join(s.Path, "scripts")
	if entries, err := os.ReadDir(scriptDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				content, _ := os.ReadFile(filepath.Join(scriptDir, entry.Name()))
				scripts[entry.Name()] = content
			}
		}
	}

	// 构建 manifest YAML
	manifestYAML := fmt.Sprintf(`---
name: %s
version: %s
description: %s`, s.Manifest.Name, s.Manifest.Version, s.Manifest.Description)

	if s.Manifest.Author != "" {
		manifestYAML += fmt.Sprintf("\nauthor: %s", s.Manifest.Author)
	}
	if len(s.Manifest.Tags) > 0 {
		manifestYAML += fmt.Sprintf("\ntags: %v", s.Manifest.Tags)
	}

	return SkillData{
		Name:       s.Manifest.Name,
		Manifest:   []byte(manifestYAML),
		Body:       s.Body,
		References: refs,
		Scripts:    scripts,
	}
}
