package ide

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/skill-home/cli/internal/skill"
)

// CursorAdapter Cursor IDE 适配器
type CursorAdapter struct {
	targetPath string
}

// NewCursorAdapter 创建 Cursor 适配器
func NewCursorAdapter(targetPath string) *CursorAdapter {
	return &CursorAdapter{
		targetPath: targetPath,
	}
}

// GetType 返回 IDE 类型
func (a *CursorAdapter) GetType() string {
	return "cursor"
}

// GetTargetPath 返回技能的目标路径
func (a *CursorAdapter) GetTargetPath(skillName string) string {
	// Cursor 使用 .mdc 格式，文件名为 {skill-name}.mdc
	return filepath.Join(a.targetPath, fmt.Sprintf("%s.mdc", skillName))
}

// InstallSkill 安装技能到 Cursor
func (a *CursorAdapter) InstallSkill(data SkillData) error {
	targetFile := a.GetTargetPath(data.Name)

	// 确保目录存在
	if err := ensureDir(filepath.Dir(targetFile)); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 将技能内容转换为 .mdc 格式
	content := a.convertToMdc(data)

	// 写入文件
	if err := os.WriteFile(targetFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("写入 .mdc 文件失败: %w", err)
	}

	return nil
}

// UninstallSkill 从 Cursor 卸载技能
func (a *CursorAdapter) UninstallSkill(skillName string) error {
	targetFile := a.GetTargetPath(skillName)
	return os.Remove(targetFile)
}

// ListSkills 列出已安装的技能
func (a *CursorAdapter) ListSkills() ([]string, error) {
	entries, err := os.ReadDir(a.targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	skills := []string{}
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".mdc" {
			skillName := entry.Name()[:len(entry.Name())-4] // 移除 .mdc 后缀
			skills = append(skills, skillName)
		}
	}

	return skills, nil
}

// SupportsSymlink 返回是否支持符号链接
func (a *CursorAdapter) SupportsSymlink() bool {
	// Cursor 使用 .mdc 单文件格式，不支持符号链接
	return false
}

// convertToMdc 将技能数据转换为 .mdc 格式
func (a *CursorAdapter) convertToMdc(data SkillData) string {
	// 提取 globs（如果有）
	globs := "**/*"

	return fmt.Sprintf(`---
title: %s
description: %s
globs: %s
---

%s`, data.Name, extractDescription(data), globs, data.Body)
}

// extractDescription 从 manifest 提取描述
func extractDescription(data SkillData) string {
	// 简单解析，实际应该解析 YAML
	// 这里简化处理，假设 manifest 中有 description 字段
	return "AI Skill"
}

// ConvertToCursorFormat 将通用技能转换为 Cursor 格式
func ConvertToCursorFormat(s *skill.Skill) SkillData {
	return SkillData{
		Name:     s.Manifest.Name,
		Manifest: []byte{}, // Cursor 不需要单独的 manifest 文件
		Body:     s.Body,
	}
}
