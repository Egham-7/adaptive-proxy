# AdaptiveProxy Library Documentation

> **Use AdaptiveProxy as a Go library to embed intelligent LLM routing in your applications**

## Quick Start

```go
package main

import (
    "log"
    "os"
    
    "adaptive-backend/pkg/config"
)

func main() {
    // Create and configure proxy
    builder := config.New().
        Port("8080").
        AddProvider("openai",
            config.NewProviderBuilder(os.Getenv("OPENAI_API_KEY")).Build(),
            "chat_completions",
        )
    
    // Start server
    srv := config.NewProxyWithBuilder(builder)
    if err := srv.Run(); err != nil {
        log.Fatal(err)
    }
}
```

## Installation

```bash
go get adaptive-backend
```

## Core Concepts

### 1. Builder Pattern

Use the fluent `Builder` API for type-safe configuration:

```go
builder := config.New().
    Port("8080").
    Environment("production").
    LogLevel("info")
```

### 2. Provider Configuration

Configure LLM providers with the `ProviderBuilder`:

```go
provider := config.NewProviderBuilder(apiKey).
    WithTimeout(30000).              // 30s timeout
    WithRateLimit(100).               // 100 req/min
    WithBaseURL("https://api.custom.com").
    Build()

builder.AddProvider("openai", provider, "chat_completions")
```

### 3. Endpoint Control

Only enable endpoints you need:

```go
// Enable chat_completions only
builder.AddProvider("openai", provider, "chat_completions")

// Enable multiple endpoints
builder.AddProvider("anthropic", provider, 
    "chat_completions", "messages")
```

**Available Endpoints:**
- `chat_completions` - OpenAI `/v1/chat/completions`
- `messages` - Anthropic `/v1/messages`
- `select_model` - Model selection `/v1/select-model`
- `generate` - Gemini `/v1/generate`
- `count_tokens` - Token counting `/v1beta/models/:model:countTokens`

## Configuration

### Server Settings

```go
builder.
    Port("8080").
    Environment("production").        // "development" or "production"
    LogLevel("warn").                 // trace, debug, info, warn, error, fatal
    AllowedOrigins("https://example.com")
```

### Providers

#### OpenAI

```go
builder.AddProvider("openai",
    config.NewProviderBuilder(os.Getenv("OPENAI_API_KEY")).
        WithTimeout(30000).
        Build(),
    "chat_completions",
)
```

#### Anthropic

```go
builder.AddProvider("anthropic",
    config.NewProviderBuilder(os.Getenv("ANTHROPIC_API_KEY")).
        WithTimeout(45000).
        Build(),
    "chat_completions", "messages",
)
```

#### Gemini

```go
builder.AddProvider("gemini",
    config.NewProviderBuilder(os.Getenv("GEMINI_API_KEY")).
        WithTimeout(30000).
        Build(),
    "generate", "count_tokens",
)
```

#### Custom Provider

```go
builder.AddProvider("custom",
    config.NewProviderBuilder(apiKey).
        WithBaseURL("https://api.custom.com/v1").
        WithAuthType("bearer").
        WithAuthHeader("X-API-Key").
        WithTimeout(45000).
        WithRateLimit(200).
        WithHeader("X-Custom-Header", "value").
        Build(),
    "chat_completions",
)
```

### Caching

Enable Redis-backed prompt-response caching:

```go
import "adaptive-backend/internal/models"

builder.WithPromptCache(models.CacheConfig{
    Enabled:           true,
    RedisURL:          "redis://localhost:6379",
    SemanticThreshold: 0.9,               // 0-1: similarity threshold
    EmbeddingModel:    "text-embedding-3-small",
    OpenAIAPIKey:      os.Getenv("OPENAI_API_KEY"),
})
```

### Intelligent Model Routing

Enable AI-powered model selection:

```go
builder.WithModelRouter(models.ModelRouterConfig{
    RouterURL:           "http://localhost:8000",
    JWTSecret:           os.Getenv("JWT_SECRET"),
    CostBias:            0.7,  // 0.0=cheapest, 1.0=best performance
    Client: models.ClientConfig{
        TimeoutMs: 3000,
    },
    SemanticCache: models.SemanticCacheConfig{
        Enabled:           true,
        SemanticThreshold: 0.95,
    },
})
```

### Fallback Configuration

Configure provider fallback behavior:

```go
builder.WithFallback(models.FallbackConfig{
    Mode:       "race",      // "race" or "sequential"
    TimeoutMs:  30000,
    MaxRetries: 3,
    CircuitBreaker: &models.CircuitBreakerConfig{
        FailureThreshold: 5,
        SuccessThreshold: 3,
        TimeoutMs:        15000,
        ResetAfterMs:     60000,
    },
})
```

**Fallback Modes:**
- `race`: Try all providers simultaneously, return first success
- `sequential`: Try providers one by one in order

## Middleware

### Rate Limiting

