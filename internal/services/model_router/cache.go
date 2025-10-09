package model_router

import (
	"context"
	"fmt"

	"adaptive-backend/internal/config"
	"adaptive-backend/internal/models"

	"github.com/botirk38/semanticcache"
	"github.com/botirk38/semanticcache/options"
	fiberlog "github.com/gofiber/fiber/v2/log"
)

const (
	defaultSemanticThreshold = 0.9
)

// DefaultCacheConfig returns default cache configuration
func DefaultCacheConfig() models.CacheConfig {
	return models.CacheConfig{
		Enabled:           true,
		SemanticThreshold: defaultSemanticThreshold,
	}
}

// ModelRouterCache wraps the semanticcache library for protocol manager specific operations
type ModelRouterCache struct {
	cache             *semanticcache.SemanticCache[string, models.ModelSelectionResponse]
	semanticThreshold float32
}

// NewModelRouterCache creates a new protocol manager cache instance
func NewModelRouterCache(cfg *config.Config) (*ModelRouterCache, error) {
	fiberlog.Info("ModelRouterCache: Initializing cache")

	// ModelRouter is optional
	if cfg.ModelRouter == nil {
		return nil, fmt.Errorf("model router not configured")
	}

	// Get semantic cache configuration
	cacheConfig := cfg.ModelRouter.Cache

	// Validate and set default threshold if invalid
	threshold := cacheConfig.SemanticThreshold
	if threshold <= 0 || threshold > 1 {
		return nil, fmt.Errorf("invalid semantic threshold %.2f; must be in (0.0, 1.0]", threshold)
	}

	fiberlog.Debugf("ModelRouterCache: Configuration - enabled=%t, backend=%s, threshold=%.2f",
		cacheConfig.Enabled, cacheConfig.Backend, threshold)

	apiKey := cacheConfig.OpenAIAPIKey
	if apiKey == "" {
		fiberlog.Error("ModelRouterCache: OpenAI API key not set in cache configuration")
		return nil, fmt.Errorf("OpenAI API key not set in cache configuration")
	}
	fiberlog.Debug("ModelRouterCache: OpenAI API key found")

	// Determine embedding model
	embedModel := cacheConfig.EmbeddingModel
	if embedModel == "" {
		embedModel = "text-embedding-3-large"
	}

	// Create semantic cache with backend based on configuration
	fiberlog.Debug("ModelRouterCache: Creating semantic cache")
	var cache *semanticcache.SemanticCache[string, models.ModelSelectionResponse]
	var err error

	backend := cacheConfig.Backend
	if backend == "" {
		backend = models.CacheBackendRedis // Default to Redis for backward compatibility
		fiberlog.Warn("ModelRouterCache: Backend not specified, defaulting to redis")
	}

	switch backend {
	case models.CacheBackendMemory:
		capacity := cacheConfig.Capacity
		if capacity <= 0 {
			capacity = 1000 // Default capacity
			fiberlog.Warnf("ModelRouterCache: Invalid or missing capacity, using default %d", capacity)
		}
		fiberlog.Debugf("ModelRouterCache: Using in-memory LRU backend with capacity=%d", capacity)
		cache, err = semanticcache.New(
			options.WithOpenAIProvider[string, models.ModelSelectionResponse](apiKey, embedModel),
			options.WithLRUBackend[string, models.ModelSelectionResponse](capacity),
		)

	case models.CacheBackendRedis:
		// Get Redis URL from cache config first, fallback to PromptCache
		redisURL := cacheConfig.RedisURL
		if redisURL == "" && cfg.PromptCache != nil {
			redisURL = cfg.PromptCache.RedisURL
		}
		if redisURL == "" {
			fiberlog.Error("ModelRouterCache: redis URL not set - please configure redis_url in cache or prompt_cache")
			return nil, fmt.Errorf("redis URL not set - please configure redis_url in cache or prompt_cache")
		}
		fiberlog.Debugf("ModelRouterCache: Using Redis backend with URL=%s", redisURL)
		cache, err = semanticcache.New(
			options.WithOpenAIProvider[string, models.ModelSelectionResponse](apiKey, embedModel),
			options.WithRedisBackend[string, models.ModelSelectionResponse](redisURL, 0),
		)

	default:
		return nil, fmt.Errorf("unsupported cache backend: %s (supported: redis, memory)", backend)
	}

	if err != nil {
		fiberlog.Errorf("ModelRouterCache: Failed to create semantic cache: %v", err)
		return nil, fmt.Errorf("failed to create semantic cache: %w", err)
	}
	fiberlog.Info("ModelRouterCache: Semantic cache created successfully")

	return &ModelRouterCache{
		cache:             cache,
		semanticThreshold: float32(threshold),
	}, nil
}

