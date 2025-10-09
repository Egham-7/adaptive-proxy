// Package config provides fluent configuration builders for AdaptiveProxy.
package config

import (
	"time"

	"github.com/Egham-7/adaptive-proxy/internal/config"
	"github.com/Egham-7/adaptive-proxy/internal/models"

	"github.com/gofiber/fiber/v2"
)

// Builder provides a fluent interface for building AdaptiveProxy configurations.
type Builder struct {
	cfg              *config.Config
	middlewares      []fiber.Handler
	rateLimitConfig  *models.RateLimitConfig
	timeoutConfig    *models.TimeoutConfig
	enabledEndpoints map[string]bool
}

// New creates a new configuration builder with minimal defaults.
func New() *Builder {
	return &Builder{
		cfg: &config.Config{
			Server: models.ServerConfig{
				Port:           "8080",
				AllowedOrigins: "*",
				Environment:    "development",
				LogLevel:       "info",
			},
			Fallback: models.FallbackConfig{
				Mode:       "race",
				TimeoutMs:  30000,
				MaxRetries: 3,
				CircuitBreaker: &models.CircuitBreakerConfig{
					FailureThreshold: 5,
					SuccessThreshold: 3,
					TimeoutMs:        15000,
					ResetAfterMs:     60000,
				},
			},
			PromptCache: nil,
			ModelRouter: nil,
			Endpoints: models.EndpointsConfig{
				ChatCompletions: models.EndpointConfig{Providers: make(map[string]models.ProviderConfig)},
				Messages:        models.EndpointConfig{Providers: make(map[string]models.ProviderConfig)},
				SelectModel:     models.EndpointConfig{Providers: make(map[string]models.ProviderConfig)},
				Generate:        models.EndpointConfig{Providers: make(map[string]models.ProviderConfig)},
				CountTokens:     models.EndpointConfig{Providers: make(map[string]models.ProviderConfig)},
			},
		},
		middlewares:      []fiber.Handler{},
		enabledEndpoints: make(map[string]bool),
	}
}

// Server configuration

// Port sets the server port.
func (b *Builder) Port(port string) *Builder {
	b.cfg.Server.Port = port
	return b
}

// AllowedOrigins sets CORS allowed origins.
func (b *Builder) AllowedOrigins(origins string) *Builder {
	b.cfg.Server.AllowedOrigins = origins
	return b
}

// Environment sets the environment (development/production).
func (b *Builder) Environment(env string) *Builder {
	b.cfg.Server.Environment = env
	return b
}

// LogLevel sets the logging level (trace, debug, info, warn, error, fatal).
func (b *Builder) LogLevel(level string) *Builder {
	b.cfg.Server.LogLevel = level
	return b
}

// Prompt Cache configuration

// WithPromptCache enables prompt-response caching.
func (b *Builder) WithPromptCache(cfg models.CacheConfig) *Builder {
	// Set defaults
	if cfg.SemanticThreshold == 0 {
		cfg.SemanticThreshold = 0.9
	}
	if cfg.EmbeddingModel == "" {
		cfg.EmbeddingModel = "text-embedding-3-small"
	}

	b.cfg.PromptCache = &cfg
	return b
}

// Model Router configuration

// WithModelRouter enables intelligent model routing.
func (b *Builder) WithModelRouter(cfg models.ModelRouterConfig) *Builder {
	// Set defaults
	if cfg.CostBias == 0 {
		cfg.CostBias = 0.9
	}
	if cfg.Client.TimeoutMs == 0 {
		cfg.Client.TimeoutMs = 3000
	}
	if cfg.Cache.SemanticThreshold == 0 {
		cfg.Cache.SemanticThreshold = 0.95
	}
	if cfg.Client.CircuitBreaker.FailureThreshold == 0 {
		cfg.Client.CircuitBreaker.FailureThreshold = 3
	}
	if cfg.Client.CircuitBreaker.SuccessThreshold == 0 {
		cfg.Client.CircuitBreaker.SuccessThreshold = 2
	}
	if cfg.Client.CircuitBreaker.TimeoutMs == 0 {
		cfg.Client.CircuitBreaker.TimeoutMs = 5000
	}
	if cfg.Client.CircuitBreaker.ResetAfterMs == 0 {
		cfg.Client.CircuitBreaker.ResetAfterMs = 30000
	}

	b.cfg.ModelRouter = &cfg
	return b
}

