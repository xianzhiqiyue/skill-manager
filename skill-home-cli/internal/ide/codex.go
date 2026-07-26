package ide

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/skill-home/cli/internal/skill"
)

// CodexAdapter Codex IDE 适配器
type CodexAdapter struct {
	targetPath string
}

// NewCodexAdapter 创建 Codex 适配器
func NewCodexAdapter(targetPath string) *CodexAdapter {
	return &CodexAdapter{
		targetPath: targetPath,
	}
}

// GetType 返回 IDE 类型
func (a *CodexAdapter) GetType() string {
	return "codex"
}

// GetTargetPath 返回技能的目标路径
func (a *CodexAdapter) GetTargetPath(skillName string) string {
	return filepath.Join(a.targetPath, skillName)
}

// InstallSkill 安装技能到 Codex
func (a *CodexAdapter) InstallSkill(data SkillData) error {
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

	// 写入 Codex 额外文件，例如 assets/ 和 agents/openai.yaml。
	for name, content := range data.AdditionalFiles {
		if !filepath.IsLocal(name) {
			return fmt.Errorf("额外文件路径不安全: %s", name)
		}
		if err := writeFile(filepath.Join(skillPath, name), content); err != nil {
			return fmt.Errorf("写入额外文件失败: %w", err)
		}
	}

	return nil
}

// UninstallSkill 从 Codex 卸载技能
func (a *CodexAdapter) UninstallSkill(skillName string) error {
	skillPath := a.GetTargetPath(skillName)
	return os.RemoveAll(skillPath)
}

// ListSkills 列出已安装的技能
func (a *CodexAdapter) ListSkills() ([]string, error) {
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
			skillFile := filepath.Join(a.targetPath, entry.Name(), "SKILL.md")
			if _, err := os.Stat(skillFile); err == nil {
				skills = append(skills, entry.Name())
			}
		}
	}

	return skills, nil
}

// SupportsSymlink 返回是否支持符号链接
func (a *CodexAdapter) SupportsSymlink() bool {
	return true
}

// ConvertToCodexFormat 将通用技能转换为 Codex 格式
func ConvertToCodexFormat(s *skill.Skill) SkillData {
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

	additionalFiles := readCodexAdditionalFiles(s.Path)

	return SkillData{
		Name:            s.Manifest.Name,
		Manifest:        []byte(manifestYAML),
		Body:            s.Body,
		References:      refs,
		Scripts:         scripts,
		AdditionalFiles: additionalFiles,
	}
}

func readCodexAdditionalFiles(skillPath string) map[string][]byte {
	files := make(map[string][]byte)
	assetsDir := filepath.Join(skillPath, "assets")

	_ = filepath.WalkDir(assetsDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		relativePath, err := filepath.Rel(skillPath, path)
		if err != nil || !filepath.IsLocal(relativePath) {
			return nil
		}
		files[relativePath] = content
		return nil
	})

	openAIConfig := filepath.Join(skillPath, "agents", "openai.yaml")
	if content, err := os.ReadFile(openAIConfig); err == nil {
		files[filepath.Join("agents", "openai.yaml")] = content
	}

	return files
}
