package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/skill-home/cli/internal/registry"
)

type searchOptions struct {
	namespace string
	tags      []string
	page      int
	perPage   int
	format    string
}

func newSearchCmd() *cobra.Command {
	opts := &searchOptions{}

	cmd := &cobra.Command{
		Use:   "search <keyword>",
		Short: "搜索技能",
		Long:  "在注册中心搜索技能",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := ""
			if len(args) > 0 {
				query = args[0]
			}
			return runSearch(query, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.namespace, "namespace", "n", "", "按命名空间筛选")
	cmd.Flags().StringArrayVarP(&opts.tags, "tag", "t", nil, "按标签筛选")
	cmd.Flags().IntVarP(&opts.page, "page", "p", 1, "页码")
	cmd.Flags().IntVar(&opts.perPage, "per-page", 20, "每页数量")
	cmd.Flags().StringVarP(&opts.format, "format", "f", "table", "输出格式 (table/json)")

	return cmd
}

func runSearch(query string, opts *searchOptions) error {
	client := newRegistryClient()
	cache, err := newDefaultRemoteCatalogCache()
	if err != nil {
		result, err := fetchRemoteSearch(client, buildSearchRemoteQuery(query, opts))
		if err != nil {
			return fmt.Errorf("搜索失败: %w", err)
		}
		return printRemoteSearchResults(query, opts, result)
	}

	return searchRemoteWithCache(query, opts, client, cache, os.Stderr)
}

func searchRemoteWithCache(query string, opts *searchOptions, client registryClient, cache *remoteCatalogCache, stderr io.Writer) error {
	remoteQuery := buildSearchRemoteQuery(query, opts)
	result, stale, err := cache.fetchWithFallback(
		remoteQuery,
		client.GetCatalogVersion,
		func() (*registry.SearchResult, error) {
			return fetchRemoteSearch(client, remoteQuery)
		},
	)
	if err != nil {
		return fmt.Errorf("搜索失败: %w", err)
	}
	if stale {
		fmt.Fprintln(stderr, "警告: 结果可能过期，当前显示的是本地缓存。")
	}
	return printRemoteSearchResults(query, opts, result)
}

func buildSearchRemoteQuery(query string, opts *searchOptions) remoteCatalogQuery {
	return remoteCatalogQuery{
		Kind:      "search",
		Namespace: opts.namespace,
		Query:     query,
		Tags:      opts.tags,
		Page:      opts.page,
		PerPage:   opts.perPage,
	}
}

func fetchRemoteSearch(client registryClient, query remoteCatalogQuery) (*registry.SearchResult, error) {
	return client.Search(query.Query, query.Namespace, query.Tags, query.Page, query.PerPage)
}

func printRemoteSearchResults(query string, opts *searchOptions, result *registry.SearchResult) error {
	if opts.format == "json" {
		return printJSON(result)
	}

	fmt.Printf("搜索: %s\n", color.CyanString(query))
	if len(opts.tags) > 0 {
		fmt.Printf("标签: %s\n", color.CyanString(strings.Join(opts.tags, ", ")))
	}
	fmt.Println()

	if len(result.Results) == 0 {
		fmt.Println("没有找到匹配的技能")
		return nil
	}

	printSkillResults(result)
	fmt.Printf("\n运行 '%s' 安装技能\n", color.YellowString("skill-home pull <name>"))

	return nil
}
