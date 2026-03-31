package cmd

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/skill-home/cli/internal/registry"
)

func TestRemoteCatalogCacheStableKey(t *testing.T) {
	cache := newRemoteCatalogCache(t.TempDir(), "https://registry.example.com")
	query := remoteCatalogQuery{
		Kind:      "search",
		Namespace: "@team",
		Query:     "lint",
		Tags:      []string{"go", "cli"},
		Page:      2,
		PerPage:   20,
	}

	first := cache.cacheKey(query)
	second := cache.cacheKey(query)
	if first != second {
		t.Fatalf("cache key changed for same query: %q vs %q", first, second)
	}
}

func TestRemoteCatalogCacheSwitchingEndpointUsesSeparateCacheNamespace(t *testing.T) {
	root := t.TempDir()
	endpointA := "https://registry-a.example.com"
	endpointB := "https://registry-b.example.com"
	query := remoteCatalogQuery{
		Kind:      "list",
		Namespace: "@team",
		Page:      1,
		PerPage:   100,
	}
	cached := &registry.SearchResult{
		Total:   1,
		Page:    1,
		PerPage: 100,
		Results: []registry.Skill{{Namespace: "team", Name: "cached"}},
	}

	cacheA := newRemoteCatalogCache(root, endpointA)
	cacheB := newRemoteCatalogCache(root, endpointB)
	if cacheA.cacheDir() == cacheB.cacheDir() {
		t.Fatalf("expected distinct cache dirs for different endpoints, got %q", cacheA.cacheDir())
	}

	if err := writeJSONFile(cacheA.queryPath(query), remoteCatalogQueryCache{
		RegistryEndpoint: endpointA,
		Kind:             "list",
		Namespace:        "@team",
		Page:             1,
		PerPage:          100,
		CatalogVersion:   7,
		CachedAt:         time.Unix(1, 0).UTC(),
		Result:           cached,
	}); err != nil {
		t.Fatalf("writeJSONFile returned error: %v", err)
	}
	if err := writeJSONFile(cacheA.statePath(), remoteCatalogState{
		RegistryEndpoint: endpointA,
		CatalogVersion:   7,
		CheckedAt:        time.Unix(2, 0).UTC(),
	}); err != nil {
		t.Fatalf("writeJSONFile returned error: %v", err)
	}

	got, stale, err := cacheB.fetchWithFallback(query,
		func() (*registry.CatalogVersionResponse, error) {
			return &registry.CatalogVersionResponse{CatalogVersion: 7}, nil
		},
		func() (*registry.SearchResult, error) {
			return nil, errors.New("remote unavailable")
		},
	)
	if err == nil {
		t.Fatal("expected error when endpoint changes and no cache exists under the new namespace")
	}
	if got != nil {
		t.Fatalf("expected no cached result, got %#v", got)
	}
	if stale {
		t.Fatal("expected stale fallback to be disabled for a different registry endpoint")
	}
}

func TestRemoteCatalogCacheRejectsVersionMismatchAsFreshCache(t *testing.T) {
	cache := newRemoteCatalogCache(t.TempDir(), "https://registry.example.com")
	query := remoteCatalogQuery{
		Kind:      "search",
		Namespace: "@team",
		Query:     "lint",
		Tags:      []string{"go"},
		Page:      1,
		PerPage:   20,
	}
	cached := &registry.SearchResult{
		Total:   1,
		Page:    1,
		PerPage: 20,
		Results: []registry.Skill{{Namespace: "team", Name: "cached"}},
	}

	if err := writeJSONFile(cache.queryPath(query), remoteCatalogQueryCache{
		RegistryEndpoint: "https://registry.example.com",
		Kind:             "search",
		Namespace:        "@team",
		Query:            "lint",
		Tags:             []string{"go"},
		Page:             1,
		PerPage:          20,
		CatalogVersion:   1,
		CachedAt:         time.Unix(1, 0).UTC(),
		Result:           cached,
	}); err != nil {
		t.Fatalf("writeJSONFile returned error: %v", err)
	}
	if err := writeJSONFile(cache.statePath(), remoteCatalogState{
		RegistryEndpoint: "https://registry.example.com",
		CatalogVersion:   2,
		CheckedAt:        time.Unix(2, 0).UTC(),
	}); err != nil {
		t.Fatalf("writeJSONFile returned error: %v", err)
	}

	got, fresh, err := cache.readFresh(query, 2)
	if err != nil {
		t.Fatalf("readFresh returned error: %v", err)
	}
	if fresh {
		t.Fatal("expected mismatched state/query versions to be treated as stale")
	}
	if got != nil {
		t.Fatalf("expected no fresh cache result, got %#v", got)
	}
}

