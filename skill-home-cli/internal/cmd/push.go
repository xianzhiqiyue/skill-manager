package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"

	"github.com/skill-home/cli/internal/config"
	"github.com/skill-home/cli/internal/registry"
	"github.com/skill-home/cli/internal/skill"
	"github.com/skill-home/cli/internal/taxonomy"
)

type pushOptions struct {
	namespace string
	version   string
	force     bool
	message   string
}

var (
	pushTerminalChecker = func() bool {
		return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
	}
	pushMetadataPrompter = promptPublishMetadata
)

func newPushCmd() *cobra.Command {
	opts := &pushOptions{}

	cmd := &cobra.Command{
		Use:   "push [path]",
		Short: "推送技能到注册中心",
		Long:  "将本地技能打包并推送到 skill-home 注册中心",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) > 0 {
				path = args[0]
			}
			return runPush(path, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.namespace, "namespace", "n", "", "命名空间 (默认使用配置中的 default_namespace)")
	cmd.Flags().StringVar(&opts.version, "version", "", "指定版本号 (默认使用 SKILL.md 中的 version)")
	cmd.Flags().BoolVarP(&opts.force, "force", "f", false, "强制推送，忽略安全警告")
	cmd.Flags().StringVarP(&opts.message, "message", "m", "", "版本说明")

	return cmd
}

func runPush(path string, opts *pushOptions) error {
	if err := requireRegistryLogin(); err != nil {
		return err
	}

	// 解析技能
	s, err := skill.Parse(path)
	if err != nil {
		return fmt.Errorf("解析技能失败: %w", err)
	}
	if err := ensurePublishMetadata(path, s); err != nil {
		return err
	}
	category, tags, err := validatePublishMetadata(s)
	if err != nil {
		return err
	}

	fmt.Printf("推送技能: %s\n", color.CyanString(s.GetFullName()))
	version := strings.TrimSpace(s.Manifest.Version)
	if opts.version != "" {
		version = strings.TrimSpace(opts.version)
	}
	if version == "" {
		return fmt.Errorf("版本号不能为空，请在 SKILL.md 中设置 version 或使用 --version")
	}
	fmt.Printf("版本: %s\n", color.CyanString(version))
	fmt.Println()

	// 确定命名空间
	namespace := opts.namespace
	if namespace == "" {
		namespace = s.Manifest.Namespace
	}
	if namespace == "" {
		namespace = config.C.Local.DefaultNamespace
	}
	namespace = strings.TrimPrefix(namespace, "@")
	if namespace == "" {
		return fmt.Errorf("命名空间不能为空，请使用 --namespace 或在配置中设置 default_namespace")
	}

	// 构建临时包路径
	tmpDir := os.TempDir()
	packName := fmt.Sprintf("%s-%s.zip", s.Manifest.Name, version)
	packPath := filepath.Join(tmpDir, packName)
	defer os.Remove(packPath)

	// 打包
	fmt.Println("正在打包技能...")
	if err := packSkill(path, packPath); err != nil {
		return fmt.Errorf("打包失败: %w", err)
	}
	fmt.Println(color.GreenString("✓"), "打包完成")

	// 创建客户端
	client := newRegistryClient()

	// 推送
	fmt.Println("正在推送到注册中心...")
	req := &registry.PublishRequest{
		Namespace:   namespace,
		Name:        strings.TrimSpace(s.Manifest.Name),
		Version:     version,
		Description: strings.TrimSpace(s.Manifest.Description),
		Category:    category,
		Tags:        tags,
		License:     strings.TrimSpace(s.Manifest.License),
		Force:       opts.force,
	}
	if req.Name == "" {
		return fmt.Errorf("技能名不能为空，请在 SKILL.md 中设置 name")
	}

	resp, err := client.Publish(packPath, req)
	if err != nil {
		// 处理特定错误
		if apiErr, ok := err.(*registry.APIError); ok {
			if apiErr.Code == "VERSION_EXISTS" {
				return fmt.Errorf("版本 %s 已存在，请更新版本号或使用 --force 覆盖", version)
			}
			if apiErr.Code == "VALIDATION_FAILED" {
				fmt.Println(color.RedString("✗"), "安全扫描未通过:")
				fmt.Println("  ", apiErr.Message)
				fmt.Println()
				fmt.Printf("使用 %s 强制推送 (不推荐)\n", color.YellowString("--force"))
				return nil
			}
		}
		return fmt.Errorf("推送失败: %w", err)
	}

	fmt.Println()
	fmt.Println(color.GreenString("✓"), "推送成功!")
	fmt.Printf("  技能: %s/%s@%s\n", color.CyanString(resp.Namespace), resp.Name, resp.Version)
	fmt.Printf("  下载: %s\n", color.CyanString(resp.DownloadURL))
	fmt.Printf("  时间: %s\n", resp.PublishedAt)

	// 自动同步（如果配置了）
	if config.C.Sync.AutoSyncOnPush {
		fmt.Println()
		fmt.Println("正在自动同步到本地 IDE...")
		if err := syncParsedSkill(s, &syncOptions{}); err != nil {
			return fmt.Errorf("自动同步失败: %w", err)
		}
	}

	return nil
}

