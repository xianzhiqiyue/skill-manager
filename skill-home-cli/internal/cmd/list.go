package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/skill-home/cli/internal/config"
	"github.com/skill-home/cli/internal/registry"
	"github.com/skill-home/cli/internal/skill"
)

type listOptions struct {
	remote    bool
	namespace string
	format    string
}

func newListCmd() *cobra.Command {
	opts := &listOptions{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "列出已安装的技能",
		Long:  "列出本地已安装或已缓存的技能",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.remote, "remote", "r", false, "列出云端已发布的技能")
	cmd.Flags().StringVarP(&opts.namespace, "namespace", "n", "", "按命名空间筛选")
	cmd.Flags().StringVarP(&opts.format, "format", "f", "table", "输出格式 (table/json)")

	return cmd
}

func runList(opts *listOptions) error {
	if opts.remote {
		return listRemoteSkills(opts)
	}
	return listLocalSkills(opts)
}

func listLocalSkills(opts *listOptions) error {
	skillsDir := config.C.Local.SkillsDir
	type localSkill struct {
		Namespace string `json:"namespace"`
		Name      string `json:"name"`
		Version   string `json:"version"`
		Path      string `json:"path"`
	}
	results := []localSkill{}

	// 检查目录是否存在
	if _, err := os.Stat(skillsDir); os.IsNotExist(err) {
		fmt.Println("本地技能目录为空")
		fmt.Printf("运行 '%s' 创建你的第一个技能\n", color.YellowString("skill-home init <name>"))
		return nil
	}

	// 扫描命名空间
	nsEntries, err := os.ReadDir(skillsDir)
	if err != nil {
		return fmt.Errorf("读取技能目录失败: %w", err)
	}

	fmt.Printf("本地技能缓存目录: %s\n\n", color.CyanString(skillsDir))

	found := false
	for _, nsEntry := range nsEntries {
		if !nsEntry.IsDir() || strings.HasPrefix(nsEntry.Name(), ".") {
			continue
		}

		ns := nsEntry.Name()
		if opts.namespace != "" && ns != opts.namespace {
			continue
		}

		// 扫描技能
		nsPath := filepath.Join(skillsDir, ns)
		skillEntries, err := os.ReadDir(nsPath)
		if err != nil {
			continue
		}

		for _, skillEntry := range skillEntries {
			if !skillEntry.IsDir() {
				continue
			}

			skillName := skillEntry.Name()
			skillPath := filepath.Join(nsPath, skillName)

			// 尝试解析技能获取版本
			version := "unknown"
			if s, err := skill.Parse(skillPath); err == nil {
				version = s.Manifest.Version
			}

			results = append(results, localSkill{
				Namespace: ns,
				Name:      skillName,
				Version:   version,
				Path:      skillPath,
			})
			found = true
		}
	}

	if opts.format == "json" {
		return printJSON(results)
	}

	for _, item := range results {
		fmt.Printf("  %s/%s@%s\n", color.GreenString("@"+item.Namespace), item.Name, color.YellowString(item.Version))
	}

	if !found {
		fmt.Println("没有找到本地技能")
		fmt.Printf("运行 '%s' 创建你的第一个技能\n", color.YellowString("skill-home init <name>"))
	}

	return nil
}

func listRemoteSkills(opts *listOptions) error {
	client := newRegistryClient()
	cache, err := newDefaultRemoteCatalogCache()
	if err != nil {
		result, err := fetchRemoteListSkills(client, buildListRemoteQuery(opts))
		if err != nil {
			return fmt.Errorf("获取远程技能列表失败: %w", err)
		}
		return printRemoteListSkills(opts, result)
	}

	return listRemoteSkillsWithCache(opts, client, cache, os.Stderr)
}

func listRemoteSkillsWithCache(opts *listOptions, client registryClient, cache *remoteCatalogCache, stderr io.Writer) error {
	query := buildListRemoteQuery(opts)
	result, stale, err := cache.fetchWithFallback(
		query,
		client.GetCatalogVersion,
		func() (*registry.SearchResult, error) {
			return fetchRemoteListSkills(client, query)
		},
	)
	if err != nil {
		return fmt.Errorf("获取远程技能列表失败: %w", err)
	}
	if stale {
		fmt.Fprintln(stderr, "警告: 结果可能过期，当前显示的是本地缓存。")
	}
	return printRemoteListSkills(opts, result)
}

func buildListRemoteQuery(opts *listOptions) remoteCatalogQuery {
	return remoteCatalogQuery{
		Kind:      "list",
		Namespace: opts.namespace,
		Page:      1,
		PerPage:   100,
	}
}

func buildListSkillsOptions(query remoteCatalogQuery) registry.ListSkillsOptions {
	return registry.ListSkillsOptions{
		Namespace: query.Namespace,
		Query:     query.Query,
		Tags:      query.Tags,
		Page:      query.Page,
		PerPage:   query.PerPage,
	}
}

func fetchRemoteListSkills(client registryClient, query remoteCatalogQuery) (*registry.SearchResult, error) {
	return client.ListSkills(buildListSkillsOptions(query))
}

func printRemoteListSkills(opts *listOptions, result *registry.SearchResult) error {
	if opts.format == "json" {
		return printJSON(result)
	}
	if len(result.Results) == 0 {
		fmt.Println("没有找到远程技能")
		return nil
	}

	fmt.Printf("远程技能列表 (%d)\n\n", result.Total)
	for _, item := range result.Results {
		printSkillSummary(item)
		fmt.Println()
	}
	if result.Total > result.PerPage*result.Page {
		fmt.Printf("使用 --page %d 查看更多结果\n", result.Page+1)
	}
	return nil
}

func stringsHasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
