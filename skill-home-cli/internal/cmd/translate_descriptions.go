package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/skill-home/cli/internal/config"
	"github.com/skill-home/cli/internal/registry"
)

type translateDescriptionsOptions struct {
	skillRef  string
	namespace string
	limit     int
	force     bool
	dryRun    bool
	model     string
	timeout   int
}

type descriptionTranslator interface {
	Translate(text string) (string, error)
}

type anthropicDescriptionTranslator struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	model      string
}

type anthropicMessagesResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func newTranslateDescriptionsCmd() *cobra.Command {
	opts := &translateDescriptionsOptions{}

	cmd := &cobra.Command{
		Use:   "translate-descriptions",
		Short: "批量把远程 skill 描述翻译成中文",
		Long:  "为缺少 description_zh 的远程 skill 生成中文描述，并回写到注册中心。",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTranslateDescriptions(opts)
		},
	}

	cmd.Flags().StringVar(&opts.skillRef, "skill", "", "只处理一个 skill，例如 @testuser/github")
	cmd.Flags().StringVarP(&opts.namespace, "namespace", "n", "", "只处理指定命名空间")
	cmd.Flags().IntVar(&opts.limit, "limit", 0, "最多处理多少个 skill，0 表示不限制")
	cmd.Flags().BoolVar(&opts.force, "force", false, "即使已有中文描述也强制覆盖")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "只预览翻译结果，不写回注册中心")
	cmd.Flags().StringVar(&opts.model, "model", "", "翻译模型，默认按当前 ANTHROPIC_BASE_URL 推断")
	cmd.Flags().IntVar(&opts.timeout, "timeout", 90, "单条翻译超时时间（秒）")

	return cmd
}

func runTranslateDescriptions(opts *translateDescriptionsOptions) error {
	if err := requireRegistryLogin(); err != nil {
		return err
	}

	client := newRegistryClient()
	candidates, err := loadTranslationCandidates(client, opts)
	if err != nil {
		return err
	}
	if len(candidates) == 0 {
		fmt.Println("没有需要翻译的 skill。")
		return nil
	}

	translator, err := newAnthropicDescriptionTranslator(opts.model, opts.timeout)
	if err != nil {
		return err
	}

	updated := 0
	skipped := 0
	failed := 0
	for _, skill := range candidates {
		description := strings.TrimSpace(skill.Description)
		if description == "" {
			skipped++
			continue
		}

		translated, err := translator.Translate(description)
		if err != nil {
			fmt.Printf("%s @%s/%s 翻译失败: %v\n", color.RedString("✗"), skill.Namespace, skill.Name, err)
			failed++
			continue
		}
		if translated == "" {
			skipped++
			continue
		}

		fullName := fmt.Sprintf("@%s/%s", strings.TrimPrefix(skill.Namespace, "@"), skill.Name)
		fmt.Printf("%s %s\n", color.CyanString(fullName), color.WhiteString(description))
		fmt.Printf("  -> %s\n", color.GreenString(translated))

		if opts.dryRun {
			continue
		}

		if _, err := client.UpdateSkill(skill.Namespace, skill.Name, &registry.UpdateSkillRequest{
			Description:   strings.TrimSpace(skill.Description),
			DescriptionZh: translated,
			Category:      skill.Category,
			Tags:          append([]string{}, skill.Tags...),
			License:       skill.License,
			IsPublic:      skill.IsPublic,
			IsOwnerOnly:   skill.IsOwnerOnly,
			IsDeprecated:  skill.IsDeprecated,
		}); err != nil {
			fmt.Printf("%s @%s/%s 回写失败: %v\n", color.RedString("✗"), skill.Namespace, skill.Name, err)
			failed++
			continue
		}

		updated++
	}

	if opts.dryRun {
		fmt.Printf("\n预览完成，共 %d 个候选 skill。\n", len(candidates))
		return nil
	}

	fmt.Printf("\n已更新 %d 个 skill，跳过 %d 个，失败 %d 个。\n", updated, skipped, failed)
	return nil
}