// Lookup searches for a cached protocol response using exact match first, then semantic similarity with custom threshold
func (pmc *ModelRouterCache) Lookup(ctx context.Context, prompt, requestID string, threshold float32) (*models.ModelSelectionResponse, string, bool) {
	fiberlog.Debugf("[%s] ModelRouterCache: Starting cache lookup", requestID)

	// 1) First try exact key matching
	fiberlog.Debugf("[%s] ModelRouterCache: Trying exact key match", requestID)
	if hit, found, err := pmc.cache.Get(ctx, prompt); found && err == nil {
		fiberlog.Infof("[%s] ModelRouterCache: Exact cache hit", requestID)
		return &hit, models.CacheTierSemanticExact, true
	} else if err != nil {
		fiberlog.Errorf("[%s] ModelRouterCache: Error during exact lookup: %v", requestID, err)
	}
	fiberlog.Debugf("[%s] ModelRouterCache: No exact match found", requestID)

	// 2) If no exact match, try semantic similarity search with provided threshold
	fiberlog.Debugf("[%s] ModelRouterCache: Trying semantic similarity search (threshold: %.2f)", requestID, threshold)
	if match, err := pmc.cache.Lookup(ctx, prompt, threshold); err == nil && match != nil {
		fiberlog.Infof("[%s] ModelRouterCache: Semantic cache hit", requestID)
		return &match.Value, models.CacheTierSemanticSimilar, true
	} else if err != nil {
		fiberlog.Errorf("[%s] ModelRouterCache: Error during semantic lookup: %v", requestID, err)
	} else {
		fiberlog.Debugf("[%s] ModelRouterCache: No semantic match found", requestID)
	}

	fiberlog.Debugf("[%s] ModelRouterCache: Cache miss", requestID)
	return nil, "", false
}

// LookupAsync searches for a cached protocol response using exact match first, then semantic similarity with custom threshold
// This method coordinates Get (exact) and Lookup (semantic) operations sequentially for proper cache tier determination
func (pmc *ModelRouterCache) LookupAsync(ctx context.Context, prompt, requestID string, threshold float32) <-chan semanticcache.LookupResult[models.ModelSelectionResponse] {
	resultCh := make(chan semanticcache.LookupResult[models.ModelSelectionResponse], 1)

	// Spawn goroutine to coordinate GetAsync -> LookupAsync sequence
	// This is necessary because we need to try GetAsync first (fast), then LookupAsync (slow) if GetAsync misses
	go func() {
		defer close(resultCh)

		fiberlog.Debugf("[%s] ModelRouterCache: Starting cache lookup", requestID)

		// Try exact key matching first (O(1) - fast)
		fiberlog.Debugf("[%s] ModelRouterCache: Trying exact key match", requestID)
		getCh := pmc.cache.GetAsync(ctx, prompt)
		getResult := <-getCh

		if getResult.Found && getResult.Error == nil {
			fiberlog.Infof("[%s] ModelRouterCache: Exact cache hit", requestID)
			resultCh <- semanticcache.LookupResult[models.ModelSelectionResponse]{
				Match: &semanticcache.Match[models.ModelSelectionResponse]{
					Value: getResult.Value,
					Score: 1.0, // Exact match score
				},
				Error: nil,
			}
			return
		} else if getResult.Error != nil {
			fiberlog.Errorf("[%s] ModelRouterCache: Error during exact lookup: %v", requestID, getResult.Error)
		}

		// Try semantic similarity search (O(n) - slower, requires embedding computation)
		fiberlog.Debugf("[%s] ModelRouterCache: Trying semantic similarity search (threshold: %.2f)", requestID, threshold)
		lookupCh := pmc.cache.LookupAsync(ctx, prompt, threshold)
		lookupResult := <-lookupCh

		if lookupResult.Match != nil && lookupResult.Error == nil {
			fiberlog.Infof("[%s] ModelRouterCache: Semantic cache hit (score: %.2f)", requestID, lookupResult.Match.Score)
			resultCh <- lookupResult
			return
		} else if lookupResult.Error != nil {
			fiberlog.Errorf("[%s] ModelRouterCache: Error during semantic lookup: %v", requestID, lookupResult.Error)
		}

		fiberlog.Debugf("[%s] ModelRouterCache: Cache miss", requestID)
		resultCh <- semanticcache.LookupResult[models.ModelSelectionResponse]{Match: nil, Error: nil}
	}()

	return resultCh
}

// StoreAsync saves a protocol response to the cache asynchronously (fire-and-forget)
func (pmc *ModelRouterCache) StoreAsync(ctx context.Context, prompt string, resp models.ModelSelectionResponse, requestID string) {
	fiberlog.Debugf("[%s] ModelRouterCache: Storing model response (fire-and-forget, model: %s/%s)", requestID, resp.Provider, resp.Model)
	pmc.cache.SetAsync(ctx, prompt, prompt, resp)
}

// DeleteAsync removes a cache entry asynchronously (fire-and-forget)
func (pmc *ModelRouterCache) DeleteAsync(ctx context.Context, prompt, provider, requestID string) {
	fiberlog.Debugf("[%s] Invalidating cache entry for provider %s (fire-and-forget)", requestID, provider)
	pmc.cache.DeleteAsync(ctx, prompt)
}

// Close closes the cache and releases resources
func (pmc *ModelRouterCache) Close() error {
	if pmc.cache != nil {
		return pmc.cache.Close()
	}
	return nil
}
