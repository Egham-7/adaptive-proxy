package cache

import (
	"context"
	"fmt"

	"adaptive-backend/internal/models"
	"adaptive-backend/internal/utils"

	"github.com/botirk38/semanticcache"
	"github.com/botirk38/semanticcache/options"
	fiberlog "github.com/gofiber/fiber/v2/log"
	"github.com/redis/go-redis/v9"
)

const (
	defaultSemanticThreshold = 0.99
)

// OpenAIPromptCache provides semantic caching for OpenAI prompt responses
type OpenAIPromptCache struct {
	client            *redis.Client
	semanticCache     *semanticcache.SemanticCache[string, models.ChatCompletion]
	semanticThreshold float32
}

// NewOpenAIPromptCache creates a new OpenAI prompt cache instance with semantic caching support
func NewOpenAIPromptCache(redisClient *redis.Client, config models.CacheConfig) (*OpenAIPromptCache, error) {
	fiberlog.Info("OpenAIPromptCache: Initializing with semantic cache support")

	pc := &OpenAIPromptCache{
		client: redisClient,
	}

	// Initialize semantic cache if enabled and properly configured
	if config.Enabled && config.OpenAIAPIKey != "" {
		threshold := config.SemanticThreshold
		if threshold <= 0 || threshold > 1 {
			threshold = defaultSemanticThreshold
			fiberlog.Warnf("OpenAIPromptCache: Invalid threshold value %.2f, using default %.2f", config.SemanticThreshold, defaultSemanticThreshold)
		}
		pc.semanticThreshold = float32(threshold)

		// Determine embedding model
		embedModel := config.EmbeddingModel
		if embedModel == "" {
			embedModel = "text-embedding-3-small"
		}

		// Determine backend type
		backend := config.Backend
		if backend == "" {
			backend = models.CacheBackendRedis // Default to Redis for backward compatibility
			fiberlog.Warn("OpenAIPromptCache: Backend not specified, defaulting to redis")
		}

		fiberlog.Debugf("OpenAIPromptCache: Initializing semantic cache with backend=%s, threshold=%.2f", backend, threshold)

		var semanticCache *semanticcache.SemanticCache[string, models.ChatCompletion]
		var err error

		switch backend {
		case models.CacheBackendMemory:
			capacity := config.Capacity
			if capacity <= 0 {
				capacity = 1000 // Default capacity
				fiberlog.Warnf("OpenAIPromptCache: Invalid or missing capacity, using default %d", capacity)
			}
			fiberlog.Debugf("OpenAIPromptCache: Using in-memory LRU backend with capacity=%d", capacity)
			semanticCache, err = semanticcache.New(
				options.WithOpenAIProvider[string, models.ChatCompletion](config.OpenAIAPIKey, embedModel),
				options.WithLRUBackend[string, models.ChatCompletion](capacity),
			)

		case models.CacheBackendRedis:
			redisURL := config.RedisURL
			if redisURL == "" {
				fiberlog.Error("OpenAIPromptCache: redis URL not set for redis backend")
				return nil, fmt.Errorf("redis URL not set for redis backend")
			}
			fiberlog.Debugf("OpenAIPromptCache: Using Redis backend with URL=%s", redisURL)
			semanticCache, err = semanticcache.New(
				options.WithOpenAIProvider[string, models.ChatCompletion](config.OpenAIAPIKey, embedModel),
				options.WithRedisBackend[string, models.ChatCompletion](redisURL, 0),
			)

		default:
			return nil, fmt.Errorf("unsupported cache backend: %s (supported: redis, memory)", backend)
		}

		if err != nil {
			fiberlog.Errorf("OpenAIPromptCache: Failed to create semantic cache: %v", err)
			return nil, fmt.Errorf("failed to create semantic cache: %w", err)
		}

		pc.semanticCache = semanticCache
		fiberlog.Info("OpenAIPromptCache: Semantic cache initialized successfully")
	} else {
		fiberlog.Info("OpenAIPromptCache: Semantic cache disabled, using basic Redis cache")
	}

	return pc, nil
}