func loadTranslationCandidates(client registryClient, opts *translateDescriptionsOptions) ([]*registry.Skill, error) {
	if opts.skillRef != "" {
		namespace, name, _, err := config.ParseSkillRef(opts.skillRef)
		if err != nil {
			return nil, err
		}
		skill, err := client.GetSkill(namespace, name)
		if err != nil {
			return nil, fmt.Errorf("获取技能详情失败: %w", wrapRegistryReadError(err))
		}
		if shouldTranslateSkillDescription(skill, opts.force) {
			return []*registry.Skill{skill}, nil
		}
		return nil, nil
	}

	summaries, err := client.GetUserSkills()
	if err != nil {
		return nil, fmt.Errorf("获取技能列表失败: %w", wrapRegistryReadError(err))
	}

	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].Namespace != summaries[j].Namespace {
			return summaries[i].Namespace < summaries[j].Namespace
		}
		return summaries[i].Name < summaries[j].Name
	})

	results := make([]*registry.Skill, 0, len(summaries))
	for _, summary := range summaries {
		if opts.namespace != "" && strings.TrimPrefix(summary.Namespace, "@") != strings.TrimPrefix(opts.namespace, "@") {
			continue
		}
		if opts.limit > 0 && len(results) >= opts.limit {
			break
		}

		skill, err := client.GetSkill(summary.Namespace, summary.Name)
		if err != nil {
			return nil, fmt.Errorf("获取 @%s/%s 详情失败: %w", summary.Namespace, summary.Name, wrapRegistryReadError(err))
		}
		if !shouldTranslateSkillDescription(skill, opts.force) {
			continue
		}
		results = append(results, skill)
	}

	return results, nil
}

func shouldTranslateSkillDescription(skill *registry.Skill, force bool) bool {
	if skill == nil {
		return false
	}

	description := strings.TrimSpace(skill.Description)
	if description == "" {
		return false
	}
	if !force && strings.TrimSpace(skill.DescriptionZh) != "" {
		return false
	}
	if !force && containsHan(description) {
		return false
	}
	return containsLatin(description)
}

func containsHan(value string) bool {
	for _, r := range value {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

func containsLatin(value string) bool {
	for _, r := range value {
		if unicode.IsLetter(r) && r <= unicode.MaxASCII {
			return true
		}
	}
	return false
}

func newAnthropicDescriptionTranslator(model string, timeoutSeconds int) (*anthropicDescriptionTranslator, error) {
	apiKey := strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY"))
	if apiKey == "" {
		return nil, fmt.Errorf("缺少 ANTHROPIC_API_KEY，无法执行批量翻译")
	}

	baseURL := strings.TrimSpace(os.Getenv("ANTHROPIC_BASE_URL"))
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}

	if strings.TrimSpace(model) == "" {
		model = strings.TrimSpace(os.Getenv("SKILL_HOME_TRANSLATE_MODEL"))
	}
	if strings.TrimSpace(model) == "" {
		if strings.Contains(baseURL, "api.kimi.com") {
			model = "kimi-k2-0711-preview"
		} else {
			model = "claude-3-5-haiku-latest"
		}
	}

	if timeoutSeconds <= 0 {
		timeoutSeconds = 90
	}

	return &anthropicDescriptionTranslator{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: time.Duration(timeoutSeconds) * time.Second,
		},
		model: model,
	}, nil
}

func (t *anthropicDescriptionTranslator) Translate(text string) (string, error) {
	payload := map[string]interface{}{
		"model":      t.model,
		"max_tokens": 120,
		"system": strings.TrimSpace(
			"你是 Skill Home 注册中心的技能简介翻译器。" +
				"把输入的一句话英文技能介绍翻译成自然、准确、简洁的中文。" +
				"只输出译文本身，不要解释，不要加引号，不要加序号。" +
				"保留 GitHub、gh、API、OpenAI、MCP、Codex 等产品名、命令名和缩写。",
		),
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": text,
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	url := t.baseURL
	if !strings.HasSuffix(url, "/v1/messages") {
		url += "/v1/messages"
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-api-key", t.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result anthropicMessagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if result.Error != nil && result.Error.Message != "" {
			return "", fmt.Errorf(result.Error.Message)
		}
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	parts := make([]string, 0, len(result.Content))
	for _, item := range result.Content {
		if item.Type != "text" {
			continue
		}
		if value := strings.TrimSpace(item.Text); value != "" {
			parts = append(parts, value)
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n")), nil
}
