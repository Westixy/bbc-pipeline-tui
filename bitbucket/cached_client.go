package bitbucket

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

// CachedClient wraps a Client with an in-memory cache for GET requests.
// POST/PUT/DELETE (mutating operations) are never cached and pass through
// to the underlying Client directly.
type CachedClient struct {
	*Client
	cache *Cache
}

// NewCachedClient creates a new CachedClient with an empty cache.
func NewCachedClient(client *Client) *CachedClient {
	return &CachedClient{
		Client: client,
		cache:  NewCache(),
	}
}

// InvalidateCache clears all cached entries.
func (cc *CachedClient) InvalidateCache() {
	cc.cache.InvalidateAll()
}

// InvalidateCacheForRepo clears cache entries for a specific workspace/repo.
func (cc *CachedClient) InvalidateCacheForRepo(workspace, repoSlug string) {
	prefix := fmt.Sprintf("/repositories/%s/%s/", url.PathEscape(workspace), url.PathEscape(repoSlug))
	cc.cache.InvalidatePrefix(prefix)
}

// InvalidateCacheForWorkspace clears cache entries for a specific workspace.
func (cc *CachedClient) InvalidateCacheForWorkspace(workspace string) {
	prefix := fmt.Sprintf("/repositories/%s", url.PathEscape(workspace))
	cc.cache.InvalidatePrefix(prefix)
}

// cacheLen returns the number of cached entries (for diagnostics).
func (cc *CachedClient) cacheLen() int {
	return cc.cache.Len()
}

// --- Cached read methods ---

// ListPipelines fetches pipelines with caching.
func (cc *CachedClient) ListPipelines(workspace, repoSlug string, params *ListPipelinesParams) (*PaginatedPipelines, error) {
	basePath := BuildPipelinePath(workspace, repoSlug, "pipelines")
	q := url.Values{}
	if params != nil {
		if params.Pagelen > 0 {
			q.Set("pagelen", fmt.Sprintf("%d", params.Pagelen))
		}
		if params.Page > 0 {
			q.Set("page", fmt.Sprintf("%d", params.Page))
		}
		if params.Sort != "" {
			q.Set("sort", params.Sort)
		}
		if params.Fields != "" {
			q.Set("fields", params.Fields)
		}
	}
	path := basePath
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	return cachedGet[PaginatedPipelines](cc, path, DefaultCacheTTL, DefaultMaxEntrySize)
}

// GetPipeline fetches a single pipeline with caching.
func (cc *CachedClient) GetPipeline(workspace, repoSlug, pipelineUUID string) (*Pipeline, error) {
	path := BuildPipelinePath(workspace, repoSlug, "pipelines/"+url.PathEscape(pipelineUUID))
	return cachedGet[Pipeline](cc, path, DefaultCacheTTL, DefaultMaxEntrySize)
}

// ListPipelineSteps fetches steps for a pipeline with caching.
func (cc *CachedClient) ListPipelineSteps(workspace, repoSlug, pipelineUUID string) (*PaginatedSteps, error) {
	path := BuildPipelinePath(workspace, repoSlug, "pipelines/"+url.PathEscape(pipelineUUID)+"/steps")
	return cachedGet[PaginatedSteps](cc, path, DefaultCacheTTL, DefaultMaxEntrySize)
}

// GetStepLog fetches a step log with caching (longer TTL, size cap).
func (cc *CachedClient) GetStepLog(workspace, repoSlug, pipelineUUID, stepUUID string) (string, error) {
	path := BuildPipelinePath(workspace, repoSlug,
		"pipelines/"+url.PathEscape(pipelineUUID)+"/steps/"+url.PathEscape(stepUUID)+"/log")

	// Check cache
	key := path
	if data, ok := cc.cache.Get(key); ok {
		return string(data), nil
	}

	// Fetch
	logContent, err := cc.Client.GetStepLog(workspace, repoSlug, pipelineUUID, stepUUID)
	if err != nil {
		return "", err
	}

	// Store
	cc.cache.Set(key, []byte(logContent), LogCacheTTL, DefaultMaxEntrySize)

	return logContent, nil
}

// GetFullURL fetches a full URL with caching.
func (cc *CachedClient) GetFullURL(fullURL string, dest interface{}) error {
	// Check cache
	if data, ok := cc.cache.Get(fullURL); ok {
		return json.Unmarshal(data, dest)
	}

	// Fetch
	if err := cc.Client.GetFullURL(fullURL, dest); err != nil {
		return err
	}

	// Store
	if data, err := json.Marshal(dest); err == nil {
		cc.cache.Set(fullURL, data, DefaultCacheTTL, DefaultMaxEntrySize)
	}

	return nil
}

// ListRepositories fetches the first page of repositories in a workspace with caching.
func (cc *CachedClient) ListRepositories(workspace string) (*PaginatedRepositories, error) {
	path := fmt.Sprintf("/repositories/%s?pagelen=50", url.PathEscape(workspace))
	return cachedGet[PaginatedRepositories](cc, path, DefaultCacheTTL, DefaultMaxEntrySize)
}

// ListAllRepositories fetches every repository in a workspace, following pagination,
// with caching.
func (cc *CachedClient) ListAllRepositories(workspace string) ([]Repository, error) {
	return cc.Client.ListAllRepositories(workspace)
}

// --- Non-cached mutating methods (passthrough) ---

// TriggerPipeline triggers a new pipeline (never cached).
func (cc *CachedClient) TriggerPipeline(workspace, repoSlug string, req TriggerPipelineRequest) (*Pipeline, error) {
	return cc.Client.TriggerPipeline(workspace, repoSlug, req)
}

// StopPipeline stops a running pipeline (never cached).
func (cc *CachedClient) StopPipeline(workspace, repoSlug, pipelineUUID string) error {
	return cc.Client.StopPipeline(workspace, repoSlug, pipelineUUID)
}

// ListPipelineVariables fetches repository-level pipeline config variables with caching.
func (cc *CachedClient) ListPipelineVariables(workspace, repoSlug string) (*PaginatedPipelineVariables, error) {
	path := BuildPipelinePath(workspace, repoSlug, "pipelines_config/variables")
	return cachedGet[PaginatedPipelineVariables](cc, path, DefaultCacheTTL, DefaultMaxEntrySize)
}

// --- Generic cached GET helper ---

func cachedGet[T any](cc *CachedClient, path string, ttl time.Duration, maxSize int) (*T, error) {
	// Check cache
	key := path
	if data, ok := cc.cache.Get(key); ok {
		var result T
		if err := json.Unmarshal(data, &result); err == nil {
			return &result, nil
		}
		// Invalid cache entry, fall through to fetch
	}

	// Fetch
	var result T
	if err := cc.doGet(path, &result); err != nil {
		return nil, err
	}

	// Store
	if data, err := json.Marshal(&result); err == nil {
		cc.cache.Set(key, data, ttl, maxSize)
	}

	return &result, nil
}