// Get retrieves a cached response for the given request using semantic similarity
func (pc *OpenAIPromptCache) Get(ctx context.Context, req *models.ChatCompletionRequest, requestID string) (*models.ChatCompletion, string, bool) {
	if req.PromptCache == nil || !req.PromptCache.Enabled {
		fiberlog.Debugf("[%s] OpenAIPromptCache: Cache disabled for request", requestID)
		return nil, "", false
	}

	if pc.semanticCache == nil {
		fiberlog.Debugf("[%s] OpenAIPromptCache: Semantic cache not initialized", requestID)
		return nil, "", false
	}

	// Use threshold override if provided in request, otherwise use cache default
	threshold := pc.semanticThreshold
	if req.PromptCache.SemanticThreshold > 0 {
		threshold = float32(req.PromptCache.SemanticThreshold)
		fiberlog.Debugf("[%s] OpenAIPromptCache: Using threshold override: %.2f", requestID, req.PromptCache.SemanticThreshold)
	}

	// Extract prompt from messages for semantic search
	prompt, err := utils.FindLastUserMessage(req.Messages)
	if err != nil {
		fiberlog.Debugf("[%s] OpenAIPromptCache: %v", requestID, err)
		return nil, "", false
	}

	fiberlog.Debugf("[%s] OpenAIPromptCache: Starting semantic cache lookup", requestID)

	// Try exact match first
	fiberlog.Debugf("[%s] OpenAIPromptCache: Trying exact key match", requestID)
	if hit, found, err := pc.semanticCache.Get(ctx, prompt); found && err == nil {
		fiberlog.Infof("[%s] OpenAIPromptCache: Exact cache hit", requestID)
		return &hit, models.CacheTierSemanticExact, true
	} else if err != nil {
		fiberlog.Errorf("[%s] OpenAIPromptCache: Error during exact lookup: %v", requestID, err)
	}

	// Try semantic similarity search with provided threshold
	fiberlog.Debugf("[%s] OpenAIPromptCache: Trying semantic similarity search (threshold: %.2f)", requestID, threshold)
	if match, err := pc.semanticCache.Lookup(ctx, prompt, threshold); err == nil && match != nil {
		fiberlog.Infof("[%s] OpenAIPromptCache: Semantic cache hit", requestID)
		return &match.Value, models.CacheTierSemanticSimilar, true
	} else if err != nil {
		fiberlog.Errorf("[%s] OpenAIPromptCache: Error during semantic lookup: %v", requestID, err)
	}

	fiberlog.Debugf("[%s] OpenAIPromptCache: Semantic cache miss", requestID)
	return nil, "", false
}

// Set stores a response in the cache with the configured TTL
func (pc *OpenAIPromptCache) Set(ctx context.Context, req *models.ChatCompletionRequest, response *models.ChatCompletion, requestID string) error {
	if req.PromptCache == nil || !req.PromptCache.Enabled {
		fiberlog.Debugf("[%s] OpenAIPromptCache: Cache disabled, skipping storage", requestID)
		return nil
	}

	if pc.semanticCache == nil {
		fiberlog.Debugf("[%s] OpenAIPromptCache: Semantic cache not initialized, skipping storage", requestID)
		return nil
	}

	// Extract prompt from messages for semantic storage
	prompt, err := utils.FindLastUserMessage(req.Messages)
	if err != nil {
		fiberlog.Debugf("[%s] OpenAIPromptCache: %v, skipping storage", requestID, err)
		return nil
	}

	fiberlog.Debugf("[%s] OpenAIPromptCache: Storing response in semantic cache", requestID)
	err = pc.semanticCache.Set(ctx, prompt, prompt, *response)
	if err != nil {
		fiberlog.Errorf("[%s] OpenAIPromptCache: Failed to store in semantic cache: %v", requestID, err)
		return fmt.Errorf("failed to store in semantic cache: %w", err)
	}

	fiberlog.Debugf("[%s] OpenAIPromptCache: Successfully stored in semantic cache", requestID)
	return nil
}

