package cmd

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/viper"

	"github.com/skill-home/cli/internal/registry"
)

func TestSearchUsesCatalogCache(t *testing.T) {
	t.Run("returns cached results when catalog version is unchanged", func(t *testing.T) {
		query := "lint"
		opts := &searchOptions{
			namespace: "@team",
			tags:      []string{"go", "cli"},
			page:      2,
			perPage:   10,
			format:    "json",
		}
		cachedResult := testRemoteSearchResult("cached-skill", opts.page, opts.perPage)
		freshResult := testRemoteSearchResult("fresh-skill", opts.page, opts.perPage)
		client := &countingSearchRegistryClient{
			fakeRegistryClient: &fakeRegistryClient{
				getCatalogVersionResp: &registry.CatalogVersionResponse{CatalogVersion: 7},
				searchResp:            freshResult,
			},
		}
		cache := setupSearchRemoteTestEnv(t, "https://registry.example.com", client)
		seedSearchSnapshot(t, cache, query, opts, 7, cachedResult)

		var err error
		stdout, stderr := captureStdStreams(t, func() {
			err = runSearch(query, opts)
		})
		if err != nil {
			t.Fatalf("runSearch returned error: %v", err)
		}
		if client.getCatalogVersionCalls != 1 {
			t.Fatalf("GetCatalogVersion calls = %d, want 1", client.getCatalogVersionCalls)
		}
		if client.searchCalls != 0 {
			t.Fatalf("Search calls = %d, want 0", client.searchCalls)
		}
		assertRemoteSearchResultJSON(t, stdout, cachedResult)
		if strings.Contains(stdout, "搜索:") {
			t.Fatalf("stdout = %q, search prelude must not appear in JSON output", stdout)
		}
		if stderr != "" {
			t.Fatalf("stderr = %q, want empty", stderr)
		}
	})

	t.Run("refreshes cache when catalog version changes", func(t *testing.T) {
		query := "lint"
		opts := &searchOptions{
			namespace: "@team",
			tags:      []string{"go"},
			page:      3,
			perPage:   5,
			format:    "json",
		}
		staleResult := testRemoteSearchResult("cached-skill", opts.page, opts.perPage)
		freshResult := testRemoteSearchResult("fresh-skill", opts.page, opts.perPage)
		client := &countingSearchRegistryClient{
			fakeRegistryClient: &fakeRegistryClient{
				getCatalogVersionResp: &registry.CatalogVersionResponse{CatalogVersion: 8},
				searchResp:            freshResult,
			},
		}
		cache := setupSearchRemoteTestEnv(t, "https://registry.example.com", client)
		seedSearchSnapshot(t, cache, query, opts, 7, staleResult)

		var err error
		stdout, stderr := captureStdStreams(t, func() {
			err = runSearch(query, opts)
		})
		if err != nil {
			t.Fatalf("runSearch returned error: %v", err)
		}
		if client.searchCalls != 1 {
			t.Fatalf("Search calls = %d, want 1", client.searchCalls)
		}
		wantQuery := buildSearchRemoteQuery(query, opts)
		if client.lastSearchQuery != wantQuery.Query {
			t.Fatalf("Search query = %q, want %q", client.lastSearchQuery, wantQuery.Query)
		}
		if client.lastSearchNamespace != wantQuery.Namespace {
			t.Fatalf("Search namespace = %q, want %q", client.lastSearchNamespace, wantQuery.Namespace)
		}
		if !reflect.DeepEqual(client.lastSearchTags, wantQuery.Tags) {
			t.Fatalf("Search tags = %#v, want %#v", client.lastSearchTags, wantQuery.Tags)
		}
		if client.lastSearchPage != wantQuery.Page {
			t.Fatalf("Search page = %d, want %d", client.lastSearchPage, wantQuery.Page)
		}
		if client.lastSearchPerPage != wantQuery.PerPage {
			t.Fatalf("Search perPage = %d, want %d", client.lastSearchPerPage, wantQuery.PerPage)
		}
		assertRemoteSearchResultJSON(t, stdout, freshResult)
		if stderr != "" {
			t.Fatalf("stderr = %q, want empty", stderr)
		}

		cached, fresh, cacheErr := cache.readFresh(buildSearchRemoteQuery(query, opts), 8)
		if cacheErr != nil {
			t.Fatalf("readFresh returned error: %v", cacheErr)
		}
		if !fresh {
			t.Fatal("expected refreshed cache to be marked fresh")
		}
		if cached == nil || cached.Results[0].Name != "fresh-skill" {
			t.Fatalf("unexpected refreshed cache: %#v", cached)
		}
	})

	t.Run("falls back to cache when catalog version lookup fails", func(t *testing.T) {
		query := "lint"
		opts := &searchOptions{
			namespace: "@team",
			tags:      []string{"go"},
			page:      1,
			perPage:   20,
			format:    "table",
		}
		cachedResult := testRemoteSearchResult("cached-skill", opts.page, opts.perPage)
		client := &countingSearchRegistryClient{
			fakeRegistryClient: &fakeRegistryClient{
				getCatalogVersionErr: errors.New("catalog unavailable"),
				searchErr:            errors.New("remote should not be called"),
			},
		}
		cache := setupSearchRemoteTestEnv(t, "https://registry.example.com", client)
		seedSearchSnapshot(t, cache, query, opts, 7, cachedResult)

		var err error
		stdout, stderr := captureStdStreams(t, func() {
			err = runSearch(query, opts)
		})
		if err != nil {
			t.Fatalf("runSearch returned error: %v", err)
		}
		if client.searchCalls != 0 {
			t.Fatalf("Search calls = %d, want 0", client.searchCalls)
		}
		if !strings.Contains(stdout, "cached-skill") {
			t.Fatalf("stdout = %q, want cached result", stdout)
		}
		if !strings.Contains(stdout, "搜索:") {
			t.Fatalf("stdout = %q, want table search prelude", stdout)
		}
		if !strings.Contains(stderr, "结果可能过期") {
			t.Fatalf("stderr = %q, want fallback warning", stderr)
		}
	})

	t.Run("falls back to cache when refresh fails", func(t *testing.T) {
		query := "lint"
		opts := &searchOptions{
			namespace: "@team",
			tags:      []string{"go"},
			page:      1,
			perPage:   20,
			format:    "table",
		}
		cachedResult := testRemoteSearchResult("cached-skill", opts.page, opts.perPage)
		client := &countingSearchRegistryClient{
			fakeRegistryClient: &fakeRegistryClient{
				getCatalogVersionResp: &registry.CatalogVersionResponse{CatalogVersion: 8},
				searchErr:             errors.New("remote unavailable"),
			},
		}
		cache := setupSearchRemoteTestEnv(t, "https://registry.example.com", client)
		seedSearchSnapshot(t, cache, query, opts, 7, cachedResult)

		var err error
		stdout, stderr := captureStdStreams(t, func() {
			err = runSearch(query, opts)
		})
		if err != nil {
			t.Fatalf("runSearch returned error: %v", err)
		}
		if client.searchCalls != 1 {
			t.Fatalf("Search calls = %d, want 1", client.searchCalls)
		}
		if !strings.Contains(stdout, "cached-skill") {
			t.Fatalf("stdout = %q, want cached result", stdout)
		}
		if !strings.Contains(stderr, "结果可能过期") {
			t.Fatalf("stderr = %q, want fallback warning", stderr)
		}
	})

	t.Run("uses different cache keys when search inputs change", func(t *testing.T) {
		cache := newTestRemoteCatalogCache(t, "https://registry.example.com")
		base := cache.cacheKey(buildSearchRemoteQuery("lint", &searchOptions{
			namespace: "@team",
			tags:      []string{"go"},
			page:      1,
			perPage:   20,
		}))

		variants := []struct {
			name string
			key  string
		}{
			{
				name: "query",
				key: cache.cacheKey(buildSearchRemoteQuery("format", &searchOptions{
					namespace: "@team",
					tags:      []string{"go"},
					page:      1,
					perPage:   20,
				})),
			},
			{
				name: "tags",
				key: cache.cacheKey(buildSearchRemoteQuery("lint", &searchOptions{
					namespace: "@team",
					tags:      []string{"shell"},
					page:      1,
					perPage:   20,
				})),
			},
			{
				name: "namespace",
				key: cache.cacheKey(buildSearchRemoteQuery("lint", &searchOptions{
					namespace: "@other",
					tags:      []string{"go"},
					page:      1,
					perPage:   20,
				})),
			},
			{
				name: "page",
				key: cache.cacheKey(buildSearchRemoteQuery("lint", &searchOptions{
					namespace: "@team",
					tags:      []string{"go"},
					page:      2,
					perPage:   20,
				})),
			},
			{
				name: "perPage",
				key: cache.cacheKey(buildSearchRemoteQuery("lint", &searchOptions{
					namespace: "@team",
					tags:      []string{"go"},
					page:      1,
					perPage:   50,
				})),
			},
		}

		for _, variant := range variants {
			if variant.key == base {
				t.Fatalf("%s variant reused base cache key %q", variant.name, base)
			}
		}
	})

	t.Run("prints fallback warning to stderr without polluting json output", func(t *testing.T) {
		query := "lint"
		opts := &searchOptions{
			namespace: "@team",
			tags:      []string{"go"},
			page:      1,
			perPage:   20,
			format:    "json",
		}
		cachedResult := testRemoteSearchResult("cached-skill", opts.page, opts.perPage)
		client := &countingSearchRegistryClient{
			fakeRegistryClient: &fakeRegistryClient{
				getCatalogVersionErr: errors.New("catalog unavailable"),
				searchErr:            errors.New("remote should not be called"),
			},
		}
		cache := setupSearchRemoteTestEnv(t, "https://registry.example.com", client)
		seedSearchSnapshot(t, cache, query, opts, 7, cachedResult)

		var err error
		stdout, stderr := captureStdStreams(t, func() {
			err = runSearch(query, opts)
		})
		if err != nil {
			t.Fatalf("runSearch returned error: %v", err)
		}
		assertRemoteSearchResultJSON(t, stdout, cachedResult)
		if strings.Contains(stdout, "结果可能过期") {
			t.Fatalf("stdout = %q, warning must not appear in JSON body", stdout)
		}
		if strings.Contains(stdout, "搜索:") || strings.Contains(stdout, "标签:") {
			t.Fatalf("stdout = %q, search prelude must not appear in JSON body", stdout)
		}
		if !strings.Contains(stderr, "结果可能过期") {
			t.Fatalf("stderr = %q, want fallback warning", stderr)
		}
	})
}

