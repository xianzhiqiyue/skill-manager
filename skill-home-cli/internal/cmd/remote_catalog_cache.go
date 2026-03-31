package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/skill-home/cli/internal/registry"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type remoteCatalogQuery struct {
	Kind      string
	Namespace string
	Query     string
	Tags      []string
	Page      int
	PerPage   int
}

type remoteCatalogState struct {
	RegistryEndpoint string    `json:"registry_endpoint"`
	CatalogVersion   int64     `json:"catalog_version"`
	CheckedAt        time.Time `json:"checked_at"`
	UpdatedAt        time.Time `json:"updated_at,omitempty"`
}

type remoteCatalogQueryCache struct {
	RegistryEndpoint string                 `json:"registry_endpoint"`
	Kind             string                 `json:"kind"`
	Namespace        string                 `json:"namespace"`
	Query            string                 `json:"query,omitempty"`
	Tags             []string               `json:"tags,omitempty"`
	Page             int                    `json:"page"`
	PerPage          int                    `json:"per_page"`
	CatalogVersion   int64                  `json:"catalog_version"`
	CachedAt         time.Time              `json:"cached_at"`
	Result           *registry.SearchResult `json:"result,omitempty"`
}

type remoteCatalogCache struct {
	baseDir  string
	endpoint string
}

var writeJSONAtomicFunc = writeJSONAtomic

func newDefaultRemoteCatalogCache() (*remoteCatalogCache, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return newRemoteCatalogCache(filepath.Join(home, ".config", "skill-home", "cache", "remote-catalog"), registryEndpoint()), nil
}

func newRemoteCatalogCache(baseDir, endpoint string) *remoteCatalogCache {
	return &remoteCatalogCache{
		baseDir:  baseDir,
		endpoint: normalizeRegistryEndpoint(endpoint),
	}
}

func (c *remoteCatalogCache) cacheDir() string {
	return filepath.Join(c.baseDir, hashString(c.endpoint))
}

func (c *remoteCatalogCache) statePath() string {
	return filepath.Join(c.cacheDir(), "state.json")
}

func (c *remoteCatalogCache) queryPath(query remoteCatalogQuery) string {
	return filepath.Join(c.cacheDir(), "queries", c.cacheKey(query)+".json")
}

func (c *remoteCatalogCache) cacheKey(query remoteCatalogQuery) string {
	key := remoteCatalogQuery{
		Kind:      query.Kind,
		Namespace: normalizeNamespace(query.Namespace),
		Query:     strings.TrimSpace(query.Query),
		Tags:      normalizeTags(query.Tags),
		Page:      query.Page,
		PerPage:   query.PerPage,
	}
	payload := struct {
		Endpoint string             `json:"endpoint"`
		Query    remoteCatalogQuery `json:"query"`
	}{
		Endpoint: c.endpoint,
		Query:    key,
	}
	data, _ := json.Marshal(payload)
	return hashBytes(data)
}

func (c *remoteCatalogCache) readFresh(query remoteCatalogQuery, catalogVersion int64) (*registry.SearchResult, bool, error) {
	queryCache, err := c.readQueryCache(query)
	if err != nil {
		return nil, false, nil
	}
	state, err := c.readState()
	if err != nil {
		return nil, false, nil
	}

	if !queryCache.isFreshCandidateFor(c.endpoint, query) {
		return nil, false, nil
	}
	if !state.isFreshCandidateFor(c.endpoint) {
		return nil, false, nil
	}
	if state.CatalogVersion != catalogVersion {
		return nil, false, nil
	}
	if queryCache.CatalogVersion != catalogVersion {
		return nil, false, nil
	}
	if state.CatalogVersion != queryCache.CatalogVersion {
		return nil, false, nil
	}

	return queryCache.Result, true, nil
}

func (c *remoteCatalogCache) fetchWithFallback(
	query remoteCatalogQuery,
	getVersion func() (*registry.CatalogVersionResponse, error),
	fetchRemote func() (*registry.SearchResult, error),
) (*registry.SearchResult, bool, error) {
	versionResp, versionErr := getVersion()
	if versionErr == nil {
		if cached, fresh, err := c.readFresh(query, versionResp.CatalogVersion); err == nil && fresh {
			return cached, false, nil
		}
		result, err := fetchRemote()
		if err == nil {
			_ = c.writeSnapshot(query, versionResp, result)
			return result, false, nil
		}
		if fallback, fallbackErr := c.readFallback(query); fallbackErr == nil {
			return fallback, true, nil
		}
		return nil, false, err
	}

	if fallback, fallbackErr := c.readFallback(query); fallbackErr == nil {
		return fallback, true, nil
	}
	return nil, false, versionErr
}

