# AdaptiveProxy Builder API Reference

## Overview

The `pkg/config` package provides a fluent, type-safe API for configuring and running AdaptiveProxy programmatically. This is the recommended approach when embedding AdaptiveProxy in your Go applications.

## Quick Start

```go
package main

import (
    "adaptive-backend/pkg/config"
)

func main() {
    builder := config.New().
        Port("8080").
        AddProvider("openai",
            config.NewProviderBuilder(os.Getenv("OPENAI_API_KEY")).Build(),
            "chat_completions",
        )
    
    srv := config.NewProxyWithBuilder(builder)
    srv.Run()
}
```

## Configuration Builder API

### Creating a Builder

```go
builder := config.New()
```

Creates a new builder with sensible defaults:
- Port: `8080`
- Environment: `development`
- Allowed Origins: `*`
- Log Level: `info`
- Fallback Mode: `race`
- Fallback Timeout: `30000ms`
- Max Retries: `3`

### Server Configuration

```go
builder.Port(port string) *Builder
```
Sets the server port (default: `8080`).

```go
builder.Environment(env string) *Builder
```
Sets the environment (`development` or `production`).

```go
builder.AllowedOrigins(origins string) *Builder
```
Sets CORS allowed origins (comma-separated or `*`).

```go
builder.LogLevel(level string) *Builder
```
Sets log level: `trace`, `debug`, `info`, `warn`, `error`, `fatal`.

### Provider Configuration

#### Type-Safe Provider Builder

```go
providerBuilder := config.NewProviderBuilder(apiKey string)
```

Creates a provider configuration builder with required API key.

**Methods:**

```go
WithBaseURL(url string) *ProviderBuilder
```
Sets custom base URL for the provider.

```go
WithAuthType(authType string) *ProviderBuilder
```
Sets authentication type: `bearer`, `api_key`, `basic`, `custom`.

```go
WithAuthHeader(name string) *ProviderBuilder
```
Sets custom auth header name.

```go
WithHealthEndpoint(endpoint string) *ProviderBuilder
```
Sets health check endpoint path.

```go
WithRateLimit(rpm int) *ProviderBuilder
```
Sets rate limit in requests per minute.

```go
WithTimeout(ms int) *ProviderBuilder
```
Sets request timeout in milliseconds.

```go
WithHeader(key, value string) *ProviderBuilder
```
Adds a custom HTTP header.

```go
Build() models.ProviderConfig
```
Builds the provider configuration.

#### Adding Providers to Endpoints

```go
builder.AddProvider(name string, cfg models.ProviderConfig, endpoints ...string) *Builder
```

Adds a provider to specific endpoints.

**Available Endpoints:**
- `chat_completions` - OpenAI-compatible `/v1/chat/completions`
- `messages` - Anthropic-compatible `/v1/messages`
- `select_model` - Model selection `/v1/select-model`
- `generate` - Gemini-compatible `/v1/generate`
- `count_tokens` - Token counting `/v1beta/models/:model:countTokens`

If no endpoints are specified, defaults to `chat_completions`.

**Example:**

```go
// OpenAI for chat completions only
builder.AddProvider("openai",
    config.NewProviderBuilder(apiKey).Build(),
    "chat_completions",
)

// Anthropic for both chat and messages
builder.AddProvider("anthropic",
    config.NewProviderBuilder(apiKey).Build(),
    "chat_completions", "messages",
)

// Custom provider with advanced config
builder.AddProvider("custom",
    config.NewProviderBuilder(apiKey).
        WithBaseURL("https://api.custom.com").
        WithTimeout(45000).
        WithRateLimit(200).
        WithHeader("X-Custom", "value").
        Build(),
    "chat_completions",
)
```

### Caching Configuration

```go
builder.WithPromptCache(cfg models.CacheConfig) *Builder
```

Enables prompt-response caching.

**CacheConfig Fields:**
```go
models.CacheConfig{
    Enabled:           bool,      // Enable caching
    RedisURL:          string,    // Redis connection URL
    SemanticThreshold: float64,   // Similarity threshold (default: 0.9)
    EmbeddingModel:    string,    // Model for embeddings (default: "text-embedding-3-small")
    OpenAIAPIKey:      string,    // API key for embeddings
}
```

**Example:**

