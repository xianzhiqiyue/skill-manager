package types

// SkillInfo 技能信息
type SkillInfo struct {
	Name        string
	Description string
	Author      string
	Version     string
	Source      string
	URL         string
	Warnings    []string
	Notes       []string
}

// Skill 转换后的技能
type Skill struct {
	Name        string
	Description string
	Version     string
	Author      string
	Tags        []string
	License     string
	Content     string // SKILL.md 完整内容
	References  map[string]string
	Scripts     map[string]string
}

// SkillImporter 技能导入器接口
type SkillImporter interface {
	// GetSkillInfo 获取技能元信息（不下载）
	GetSkillInfo() (*SkillInfo, error)

	// Download 下载技能到指定目录
	Download(destPath string) error

	// ConvertToSkill 将下载的内容转换为通用 Skill 格式
	ConvertToSkill(sourcePath string) (*Skill, error)
}

// ImportConfig 导入配置
type ImportConfig struct {
	SourceURL   string
	SourceType  string
	OutputDir   string
	Force       bool
	ConvertOnly bool
}

// ConversionResult 转换结果
type ConversionResult struct {
	Skill    *Skill
	Warnings []string
	Notes    []string
}