// packSkill 打包技能到指定路径
func packSkill(srcPath, dstPath string) error {
	// 复用 pack 命令的逻辑
	opts := &packOptions{
		output:   dstPath,
		compress: true,
	}
	return runPack(srcPath, opts)
}

func validatePublishMetadata(s *skill.Skill) (string, []string, error) {
	if s == nil {
		return "", nil, fmt.Errorf("技能内容不能为空")
	}

	definition, err := taxonomy.Load()
	if err != nil {
		return "", nil, fmt.Errorf("加载 taxonomy 失败: %w", err)
	}

	category := strings.ToLower(strings.TrimSpace(s.Manifest.Category))
	if category == "" {
		return "", nil, fmt.Errorf("缺少 category，请在 SKILL.md 中设置 category")
	}
	if !definition.HasCategory(category) {
		return "", nil, fmt.Errorf("category %q 不在官方词表中", s.Manifest.Category)
	}

	if len(s.Manifest.Tags) == 0 {
		return "", nil, fmt.Errorf("tags 至少需要 1 个官方标签")
	}
	if len(s.Manifest.Tags) > 4 {
		return "", nil, fmt.Errorf("官方 tags 最多只能填写 4 个")
	}

	normalizedTags := make([]string, 0, len(s.Manifest.Tags))
	seen := make(map[string]struct{}, len(s.Manifest.Tags))
	for _, rawTag := range s.Manifest.Tags {
		tag := definition.NormalizeTag(rawTag)
		if tag == "" {
			continue
		}
		if !definition.HasOfficialTag(tag) {
			return "", nil, fmt.Errorf("tag %q 不在官方词表中", rawTag)
		}
		if _, exists := seen[tag]; exists {
			continue
		}
		seen[tag] = struct{}{}
		normalizedTags = append(normalizedTags, tag)
	}

	if len(normalizedTags) == 0 {
		return "", nil, fmt.Errorf("tags 至少需要 1 个官方标签")
	}

	return category, normalizedTags, nil
}

func ensurePublishMetadata(path string, s *skill.Skill) error {
	if _, _, err := validatePublishMetadata(s); err == nil {
		return nil
	}

	if !pushTerminalChecker() {
		_, _, err := validatePublishMetadata(s)
		return err
	}

	fmt.Println(color.YellowString("!"), "检测到 SKILL.md 缺少或包含非法的官方分类元数据，先补齐后继续发布。")

	definition, err := taxonomy.Load()
	if err != nil {
		return fmt.Errorf("加载 taxonomy 失败: %w", err)
	}

	category, normalizedExistingTags := collectExistingMetadata(definition, s)
	selectedCategory, selectedTags, err := pushMetadataPrompter(category, normalizedExistingTags)
	if err != nil {
		return fmt.Errorf("补齐分类元数据失败: %w", err)
	}

	normalizedSkill := *s
	normalizedSkill.Manifest = s.Manifest
	normalizedSkill.Manifest.Category = strings.TrimSpace(selectedCategory)
	normalizedSkill.Manifest.Tags = append([]string{}, selectedTags...)

	validCategory, validTags, err := validatePublishMetadata(&normalizedSkill)
	if err != nil {
		return err
	}

	if err := writePublishMetadata(path, validCategory, validTags); err != nil {
		return err
	}

	s.Manifest.Category = validCategory
	s.Manifest.Tags = validTags
	return nil
}

