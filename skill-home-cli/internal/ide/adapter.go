package ide

import (
	"fmt"
	"os"
	"path/filepath"
)

// Adapter IDE 适配器接口
type Adapter interface {
	// GetType 返回 IDE 类型
	GetType() string

	// GetTargetPath 返回技能的目标路径
	GetTargetPath(skillName string) string

	// InstallSkill 安装技能到 IDE
	InstallSkill(skill SkillData) error

	// UninstallSkill 从 IDE 卸载技能
	UninstallSkill(skillName string) error

	// ListSkills 列出已安装的技能
	ListSkills() ([]string, error)

	// SupportsSymlink 返回是否支持符号链接
	SupportsSymlink() bool
}

// SkillData 技能数据
type SkillData struct {
	Name        string
	Manifest    []byte
	Body        string
	References  map[string][]byte
	Scripts     map[string][]byte
}

// NewAdapter 创建 IDE 适配器
func NewAdapter(ideType, targetPath string) (Adapter, error) {
	switch ideType {
	case "claude":
		return NewClaudeAdapter(targetPath), nil
	case "cursor":
		return NewCursorAdapter(targetPath), nil
	case "codex":
		return NewCodexAdapter(targetPath), nil
	default:
		return nil, fmt.Errorf("不支持的 IDE 类型: %s", ideType)
	}
}

// ensureDir 确保目录存在
func ensureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// writeFile 写入文件
func writeFile(path string, content []byte) error {
	dir := filepath.Dir(path)
	if err := ensureDir(dir); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0644)
}