```go
builder.WithPromptCache(models.CacheConfig{
    Enabled:           true,
    RedisURL:          "redis://localhost:6379",
    SemanticThreshold: 0.9,
    EmbeddingModel:    "text-embedding-3-small",
    OpenAIAPIKey:      os.Getenv("OPENAI_API_KEY"),
})
```

### Model Router Configuration

```go
builder.WithModelRouter(cfg config.ModelRouterConfig) *Builder
```

Enables intelligent model routing.

**ModelRouterConfig Fields:**
```go
config.ModelRouterConfig{
    RouterURL:            string,   // Required: Model router service URL
    JWTSecret:            string,   // Required: JWT secret for auth
    CostBias:             float32,  // 0.0 = cheapest, 1.0 = best (default: 0.9)
    TimeoutMs:            int,      // Request timeout (default: 3000ms)
    EnableSemanticCache:  bool,     // Enable semantic caching
    SemanticThreshold:    float64,  // Similarity threshold (default: 0.95)
    CircuitBreakerConfig: *models.CircuitBreakerConfig, // Optional
}
```

**Example:**

```go
builder.WithModelRouter(config.ModelRouterConfig{
    RouterURL:           "http://localhost:8000",
    JWTSecret:           "my-secret",
    CostBias:            0.7, // Balanced
    TimeoutMs:           3000,
    EnableSemanticCache: true,
    SemanticThreshold:   0.95,
})
```

### Fallback Configuration

```go
builder.WithFallback(cfg config.FallbackConfig) *Builder
```

Configures fallback behavior when providers fail.

**FallbackConfig Fields:**
```go
config.FallbackConfig{
    Mode:                 string, // "race" or "sequential" (default: "race")
    TimeoutMs:            int,    // Timeout per provider (default: 30000ms)
    MaxRetries:           int,    // Max retries (default: 3)
    CircuitBreakerConfig: *models.CircuitBreakerConfig, // Optional
}
```

**Example:**

```go
builder.WithFallback(config.FallbackConfig{
    Mode:       "race", // Try all providers simultaneously
    TimeoutMs:  30000,
    MaxRetries: 3,
})
```

### Middleware Configuration

#### Rate Limiting

```go
builder.WithRateLimit(max int, expiration time.Duration, keyFunc ...func(*fiber.Ctx) string) *Builder
```

Configures rate limiting middleware.

**Parameters:**
- `max`: Maximum number of requests
- `expiration`: Time window
- `keyFunc` (optional): Custom key generator function

**Example:**

```go
// 500 requests per minute per API key
builder.WithRateLimit(500, 1*time.Minute)

// Custom key function
builder.WithRateLimit(500, 1*time.Minute, func(c *fiber.Ctx) string {
    return c.Get("X-User-ID")
})
```

#### Request Timeout

```go
builder.WithTimeout(timeout time.Duration) *Builder
```

Sets global request timeout.

**Example:**

```go
builder.WithTimeout(60 * time.Second)
```

#### Custom Middleware

```go
builder.WithMiddleware(middleware fiber.Handler) *Builder
```

Adds custom Fiber middleware.

**Example:**

```go
builder.WithMiddleware(func(c *fiber.Ctx) error {
    log.Printf("[CUSTOM] %s %s", c.Method(), c.Path())
    return c.Next()
})

// Add authentication middleware
builder.WithMiddleware(authMiddleware)

// Add metrics middleware
builder.WithMiddleware(metricsMiddleware)
```

### Building Configuration

```go
cfg := builder.Build() *config.Config
```

Returns the built configuration.

## Proxy Server API

### Creating a Proxy Server

#### From Builder (Recommended)

```go
srv := config.NewProxyWithBuilder(builder *config.Builder) *Proxy
```

Creates a proxy with full middleware and endpoint control.

#### From Configuration

```go
srv := config.NewProxy(cfg *config.Config) *Proxy
```

Creates a proxy with default middleware settings.

### Running the Server

```go
err := srv.Run() error
```

Starts the server and blocks until shutdown.

**Features:**
- Validates configuration
- Sets up middleware
- Creates Redis client (if configured)
- Registers routes for enabled endpoints only
- Graceful shutdown on SIGINT/SIGTERM

## Advanced Examples

### Multi-Provider Setup with Fallback

