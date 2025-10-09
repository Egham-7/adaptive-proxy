# Intelligent Routing

AdaptiveProxy's Model Router uses AI to select the optimal LLM provider and model for each request, optimizing for cost, performance, and capability.

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
            config.NewProviderBuilder(os.Getenv("OPENAI_API_KEY")).Build(),
        ).
        AddAnthropicCompatibleProvider("anthropic",
            config.NewProviderBuilder(os.Getenv("ANTHROPIC_API_KEY")).Build(),
        ).
        AddGeminiCompatibleProvider("gemini",
            config.NewProviderBuilder(os.Getenv("GEMINI_API_KEY")).Build(),
        ).
        WithModelRouter(models.ModelRouterConfig{
            CostBias: 0.9,  // Prioritize cost savings
        })
    
    srv := config.NewProxyWithBuilder(builder)
    srv.Run()
}
```

Send request without specifying model:

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [{"role": "user", "content": "Explain quantum computing"}]
  }'
```

The router automatically selects the best model based on prompt complexity and cost.

## How It Works

### 1. Automatic Model Selection

When no model is specified, AdaptiveProxy:

1. **Analyzes** the prompt for complexity, tools, and context
2. **Calls AI service** to score available models
3. **Selects optimal model** balancing cost and capability
4. **Caches decision** for similar prompts (optional)

### 2. Cost Bias

The `cost_bias` parameter controls the cost-performance tradeoff:

```go
// Aggressive cost savings (may sacrifice quality)
builder.WithModelRouter(models.ModelRouterConfig{
    CostBias: 0.95,
})

// Balanced (recommended)
builder.WithModelRouter(models.ModelRouterConfig{
    CostBias: 0.9,
})

// Performance-first (higher quality, higher cost)
builder.WithModelRouter(models.ModelRouterConfig{
    CostBias: 0.7,
})
```

**Range:** `0.0 - 1.0`
- **0.0**: Pure performance (ignores cost)
- **1.0**: Pure cost (ignores performance)
- **0.9**: Sweet spot for most use cases

### 3. Provider Filtering

Router only considers configured providers:

```yaml
model_router:
  cost_bias: 0.9

endpoints:
  chat_completions:
    providers:
      openai:
        api_key: ${OPENAI_API_KEY}
      anthropic:
        api_key: ${ANTHROPIC_API_KEY}
      gemini:
        api_key: ${GEMINI_API_KEY}
```

## Configuration

### Via Builder API

```go
import "github.com/Egham-7/adaptive-proxy/internal/models"

builder := config.New().
    AddOpenAICompatibleProvider("openai", openaiCfg).
    AddAnthropicCompatibleProvider("anthropic", anthropicCfg).
    AddGeminiCompatibleProvider("gemini", geminiCfg).
    WithModelRouter(models.ModelRouterConfig{
        CostBias: 0.9,
        SemanticCache: models.SemanticCacheConfig{
            Enabled:           true,
            SemanticThreshold: 0.90,
        },
    })
```

### Via YAML

```yaml
model_router:
  cost_bias: 0.9
  cache:
    enabled: true
    semantic_threshold: 0.90
    ttl_seconds: 86400

endpoints:
  chat_completions:
    providers:
      openai:
        api_key: ${OPENAI_API_KEY}
        base_url: https://api.openai.com/v1
      anthropic:
        api_key: ${ANTHROPIC_API_KEY}
        base_url: https://api.anthropic.com
      gemini:
        api_key: ${GEMINI_API_KEY}
        base_url: https://generativelanguage.googleapis.com
```

## Router Cache

### Overview

Router cache stores AI service decisions to reduce API calls and latency.

**Benefits:**
- **70%+ reduction** in AI service calls
- **Sub-10ms** cache lookups vs 200-500ms AI calls
- **Cost savings** on routing decisions

### Configuration

```go
builder.WithModelRouter(models.ModelRouterConfig{
    CostBias: 0.9,
    SemanticCache: models.SemanticCacheConfig{
        Enabled:           true,
        SemanticThreshold: 0.90,
    },
})
```

