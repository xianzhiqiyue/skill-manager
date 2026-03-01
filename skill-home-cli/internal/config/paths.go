package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PathResolver 路径解析器
type PathResolver struct {
	projectRoot string
}

// NewPathResolver 创建路径解析器
func NewPathResolver() (*PathResolver, error) {
	root, err := findProjectRoot()
	if err != nil {
		return nil, err
	}
	return &PathResolver{projectRoot: root}, nil
}

// GetProjectRoot 获取项目根目录
func (p *PathResolver) GetProjectRoot() string {
	return p.projectRoot
}

// GetIDEProjectPath 获取 IDE 项目级路径
func (p *PathResolver) GetIDEProjectPath(ideType string) (string, error) {
	switch ideType {
	case "claude":
		return filepath.Join(p.projectRoot, C.IDE.Claude.ProjectPath), nil
	case "cursor":
		return filepath.Join(p.projectRoot, C.IDE.Cursor.ProjectPath), nil
	case "codex":
		return filepath.Join(p.projectRoot, C.IDE.Codex.ProjectPath), nil
	default:
		return "", fmt.Errorf("未知的 IDE 类型: %s", ideType)
	}
}

// GetIDEGlobalPath 获取 IDE 全局路径
func (p *PathResolver) GetIDEGlobalPath(ideType string) (string, error) {
	switch ideType {
	case "claude":
		return C.IDE.Claude.GlobalPath, nil
	case "cursor":
		return "", fmt.Errorf("Cursor 不支持全局路径")
	case "codex":
		return C.IDE.Codex.GlobalPath, nil
	default:
		return "", fmt.Errorf("未知的 IDE 类型: %s", ideType)
	}
}

// EnsureDir 确保目录存在
func EnsureDir(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("创建目录失败 %s: %w", path, err)
	}
	return nil
}

// findProjectRoot 查找项目根目录
func findProjectRoot() (string, error) {
	// 从当前目录开始向上查找
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("获取当前目录失败: %w", err)
	}

	// 向上查找直到找到 .git 目录或到达根目录
	for {
		// 检查是否是 git 仓库
		gitDir := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
			return dir, nil
		}

		// 检查是否包含 package.json 等常见项目文件
		markers := []string{"package.json", "go.mod", "Cargo.toml", "pom.xml"}
		for _, marker := range markers {
			if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
				return dir, nil
			}
		}

		// 到达根目录
		parent := filepath.Dir(dir)
		if parent == dir {
			// 没有找到项目根目录，返回当前目录
			cwd, _ := os.Getwd()
			return cwd, nil
		}
		dir = parent
	}
}

// GetSkillCacheDir 获取技能缓存目录
func GetSkillCacheDir(namespace, name, version string) string {
	skillDir := filepath.Join(C.Local.SkillsDir, namespace, name, version)
	return skillDir
}

// GetSkillSourcePath 获取技能源路径
func GetSkillSourcePath(namespace, name string) string {
	return filepath.Join(C.Local.SkillsDir, namespace, name)
}

// ParseSkillRef 解析技能引用
// 格式: @namespace/name@version 或 name@version 或 name
func ParseSkillRef(ref string) (namespace, name, version string, err error) {
	// 移除前导的 @
	ref = strings.TrimPrefix(ref, "@")

	// 分离命名空间和名称
	parts := strings.Split(ref, "/")
	if len(parts) == 2 {
		namespace = parts[0]
		name = parts[1]
	} else if len(parts) == 1 {
		namespace = C.Local.DefaultNamespace
		name = parts[0]
	} else {
		return "", "", "", fmt.Errorf("无效的技能引用格式: %s", ref)
	}

	// 分离名称和版本
	nameParts := strings.Split(name, "@")
	if len(nameParts) == 2 {
		name = nameParts[0]
		version = nameParts[1]
	}

	return namespace, name, version, nil
}
