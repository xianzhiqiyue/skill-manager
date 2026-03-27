package ide

import (
	"fmt"
	"os"
	"path/filepath"
)

// OpenClawAdapter OpenClaw 适配器
type OpenClawAdapter struct {
	targetPath string
}

// NewOpenClawAdapter 创建 OpenClaw 适配器
func NewOpenClawAdapter(targetPath string) *OpenClawAdapter {
	return &OpenClawAdapter{
		targetPath: targetPath,
	}
}

// GetType 返回 IDE 类型
func (a *OpenClawAdapter) GetType() string {
	return "openclaw"
}

// GetTargetPath 返回技能的目标路径
func (a *OpenClawAdapter) GetTargetPath(skillName string) string {
	return filepath.Join(a.targetPath, skillName)
}

// InstallSkill 安装技能到 OpenClaw
func (a *OpenClawAdapter) InstallSkill(data SkillData) error {
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

	refDir := filepath.Join(skillPath, "references")
	if err := ensureDir(refDir); err != nil {
		return fmt.Errorf("创建 references 目录失败: %w", err)
	}
	if len(data.References) > 0 {
		for name, content := range data.References {
			refPath := filepath.Join(refDir, name)
			if err := writeFile(refPath, content); err != nil {
				return fmt.Errorf("写入 reference 失败: %w", err)
			}
		}
	}

	scriptDir := filepath.Join(skillPath, "scripts")
	if err := ensureDir(scriptDir); err != nil {
		return fmt.Errorf("创建 scripts 目录失败: %w", err)
	}
	if len(data.Scripts) > 0 {
		for name, content := range data.Scripts {
			scriptPath := filepath.Join(scriptDir, name)
			if err := writeExecutableFile(scriptPath, content); err != nil {
				return fmt.Errorf("写入 script 失败: %w", err)
			}
		}
	}

	return nil
}

// UninstallSkill 从 OpenClaw 卸载技能
func (a *OpenClawAdapter) UninstallSkill(skillName string) error {
	return os.RemoveAll(a.GetTargetPath(skillName))
}

// ListSkills 列出已安装的技能
func (a *OpenClawAdapter) ListSkills() ([]string, error) {
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
func (a *OpenClawAdapter) SupportsSymlink() bool {
	return true
}

func writeExecutableFile(path string, content []byte) error {
	dir := filepath.Dir(path)
	if err := ensureDir(dir); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0755)
}
