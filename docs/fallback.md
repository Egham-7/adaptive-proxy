# Fallback & Resilience

AdaptiveProxy provides production-grade resilience with automatic provider fallback, circuit breakers, and configurable retry strategies.

## Overview

When a provider fails (rate limits, timeouts, errors), AdaptiveProxy automatically:
1. Detects the failure and opens a circuit breaker
2. Routes to the next available provider
3. Retries with exponential backoff
4. Tracks provider health in real-time

## Execution Modes

### Sequential Mode (Default)

Tries providers one-by-one in order until success.

```go
builder := config.New().
    AddOpenAICompatibleProvider("openai",
        config.NewProviderBuilder(openaiKey).Build(),
    ).
    AddAnthropicCompatibleProvider("anthropic",
        config.NewProviderBuilder(anthropicKey).Build(),
    ).
    AddGeminiCompatibleProvider("gemini",
        config.NewProviderBuilder(geminiKey).Build(),
    ).
    SetFallbackMode("sequential")
```

**Flow**: OpenAI → (fails) → Anthropic → (fails) → Gemini → (fails) → Return error

**Best for**: Cost optimization (use cheaper providers first), predictable billing

### Race Mode

Sends requests to all providers simultaneously, returns first success.

```go
builder := config.New().
    AddOpenAICompatibleProvider("openai",
        config.NewProviderBuilder(openaiKey).Build(),
    ).
    AddAnthropicCompatibleProvider("anthropic",
        config.NewProviderBuilder(anthropicKey).Build(),
    ).
    SetFallbackMode("race")
```

**Flow**: OpenAI + Anthropic + Gemini (all at once) → Return fastest success

**Best for**: Ultra-low latency requirements, high availability, less cost-sensitive

**⚠️ Warning**: Race mode multiplies your API costs by the number of providers!

## Circuit Breakers

Circuit breakers prevent cascading failures by temporarily blocking requests to unhealthy providers.

### States

1. **Closed (Healthy)**: All requests pass through
2. **Open (Failing)**: Requests fail fast without calling provider
3. **Half-Open (Testing)**: Limited requests to test recovery

### Configuration

```go
builder := config.New().
    AddOpenAICompatibleProvider("openai",
        config.NewProviderBuilder(openaiKey).
            SetCircuitBreaker(
                5,     // maxFailures: Open after 5 consecutive failures
                60,    // timeout: Stay open for 60 seconds
                3,     // maxRequests: Allow 3 test requests in half-open state
            ).
            Build(),
    )
```

### How It Works

```
Closed → [5 failures] → Open → [60s timeout] → Half-Open → [3 successes] → Closed
                                                       ↓ [1 failure]
                                                      Open
```

### Per-Request Override

```yaml
# Bypass circuit breaker for critical requests
circuit_breaker:
  enabled: false
```

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Critical request"}],
    "circuit_breaker": {"enabled": false}
  }'
```

## Retry Strategies

### Exponential Backoff

```go
builder := config.New().
    SetRetryConfig(
        3,     // maxRetries
        500,   // initialDelay (ms)
        2.0,   // backoffMultiplier
    )
```

**Delays**: 500ms → 1000ms → 2000ms

### Per-Provider Retry

```go
builder := config.New().
    AddOpenAICompatibleProvider("openai",
        config.NewProviderBuilder(openaiKey).
            SetRetries(5, 1000, 2.0).  // More aggressive retries
            Build(),
    ).
    AddAnthropicCompatibleProvider("anthropic",
        config.NewProviderBuilder(anthropicKey).
            SetRetries(2, 500, 1.5).  // Faster failover
            Build(),
    )
```

### Retry Conditions

AdaptiveProxy retries on:
- 5xx server errors
- Network timeouts
- Rate limits (429) with `Retry-After` header
- Connection errors

Does NOT retry on:
- 4xx client errors (except 429)
- Invalid API keys
- Malformed requests

## Multi-Provider Resilience Patterns

### Cost-Optimized Fallback

Use cheaper providers first, expensive ones as backup:

```go
builder := config.New().
    AddOpenAICompatibleProvider("groq",           // Fastest, cheapest
        config.NewProviderBuilder(groqKey).Build(),
    ).
    AddOpenAICompatibleProvider("deepseek",       // Cheap, good quality
        config.NewProviderBuilder(deepseekKey).Build(),
    ).
    AddOpenAICompatibleProvider("openai",         // Expensive, most reliable
        config.NewProviderBuilder(openaiKey).Build(),
    ).
    SetFallbackMode("sequential")