func TestRemoteCatalogCacheAllowsExpiredFallbackWhenRemoteFails(t *testing.T) {
	cache := newRemoteCatalogCache(t.TempDir(), "https://registry.example.com")
	query := remoteCatalogQuery{
		Kind:      "search",
		Namespace: "@team",
		Query:     "lint",
		Tags:      []string{"go"},
		Page:      1,
		PerPage:   20,
	}
	cached := &registry.SearchResult{
		Total:   1,
		Page:    1,
		PerPage: 20,
		Results: []registry.Skill{{Namespace: "team", Name: "cached"}},
	}

	if err := writeJSONFile(cache.queryPath(query), remoteCatalogQueryCache{
		RegistryEndpoint: "https://registry.example.com",
		Kind:             "search",
		Namespace:        "@team",
		Query:            "lint",
		Tags:             []string{"go"},
		Page:             1,
		PerPage:          20,
		CatalogVersion:   1,
		CachedAt:         time.Unix(1, 0).UTC(),
		Result:           cached,
	}); err != nil {
		t.Fatalf("writeJSONFile returned error: %v", err)
	}
	if err := writeJSONFile(cache.statePath(), remoteCatalogState{
		RegistryEndpoint: "https://registry.example.com",
		CatalogVersion:   2,
		CheckedAt:        time.Unix(2, 0).UTC(),
	}); err != nil {
		t.Fatalf("writeJSONFile returned error: %v", err)
	}

	got, stale, err := cache.fetchWithFallback(query,
		func() (*registry.CatalogVersionResponse, error) {
			return &registry.CatalogVersionResponse{CatalogVersion: 2}, nil
		},
		func() (*registry.SearchResult, error) {
			return nil, errors.New("remote unavailable")
		},
	)
	if err != nil {
		t.Fatalf("fetchWithFallback returned error: %v", err)
	}
	if !stale {
		t.Fatal("expected stale fallback when remote request fails")
	}
	if !reflect.DeepEqual(got, cached) {
		t.Fatalf("unexpected fallback result: got %#v want %#v", got, cached)
	}
}

func TestRemoteCatalogCacheWriteSnapshotWritesQueryAndStateFiles(t *testing.T) {
	cache := newRemoteCatalogCache(t.TempDir(), "https://registry.example.com")
	query := remoteCatalogQuery{
		Kind:      "search",
		Namespace: "@team",
		Query:     "lint",
		Tags:      []string{"go"},
		Page:      1,
		PerPage:   20,
	}
	version := &registry.CatalogVersionResponse{
		CatalogVersion: 7,
		UpdatedAt:      time.Unix(3, 0).UTC(),
	}
	result := &registry.SearchResult{
		Total:   1,
		Page:    1,
		PerPage: 20,
		Results: []registry.Skill{{Namespace: "team", Name: "cached"}},
	}

	if err := cache.writeSnapshot(query, version, result); err != nil {
		t.Fatalf("writeSnapshot returned error: %v", err)
	}

	if _, err := os.Stat(cache.queryPath(query)); err != nil {
		t.Fatalf("query cache file missing: %v", err)
	}
	if _, err := os.Stat(cache.statePath()); err != nil {
		t.Fatalf("state file missing: %v", err)
	}

	writtenQuery, err := cache.readQueryCache(query)
	if err != nil {
		t.Fatalf("readQueryCache returned error: %v", err)
	}
	if writtenQuery.CatalogVersion != 7 {
		t.Fatalf("unexpected query catalog version: %d", writtenQuery.CatalogVersion)
	}
	if !reflect.DeepEqual(writtenQuery.Result, result) {
		t.Fatalf("unexpected query result: got %#v want %#v", writtenQuery.Result, result)
	}

	writtenState, err := cache.readState()
	if err != nil {
		t.Fatalf("readState returned error: %v", err)
	}
	if writtenState.CatalogVersion != 7 {
		t.Fatalf("unexpected state catalog version: %d", writtenState.CatalogVersion)
	}
	if writtenState.RegistryEndpoint != "https://registry.example.com" {
		t.Fatalf("unexpected state registry endpoint: %q", writtenState.RegistryEndpoint)
	}
}

func TestRemoteCatalogCacheFetchWithFallbackIgnoresSnapshotWriteFailure(t *testing.T) {
	cache := newRemoteCatalogCache(t.TempDir(), "https://registry.example.com")
	query := remoteCatalogQuery{
		Kind:      "search",
		Namespace: "@team",
		Query:     "lint",
		Tags:      []string{"go"},
		Page:      1,
		PerPage:   20,
	}
	result := &registry.SearchResult{
		Total:   1,
		Page:    1,
		PerPage: 20,
		Results: []registry.Skill{{Namespace: "team", Name: "remote"}},
	}
	version := &registry.CatalogVersionResponse{CatalogVersion: 11}

	oldWriter := writeJSONAtomicFunc
	writeJSONAtomicFunc = func(path string, value any) error {
		return errors.New("disk full")
	}
	t.Cleanup(func() {
		writeJSONAtomicFunc = oldWriter
	})

	got, stale, err := cache.fetchWithFallback(query,
		func() (*registry.CatalogVersionResponse, error) {
			return version, nil
		},
		func() (*registry.SearchResult, error) {
			return result, nil
		},
	)
	if err != nil {
		t.Fatalf("fetchWithFallback returned error: %v", err)
	}
	if stale {
		t.Fatal("expected fresh remote result, got stale fallback")
	}
	if got != result {
		t.Fatalf("unexpected result pointer: got %#v want %#v", got, result)
	}
}