**Parameters:**
- `Enabled`: Enable semantic cache for router
- `SemanticThreshold`: 0.0-1.0 (0.90 recommended)

### How It Works

1. **Check cache** for similar prompts (semantic search)
2. **Cache hit**: Return cached model selection (< 10ms)
3. **Cache miss**: Call AI service, cache decision
4. **Validate**: Skip models with open circuit breakers

### Per-Request Overrides

Disable router cache for specific requests:

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [{"role": "user", "content": "Hello"}],
    "model_router": {
      "cost_bias": 0.95,
      "cache": {
        "enabled": false
      }
    }
  }'
```

Adjust cost bias per request:

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [{"role": "user", "content": "Complex analysis task"}],
    "model_router": {
      "cost_bias": 0.7
    }
  }'
```

## Circuit Breaker Integration

Router automatically filters out unhealthy providers using circuit breakers.

**Example flow:**
1. OpenAI circuit breaker opens (5 failures)
2. Router excludes OpenAI from selection
3. Selects next best provider (Anthropic/Gemini)
4. Logs filtered providers

```
[request-123] 🚫 Filtering out provider openai (circuit breaker open)
[request-123] ⚠️  Provider filtering: 3 -> 2 models (filtered: [openai])
[request-123] ✅ AI service selected PRIMARY: anthropic/claude-3-5-sonnet
```

See [Fallback](fallback.md) for circuit breaker configuration.

## Model Capabilities

Router considers model capabilities for selection:

```yaml
model_router:
  models:
    - provider: openai
      model_name: gpt-4o-mini
      supports_tools: true
      supports_vision: false
      max_tokens: 16384
      
    - provider: anthropic
      model_name: claude-3-5-sonnet-20241022
      supports_tools: true
      supports_vision: true
      max_tokens: 8192
```

**Fields:**
- `provider`: Provider name (openai, anthropic, gemini)
- `model_name`: Specific model identifier
- `supports_tools`: Function calling support
- `supports_vision`: Image input support
- `max_tokens`: Maximum context window

## Selection Algorithm

### 1. Prompt Analysis
- Length and complexity
- Tool/function call requirements
- Vision/multimodal needs

### 2. Model Scoring
- **Cost**: Token pricing (input + output)
- **Performance**: Latency, throughput
- **Capability**: Features match
- **Availability**: Circuit breaker status

### 3. Weighted Selection
```
score = (capability_score * (1 - cost_bias)) + (cost_score * cost_bias)
```

### 4. Response
```json
{
  "provider": "gemini",
  "model": "gemini-1.5-flash-002",
  "alternatives": [
    {"provider": "openai", "model": "gpt-4o-mini"},
    {"provider": "anthropic", "model": "claude-3-5-haiku-20241022"}
  ]
}
```

## Use Cases

### 1. Cost Optimization

Maximize cost savings for simple queries:

```go
builder := config.New().
    AddOpenAICompatibleProvider("openai", openaiCfg).
    AddGeminiCompatibleProvider("gemini", geminiCfg).  // Gemini often cheaper
    WithModelRouter(models.ModelRouterConfig{
        CostBias: 0.95,  // Aggressive cost savings
    })
```

### 2. Quality-First

Prioritize response quality:

```go
builder := config.New().
    AddOpenAICompatibleProvider("openai", openaiCfg).
    AddAnthropicCompatibleProvider("anthropic", anthropicCfg).
    WithModelRouter(models.ModelRouterConfig{
        CostBias: 0.7,  // Quality over cost
    })
```

## Logs

Router logs detailed selection info:

```
[request-123] ═══ Model Selection Started ═══
[request-123] User: user-456 | Prompt length: 234 chars | Cost bias: 0.90
[request-123] 🔍 Cache enabled - checking semantic cache (threshold: 0.90)
[request-123] ❌ Cache miss - proceeding to AI service
[request-123] 🤖 Calling AI model selection service
[request-123] ✅ AI service selected PRIMARY: gemini/gemini-1.5-flash-002
[request-123] 📋 ALTERNATIVES (2):
[request-123]    1. openai/gpt-4o-mini
[request-123]    2. anthropic/claude-3-5-haiku-20241022
[request-123] ═══ Model Selection Complete (AI Service) ═══
```

## Troubleshooting

### No Model Selected

**Symptom**: Empty response or errors

**Solutions:**
1. Verify providers configured:
   ```go
   builder := config.New().
       AddOpenAICompatibleProvider("openai", cfg).  // At least one provider required
       WithModelRouter(models.ModelRouterConfig{CostBias: 0.9})
   ```

2. Check circuit breakers:
   ```
   [request-123] ⚠️  All cached models unavailable (circuit breakers open)
   ```

3. Verify AI service running:
   ```bash
   curl http://localhost:8000/health
   ```

### Wrong Model Selected

**Symptom**: Too expensive or incapable model

**Solutions:**
1. Adjust cost bias:
   ```go
   WithModelRouter(models.ModelRouterConfig{
       CostBias: 0.95,  // More cost-sensitive
   })
   ```

2. Configure model capabilities in YAML:
   ```yaml
   model_router:
     models:
       - provider: openai
         model_name: gpt-4o-mini
         supports_tools: true
   ```

3. Override per request:
   ```json
   {
     "model_router": {
       "cost_bias": 0.99
     }
   }
   ```

### Cache Issues

**Symptom**: Low cache hit rate (< 30%)

**Solutions:**
1. Lower semantic threshold:
   ```go
   SemanticCache: models.SemanticCacheConfig{
       Enabled:           true,
       SemanticThreshold: 0.85,  // From 0.90
   }
   ```

2. Check prompt variability:
   - Prompts with timestamps won't match
   - Use consistent formatting

3. Verify Redis connection:
   ```bash
   redis-cli PING
   ```

### AI Service Errors

**Symptom**: Router timeouts or failures

**Solutions:**
1. Check AI service health:
   ```bash
   curl http://localhost:8000/health
   ```

2. Verify network connectivity:
   ```bash
   ping ai-service-host
   ```

3. Review AI service logs

## Best Practices

1. **Start with balanced cost_bias (0.9)**
   - Adjust based on quality needs
   - Monitor cost savings vs quality

2. **Enable router cache**
   - 70%+ reduction in AI calls
   - Use threshold 0.90 for production

3. **Configure accurate model capabilities**
   - Ensures correct selection
   - Prevents capability mismatches

4. **Monitor circuit breakers**
   - Router auto-filters unhealthy providers
   - Review filtered provider logs

5. **Use per-request overrides sparingly**
   - Prefer YAML config for consistency
   - Override only for special cases

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
        AddGeminiCompatibleProvider("gemini",
            config.NewProviderBuilder(os.Getenv("GEMINI_API_KEY")).
                WithBaseURL("https://generativelanguage.googleapis.com").
                Build(),
        ).
        
        // Model router with aggressive cost savings
        WithModelRouter(models.ModelRouterConfig{
            CostBias: 0.95,
            SemanticCache: models.SemanticCacheConfig{
                Enabled:           true,
                SemanticThreshold: 0.90,
            },
        }).
        
        // Circuit breakers for resilience
        WithFallback(models.FallbackConfig{
            Mode:       "race",
            TimeoutMs:  30000,
            MaxRetries: 3,
            CircuitBreaker: &models.CircuitBreakerConfig{
                FailureThreshold: 5,
                SuccessThreshold: 3,
                TimeoutMs:        15000,
                ResetAfterMs:     60000,
            },
        })

    srv := config.NewProxyWithBuilder(builder)
    if err := srv.Run(); err != nil {
        log.Fatal(err)
    }
}
```

## Next Steps

- [Fallback](fallback.md) - Multi-provider resilience
- [Caching](caching.md) - Redis configuration
- [Providers](providers.md) - Provider setup