func (c *remoteCatalogCache) writeSnapshot(query remoteCatalogQuery, version *registry.CatalogVersionResponse, result *registry.SearchResult) error {
	if version == nil {
		return errors.New("missing catalog version")
	}
	if result == nil {
		return errors.New("missing search result")
	}
	normalized := normalizeQuery(query)
	queryCache := remoteCatalogQueryCache{
		RegistryEndpoint: c.endpoint,
		Kind:             normalized.Kind,
		Namespace:        normalized.Namespace,
		Query:            normalized.Query,
		Tags:             normalized.Tags,
		Page:             normalized.Page,
		PerPage:          normalized.PerPage,
		CatalogVersion:   version.CatalogVersion,
		CachedAt:         time.Now().UTC(),
		Result:           result,
	}
	state := remoteCatalogState{
		RegistryEndpoint: c.endpoint,
		CatalogVersion:   version.CatalogVersion,
		CheckedAt:        time.Now().UTC(),
		UpdatedAt:        version.UpdatedAt,
	}

	if err := writeJSONAtomicFunc(c.queryPath(query), queryCache); err != nil {
		return err
	}
	if err := writeJSONAtomicFunc(c.statePath(), state); err != nil {
		return err
	}
	return nil
}

func (c *remoteCatalogCache) readState() (*remoteCatalogState, error) {
	data, err := os.ReadFile(c.statePath())
	if err != nil {
		return nil, err
	}
	var state remoteCatalogState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func (c *remoteCatalogCache) readQueryCache(query remoteCatalogQuery) (*remoteCatalogQueryCache, error) {
	data, err := os.ReadFile(c.queryPath(query))
	if err != nil {
		return nil, err
	}
	var cached remoteCatalogQueryCache
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, err
	}
	return &cached, nil
}

func (c *remoteCatalogCache) readFallback(query remoteCatalogQuery) (*registry.SearchResult, error) {
	cached, err := c.readQueryCache(query)
	if err != nil {
		return nil, err
	}
	if !cached.isFallbackCandidateFor(c.endpoint, query) {
		return nil, errors.New("cached query is not usable")
	}
	return cached.Result, nil
}

func (q remoteCatalogQueryCache) isFallbackCandidateFor(endpoint string, query remoteCatalogQuery) bool {
	if normalizeRegistryEndpoint(q.RegistryEndpoint) != endpoint {
		return false
	}
	norm := normalizeQuery(query)
	if q.Kind != norm.Kind || normalizeNamespace(q.Namespace) != norm.Namespace || strings.TrimSpace(q.Query) != norm.Query {
		return false
	}
	if q.Page != norm.Page || q.PerPage != norm.PerPage {
		return false
	}
	if !equalTags(q.Tags, norm.Tags) {
		return false
	}
	return q.hasCompleteStructure()
}

func (q remoteCatalogQueryCache) isFreshCandidateFor(endpoint string, query remoteCatalogQuery) bool {
	return q.isFallbackCandidateFor(endpoint, query)
}

func (q remoteCatalogQueryCache) hasCompleteStructure() bool {
	if normalizeRegistryEndpoint(q.RegistryEndpoint) == "" {
		return false
	}
	if q.Kind == "" || q.Page <= 0 || q.PerPage <= 0 {
		return false
	}
	if q.CachedAt.IsZero() {
		return false
	}
	return searchResultHasCompleteStructure(q.Result)
}

func (s remoteCatalogState) isFreshCandidateFor(endpoint string) bool {
	return normalizeRegistryEndpoint(s.RegistryEndpoint) == endpoint && !s.CheckedAt.IsZero()
}

func searchResultHasCompleteStructure(r *registry.SearchResult) bool {
	if r == nil {
		return false
	}
	if r.Page <= 0 || r.PerPage <= 0 {
		return false
	}
	return true
}

func normalizeQuery(query remoteCatalogQuery) remoteCatalogQuery {
	return remoteCatalogQuery{
		Kind:      query.Kind,
		Namespace: normalizeNamespace(query.Namespace),
		Query:     strings.TrimSpace(query.Query),
		Tags:      normalizeTags(query.Tags),
		Page:      query.Page,
		PerPage:   query.PerPage,
	}
}

func normalizeRegistryEndpoint(endpoint string) string {
	return strings.TrimRight(strings.TrimSpace(endpoint), "/")
}

func normalizeNamespace(namespace string) string {
	return strings.TrimPrefix(strings.TrimSpace(namespace), "@")
}

func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return []string{}
	}
	cleaned := make([]string, len(tags))
	copy(cleaned, tags)
	sort.Strings(cleaned)
	return cleaned
}

func equalTags(left, right []string) bool {
	if len(left) == 0 && len(right) == 0 {
		return true
	}
	leftCopy := normalizeTags(left)
	rightCopy := normalizeTags(right)
	if len(leftCopy) != len(rightCopy) {
		return false
	}
	for i := range leftCopy {
		if leftCopy[i] != rightCopy[i] {
			return false
		}
	}
	return true
}

func hashString(value string) string {
	return hashBytes([]byte(value))
}

func hashBytes(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

func writeJSONAtomic(path string, value any) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".tmp-remote-catalog-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()

	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
