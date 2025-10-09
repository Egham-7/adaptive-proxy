# Caching

AdaptiveProxy provides powerful Redis-backed caching to reduce costs and improve response times.

## Quick Start

```go
import (
    "os"
    "github.com/Egham-7/adaptive-proxy/pkg/config"
    "github.com/Egham-7/adaptive-proxy/internal/models"
)

func main() {
    builder := config.New().
        AddOpenAICompatibleProvider("openai",
            config.NewProviderBuilder(os.Getenv("OPENAI_API_KEY")).
                WithBaseURL("https://api.openai.com/v1").
                Build(),
        ).
        WithSemanticCache(models.SemanticCacheConfig{
            Enabled:           true,
            SemanticThreshold: 0.85,
            RedisURL:          "redis://localhost:6379",
        })
    
    srv := config.NewProxyWithBuilder(builder)
    srv.Run()
}
```

## Cache Types

### 1. Prompt Cache (Exact & Semantic)

Caches prompt-response pairs with both exact and semantic matching.

**Configuration:**
```go
builder := config.New().
    AddOpenAICompatibleProvider("openai", openaiCfg).  // Used for embeddings
    WithSemanticCache(models.SemanticCacheConfig{
        Enabled:           true,
        SemanticThreshold: 0.85,
        RedisURL:          "redis://localhost:6379",
    })
```

**How it works:**
1. **Exact match**: Checks for identical prompts first
2. **Semantic match**: Uses embeddings to find similar prompts (if threshold met)
3. **Cache miss**: Falls through to provider, then caches response

**Semantic threshold:**
- `0.0 - 1.0` (cosine similarity)
- `0.85` recommended for production
- Higher = stricter matching
- Lower = more cache hits, but less precise

### 2. Model Router Cache

Caches AI service decisions for model selection.

**Configuration:**
```yaml
model_router:
  cost_bias: 0.9
  cache:
    enabled: true
    semantic_threshold: 0.90
    ttl_seconds: 86400  # 24 hours
```

**Via Builder:**
```go
builder := config.New().
    AddOpenAICompatibleProvider("openai", openaiCfg).
    WithModelRouter(models.ModelRouterConfig{
        CostBias: 0.9,
        SemanticCache: models.SemanticCacheConfig{
            Enabled:           true,
            SemanticThreshold: 0.90,
        },
    })
```

**How it works:**
- Caches model selection decisions based on prompts
- Reuses previous AI decisions for similar requests
- Reduces AI service API calls by ~70%

## Redis Configuration

### Connection

```go
// Basic
builder := config.New().
    WithRedis("redis://localhost:6379")

// With auth
builder := config.New().
    WithRedis("redis://:password@localhost:6379")

// TLS
builder := config.New().
    WithRedis("rediss://user:password@host:6379")
```

### YAML Configuration

```yaml
prompt_cache:
  redis_url: ${REDIS_URL:-redis://localhost:6379}
  enabled: true
  semantic_threshold: 0.85
  openai_api_key: ${OPENAI_API_KEY}
  embedding_model: text-embedding-3-small
```

## Cache Keys & TTL

### Key Structure

```
# Exact cache
adaptive:prompt:sha256:<hash>

# Semantic cache
adaptive:semantic:embedding:<provider>:<model>:<embedding_hash>

# Model router cache
adaptive:model_router:embedding:<hash>
```

### Time-to-Live (TTL)

Default TTLs:
- **Prompt cache**: 24 hours
- **Semantic cache**: 24 hours  
- **Model router cache**: 24 hours (configurable)

## Per-Request Overrides

Disable caching for specific requests:

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "Hello"}],
    "prompt_cache": {
      "enabled": false
    }
  }'
```

Adjust semantic threshold:

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "Hello"}],
    "prompt_cache": {
      "enabled": true,
      "semantic_threshold": 0.95
    }
  }'
```

## Cache Strategies

### Development

Fast iterations, lower threshold:

```go
builder := config.New().
    WithRedis("redis://localhost:6379").
    WithSemanticCache(models.SemanticCacheConfig{
        Enabled:           true,
        SemanticThreshold: 0.75,  // Lower threshold = more hits
    })
```

### Production

Precision caching, higher threshold:

```go
builder := config.New().
    WithRedis("redis://localhost:6379").
    WithSemanticCache(models.SemanticCacheConfig{
        Enabled:           true,
        SemanticThreshold: 0.90,  // Higher threshold = more precise
    })
```

### Cost Optimization

Aggressive caching with model router:

```go
builder := config.New().
    WithRedis("redis://localhost:6379").
    AddOpenAICompatibleProvider("openai", openaiCfg).
    WithSemanticCache(models.SemanticCacheConfig{
        Enabled:           true,
        SemanticThreshold: 0.85,
    }).
    WithModelRouter(models.ModelRouterConfig{
        CostBias: 0.9,
        SemanticCache: models.SemanticCacheConfig{
            Enabled:           true,
            SemanticThreshold: 0.90,
        },
    })
```

### Production

Precision caching, higher threshold:

```go
builder := config.New().
    WithRedis("redis://localhost:6379").
    WithSemanticCache(models.SemanticCacheConfig{
        Enabled:           true,
        SemanticThreshold: 0.90,  // Higher threshold = more precise
    })
```

### Cost Optimization

Aggressive caching with model router:

```go
builder := config.New().
    WithRedis("redis://localhost:6379").
    AddOpenAICompatibleProvider("openai", openaiCfg).
    WithSemanticCache(models.SemanticCacheConfig{
        Enabled:           true,
        SemanticThreshold: 0.85,
    }).
    WithModelRouter(models.ModelRouterConfig{
        CostBias: 0.9,
        SemanticCache: models.SemanticCacheConfig{
            Enabled:           true,
            SemanticThreshold: 0.90,
        },
    })
```

## Monitoring Cache Performance

### Cache Hit Metrics

AdaptiveProxy logs cache hits/misses:

```
[CACHE] Exact hit for prompt hash: abc123
[CACHE] Semantic hit (similarity: 0.92) for prompt: "Hello world"
[CACHE] Cache miss, forwarding to provider
```

### Redis Monitoring

Monitor Redis with:

```bash
# Connect to Redis CLI
redis-cli

# View all cache keys
KEYS adaptive:*

# Check cache stats
INFO stats

# Monitor real-time operations
MONITOR
```

## Embedding Models

### Supported Models

Default: `text-embedding-3-small` (OpenAI)

**Options:**
- `text-embedding-3-small` - Fast, cost-effective
- `text-embedding-3-large` - Higher quality, slower
- `text-embedding-ada-002` - Legacy model

### Configuration

```yaml
prompt_cache:
  openai_api_key: ${OPENAI_API_KEY}
  embedding_model: text-embedding-3-small
```

**Or via builder:**
```go
builder := config.New().
    WithRedis("redis://localhost:6379").
    WithModelRouter(models.ModelRouterConfig{
        CostBias: 0.9,
        SemanticCache: models.SemanticCacheConfig{
            Enabled:           true,
            SemanticThreshold: 0.90,
        },
    })
```

## Troubleshooting

### Cache Not Working

**Symptom**: No cache hits

**Solutions:**
1. Verify Redis connection:
   ```bash
   redis-cli PING
   ```

2. Check OpenAI API key (for semantic cache):
   ```bash
   echo $OPENAI_API_KEY
   ```

3. Verify cache is enabled:
   ```go
   builder := config.New().
       WithRedis("redis://localhost:6379").
       WithSemanticCache(models.SemanticCacheConfig{
           Enabled:           true,  // Make sure enabled=true
           SemanticThreshold: 0.85,
       })
   ```

### Low Cache Hit Rate

**Symptom**: < 30% hit rate

**Solutions:**
1. Lower semantic threshold:
   ```go
   SemanticThreshold: 0.75  // From 0.85
   ```

2. Check prompt variability:
   - Prompts with timestamps/random data won't match
   - Use consistent formatting

3. Monitor similarity scores:
   ```
   [CACHE] Semantic miss (similarity: 0.72 < 0.85)
   ```

### Redis Connection Errors

**Symptom**: `connection refused` errors

**Solutions:**
1. Start Redis:
   ```bash
   redis-server
   ```

2. Check Redis URL format:
   ```
   redis://localhost:6379  ✓
   localhost:6379          ✗
   ```

3. Verify network access (Docker/remote):
   ```bash
   telnet localhost 6379
   ```

## Best Practices

1. **Use semantic caching for similar prompts**
   - Threshold `0.85-0.90` for production
   - Threshold `0.75-0.80` for development

2. **Enable model router cache to reduce AI costs**
   - Caches routing decisions
   - 70%+ reduction in AI service calls

3. **Monitor cache hit rates**
   - Target 50-70% hit rate for cost savings
   - Adjust thresholds based on metrics

4. **Use Redis persistence**
   - Enable RDB/AOF for cache durability
   - Survives restarts

5. **Set appropriate TTLs**
   - 24h default balances freshness/cost
   - Reduce for rapidly changing data
   - Increase for stable datasets

## Example: Production Setup

```go
package main

import (
    "log"
    "os"
    "github.com/Egham-7/adaptive-proxy/pkg/config"
    "github.com/Egham-7/adaptive-proxy/internal/models"
)

func main() {
    builder := config.New().
        Port("8080").
        Environment("production").
        
        // Redis connection
        WithRedis("redis://:password@production.redis:6379").
        
        // Providers
        AddOpenAICompatibleProvider("openai",
            config.NewProviderBuilder(os.Getenv("OPENAI_API_KEY")).
                WithBaseURL("https://api.openai.com/v1").
                Build(),
        ).
        AddAnthropicCompatibleProvider("anthropic",
            config.NewProviderBuilder(os.Getenv("ANTHROPIC_API_KEY")).
                WithBaseURL("https://api.anthropic.com").
                Build(),
        ).
        
        // Prompt caching (aggressive)
        WithSemanticCache(models.SemanticCacheConfig{
            Enabled:           true,
            SemanticThreshold: 0.85,
        }).
        
        // Model router with caching
        WithModelRouter(models.ModelRouterConfig{
            CostBias: 0.9,
            SemanticCache: models.SemanticCacheConfig{
                Enabled:           true,
                SemanticThreshold: 0.90,
            },
        })
    
    srv := config.NewProxyWithBuilder(builder)
    if err := srv.Run(); err != nil {
        log.Fatal(err)
    }
}
```

## Next Steps

- [Routing](routing.md) - Intelligent model selection
- [Fallback](fallback.md) - Multi-provider resilience
- [Providers](providers.md) - Provider setup