```go
builder := config.New().
    Port("8080").
    
    // Primary: OpenAI
    AddProvider("openai",
        config.NewProviderBuilder(openaiKey).Build(),
        "chat_completions",
    ).
    
    // Fallback: Anthropic
    AddProvider("anthropic",
        config.NewProviderBuilder(anthropicKey).Build(),
        "chat_completions",
    ).
    
    // Fallback: DeepSeek
    AddProvider("deepseek",
        config.NewProviderBuilder(deepseekKey).Build(),
        "chat_completions",
    ).
    
    // Race mode: Try all simultaneously
    WithFallback(config.FallbackConfig{
        Mode:       "race",
        TimeoutMs:  30000,
        MaxRetries: 3,
    })
```

### Provider-Specific Endpoints

```go
builder := config.New().
    // OpenAI for chat
    AddProvider("openai",
        config.NewProviderBuilder(openaiKey).Build(),
        "chat_completions",
    ).
    
    // Anthropic for messages
    AddProvider("anthropic",
        config.NewProviderBuilder(anthropicKey).Build(),
        "messages",
    ).
    
    // Gemini for generation
    AddProvider("gemini",
        config.NewProviderBuilder(geminiKey).Build(),
        "generate", "count_tokens",
    )
```

This ensures only the necessary endpoints are registered and initialized.

### Custom Provider with Headers

```go
builder.AddProvider("custom",
    config.NewProviderBuilder(apiKey).
        WithBaseURL("https://api.custom.com/v1").
        WithAuthType("bearer").
        WithAuthHeader("X-API-Key").
        WithTimeout(45000).
        WithRateLimit(200).
        WithHeader("X-Environment", "production").
        WithHeader("X-Client-ID", "adaptive-proxy").
        Build(),
    "chat_completions",
)
```

### Production Configuration

```go
builder := config.New().
    Port("8080").
    Environment("production").
    LogLevel("warn").
    AllowedOrigins("https://app.example.com,https://admin.example.com").
    
    // Production rate limiting
    WithRateLimit(1000, 1*time.Minute).
    WithTimeout(120 * time.Second).
    
    // Enable caching
    WithPromptCache(models.CacheConfig{
        Enabled:           true,
        RedisURL:          os.Getenv("REDIS_URL"),
        SemanticThreshold: 0.95,
        EmbeddingModel:    "text-embedding-3-small",
        OpenAIAPIKey:      os.Getenv("OPENAI_API_KEY"),
    }).
    
    // Intelligent routing
    WithModelRouter(config.ModelRouterConfig{
        RouterURL:           os.Getenv("ROUTER_URL"),
        JWTSecret:           os.Getenv("JWT_SECRET"),
        CostBias:            0.5, // Prioritize cost savings
        TimeoutMs:           5000,
        EnableSemanticCache: true,
    }).
    
    // Add monitoring middleware
    WithMiddleware(prometheusMiddleware).
    WithMiddleware(tracingMiddleware)
```

## Helper Methods

### Accessing Builder State

```go
middlewares := builder.GetMiddlewares() []fiber.Handler
```

Returns all configured custom middlewares.

```go
rateLimitCfg := builder.GetRateLimitConfig() *RateLimitConfig
```

Returns rate limit configuration.

```go
timeoutCfg := builder.GetTimeoutConfig() *TimeoutConfig
```

Returns timeout configuration.

```go
endpoints := builder.GetEnabledEndpoints() map[string]bool
```

Returns map of enabled endpoints.

## Loading YAML Configuration

```go
cfg, err := config.LoadDefault() (*config.Config, error)
```

Loads from default `config.yaml`.

```go
cfg, err := config.LoadFromYAML(path string) (*config.Config, error)
```

Loads from custom YAML file.

**Example:**

```go
// Load YAML config
cfg, err := config.LoadFromYAML("production-config.yaml")
if err != nil {
    log.Fatal(err)
}

// Use with proxy
srv := config.NewProxy(cfg)
srv.Run()
```

## Error Handling

All configuration validation happens at runtime during `srv.Run()`. Common errors:

- Invalid Redis URL
- Missing required provider API keys
- Invalid timeout values
- Network errors connecting to Redis

Always check the error returned by `Run()`:

```go
if err := srv.Run(); err != nil {
    log.Fatalf("Server failed: %v", err)
}
```
