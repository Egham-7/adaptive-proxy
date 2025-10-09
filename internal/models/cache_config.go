package models

// CacheBackendType represents the type of cache backend to use
type CacheBackendType string

const (
	CacheBackendRedis  CacheBackendType = "redis"
	CacheBackendMemory CacheBackendType = "memory"
)

// CacheConfig holds configuration for prompt caching (optional)
type CacheConfig struct {
	// Backend configuration
	Backend  CacheBackendType `json:"backend,omitzero" yaml:"backend"`     // "redis" or "memory"
	RedisURL string           `json:"redis_url,omitzero" yaml:"redis_url"` // Required if backend is "redis"
	Capacity int              `json:"capacity,omitzero" yaml:"capacity"`   // Required if backend is "memory" (LRU cache size)

	// Cache behavior
	Enabled           bool    `json:"enabled,omitzero" yaml:"enabled"`
	SemanticThreshold float64 `json:"semantic_threshold,omitzero" yaml:"semantic_threshold"`
	OpenAIAPIKey      string  `json:"openai_api_key,omitzero" yaml:"openai_api_key"`
	EmbeddingModel    string  `json:"embedding_model,omitzero" yaml:"embedding_model"`
}
