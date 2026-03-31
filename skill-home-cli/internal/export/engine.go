package export

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/skill-home/cli/internal/skill"
)

// Result 导出结果
type Result struct {
	Platform   string
	Files      map[string][]byte
	TargetPath string
}

// Engine 导出引擎
type Engine struct {
	skillsDir string
}

// NewEngine 创建导出引擎
func NewEngine(skillsDir string) *Engine {
	return &Engine{
		skillsDir: skillsDir,
	}
}

// Export 导出技能到指定平台
func (e *Engine) Export(s *skill.Skill, platform string) (*Result, error) {
	switch platform {
	case "claude":
		return e.exportToClaude(s)
	case "copilot":
		return e.exportToCopilot(s)
	case "codex":
		return e.exportToCodex(s)
	case "cursor":
		return e.exportToCursor(s)
	default:
		return nil, fmt.Errorf("不支持的平台: %s", platform)
	}
}

func buildBaseManifest(s *skill.Skill) map[string]interface{} {
	manifest := map[string]interface{}{
		"name":        s.Manifest.Name,
		"version":     s.Manifest.Version,
		"description": s.Manifest.Description,
	}

	if s.Manifest.Namespace != "" {
		manifest["namespace"] = s.Manifest.Namespace
	}
	if s.Manifest.DescriptionZh != "" {
		manifest["description_zh"] = s.Manifest.DescriptionZh
	}
	if s.Manifest.Author != "" {
		manifest["author"] = s.Manifest.Author
	}
	if len(s.Manifest.Tags) > 0 {
		manifest["tags"] = s.Manifest.Tags
	}
	if s.Manifest.License != "" {
		manifest["license"] = s.Manifest.License
	}
	if s.Manifest.Homepage != "" {
		manifest["homepage"] = s.Manifest.Homepage
	}
	if s.Manifest.Repository != "" {
		manifest["repository"] = s.Manifest.Repository
	}
	requires := s.Manifest.Requires
	if len(requires) == 0 {
		requires = deriveRequiresFromMetadata(s.Manifest.Metadata)
	}
	if len(requires) > 0 {
		manifest["requires"] = requires
	}
	if len(s.Manifest.Permissions) > 0 {
		manifest["permissions"] = s.Manifest.Permissions
	}
	if len(s.Manifest.Engines) > 0 {
		manifest["engines"] = s.Manifest.Engines
	}
	if len(s.Manifest.Metadata) > 0 {
		manifest["metadata"] = s.Manifest.Metadata
	}

	return manifest
}

func deriveRequiresFromMetadata(metadata map[string]interface{}) []string {
	if len(metadata) == 0 {
		return nil
	}

	openclaw, ok := metadata["openclaw"].(map[string]interface{})
	if !ok {
		return nil
	}
	rawCredentials, ok := openclaw["credentials"].([]interface{})
	if !ok || len(rawCredentials) == 0 {
		return nil
	}

	requires := make([]string, 0, len(rawCredentials))
	seen := make(map[string]struct{}, len(rawCredentials))
	for _, rawCredential := range rawCredentials {
		credential, ok := rawCredential.(map[string]interface{})
		if !ok {
			continue
		}
		env, ok := credential["env"].(string)
		if !ok {
			continue
		}
		env = strings.TrimSpace(env)
		if env == "" {
			continue
		}
		if _, exists := seen[env]; exists {
			continue
		}
		seen[env] = struct{}{}
		requires = append(requires, env)
	}
	if len(requires) == 0 {
		return nil
	}
	return requires
}

// exportToCopilot 导出到 GitHub Copilot 格式
func (e *Engine) exportToCopilot(s *skill.Skill) (*Result, error) {
	result := &Result{
		Platform: "copilot",
		Files:    make(map[string][]byte),
	}

	manifest := buildBaseManifest(s)
	if s.Manifest.IDEConfig.Copilot != nil {
		cfg := s.Manifest.IDEConfig.Copilot
		if len(cfg.Globs) > 0 {
			manifest["globs"] = cfg.Globs
		}
		manifest["auto_activate"] = cfg.AutoActivate
		manifest["file_context"] = cfg.FileContext
	}

	manifestBytes, err := yaml.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("序列化 manifest 失败: %w", err)
	}

	content := fmt.Sprintf("---\n%s---\n\n%s", string(manifestBytes), s.Body)
	result.Files["SKILL.md"] = []byte(content)

	for _, ref := range s.References {
		refPath := filepath.Join(s.Path, "references", ref)
		if data, err := os.ReadFile(refPath); err == nil {
			result.Files[filepath.Join("references", ref)] = data
		}
	}

	for _, script := range s.Scripts {
		scriptPath := filepath.Join(s.Path, "scripts", script)
		if data, err := os.ReadFile(scriptPath); err == nil {
			result.Files[filepath.Join("scripts", script)] = data
		}
	}

	return result, nil
}