type countingSearchRegistryClient struct {
	*fakeRegistryClient

	getCatalogVersionCalls int
	searchCalls            int
	lastSearchQuery        string
	lastSearchNamespace    string
	lastSearchTags         []string
	lastSearchPage         int
	lastSearchPerPage      int
}

func (c *countingSearchRegistryClient) GetCatalogVersion() (*registry.CatalogVersionResponse, error) {
	c.getCatalogVersionCalls++
	return c.fakeRegistryClient.GetCatalogVersion()
}

func (c *countingSearchRegistryClient) Search(query, namespace string, tags []string, page, perPage int) (*registry.SearchResult, error) {
	c.searchCalls++
	c.lastSearchQuery = query
	c.lastSearchNamespace = namespace
	c.lastSearchTags = append([]string(nil), tags...)
	c.lastSearchPage = page
	c.lastSearchPerPage = perPage
	return c.fakeRegistryClient.Search(query, namespace, tags, page, perPage)
}

func setupSearchRemoteTestEnv(t *testing.T, endpoint string, client registryClient) *remoteCatalogCache {
	t.Helper()

	t.Cleanup(viper.Reset)
	viper.Reset()
	viper.Set("registry.endpoint", endpoint)
	t.Setenv("HOME", t.TempDir())

	restore := swapRegistryClientFactory(func() registryClient {
		return client
	})
	t.Cleanup(restore)

	cache, err := newDefaultRemoteCatalogCache()
	if err != nil {
		t.Fatalf("newDefaultRemoteCatalogCache returned error: %v", err)
	}
	return cache
}

