package ide

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/skill-home/cli/internal/skill"
)

// CopilotAdapter GitHub Copilot IDE 适配器
type CopilotAdapter struct {
	targetPath string
}

// NewCopilotAdapter 创建 Copilot 适配器
func NewCopilotAdapter(targetPath string) *CopilotAdapter {
	return &CopilotAdapter{
		targetPath: targetPath,
	}
}

// GetType 返回 IDE 类型
func (a *CopilotAdapter) GetType() string {
	return "copilot"
}

// GetTargetPath 返回技能的目标路径
func (a *CopilotAdapter) GetTargetPath(skillName string) string {
	return filepath.Join(a.targetPath, skillName)
}

// InstallSkill 安装技能到 GitHub Copilot
func (a *CopilotAdapter) InstallSkill(data SkillData) error {
	skillPath := a.GetTargetPath(data.Name)

	if err := ensureDir(skillPath); err != nil {
		return fmt.Errorf("创建技能目录失败: %w", err)
	}

	skillFile := filepath.Join(skillPath, "SKILL.md")
	content := append(data.Manifest, '\n', '-', '-', '-', '\n', '\n')
	content = append(content, []byte(data.Body)...)

	if err := writeFile(skillFile, content); err != nil {
		return fmt.Errorf("写入 SKILL.md 失败: %w", err)
	}

	if len(data.References) > 0 {
		refDir := filepath.Join(skillPath, "references")
		for name, content := range data.References {
			refPath := filepath.Join(refDir, name)
			if err := writeFile(refPath, content); err != nil {
				return fmt.Errorf("写入 reference 失败: %w", err)
			}
		}
	}

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

// UninstallSkill 从 GitHub Copilot 卸载技能
func (a *CopilotAdapter) UninstallSkill(skillName string) error {
	return os.RemoveAll(a.GetTargetPath(skillName))
}

// ListSkills 列出已安装的技能
func (a *CopilotAdapter) ListSkills() ([]string, error) {
	entries, err := os.ReadDir(a.targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	skills := []string{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillFile := filepath.Join(a.targetPath, entry.Name(), "SKILL.md")
		if _, err := os.Stat(skillFile); err == nil {
			skills = append(skills, entry.Name())
		}
	}

	return skills, nil
}

// SupportsSymlink 返回是否支持符号链接
func (a *CopilotAdapter) SupportsSymlink() bool {
	return true
}

// ConvertToCopilotFormat 将通用技能转换为 Copilot 格式
func ConvertToCopilotFormat(s *skill.Skill) SkillData {
	refs := make(map[string][]byte)
	refDir := filepath.Join(s.Path, "references")
	if entries, err := os.ReadDir(refDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			content, _ := os.ReadFile(filepath.Join(refDir, entry.Name()))
			refs[entry.Name()] = content
		}
	}

	scripts := make(map[string][]byte)
	scriptDir := filepath.Join(s.Path, "scripts")
	if entries, err := os.ReadDir(scriptDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			content, _ := os.ReadFile(filepath.Join(scriptDir, entry.Name()))
			scripts[entry.Name()] = content
		}
	}

	manifestYAML := fmt.Sprintf(`---
name: %s
version: %s
description: %s`, s.Manifest.Name, s.Manifest.Version, s.Manifest.Description)

	if s.Manifest.Author != "" {
		manifestYAML += fmt.Sprintf("\nauthor: %s", s.Manifest.Author)
	}

	if s.Manifest.IDEConfig.Copilot != nil && len(s.Manifest.IDEConfig.Copilot.Globs) > 0 {
		manifestYAML += fmt.Sprintf("\nglobs: [%s]", joinQuoted(s.Manifest.IDEConfig.Copilot.Globs))
	}

	return SkillData{
		Name:       s.Manifest.Name,
		Manifest:   []byte(manifestYAML),
		Body:       s.Body,
		References: refs,
		Scripts:    scripts,
	}
}

func joinQuoted(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, fmt.Sprintf("%q", value))
	}
	return strings.Join(quoted, ", ")
}
