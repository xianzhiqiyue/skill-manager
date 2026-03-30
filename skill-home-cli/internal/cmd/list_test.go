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

func TestListRemoteUsesCatalogCache(t *testing.T) {
	t.Run("returns cached results when catalog version is unchanged", func(t *testing.T) {
		opts := &listOptions{remote: true, namespace: "@team", format: "json"}
		cachedResult := testRemoteListResult("cached-skill")
		freshResult := testRemoteListResult("fresh-skill")
		client := &countingRegistryClient{
			fakeRegistryClient: &fakeRegistryClient{
				getCatalogVersionResp: &registry.CatalogVersionResponse{CatalogVersion: 7},
				listSkillsResp:        freshResult,
			},
		}
		cache := setupListRemoteTestEnv(t, "https://registry.example.com", client)
		seedListSnapshot(t, cache, opts, 7, cachedResult)

		var err error
		stdout, stderr := captureStdStreams(t, func() {
			err = listRemoteSkills(opts)
		})
		if err != nil {
			t.Fatalf("listRemoteSkills returned error: %v", err)
		}
		if client.getCatalogVersionCalls != 1 {
			t.Fatalf("GetCatalogVersion calls = %d, want 1", client.getCatalogVersionCalls)
		}
		if client.listSkillsCalls != 0 {
			t.Fatalf("ListSkills calls = %d, want 0", client.listSkillsCalls)
		}
		assertSearchResultJSON(t, stdout, cachedResult)
		if stderr != "" {
			t.Fatalf("stderr = %q, want empty", stderr)
		}
	})

	t.Run("refreshes cache when catalog version changes", func(t *testing.T) {
		opts := &listOptions{remote: true, namespace: "@team", format: "json"}
		staleResult := testRemoteListResult("cached-skill")
		freshResult := testRemoteListResult("fresh-skill")
		client := &countingRegistryClient{
			fakeRegistryClient: &fakeRegistryClient{
				getCatalogVersionResp: &registry.CatalogVersionResponse{CatalogVersion: 8},
				listSkillsResp:        freshResult,
			},
		}
		cache := setupListRemoteTestEnv(t, "https://registry.example.com", client)
		seedListSnapshot(t, cache, opts, 7, staleResult)

		var err error
		stdout, stderr := captureStdStreams(t, func() {
			err = listRemoteSkills(opts)
		})
		if err != nil {
			t.Fatalf("listRemoteSkills returned error: %v", err)
		}
		if client.listSkillsCalls != 1 {
			t.Fatalf("ListSkills calls = %d, want 1", client.listSkillsCalls)
		}
		wantListOpts := buildListSkillsOptions(buildListRemoteQuery(opts))
		if !reflect.DeepEqual(client.lastListSkillsOpts, wantListOpts) {
			t.Fatalf("ListSkills opts = %#v, want %#v", client.lastListSkillsOpts, wantListOpts)
		}
		assertSearchResultJSON(t, stdout, freshResult)
		if stderr != "" {
			t.Fatalf("stderr = %q, want empty", stderr)
		}

		cached, fresh, cacheErr := cache.readFresh(buildListRemoteQuery(opts), 8)
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
		opts := &listOptions{remote: true, namespace: "@team", format: "table"}
		cachedResult := testRemoteListResult("cached-skill")
		client := &countingRegistryClient{
			fakeRegistryClient: &fakeRegistryClient{
				getCatalogVersionErr: errors.New("catalog unavailable"),
				listSkillsErr:        errors.New("remote should not be called"),
			},
		}
		cache := setupListRemoteTestEnv(t, "https://registry.example.com", client)
		seedListSnapshot(t, cache, opts, 7, cachedResult)

		var err error
		stdout, stderr := captureStdStreams(t, func() {
			err = listRemoteSkills(opts)
		})
		if err != nil {
			t.Fatalf("listRemoteSkills returned error: %v", err)
		}
		if client.listSkillsCalls != 0 {
			t.Fatalf("ListSkills calls = %d, want 0", client.listSkillsCalls)
		}
		if !strings.Contains(stdout, "cached-skill") {
			t.Fatalf("stdout = %q, want cached result", stdout)
		}
		if !strings.Contains(stderr, "结果可能过期") {
			t.Fatalf("stderr = %q, want fallback warning", stderr)
		}
	})

	t.Run("falls back to cache when refresh fails", func(t *testing.T) {
		opts := &listOptions{remote: true, namespace: "@team", format: "table"}
		cachedResult := testRemoteListResult("cached-skill")
		client := &countingRegistryClient{
			fakeRegistryClient: &fakeRegistryClient{
				getCatalogVersionResp: &registry.CatalogVersionResponse{CatalogVersion: 8},
				listSkillsErr:         errors.New("remote unavailable"),
			},
		}
		cache := setupListRemoteTestEnv(t, "https://registry.example.com", client)
		seedListSnapshot(t, cache, opts, 7, cachedResult)

		var err error
		stdout, stderr := captureStdStreams(t, func() {
			err = listRemoteSkills(opts)
		})
		if err != nil {
			t.Fatalf("listRemoteSkills returned error: %v", err)
		}
		if client.listSkillsCalls != 1 {
			t.Fatalf("ListSkills calls = %d, want 1", client.listSkillsCalls)
		}
		if !strings.Contains(stdout, "cached-skill") {
			t.Fatalf("stdout = %q, want cached result", stdout)
		}
		if !strings.Contains(stderr, "结果可能过期") {
			t.Fatalf("stderr = %q, want fallback warning", stderr)
		}
	})

	t.Run("returns error when no cache exists and remote refresh fails", func(t *testing.T) {
		opts := &listOptions{remote: true, namespace: "@team", format: "table"}
		client := &countingRegistryClient{
			fakeRegistryClient: &fakeRegistryClient{
				getCatalogVersionResp: &registry.CatalogVersionResponse{CatalogVersion: 8},
				listSkillsErr:         errors.New("remote unavailable"),
			},
		}
		setupListRemoteTestEnv(t, "https://registry.example.com", client)

		var err error
		stdout, stderr := captureStdStreams(t, func() {
			err = listRemoteSkills(opts)
		})
		if err == nil {
			t.Fatal("listRemoteSkills returned nil error, want failure")
		}
		if !strings.Contains(err.Error(), "获取远程技能列表失败") {
			t.Fatalf("error = %v, want wrapped remote failure", err)
		}
		if stdout != "" {
			t.Fatalf("stdout = %q, want empty", stdout)
		}
		if stderr != "" {
			t.Fatalf("stderr = %q, want empty", stderr)
		}
	})

	t.Run("prints fallback warning to stderr without polluting json output", func(t *testing.T) {
		opts := &listOptions{remote: true, namespace: "@team", format: "json"}
		cachedResult := testRemoteListResult("cached-skill")
		client := &countingRegistryClient{
			fakeRegistryClient: &fakeRegistryClient{
				getCatalogVersionErr: errors.New("catalog unavailable"),
				listSkillsErr:        errors.New("remote should not be called"),
			},
		}
		cache := setupListRemoteTestEnv(t, "https://registry.example.com", client)
		seedListSnapshot(t, cache, opts, 7, cachedResult)

		var err error
		stdout, stderr := captureStdStreams(t, func() {
			err = listRemoteSkills(opts)
		})
		if err != nil {
			t.Fatalf("listRemoteSkills returned error: %v", err)
		}
		assertSearchResultJSON(t, stdout, cachedResult)
		if strings.Contains(stdout, "结果可能过期") {
			t.Fatalf("stdout = %q, warning must not appear in JSON body", stdout)
		}
		if !strings.Contains(stderr, "结果可能过期") {
			t.Fatalf("stderr = %q, want fallback warning", stderr)
		}
	})
}

