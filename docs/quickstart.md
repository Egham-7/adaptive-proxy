# 🚀 Quick Start (5 minutes)

Get AdaptiveProxy running in your Go application in under 5 minutes.

## Prerequisites

- Go 1.25 or later
- API key from at least one LLM provider (OpenAI, Anthropic, Gemini, etc.)

## Step 1: Install

```bash
go get adaptive-backend
```

## Step 2: Create Your First Proxy

Create `main.go`:

```go
package main

import (
    "log"
    "os"
    
    "adaptive-backend/pkg/config"
)

func main() {
    // Configure proxy with OpenAI
    builder := config.New().
        Port("8080").
        AddOpenAICompatibleProvider("openai",
            config.NewProviderBuilder(os.Getenv("OPENAI_API_KEY")).Build(),
        )
    
    // Start server
    srv := config.NewProxyWithBuilder(builder)
    if err := srv.Run(); err != nil {
        log.Fatal(err)
    }
}
```

## Step 3: Set Your API Key

```bash
export OPENAI_API_KEY="sk-..."
```

## Step 4: Run It

```bash
go run main.go
```

You should see:

```
🚀 AdaptiveProxy starting on :8080
   Environment: development
   Go version: go1.25.0
   GOMAXPROCS: 8
```

## Step 5: Test It

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ]
  }'
```

**Success!** 🎉 You now have a working LLM proxy.

## Next Steps

### Add Fallback (Recommended)

Make your proxy more resilient:

```go
builder := config.New().
    AddOpenAICompatibleProvider("openai",
        config.NewProviderBuilder(os.Getenv("OPENAI_API_KEY")).Build(),
    ).
    AddAnthropicCompatibleProvider("anthropic",
        config.NewProviderBuilder(os.Getenv("ANTHROPIC_API_KEY")).Build(),
    )
```

Now if OpenAI fails, requests automatically fall back to Anthropic.

### Enable Caching (Save 60-80% on costs)

```go
import "adaptive-backend/internal/models"

builder.WithPromptCache(models.CacheConfig{
    Enabled:           true,
    RedisURL:          "redis://localhost:6379",
    SemanticThreshold: 0.9,
    EmbeddingModel:    "text-embedding-3-small",
    OpenAIAPIKey:      os.Getenv("OPENAI_API_KEY"),
})
```

Install Redis:
```bash
# macOS
brew install redis && brew services start redis

# Linux
sudo apt install redis-server && sudo systemctl start redis

# Docker
docker run -d -p 6379:6379 redis:alpine
```

### Add Rate Limiting

```go
import "time"

builder.WithRateLimit(1000, 1*time.Minute)  // 1000 req/min
```

## Common Patterns

### Multi-Provider with Race Mode

Try all providers simultaneously, return fastest:

```go
import "adaptive-backend/internal/models"

builder := config.New().
    AddOpenAICompatibleProvider("openai", openaiConfig).
    AddAnthropicCompatibleProvider("anthropic", anthropicConfig).
    AddOpenAICompatibleProvider("deepseek", deepseekConfig).
    WithFallback(models.FallbackConfig{
        Mode:       "race",  // Try all at once
        TimeoutMs:  30000,
        MaxRetries: 3,
    })
```

### Production Setup

```go
builder := config.New().
    Port("8080").
    Environment("production").
    LogLevel("warn").
    AllowedOrigins("https://yourdomain.com").
    WithRateLimit(1000, 1*time.Minute).
    WithTimeout(120 * time.Second).
    AddOpenAICompatibleProvider("openai", openaiConfig).
    AddAnthropicCompatibleProvider("anthropic", anthropicConfig)
```

## Troubleshooting

### "connection refused"
- Check Redis is running: `redis-cli ping` (should return `PONG`)
- Disable caching if Redis not needed: Remove `.WithPromptCache()`

### "invalid API key"
- Verify your API key: `echo $OPENAI_API_KEY`
- Check key has correct prefix: `sk-...` for OpenAI, `sk-ant-...` for Anthropic

### "address already in use"
- Change port: `builder.Port("8081")`
- Kill existing process: `lsof -ti:8080 | xargs kill`

## What's Next?

- **Learn the basics**: [Basic Usage](./basic-usage.md)
- **Configure providers**: [Providers Guide](./providers.md)
- **Go to production**: [Production Guide](./production.md)
- **See examples**: [Examples](./examples/)

## Getting Help

- 📖 **Documentation**: [Full docs](./README.md)
- 🐛 **Issues**: [GitHub Issues](https://github.com/Egham-7/adaptive-proxy/issues)
- 💬 **Questions**: [Discussions](https://github.com/Egham-7/adaptive-proxy/discussions)