```

### Geographic Redundancy

Failover between regions:

```go
builder := config.New().
    AddOpenAICompatibleProvider("openai-us",
        config.NewProviderBuilder(openaiKey).
            SetBaseURL("https://api.openai.com/v1").
            Build(),
    ).
    AddOpenAICompatibleProvider("openai-eu",
        config.NewProviderBuilder(openaiKey).
            SetBaseURL("https://api.openai.eu/v1").
            Build(),
    ).
    SetFallbackMode("sequential")
```

### Capability-Based Routing

Different providers for different models:

```go
builder := config.New().
    AddOpenAICompatibleProvider("openai",
        config.NewProviderBuilder(openaiKey).Build(),
    ).
    AddAnthropicCompatibleProvider("anthropic",
        config.NewProviderBuilder(anthropicKey).Build(),
    )

// YAML or request body
```

```yaml
routing:
  rules:
    - model: "gpt-4"
      provider: "openai"
    - model: "claude-3-opus"
      provider: "anthropic"
```

## Error Handling

### Error Types

AdaptiveProxy distinguishes between:

1. **Transient errors**: Retry automatically
   - Network timeouts
   - 503 Service Unavailable
   - 429 Rate Limit

2. **Permanent errors**: Fail immediately
   - 401 Unauthorized
   - 400 Bad Request
   - Invalid model

3. **Provider-specific errors**: Custom handling
   - OpenAI moderation flags
   - Anthropic content policy
   - Gemini safety filters

### Custom Error Handlers

```go
import "adaptive-backend/pkg/middleware"

builder := config.New().
    AddMiddleware(middleware.ErrorHandler(func(err error) error {
        // Custom error handling
        log.Printf("Error: %v", err)
        return err
    }))
```

## Health Checks

### Provider Health Monitoring

```go
// Built-in health endpoint
GET http://localhost:8080/health

{
  "status": "healthy",
  "providers": {
    "openai": {
      "status": "healthy",
      "circuit": "closed",
      "latency_p95": 234
    },
    "anthropic": {
      "status": "degraded",
      "circuit": "half-open",
      "latency_p95": 1200
    }
  }
}
```

### Custom Health Checks

```go
builder := config.New().
    SetHealthCheckInterval(30)  // Check every 30 seconds
```

## Production Best Practices

### 1. Always Configure Circuit Breakers

```go
config.NewProviderBuilder(apiKey).
    SetCircuitBreaker(5, 60, 3).  // Prevent cascading failures
    Build()
```

### 2. Use Timeouts

```go
builder := config.New().
    SetTimeout(30 * time.Second)  // Global timeout
```

### 3. Set Up Alerts

- Circuit breaker opens → Page on-call
- All providers failing → Critical alert
- High latency (p95 > 5s) → Warning

### 4. Test Failover Scenarios

Test your fallback configuration with simulated failures to ensure proper behavior.

## Complete Example

```go
package main

import (
    "log"
    "os"
    "time"
    
    "adaptive-backend/pkg/config"
    "adaptive-backend/pkg/middleware"
)

func main() {
    builder := config.New().
        // Primary: Fast and cheap
        AddOpenAICompatibleProvider("groq",
            config.NewProviderBuilder(os.Getenv("GROQ_API_KEY")).
                SetCircuitBreaker(3, 30, 2).
                SetRetries(2, 500, 1.5).
                SetTimeout(10 * time.Second).
                Build(),
        ).
        // Secondary: Reliable fallback
        AddOpenAICompatibleProvider("openai",
            config.NewProviderBuilder(os.Getenv("OPENAI_API_KEY")).
                SetCircuitBreaker(5, 60, 3).
                SetRetries(3, 1000, 2.0).
                SetTimeout(30 * time.Second).
                Build(),
        ).
        // Tertiary: Last resort
        AddAnthropicCompatibleProvider("anthropic",
            config.NewProviderBuilder(os.Getenv("ANTHROPIC_API_KEY")).
                SetCircuitBreaker(5, 60, 3).
                SetRetries(3, 1000, 2.0).
                SetTimeout(30 * time.Second).
                Build(),
        ).
        SetFallbackMode("sequential").
        SetTimeout(60 * time.Second).  // Global timeout
        AddMiddleware(middleware.RateLimit(1000, time.Minute)).
        AddMiddleware(middleware.Prometheus())
    
    srv := config.NewProxyWithBuilder(builder)
    log.Fatal(srv.Run())
}
```

## Next Steps

- [Routing](./routing.md) - Intelligent model selection
- [Caching](./caching.md) - Redis configuration
- [Providers](./providers.md) - Provider setup