```go
import "time"

// 500 requests per minute per API key
builder.WithRateLimit(500, 1*time.Minute)

// Custom key function
builder.WithRateLimit(500, 1*time.Minute, func(c *fiber.Ctx) string {
    return c.Get("X-User-ID")  // Rate limit per user
})
```

### Request Timeout

```go
builder.WithTimeout(60 * time.Second)
```

### Custom Middleware

```go
import "github.com/gofiber/fiber/v2"

// Add logging middleware
builder.WithMiddleware(func(c *fiber.Ctx) error {
    log.Printf("[REQUEST] %s %s", c.Method(), c.Path())
    return c.Next()
})

// Add authentication middleware
builder.WithMiddleware(authMiddleware)

// Add metrics middleware
builder.WithMiddleware(prometheusMiddleware)
```

## Complete Examples

### Minimal Setup

```go
package main

import (
    "log"
    "os"
    
    "adaptive-backend/pkg/config"
)

func main() {
    builder := config.New().
        AddProvider("openai",
            config.NewProviderBuilder(os.Getenv("OPENAI_API_KEY")).Build(),
            "chat_completions",
        )
    
    srv := config.NewProxyWithBuilder(builder)
    if err := srv.Run(); err != nil {
        log.Fatal(err)
    }
}
```

### Multi-Provider with Fallback

```go
package main

import (
    "log"
    "os"
    "time"
    
    "adaptive-backend/internal/models"
    "adaptive-backend/pkg/config"
)

func main() {
    builder := config.New().
        Port("8080").
        Environment("production").
        LogLevel("info").
        
        // Primary: OpenAI
        AddProvider("openai",
            config.NewProviderBuilder(os.Getenv("OPENAI_API_KEY")).
                WithTimeout(30000).
                Build(),
            "chat_completions",
        ).
        
        // Fallback: Anthropic
        AddProvider("anthropic",
            config.NewProviderBuilder(os.Getenv("ANTHROPIC_API_KEY")).
                WithTimeout(30000).
                Build(),
            "chat_completions",
        ).
        
        // Fallback: DeepSeek
        AddProvider("deepseek",
            config.NewProviderBuilder(os.Getenv("DEEPSEEK_API_KEY")).
                WithTimeout(30000).
                Build(),
            "chat_completions",
        ).
        
        // Race mode: try all simultaneously
        WithFallback(models.FallbackConfig{
            Mode:       "race",
            TimeoutMs:  30000,
            MaxRetries: 3,
        }).
        
        // Rate limiting
        WithRateLimit(1000, 1*time.Minute).
        WithTimeout(120 * time.Second)
    
    srv := config.NewProxyWithBuilder(builder)
    if err := srv.Run(); err != nil {
        log.Fatal(err)
    }
}
```

### Full Production Setup

```go
package main

import (
    "log"
    "os"
    "time"
    
    "adaptive-backend/internal/models"
    "adaptive-backend/pkg/config"
    "github.com/gofiber/fiber/v2"
)

func main() {
    builder := config.New().
        Port("8080").
        Environment("production").
        LogLevel("warn").
        AllowedOrigins("https://app.example.com").
        
        // Multi-provider setup
        AddProvider("openai",
            config.NewProviderBuilder(os.Getenv("OPENAI_API_KEY")).
                WithTimeout(30000).
                WithRateLimit(100).
                Build(),
            "chat_completions",
        ).
        AddProvider("anthropic",
            config.NewProviderBuilder(os.Getenv("ANTHROPIC_API_KEY")).
                WithTimeout(45000).
                Build(),
            "chat_completions", "messages",
        ).
        AddProvider("gemini",
            config.NewProviderBuilder(os.Getenv("GEMINI_API_KEY")).
                WithTimeout(30000).
                Build(),
            "generate", "count_tokens",
        ).
        
        // Enable caching
        WithPromptCache(models.CacheConfig{
            Enabled:           true,
            RedisURL:          os.Getenv("REDIS_URL"),
            SemanticThreshold: 0.95,
            EmbeddingModel:    "text-embedding-3-small",
            OpenAIAPIKey:      os.Getenv("OPENAI_API_KEY"),
        }).
        
        // Intelligent routing
        WithModelRouter(models.ModelRouterConfig{
            RouterURL: os.Getenv("ROUTER_URL"),
            JWTSecret: os.Getenv("JWT_SECRET"),
            CostBias:  0.5, // Balance cost vs performance
            Client: models.ClientConfig{
                TimeoutMs: 5000,
            },
            SemanticCache: models.SemanticCacheConfig{
                Enabled:           true,
                SemanticThreshold: 0.95,
            },
        }).
        
        // Fallback configuration
        WithFallback(models.FallbackConfig{
            Mode:       "race",
            TimeoutMs:  30000,
            MaxRetries: 3,
        }).
        
        // Middleware
        WithRateLimit(1000, 1*time.Minute).
        WithTimeout(120 * time.Second).
        WithMiddleware(loggingMiddleware).
        WithMiddleware(metricsMiddleware)
    
    srv := config.NewProxyWithBuilder(builder)
    if err := srv.Run(); err != nil {
        log.Fatal(err)
    }
}

func loggingMiddleware(c *fiber.Ctx) error {
    start := time.Now()
    err := c.Next()
    log.Printf("%s %s - %v", c.Method(), c.Path(), time.Since(start))
    return err
}

func metricsMiddleware(c *fiber.Ctx) error {
    // Your metrics logic
    return c.Next()
}
```

