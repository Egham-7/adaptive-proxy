# Basic Usage

Learn the fundamental concepts and patterns for using AdaptiveProxy.

## Core Concepts

### 1. Builder Pattern

AdaptiveProxy uses a fluent builder API for configuration:

```go
builder := config.New().
    Port("8080").
    AddOpenAICompatibleProvider("openai", providerConfig)
```

**Why?**
- Type-safe configuration
- IDE autocomplete
- Clear, readable code
- Compile-time validation

### 2. Providers

Providers are LLM services (OpenAI, Anthropic, etc.):

```go
provider := config.NewProviderBuilder(apiKey).
    WithTimeout(30000).     // 30 seconds
    WithRateLimit(100).     // 100 req/min
    Build()
```

### 3. Endpoints

Endpoints are API routes you expose:

- `chat_completions` - OpenAI-compatible chat
- `messages` - Anthropic-compatible messages
- `generate` - Gemini-compatible generation
- `select_model` - Model selection API
- `count_tokens` - Token counting

### 4. Proxy Server

The server that handles requests:

```go
srv := config.NewProxyWithBuilder(builder)
srv.Run()  // Blocks until shutdown
```

## Basic Example

```go
package main

import (
    "log"
    "os"
    
    "adaptive-backend/pkg/config"
)

func main() {
    // 1. Create builder
    builder := config.New()
    
    // 2. Configure server
    builder.Port("8080")
    
    // 3. Add provider
    builder.AddOpenAICompatibleProvider("openai",
        config.NewProviderBuilder(os.Getenv("OPENAI_API_KEY")).Build(),
    )
    
    // 4. Create and run proxy
    srv := config.NewProxyWithBuilder(builder)
    if err := srv.Run(); err != nil {
        log.Fatal(err)
    }
}
```

## Using the API

Once running, make requests to your proxy:

### Chat Completions (OpenAI-compatible)

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

Response:
```json
{
  "id": "chatcmpl-...",
  "object": "chat.completion",
  "created": 1234567890,
  "model": "gpt-4",
  "choices": [{
    "index": 0,
    "message": {
      "role": "assistant",
      "content": "Hello! How can I help you today?"
    },
    "finish_reason": "stop"
  }],
  "usage": {
    "prompt_tokens": 10,
    "completion_tokens": 9,
    "total_tokens": 19
  }
}
```

### Messages (Anthropic-compatible)

```bash
curl http://localhost:8080/v1/messages \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-3-5-sonnet-20241022",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ],
    "max_tokens": 1024
  }'
```

### Streaming

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Count to 5"}],
    "stream": true
  }'
```

Response (Server-Sent Events):
```
data: {"choices":[{"delta":{"content":"1"}}]}
data: {"choices":[{"delta":{"content":","}}]}
data: {"choices":[{"delta":{"content":" 2"}}]}
...
data: [DONE]
```

## Provider Override

Specify provider explicitly using `provider:model` format:

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "anthropic:claude-3-5-sonnet-20241022",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

This bypasses intelligent routing and uses Anthropic directly.

## Configuration Patterns

### Single Provider

```go
builder := config.New().
    AddOpenAICompatibleProvider("openai",
        config.NewProviderBuilder(os.Getenv("OPENAI_API_KEY")).Build(),
    )
```

Use case: Simple proxy to OpenAI with rate limiting and monitoring.

### Multi-Provider

```go
builder := config.New().
    AddOpenAICompatibleProvider("openai", openaiConfig).
    AddAnthropicCompatibleProvider("anthropic", anthropicConfig).
    AddGeminiCompatibleProvider("gemini", geminiConfig)
```

Use case: Automatic fallback when one provider fails.

### Provider-Specific Endpoints

```go
builder := config.New().
    AddOpenAICompatibleProvider("openai", openaiConfig).
    AddAnthropicCompatibleProvider("anthropic", anthropicConfig).
    AddGeminiCompatibleProvider("gemini", geminiConfig)
```

Use case: Expose different APIs for different providers.

## Server Configuration

### Port

```go
builder.Port("8080")  // Default: "8080"
```

### Environment

```go
builder.Environment("production")  // "development" or "production"
```

**Development mode:**
- Verbose logging
- Stack traces in errors
- Route printing
- pprof profiler enabled

**Production mode:**
- Minimal logging
- Sanitized errors
- No route printing
- No profiler

### Log Level

```go
builder.LogLevel("warn")  // trace, debug, info, warn, error, fatal
```

### CORS

```go
builder.AllowedOrigins("https://example.com,https://app.example.com")
```

Or allow all (development only):
```go
builder.AllowedOrigins("*")  // Default
```

## Error Handling

AdaptiveProxy returns standardized error responses:

### Invalid Request (400)

```json
{
  "error": "invalid request: missing model field",
  "type": "invalid_request_error",
  "code": 400
}
```

### Rate Limit (429)

```json
{
  "error": "rate limit exceeded: 1000 requests per minute",
  "type": "rate_limit_error",
  "code": 429,
  "retryable": true,
  "retry_after": "60s"
}
```

### Provider Error (502)

```json
{
  "error": "provider request failed",
  "type": "provider_error",
  "code": 502,
  "retryable": true
}
```

### Internal Error (500)

```json
{
  "error": "internal server error",
  "type": "internal_error",
  "code": 500
}
```

## Health Checks

Check if the proxy is healthy:

```bash
curl http://localhost:8080/health
```

Response:
```json
{
  "status": "healthy",
  "redis": "connected",
  "timestamp": "2024-01-01T12:00:00Z"
}
```

## Graceful Shutdown

AdaptiveProxy handles shutdown gracefully:

```bash
# Send SIGINT (Ctrl+C) or SIGTERM
kill -TERM <pid>
```

Server will:
1. Stop accepting new requests
2. Complete in-flight requests (30s timeout)
3. Close Redis connections
4. Exit cleanly

## Next Steps

- **Configure providers**: [Providers Guide](./providers.md)
- **Enable caching**: [Caching Guide](./caching.md)
- **Add fallback**: [Fallback Guide](./fallback.md)

## Common Patterns

### Development Server

```go
config.New().
    Environment("development").
    LogLevel("debug").
    AddOpenAICompatibleProvider("openai", openaiConfig)
```

### Staging Server

```go
config.New().
    Environment("production").
    LogLevel("info").
    AllowedOrigins("https://staging.example.com").
    WithRateLimit(500, 1*time.Minute).
    AddOpenAICompatibleProvider("openai", openaiConfig).
    AddAnthropicCompatibleProvider("anthropic", anthropicConfig)
```

### Production Server

```go
config.New().
    Environment("production").
    LogLevel("warn").
    AllowedOrigins("https://example.com").
    WithRateLimit(1000, 1*time.Minute).
    WithTimeout(120 * time.Second).
    AddOpenAICompatibleProvider("openai", openaiConfig).
    AddAnthropicCompatibleProvider("anthropic", anthropicConfig).
    WithPromptCache(cacheConfig).
    WithFallback(fallbackConfig)
```

## Tips

1. **Always set timeouts** - Prevents hanging requests
2. **Use environment variables** - Never hardcode API keys
3. **Enable rate limiting** - Prevents abuse and cost overruns
4. **Add multiple providers** - Increases reliability
5. **Monitor health endpoint** - Set up alerts for downtime
6. **Use production mode** - Reduces attack surface

Ready to configure providers? → [Providers Guide](./providers.md)