func seedSearchSnapshot(t *testing.T, cache *remoteCatalogCache, query string, opts *searchOptions, version int64, result *registry.SearchResult) {
	t.Helper()

	if err := cache.writeSnapshot(buildSearchRemoteQuery(query, opts), &registry.CatalogVersionResponse{
		CatalogVersion: version,
	}, result); err != nil {
		t.Fatalf("writeSnapshot returned error: %v", err)
	}
}

func testRemoteSearchResult(name string, page, perPage int) *registry.SearchResult {
	return &registry.SearchResult{
		Total:   1,
		Page:    page,
		PerPage: perPage,
		Results: []registry.Skill{
			{
				Namespace:     "team",
				Name:          name,
				Description:   "test skill",
				LatestVersion: "1.0.0",
				Tags:          []string{"go"},
			},
		},
	}
}

func assertRemoteSearchResultJSON(t *testing.T, stdout string, want *registry.SearchResult) {
	t.Helper()

	var got registry.SearchResult
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\nstdout=%q", err, stdout)
	}
	if got.Total != want.Total || got.Page != want.Page || got.PerPage != want.PerPage {
		t.Fatalf("unexpected pagination: got %#v want %#v", got, *want)
	}
	if len(got.Results) != len(want.Results) {
		t.Fatalf("result length = %d, want %d", len(got.Results), len(want.Results))
	}
	for i := range want.Results {
		if got.Results[i].Namespace != want.Results[i].Namespace || got.Results[i].Name != want.Results[i].Name {
			t.Fatalf("unexpected result[%d]: got %#v want %#v", i, got.Results[i], want.Results[i])
		}
	}
}
