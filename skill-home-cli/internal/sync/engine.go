package sync

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/skill-home/cli/internal/ide"
	"github.com/skill-home/cli/internal/skill"
)

// SyncMode 同步模式
type SyncMode string

const (
	ModeAuto    SyncMode = "auto"    // 自动检测
	ModeSymlink SyncMode = "symlink" // 符号链接
	ModeMirror  SyncMode = "mirror"  // 物理镜像
)

// Engine 同步引擎
type Engine struct {
	mode SyncMode
}

// NewEngine 创建同步引擎
func NewEngine(mode SyncMode) *Engine {
	return &Engine{
		mode: mode,
	}
}

// Sync 同步技能到 IDE
func (e *Engine) Sync(s *skill.Skill, adapter ide.Adapter) error {
	// 确定同步模式
	mode := e.resolveMode(adapter)

	// 转换技能数据
	data := e.convertSkill(s, adapter.GetType())

	// 执行同步
	switch mode {
	case ModeSymlink:
		return e.syncSymlink(s, adapter)
	case ModeMirror:
		return e.syncMirror(data, adapter)
	}

	return fmt.Errorf("未知的同步模式: %s", mode)
}

// resolveMode 确定同步模式
func (e *Engine) resolveMode(adapter ide.Adapter) SyncMode {
	if e.mode != ModeAuto {
		return e.mode
	}

	// 检测目标 IDE 是否支持符号链接
	if adapter.SupportsSymlink() {
		return ModeSymlink
	}
	return ModeMirror
}

// convertSkill 转换技能为 IDE 特定格式
func (e *Engine) convertSkill(s *skill.Skill, ideType string) ide.SkillData {
	switch ideType {
	case "claude":
		return ide.ConvertToClaudeFormat(s)
	case "cursor":
		return ide.ConvertToCursorFormat(s)
	case "codex":
		return ide.ConvertToCodexFormat(s)
	default:
		return ide.ConvertToClaudeFormat(s)
	}
}

// syncSymlink 使用符号链接同步
func (e *Engine) syncSymlink(s *skill.Skill, adapter ide.Adapter) error {
	targetPath := adapter.GetTargetPath(s.Manifest.Name)

	// 如果目标已存在，先删除
	if info, err := os.Lstat(targetPath); err == nil {
		// 如果是符号链接，直接删除
		if info.Mode()&os.ModeSymlink != 0 {
			os.Remove(targetPath)
		} else if info.IsDir() {
			// 如果是目录，删除整个目录
			os.RemoveAll(targetPath)
		} else {
			// 如果是普通文件，删除
			os.Remove(targetPath)
		}
	}

	// 确保目标目录存在
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	// 创建符号链接
	// 注意：Windows 需要管理员权限才能创建符号链接
	// 这里使用相对路径
	relSrc, err := filepath.Rel(targetDir, s.Path)
	if err != nil {
		// 如果无法获取相对路径，使用绝对路径
		relSrc = s.Path
	}

	if err := os.Symlink(relSrc, targetPath); err != nil {
		// 符号链接失败，降级为物理镜像
		return e.syncMirror(e.convertSkill(s, adapter.GetType()), adapter)
	}

	return nil
}

// syncMirror 使用物理镜像同步
func (e *Engine) syncMirror(data ide.SkillData, adapter ide.Adapter) error {
	return adapter.InstallSkill(data)
}

// detectSymlinkSupport 检测系统是否支持符号链接
func detectSymlinkSupport() bool {
	// 创建临时文件测试
	tmpDir := os.TempDir()
	testLink := filepath.Join(tmpDir, "symlink_test")
	testTarget := filepath.Join(tmpDir, "symlink_target")

	// 创建测试目标
	os.WriteFile(testTarget, []byte("test"), 0644)
	defer os.Remove(testTarget)

	// 尝试创建符号链接
	err := os.Symlink(testTarget, testLink)
	if err == nil {
		os.Remove(testLink)
		return true
	}

	return false
}

// SymlinkSupported 是否支持符号链接
var SymlinkSupported = detectSymlinkSupport()