type countingRegistryClient struct {
	*fakeRegistryClient

	getCatalogVersionCalls int
	listSkillsCalls        int
	lastListSkillsOpts     registry.ListSkillsOptions
}

func (c *countingRegistryClient) GetCatalogVersion() (*registry.CatalogVersionResponse, error) {
	c.getCatalogVersionCalls++
	return c.fakeRegistryClient.GetCatalogVersion()
}

func (c *countingRegistryClient) ListSkills(opts registry.ListSkillsOptions) (*registry.SearchResult, error) {
	c.listSkillsCalls++
	c.lastListSkillsOpts = opts
	return c.fakeRegistryClient.ListSkills(opts)
}

func setupListRemoteTestEnv(t *testing.T, endpoint string, client registryClient) *remoteCatalogCache {
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

func seedListSnapshot(t *testing.T, cache *remoteCatalogCache, opts *listOptions, version int64, result *registry.SearchResult) {
	t.Helper()

	if err := cache.writeSnapshot(buildListRemoteQuery(opts), &registry.CatalogVersionResponse{
		CatalogVersion: version,
	}, result); err != nil {
		t.Fatalf("writeSnapshot returned error: %v", err)
	}
}

func testRemoteListResult(name string) *registry.SearchResult {
	return &registry.SearchResult{
		Total:   1,
		Page:    1,
		PerPage: 100,
		Results: []registry.Skill{
			{
				Namespace:     "team",
				Name:          name,
				Description:   "test skill",
				LatestVersion: "1.0.0",
			},
		},
	}
}

func assertSearchResultJSON(t *testing.T, stdout string, want *registry.SearchResult) {
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
