package ide

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/skill-home/cli/internal/skill"
	"gopkg.in/yaml.v3"
)

// XiguaAdapter Xigua Agent 适配器
type XiguaAdapter struct {
	targetPath string
}

// NewXiguaAdapter 创建 Xigua 适配器
func NewXiguaAdapter(targetPath string) *XiguaAdapter {
	return &XiguaAdapter{
		targetPath: targetPath,
	}
}

// GetType 返回 IDE 类型
func (a *XiguaAdapter) GetType() string {
	return "xigua"
}

// GetTargetPath 返回技能的目标路径
func (a *XiguaAdapter) GetTargetPath(skillName string) string {
	return filepath.Join(a.targetPath, skillName)
}

// InstallSkill 安装技能到 Xigua Agent
func (a *XiguaAdapter) InstallSkill(data SkillData) error {
	skillPath := a.GetTargetPath(data.Name)

	if err := ensureDir(skillPath); err != nil {
		return fmt.Errorf("创建技能目录失败: %w", err)
	}

	content := append(data.Manifest, '\n', '-', '-', '-', '\n', '\n')
	content = append(content, []byte(data.Body)...)
	if err := writeFile(filepath.Join(skillPath, "SKILL.md"), content); err != nil {
		return fmt.Errorf("写入 SKILL.md 失败: %w", err)
	}

	metadata := parseXiguaSkillMetadata(data)
	manifestBytes, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化 skill.json 失败: %w", err)
	}
	manifestBytes = append(manifestBytes, '\n')
	if err := writeFile(filepath.Join(skillPath, "skill.json"), manifestBytes); err != nil {
		return fmt.Errorf("写入 skill.json 失败: %w", err)
	}

	if len(data.References) > 0 {
		refDir := filepath.Join(skillPath, "references")
		for name, content := range data.References {
			if err := writeFile(filepath.Join(refDir, name), content); err != nil {
				return fmt.Errorf("写入 reference 失败: %w", err)
			}
		}
	}

	if len(data.Scripts) > 0 {
		scriptDir := filepath.Join(skillPath, "scripts")
		for name, content := range data.Scripts {
			if err := writeExecutableFile(filepath.Join(scriptDir, name), content); err != nil {
				return fmt.Errorf("写入 script 失败: %w", err)
			}
		}
	}

	return nil
}

// UninstallSkill 从 Xigua Agent 卸载技能
func (a *XiguaAdapter) UninstallSkill(skillName string) error {
	return os.RemoveAll(a.GetTargetPath(skillName))
}

// ListSkills 列出已安装的技能
func (a *XiguaAdapter) ListSkills() ([]string, error) {
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
func (a *XiguaAdapter) SupportsSymlink() bool {
	return true
}

// ConvertToXiguaFormat 将通用技能转换为 Xigua Agent 包格式
func ConvertToXiguaFormat(s *skill.Skill) SkillData {
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

	manifestYAML := fmt.Sprintf(`---
name: %s
slug: %s
version: %s
description: %s`, s.Manifest.Name, xiguaSlug(s.Manifest.Name), s.Manifest.Version, s.Manifest.Description)

	if s.Manifest.Author != "" {
		manifestYAML += fmt.Sprintf("\nauthor: %s", s.Manifest.Author)
	}
	if len(s.Manifest.Tags) > 0 {
		manifestYAML += fmt.Sprintf("\ntags: [%s]", joinQuotedStrings(s.Manifest.Tags))
	}

	return SkillData{
		Name:       s.Manifest.Name,
		Manifest:   []byte(manifestYAML),
		Body:       s.Body,
		References: refs,
		Scripts:    scripts,
	}
}

type xiguaSkillJSON struct {
	Name           string `json:"name"`
	Slug           string `json:"slug"`
	Version        string `json:"version,omitempty"`
	Description    string `json:"description,omitempty"`
	SourceType     string `json:"sourceType"`
	OriginalFormat string `json:"originalFormat"`
}

func parseXiguaSkillMetadata(data SkillData) xiguaSkillJSON {
	manifest := map[string]interface{}{}
	raw := strings.TrimSpace(string(data.Manifest))
	raw = strings.TrimPrefix(raw, "---")
	raw = strings.TrimSpace(raw)
	_ = yaml.Unmarshal([]byte(raw), &manifest)

	name := data.Name
	if value, ok := manifest["name"].(string); ok && strings.TrimSpace(value) != "" {
		name = strings.TrimSpace(value)
	}
	slug := xiguaSlug(name)
	if value, ok := manifest["slug"].(string); ok && strings.TrimSpace(value) != "" {
		slug = xiguaSlug(value)
	}

	return xiguaSkillJSON{
		Name:           name,
		Slug:           slug,
		Version:        stringValue(manifest["version"]),
		Description:    stringValue(manifest["description"]),
		SourceType:     "skill-home",
		OriginalFormat: "skill-home",
	}
}

var xiguaSlugPattern = regexp.MustCompile(`[^a-z0-9]+`)

func xiguaSlug(input string) string {
	slug := strings.Trim(strings.ToLower(input), " \t\r\n")
	slug = xiguaSlugPattern.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return "skill"
	}
	return slug
}

func stringValue(value interface{}) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return ""
	}
}

func joinQuotedStrings(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, fmt.Sprintf("%q", value))
	}
	return strings.Join(quoted, ", ")
}