func TestRemoteCatalogCacheRejectsZeroValueResultStructure(t *testing.T) {
	cache := newRemoteCatalogCache(t.TempDir(), "https://registry.example.com")
	query := remoteCatalogQuery{
		Kind:      "search",
		Namespace: "@team",
		Query:     "lint",
		Tags:      []string{"go"},
		Page:      1,
		PerPage:   20,
	}

	if err := writeJSONFile(cache.queryPath(query), remoteCatalogQueryCache{
		RegistryEndpoint: "https://registry.example.com",
		Kind:             "search",
		Namespace:        "@team",
		Query:            "lint",
		Tags:             []string{"go"},
		Page:             1,
		PerPage:          20,
		CatalogVersion:   1,
		CachedAt:         time.Unix(1, 0).UTC(),
		Result:           &registry.SearchResult{},
	}); err != nil {
		t.Fatalf("writeJSONFile returned error: %v", err)
	}
	if err := writeJSONFile(cache.statePath(), remoteCatalogState{
		RegistryEndpoint: "https://registry.example.com",
		CatalogVersion:   1,
		CheckedAt:        time.Unix(2, 0).UTC(),
	}); err != nil {
		t.Fatalf("writeJSONFile returned error: %v", err)
	}

	got, fresh, err := cache.readFresh(query, 1)
	if err != nil {
		t.Fatalf("readFresh returned error: %v", err)
	}
	if fresh {
		t.Fatal("expected zero-value result structure to be rejected as fresh")
	}
	if got != nil {
		t.Fatalf("expected no fresh result, got %#v", got)
	}

	fallback, _, fallbackErr := cache.fetchWithFallback(query,
		func() (*registry.CatalogVersionResponse, error) {
			return nil, errors.New("version unavailable")
		},
		func() (*registry.SearchResult, error) {
			return nil, errors.New("remote unavailable")
		},
	)
	if fallbackErr == nil {
		t.Fatal("expected fallback to be rejected for zero-value result structure")
	}
	if fallback != nil {
		t.Fatalf("expected no fallback result, got %#v", fallback)
	}
}

func TestRemoteCatalogCacheRejectsIncompleteFallbackCache(t *testing.T) {
	t.Run("missing source", func(t *testing.T) {
		cache := newRemoteCatalogCache(t.TempDir(), "https://registry.example.com")
		query := remoteCatalogQuery{
			Kind:      "search",
			Namespace: "@team",
			Query:     "lint",
			Tags:      []string{"go"},
			Page:      1,
			PerPage:   20,
		}

		if err := writeJSONFile(cache.queryPath(query), remoteCatalogQueryCache{
			Kind:           "search",
			Namespace:      "@team",
			Query:          "lint",
			Tags:           []string{"go"},
			Page:           1,
			PerPage:        20,
			CatalogVersion: 1,
			CachedAt:       time.Unix(1, 0).UTC(),
			Result:         &registry.SearchResult{Total: 1, Page: 1, PerPage: 20},
		}); err != nil {
			t.Fatalf("writeJSONFile returned error: %v", err)
		}

		got, stale, err := cache.fetchWithFallback(query,
			func() (*registry.CatalogVersionResponse, error) {
				return nil, errors.New("version unavailable")
			},
			func() (*registry.SearchResult, error) {
				return nil, errors.New("remote unavailable")
			},
		)
		if err == nil {
			t.Fatal("expected error when fallback cache is missing source information")
		}
		if got != nil {
			t.Fatalf("expected no cached result, got %#v", got)
		}
		if stale {
			t.Fatal("expected stale fallback to be rejected when source is missing")
		}
	})

	t.Run("corrupt structure", func(t *testing.T) {
		cache := newRemoteCatalogCache(t.TempDir(), "https://registry.example.com")
		query := remoteCatalogQuery{
			Kind:      "search",
			Namespace: "@team",
			Query:     "lint",
			Tags:      []string{"go"},
			Page:      1,
			PerPage:   20,
		}

		if err := os.MkdirAll(filepath.Dir(cache.queryPath(query)), 0755); err != nil {
			t.Fatalf("MkdirAll returned error: %v", err)
		}
		if err := os.WriteFile(cache.queryPath(query), []byte(`{"registry_endpoint":"https://registry.example.com","kind":"search","namespace":"@team"`), 0600); err != nil {
			t.Fatalf("WriteFile returned error: %v", err)
		}

		got, stale, err := cache.fetchWithFallback(query,
			func() (*registry.CatalogVersionResponse, error) {
				return nil, errors.New("version unavailable")
			},
			func() (*registry.SearchResult, error) {
				return nil, errors.New("remote unavailable")
			},
		)
		if err == nil {
			t.Fatal("expected error when fallback cache is structurally corrupt")
		}
		if got != nil {
			t.Fatalf("expected no cached result, got %#v", got)
		}
		if stale {
			t.Fatal("expected stale fallback to be rejected when cache structure is corrupt")
		}
	})
}

func writeJSONFile(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
