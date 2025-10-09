# Middleware

**Rate limiting, timeouts, and request processing**

AdaptiveProxy provides built-in middleware through the builder API. Configure rate limiting, timeouts, and custom request processing without touching Fiber internals.

## Quick Example

```go
import "time"

builder := config.New().
    AddOpenAICompatibleProvider("openai", openaiCfg).
    WithRateLimit(100, 1*time.Minute).
    WithTimeout(30 * time.Second)

srv := config.NewProxyWithBuilder(builder)
srv.Run()
```

---

## Rate Limiting

### Basic Configuration

Limit requests globally:

```go
import "time"

builder := config.New().
    WithRateLimit(100, 1*time.Minute)  // 100 requests per minute
```

### Per-IP Rate Limiting (Default)

By default, rate limiting is per IP address:

```go
builder.WithRateLimit(100, 1*time.Minute)
```

**Behavior**: Each IP address can make 100 requests per minute.

### Custom Rate Limit Key

Rate limit by a custom key (e.g., API key from header):

```go
import "github.com/gofiber/fiber/v2"

builder.WithRateLimit(1000, 1*time.Hour, func(c *fiber.Ctx) string {
    // Extract API key from Authorization header
    apiKey := c.Get("Authorization")
    if apiKey == "" {
        return c.IP() // Fallback to IP
    }
    return apiKey
})
```

**Use Case**: Higher limits for authenticated users, lower for anonymous.

---

## Timeouts

### Global Timeout

Cancel requests that exceed timeout:

```go
builder.WithTimeout(30 * time.Second)
```

**Behavior**:
- Request cancelled after 30 seconds
- Returns 408 Request Timeout
- Triggers fallback to next provider

### Per-Provider Timeouts

Different timeouts for different providers:

```go
openaiCfg := config.NewProviderBuilder(key).
    WithTimeout(10000).  // 10 seconds
    Build()

anthropicCfg := config.NewProviderBuilder(key).
    WithTimeout(30000).  // 30 seconds (Claude can be slower)
    Build()

builder.
    AddOpenAICompatibleProvider("openai", openaiCfg).
    AddAnthropicCompatibleProvider("anthropic", anthropicCfg)
```

---

## Custom Middleware

### Request Logging

Log all incoming requests:

```go
import (
    "log"
    "time"
    "github.com/gofiber/fiber/v2"
)

builder.WithMiddleware(func(c *fiber.Ctx) error {
    start := time.Now()
    
    // Process request
    err := c.Next()
    
    // Log request
    log.Printf(
        "[%s] %s %s - %d (%v)",
        c.Method(),
        c.Path(),
        c.IP(),
        c.Response().StatusCode(),
        time.Since(start),
    )
    
    return err
})
```

### Authentication

Validate API keys:

```go
builder.WithMiddleware(func(c *fiber.Ctx) error {
    apiKey := c.Get("Authorization")
    
    if !isValidAPIKey(apiKey) {
        return c.Status(401).JSON(fiber.Map{
            "error": "Invalid API key",
        })
    }
    
    return c.Next()
})
```

### Custom Headers

```go
import "github.com/google/uuid"

builder.WithMiddleware(func(c *fiber.Ctx) error {
    c.Set("X-Request-ID", uuid.New().String())
    
    err := c.Next()
    
    c.Set("X-Powered-By", "AdaptiveProxy")
    
    return err
})
```

---

## Complete Example

```go
package main

import (
    "log"
    "os"
    "time"
    
    "github.com/Egham-7/adaptive-proxy/pkg/config"
)

func main() {
    builder := config.New().
        // Server config
        Port("8080").
        AllowedOrigins("https://yourdomain.com").
        Environment("production").
        LogLevel("info").
        
        // Providers
        AddOpenAICompatibleProvider("openai",
            config.NewProviderBuilder(os.Getenv("OPENAI_API_KEY")).
                WithBaseURL("https://api.openai.com/v1").
                WithTimeout(10000).
                Build(),
        ).
        
        // Middleware
        WithRateLimit(1000, 1*time.Minute).
        WithTimeout(30 * time.Second)
    
    srv := config.NewProxyWithBuilder(builder)
    log.Fatal(srv.Run())
}
```

---

## Complete Example

```go
package main

import (
    "log"
    "os"
    "time"
    
    "github.com/Egham-7/adaptive-proxy/pkg/config"
    "github.com/gofiber/fiber/v2"
)

func main() {
    builder := config.New().
        // Providers
        AddOpenAICompatibleProvider("openai",
            config.NewProviderBuilder(os.Getenv("OPENAI_API_KEY")).
                WithBaseURL("https://api.openai.com/v1").
                WithTimeout(30000).
                Build(),
        ).
        AddOpenAICompatibleProvider("groq",
            config.NewProviderBuilder(os.Getenv("GROQ_API_KEY")).
                WithBaseURL("https://api.groq.com/openai/v1").
                WithTimeout(10000).
                Build(),
        ).
        
        // Middleware
        WithRateLimit(100, 1*time.Minute).
        WithTimeout(30 * time.Second).
        WithMiddleware(loggingMiddleware()).
        WithMiddleware(authMiddleware())
    
    srv := config.NewProxyWithBuilder(builder)
    log.Fatal(srv.Run())
}

func loggingMiddleware() fiber.Handler {
    return func(c *fiber.Ctx) error {
        start := time.Now()
        err := c.Next()
        log.Printf("[%s] %s - %v", c.Method(), c.Path(), time.Since(start))
        return err
    }
}

func authMiddleware() fiber.Handler {
    return func(c *fiber.Ctx) error {
        apiKey := c.Get("Authorization")
        if apiKey == "" {
            return c.Status(401).SendString("Missing API key")
        }
        return c.Next()
    }
}
```

---

## Production Best Practices

### 1. Always Set Timeouts

Prevent hanging requests:

```go
builder.WithTimeout(30 * time.Second)
```

### 2. Use Rate Limiting

Protect against abuse:

```go
builder.WithRateLimit(1000, 1*time.Hour)
```

### 3. Add Request IDs

For debugging and tracing:

```go
import "github.com/google/uuid"

builder.WithMiddleware(func(c *fiber.Ctx) error {
    c.Set("X-Request-ID", uuid.New().String())
    return c.Next()
})
```

### 4. Log All Requests

For monitoring and debugging:

```go
builder.WithMiddleware(loggingMiddleware())
```

### 5. CORS Configuration

Set allowed origins in production:

```go
builder.AllowedOrigins("https://yourdomain.com,https://app.yourdomain.com")
```

### 2. Use Rate Limiting

Protect against abuse:

```go
builder.WithRateLimit(1000, 1*time.Hour)
```

### 3. Add Request IDs

For debugging and tracing:

```go
import "github.com/google/uuid"

builder.WithMiddleware(func(c *fiber.Ctx) error {
    c.Set("X-Request-ID", uuid.New().String())
    return c.Next()
})
```

### 4. Log All Requests

For monitoring and debugging:

```go
builder.WithMiddleware(loggingMiddleware())
```

### 5. CORS Configuration

Set allowed origins in production:

```go
builder.AllowedOrigins("https://yourdomain.com,https://app.yourdomain.com")
```

---

## Next Steps

- [Fallback](./fallback.md) - Multi-provider resilience
- [Production Guide](./production.md) - Production deployment
- [Performance](./performance.md) - Optimization tips