// SetAsync stores a response asynchronously in the cache
func (pc *OpenAIPromptCache) SetAsync(ctx context.Context, req *models.ChatCompletionRequest, response *models.ChatCompletion, requestID string) <-chan error {
	if req.PromptCache == nil || !req.PromptCache.Enabled || pc.semanticCache == nil {
		errCh := make(chan error, 1)
		errCh <- nil
		close(errCh)
		return errCh
	}

	prompt, err := utils.FindLastUserMessage(req.Messages)
	if err != nil {
		errCh := make(chan error, 1)
		errCh <- nil
		close(errCh)
		return errCh
	}

	fiberlog.Debugf("[%s] OpenAIPromptCache: Storing response in semantic cache (async)", requestID)
	return pc.semanticCache.SetAsync(ctx, prompt, prompt, *response)
}

// GetAsync retrieves a cached response asynchronously using semantic similarity
func (pc *OpenAIPromptCache) GetAsync(ctx context.Context, req *models.ChatCompletionRequest, requestID string) <-chan semanticcache.GetResult[models.ChatCompletion] {
	if req.PromptCache == nil || !req.PromptCache.Enabled || pc.semanticCache == nil {
		resultCh := make(chan semanticcache.GetResult[models.ChatCompletion], 1)
		resultCh <- semanticcache.GetResult[models.ChatCompletion]{Found: false}
		close(resultCh)
		return resultCh
	}

	prompt, err := utils.FindLastUserMessage(req.Messages)
	if err != nil {
		resultCh := make(chan semanticcache.GetResult[models.ChatCompletion], 1)
		resultCh <- semanticcache.GetResult[models.ChatCompletion]{Error: err}
		close(resultCh)
		return resultCh
	}

	fiberlog.Debugf("[%s] OpenAIPromptCache: Getting from semantic cache (async)", requestID)
	return pc.semanticCache.GetAsync(ctx, prompt)
}

// LookupAsync performs semantic similarity lookup asynchronously
func (pc *OpenAIPromptCache) LookupAsync(ctx context.Context, req *models.ChatCompletionRequest, requestID string) <-chan semanticcache.LookupResult[models.ChatCompletion] {
	if req.PromptCache == nil || !req.PromptCache.Enabled || pc.semanticCache == nil {
		resultCh := make(chan semanticcache.LookupResult[models.ChatCompletion], 1)
		resultCh <- semanticcache.LookupResult[models.ChatCompletion]{Match: nil}
		close(resultCh)
		return resultCh
	}

	prompt, err := utils.FindLastUserMessage(req.Messages)
	if err != nil {
		resultCh := make(chan semanticcache.LookupResult[models.ChatCompletion], 1)
		resultCh <- semanticcache.LookupResult[models.ChatCompletion]{Error: err}
		close(resultCh)
		return resultCh
	}

	threshold := pc.semanticThreshold
	if req.PromptCache.SemanticThreshold > 0 {
		threshold = float32(req.PromptCache.SemanticThreshold)
	}

	fiberlog.Debugf("[%s] OpenAIPromptCache: Semantic lookup (async, threshold: %.2f)", requestID, threshold)
	return pc.semanticCache.LookupAsync(ctx, prompt, threshold)
}

// Flush clears all prompt cache entries
func (pc *OpenAIPromptCache) Flush(ctx context.Context) error {
	if pc.semanticCache == nil {
		fiberlog.Debug("OpenAIPromptCache: Semantic cache not initialized, nothing to flush")
		return nil
	}

	if err := pc.semanticCache.Flush(ctx); err != nil {
		fiberlog.Errorf("OpenAIPromptCache: Failed to flush semantic cache: %v", err)
		return fmt.Errorf("failed to flush semantic cache: %w", err)
	}

	return nil
}

// Close closes the Redis connection and semantic cache
func (pc *OpenAIPromptCache) Close() error {
	if pc.semanticCache != nil {
		if err := pc.semanticCache.Close(); err != nil {
			// Return first error encountered
			if pc.client != nil {
				_ = pc.client.Close() // Still attempt to close client
			}
			return err
		}
	}
	if pc.client != nil {
		return pc.client.Close()
	}
	return nil
}