### Using YAML Configuration

```go
package main

import (
    "log"
    
    "adaptive-backend/internal/config"
    pkgconfig "adaptive-backend/pkg/config"
)

func main() {
    // Load environment files
    envFiles := []string{".env.local", ".env.development", ".env"}
    config.LoadEnvFiles(envFiles)
    
    // Load YAML config
    cfg, err := config.LoadFromFile("config.yaml")
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }
    
    // Create proxy
    srv := pkgconfig.NewProxy(cfg)
    if err := srv.Run(); err != nil {
        log.Fatal(err)
    }
}
```

Or using the builder:

```go
// Load YAML and env files in one call
builder, err := pkgconfig.FromYAML("config.yaml", []string{".env.local", ".env"})
if err != nil {
    log.Fatal(err)
}

// Add additional providers or middleware
builder.WithMiddleware(customMiddleware)

srv := pkgconfig.NewProxyWithBuilder(builder)
srv.Run()
```

## API Reference

### Builder Methods

| Method | Description |
|--------|-------------|
| `New()` | Create new builder with defaults |
| `Port(string)` | Set server port |
| `Environment(string)` | Set environment (development/production) |
| `LogLevel(string)` | Set log level (trace/debug/info/warn/error/fatal) |
| `AllowedOrigins(string)` | Set CORS allowed origins |
| `AddProvider(name, cfg, ...endpoints)` | Add provider to endpoints |
| `WithPromptCache(cfg)` | Enable prompt caching |
| `WithModelRouter(cfg)` | Enable intelligent routing |
| `WithFallback(cfg)` | Configure fallback behavior |
| `WithRateLimit(max, duration, ...keyFunc)` | Configure rate limiting |
| `WithTimeout(duration)` | Set global timeout |
| `WithMiddleware(handler)` | Add custom middleware |
| `Build()` | Build configuration |

### ProviderBuilder Methods

| Method | Description |
|--------|-------------|
| `NewProviderBuilder(apiKey)` | Create provider builder |
| `WithBaseURL(url)` | Set custom base URL |
| `WithAuthType(type)` | Set auth type (bearer/api_key/basic/custom) |
| `WithAuthHeader(name)` | Set custom auth header |
| `WithHealthEndpoint(path)` | Set health check endpoint |
| `WithRateLimit(rpm)` | Set rate limit (requests per minute) |
| `WithTimeout(ms)` | Set timeout (milliseconds) |
| `WithHeader(key, value)` | Add custom header |
| `Build()` | Build provider config |

### Proxy Methods

| Method | Description |
|--------|-------------|
| `NewProxy(cfg)` | Create proxy with config |
| `NewProxyWithBuilder(builder)` | Create proxy with builder |
| `Run()` | Start server (blocks until shutdown) |

## Environment Variables

When using YAML configuration, you can use environment variable substitution:

```yaml
server:
  port: "${PORT:-8080}"

endpoints:
  chat_completions:
    providers:
      openai:
        api_key: "${OPENAI_API_KEY}"
```

Load environment files explicitly:

```go
config.LoadEnvFiles([]string{".env.local", ".env"})
```

## Error Handling

All configuration validation happens during `Run()`:

```go
if err := srv.Run(); err != nil {
    log.Fatalf("Server failed: %v", err)
}
```

Common errors:
- Invalid Redis URL
- Missing API keys
- Invalid timeout values
- Redis connection failures

## Graceful Shutdown

The server handles graceful shutdown automatically:
- Listens for `SIGINT`/`SIGTERM`
- 30-second shutdown timeout
- Closes Redis connections
- Completes in-flight requests

## Performance Tips

1. **Use Redis caching** for repeated prompts (60-80% cost reduction)
2. **Enable race fallback mode** for lowest latency
3. **Configure circuit breakers** to prevent cascading failures
4. **Set appropriate timeouts** per provider characteristics
5. **Use semantic caching** for similar prompts
6. **Monitor rate limits** to avoid provider throttling

## Next Steps

- See [CLAUDE.md](./CLAUDE.md) for architecture details
- See [AGENTS.md](./AGENTS.md) for development guidelines
- See [pkg/README.md](./pkg/README.md) for detailed API reference
- Check [config.yaml](./config.yaml) for YAML configuration examples

## Support

For issues and questions:
- GitHub Issues: [adaptive-proxy](https://github.com/yourusername/adaptive-proxy/issues)
- Documentation: [CLAUDE.md](./CLAUDE.md)