// exportToClaude 导出到 Claude Code 格式
func (e *Engine) exportToClaude(s *skill.Skill) (*Result, error) {
	result := &Result{
		Platform: "claude",
		Files:    make(map[string][]byte),
	}

	// 构建 manifest
	manifest := buildBaseManifest(s)

	// 添加 Claude 特定配置
	if s.Manifest.IDEConfig.Claude != nil {
		cfg := s.Manifest.IDEConfig.Claude
		if len(cfg.Globs) > 0 {
			manifest["globs"] = cfg.Globs
		}
		manifest["auto_activate"] = cfg.AutoActivate
		manifest["file_context"] = cfg.FileContext
	}

	// 序列化 manifest
	manifestBytes, err := yaml.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("序列化 manifest 失败: %w", err)
	}

	// 构建 SKILL.md 内容
	content := fmt.Sprintf("---\n%s---\n\n%s", string(manifestBytes), s.Body)
	result.Files["SKILL.md"] = []byte(content)

	// 添加 references
	for _, ref := range s.References {
		refPath := filepath.Join(s.Path, "references", ref)
		if data, err := os.ReadFile(refPath); err == nil {
			result.Files[filepath.Join("references", ref)] = data
		}
	}

	// 添加 scripts
	for _, script := range s.Scripts {
		scriptPath := filepath.Join(s.Path, "scripts", script)
		if data, err := os.ReadFile(scriptPath); err == nil {
			result.Files[filepath.Join("scripts", script)] = data
		}
	}

	return result, nil
}

// exportToCodex 导出到 Codex 格式
func (e *Engine) exportToCodex(s *skill.Skill) (*Result, error) {
	result := &Result{
		Platform: "codex",
		Files:    make(map[string][]byte),
	}

	// 构建 manifest
	manifest := map[string]interface{}{
		"name":        s.Manifest.Name,
		"version":     s.Manifest.Version,
		"description": s.Manifest.Description,
	}

	if s.Manifest.Author != "" {
		manifest["author"] = s.Manifest.Author
	}

	// 添加 Codex 特定配置
	if s.Manifest.IDEConfig.Codex != nil {
		cfg := s.Manifest.IDEConfig.Codex
		if len(cfg.Globs) > 0 {
			// Codex 使用单个 glob 字符串
			manifest["glob"] = strings.Join(cfg.Globs, ", ")
		}
		if len(cfg.Tools) > 0 {
			manifest["tools"] = cfg.Tools
		}
	}

	// 序列化 manifest
	manifestBytes, err := yaml.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("序列化 manifest 失败: %w", err)
	}

	// 构建 .mdc 文件内容
	content := fmt.Sprintf("---\n%s---\n\n%s", string(manifestBytes), s.Body)
	result.Files[s.Manifest.Name+".mdc"] = []byte(content)

	for _, ref := range s.References {
		refPath := filepath.Join(s.Path, "references", ref)
		if data, err := os.ReadFile(refPath); err == nil {
			result.Files[filepath.Join("references", ref)] = data
		}
	}

	for _, script := range s.Scripts {
		scriptPath := filepath.Join(s.Path, "scripts", script)
		if data, err := os.ReadFile(scriptPath); err == nil {
			result.Files[filepath.Join("scripts", script)] = data
		}
	}

	return result, nil
}

// exportToCursor 导出到 Cursor 格式
func (e *Engine) exportToCursor(s *skill.Skill) (*Result, error) {
	result := &Result{
		Platform: "cursor",
		Files:    make(map[string][]byte),
	}

	// 构建 frontmatter
	frontmatter := map[string]interface{}{
		"title":       s.Manifest.Name,
		"description": s.Manifest.Description,
	}

	// 添加 Cursor 特定配置
	globs := "*"
	if s.Manifest.IDEConfig.Cursor != nil {
		cfg := s.Manifest.IDEConfig.Cursor
		if len(cfg.Globs) > 0 {
			globs = strings.Join(cfg.Globs, ", ")
		}
	}
	frontmatter["globs"] = globs

	// 序列化 frontmatter
	manifestBytes, err := yaml.Marshal(frontmatter)
	if err != nil {
		return nil, fmt.Errorf("序列化 frontmatter 失败: %w", err)
	}

	// 构建 .mdc 文件内容
	content := fmt.Sprintf("---\n%s---\n\n%s", string(manifestBytes), s.Body)
	result.Files[s.Manifest.Name+".mdc"] = []byte(content)

	return result, nil
}

// Install 安装导出结果到目标路径
func (e *Engine) Install(result *Result, targetPath string) error {
	skillDir := filepath.Join(targetPath, result.Platform)

	for filePath, content := range result.Files {
		fullPath := filepath.Join(skillDir, filePath)

		// 确保目录存在
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建目录失败 %s: %w", dir, err)
		}

		// 写入文件
		if err := os.WriteFile(fullPath, content, 0644); err != nil {
			return fmt.Errorf("写入文件失败 %s: %w", fullPath, err)
		}
	}

	return nil
}

// GetDefaultTargetPath 获取平台默认目标路径
func GetDefaultTargetPath(platform string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	switch platform {
	case "claude":
		return filepath.Join(home, ".claude", "skills"), nil
	case "copilot":
		return filepath.Join(home, ".copilot", "skills"), nil
	case "codex":
		return filepath.Join(".codex", "agents"), nil
	case "cursor":
		return filepath.Join(".cursor", "rules"), nil
	default:
		return "", fmt.Errorf("不支持的平台: %s", platform)
	}
}