// Fallback configuration

// WithFallback configures fallback behavior.
func (b *Builder) WithFallback(cfg models.FallbackConfig) *Builder {
	// Set defaults
	if cfg.Mode == "" {
		cfg.Mode = "race"
	}
	if cfg.TimeoutMs == 0 {
		cfg.TimeoutMs = 30000
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}

	b.cfg.Fallback = cfg
	return b
}

// Provider configuration

// ProviderBuilder provides a type-safe way to configure providers.
type ProviderBuilder struct {
	apiKey         string
	baseURL        string
	authType       string
	authHeaderName string
	healthEndpoint string
	rateLimitRpm   *int
	timeoutMs      int
	headers        map[string]string
}

// NewProviderBuilder creates a new provider configuration builder.
func NewProviderBuilder(apiKey string) *ProviderBuilder {
	return &ProviderBuilder{
		apiKey:  apiKey,
		headers: make(map[string]string),
	}
}

// WithBaseURL sets a custom base URL for the provider.
func (pb *ProviderBuilder) WithBaseURL(url string) *ProviderBuilder {
	pb.baseURL = url
	return pb
}

// WithAuthType sets the authentication type.
func (pb *ProviderBuilder) WithAuthType(authType string) *ProviderBuilder {
	pb.authType = authType
	return pb
}

// WithAuthHeader sets a custom auth header name.
func (pb *ProviderBuilder) WithAuthHeader(name string) *ProviderBuilder {
	pb.authHeaderName = name
	return pb
}

// WithHealthEndpoint sets the health check endpoint.
func (pb *ProviderBuilder) WithHealthEndpoint(endpoint string) *ProviderBuilder {
	pb.healthEndpoint = endpoint
	return pb
}

// WithRateLimit sets rate limit in requests per minute.
func (pb *ProviderBuilder) WithRateLimit(rpm int) *ProviderBuilder {
	pb.rateLimitRpm = &rpm
	return pb
}

// WithTimeout sets the request timeout in milliseconds.
func (pb *ProviderBuilder) WithTimeout(ms int) *ProviderBuilder {
	pb.timeoutMs = ms
	return pb
}

// WithHeader adds a custom header.
func (pb *ProviderBuilder) WithHeader(key, value string) *ProviderBuilder {
	pb.headers[key] = value
	return pb
}

// Build builds the provider configuration.
func (pb *ProviderBuilder) Build() models.ProviderConfig {
	return models.ProviderConfig{
		APIKey:         pb.apiKey,
		BaseURL:        pb.baseURL,
		AuthType:       pb.authType,
		AuthHeaderName: pb.authHeaderName,
		HealthEndpoint: pb.healthEndpoint,
		RateLimitRpm:   pb.rateLimitRpm,
		TimeoutMs:      pb.timeoutMs,
		Headers:        pb.headers,
	}
}

// AddOpenAICompatibleProvider adds a provider using OpenAI-compatible API (chat/completions).
// Automatically registers the provider to chat_completions endpoint.
// name: provider identifier (e.g., "openai", "groq", "deepseek")
// cfg: provider configuration (API key, base URL, etc.)
func (b *Builder) AddOpenAICompatibleProvider(name string, cfg models.ProviderConfig) *Builder {
	b.cfg.Endpoints.ChatCompletions.Providers[name] = cfg
	b.enabledEndpoints["chat_completions"] = true
	return b
}

// AddAnthropicCompatibleProvider adds a provider using Anthropic-compatible API (messages).
// Automatically registers the provider to messages endpoint.
// name: provider identifier (e.g., "anthropic")
// cfg: provider configuration (API key, base URL, etc.)
func (b *Builder) AddAnthropicCompatibleProvider(name string, cfg models.ProviderConfig) *Builder {
	b.cfg.Endpoints.Messages.Providers[name] = cfg
	b.enabledEndpoints["messages"] = true
	return b
}