func collectExistingMetadata(definition *taxonomy.Definition, s *skill.Skill) (string, []string) {
	category := strings.ToLower(strings.TrimSpace(s.Manifest.Category))
	if definition == nil || !definition.HasCategory(category) {
		category = ""
	}

	normalizedTags := make([]string, 0, len(s.Manifest.Tags))
	seen := make(map[string]struct{}, len(s.Manifest.Tags))
	for _, rawTag := range s.Manifest.Tags {
		tag := definition.NormalizeTag(rawTag)
		if tag == "" || !definition.HasOfficialTag(tag) {
			continue
		}
		if _, exists := seen[tag]; exists {
			continue
		}
		seen[tag] = struct{}{}
		normalizedTags = append(normalizedTags, tag)
	}

	return category, normalizedTags
}

func promptPublishMetadata(category string, tags []string) (string, []string, error) {
	categories, err := categoryOptions()
	if err != nil {
		return "", nil, err
	}
	if len(categories) == 0 {
		return "", nil, fmt.Errorf("taxonomy 中没有可用的分类")
	}

	defaultCategory := category
	if defaultCategory == "" {
		defaultCategory = categories[0]
	}

	selectedCategory := defaultCategory
	if err := survey.AskOne(&survey.Select{
		Message: "选择一级分类:",
		Options: categories,
		Default: defaultCategory,
	}, &selectedCategory); err != nil {
		return "", nil, err
	}

	definition, err := taxonomy.Load()
	if err != nil {
		return "", nil, err
	}
	options := make([]string, 0, len(definition.OfficialTags))
	for _, tag := range definition.OfficialTags {
		options = append(options, tag.ID)
	}

	selectedTags := append([]string{}, tags...)
	if err := survey.AskOne(&survey.MultiSelect{
		Message: "选择 1-4 个官方标签:",
		Options: options,
		Default: selectedTags,
		Help:    "这些标签会写回 SKILL.md，并用于目录筛选和展示",
	}, &selectedTags); err != nil {
		return "", nil, err
	}
	if len(selectedTags) == 0 {
		return "", nil, fmt.Errorf("至少选择 1 个官方标签")
	}
	if len(selectedTags) > 4 {
		return "", nil, fmt.Errorf("最多选择 4 个官方标签")
	}

	return selectedCategory, selectedTags, nil
}

func writePublishMetadata(path string, category string, tags []string) error {
	skillFile := filepath.Join(path, "SKILL.md")
	content, err := os.ReadFile(skillFile)
	if err != nil {
		return fmt.Errorf("读取 SKILL.md 失败: %w", err)
	}

	frontmatter, body, err := skill.ParseFrontmatter(string(content))
	if err != nil {
		return err
	}

	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(frontmatter), &doc); err != nil {
		return fmt.Errorf("解析 SKILL.md frontmatter 失败: %w", err)
	}

	mapping := &doc
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		mapping = doc.Content[0]
	}
	if mapping.Kind != yaml.MappingNode {
		return fmt.Errorf("SKILL.md frontmatter 必须是 YAML 对象")
	}

	upsertYAMLString(mapping, "category", category)
	upsertYAMLStringList(mapping, "tags", tags)

	frontmatterBytes, err := yaml.Marshal(&doc)
	if err != nil {
		return fmt.Errorf("序列化 SKILL.md frontmatter 失败: %w", err)
	}

	body = strings.TrimSpace(body)
	updated := fmt.Sprintf("---\n%s---\n\n%s\n", strings.TrimSpace(string(frontmatterBytes)), body)
	if err := os.WriteFile(skillFile, []byte(updated), 0644); err != nil {
		return fmt.Errorf("写入 SKILL.md 失败: %w", err)
	}
	return nil
}

func upsertYAMLString(mapping *yaml.Node, key string, value string) {
	removeYAMLKey(mapping, key)
	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value},
	)
}

func upsertYAMLStringList(mapping *yaml.Node, key string, values []string) {
	removeYAMLKey(mapping, key)

	sequence := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	for _, value := range values {
		sequence.Content = append(sequence.Content, &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!str",
			Value: value,
		})
	}

	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		sequence,
	)
}

func removeYAMLKey(mapping *yaml.Node, key string) {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return
	}

	filtered := make([]*yaml.Node, 0, len(mapping.Content))
	for i := 0; i < len(mapping.Content); i += 2 {
		if i+1 >= len(mapping.Content) {
			break
		}
		if mapping.Content[i].Value == key {
			continue
		}
		filtered = append(filtered, mapping.Content[i], mapping.Content[i+1])
	}
	mapping.Content = filtered
}