// AddGeminiCompatibleProvider adds a provider using Gemini-compatible API (generateContent).
// Automatically registers the provider to generate and count_tokens endpoints.
// name: provider identifier (e.g., "gemini", "google")
// cfg: provider configuration (API key, base URL, etc.)
func (b *Builder) AddGeminiCompatibleProvider(name string, cfg models.ProviderConfig) *Builder {
	b.cfg.Endpoints.Generate.Providers[name] = cfg
	b.cfg.Endpoints.CountTokens.Providers[name] = cfg
	b.enabledEndpoints["generate"] = true
	b.enabledEndpoints["count_tokens"] = true
	return b
}

// Middleware configuration

// WithRateLimit configures rate limiting middleware.
func (b *Builder) WithRateLimit(max int, expiration time.Duration, keyFunc ...func(*fiber.Ctx) string) *Builder {
	cfg := &models.RateLimitConfig{
		Max:        max,
		Expiration: expiration,
	}
	if len(keyFunc) > 0 {
		cfg.KeyFunc = keyFunc[0]
	}
	b.rateLimitConfig = cfg
	return b
}

// WithTimeout configures request timeout middleware.
func (b *Builder) WithTimeout(timeout time.Duration) *Builder {
	b.timeoutConfig = &models.TimeoutConfig{
		Timeout: timeout,
	}
	return b
}

// WithMiddleware adds a custom middleware.
func (b *Builder) WithMiddleware(middleware fiber.Handler) *Builder {
	b.middlewares = append(b.middlewares, middleware)
	return b
}

// GetMiddlewares returns all configured middlewares.
func (b *Builder) GetMiddlewares() []fiber.Handler {
	return b.middlewares
}

// GetRateLimitConfig returns the rate limit configuration.
func (b *Builder) GetRateLimitConfig() *models.RateLimitConfig {
	return b.rateLimitConfig
}

// GetTimeoutConfig returns the timeout configuration.
func (b *Builder) GetTimeoutConfig() *models.TimeoutConfig {
	return b.timeoutConfig
}

// GetEnabledEndpoints returns a map of enabled endpoints.
func (b *Builder) GetEnabledEndpoints() map[string]bool {
	return b.enabledEndpoints
}

// Build returns the constructed configuration.
func (b *Builder) Build() *config.Config {
	return b.cfg
}

// FromYAML creates a Builder from a YAML configuration file.
// The envFiles parameter specifies which .env files to load before parsing the YAML config.
// Files are loaded in order (first has highest priority).
// Example: builder, err := config.FromYAML("config.yaml", []string{".env.local", ".env"})
func FromYAML(path string, envFiles []string) (*Builder, error) {
	// Load env files first if specified
	if len(envFiles) > 0 {
		config.LoadEnvFiles(envFiles)
	}

	cfg, err := config.LoadFromFile(path)
	if err != nil {
		return nil, err
	}

	return builderFromConfig(cfg), nil
}

// builderFromConfig creates a Builder from an existing config and marks enabled endpoints
func builderFromConfig(cfg *config.Config) *Builder {
	builder := &Builder{
		cfg:              cfg,
		middlewares:      []fiber.Handler{},
		enabledEndpoints: make(map[string]bool),
	}

	// Mark all endpoints with providers as enabled
	if len(cfg.Endpoints.ChatCompletions.Providers) > 0 {
		builder.enabledEndpoints["chat_completions"] = true
	}
	if len(cfg.Endpoints.Messages.Providers) > 0 {
		builder.enabledEndpoints["messages"] = true
	}
	if len(cfg.Endpoints.SelectModel.Providers) > 0 {
		builder.enabledEndpoints["select_model"] = true
	}
	if len(cfg.Endpoints.Generate.Providers) > 0 {
		builder.enabledEndpoints["generate"] = true
	}
	if len(cfg.Endpoints.CountTokens.Providers) > 0 {
		builder.enabledEndpoints["count_tokens"] = true
	}

	return builder
}
