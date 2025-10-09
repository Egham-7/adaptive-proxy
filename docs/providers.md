# Providers Configuration

Configure LLM providers (OpenAI, Anthropic, Gemini, DeepSeek, Groq, and custom providers).

## Provider Builder API

Every provider uses the fluent `ProviderBuilder`:

```go
provider := config.NewProviderBuilder(apiKey).
    WithTimeout(30000).              // milliseconds
    WithRateLimit(100).               // requests per minute
    WithBaseURL("https://...").       // custom URL
    WithHeader("X-Custom", "value").  // custom headers
    Build()
```

## Adding Providers by API Compatibility

AdaptiveProxy provides convenience methods to add providers based on their API specification:

- `AddOpenAICompatibleProvider` - For OpenAI-compatible APIs (chat/completions endpoint)
- `AddAnthropicCompatibleProvider` - For Anthropic-compatible APIs (messages endpoint)
- `AddGeminiCompatibleProvider` - For Gemini-compatible APIs (generateContent endpoint)

## Built-in Providers

### OpenAI

```go
import "os"

openaiProvider := config.NewProviderBuilder(os.Getenv("OPENAI_API_KEY")).
    WithTimeout(30000).    // 30 seconds
    WithRateLimit(500).    // 500 req/min
    WithBaseURL("https://api.openai.com/v1").
    Build()

builder.AddOpenAICompatibleProvider("openai", openaiProvider)
```

**API Compatibility:** OpenAI-compatible (uses `/v1/chat/completions`)

**Supported Models:** Any OpenAI model (we're just a proxy)
- Examples: `gpt-4o`, `gpt-4o-mini`, `gpt-4-turbo`, `o1`, `o3-mini`, etc.

**Environment Variable:**
```bash
export OPENAI_API_KEY="sk-..."
```

### Anthropic

```go
anthropicProvider := config.NewProviderBuilder(os.Getenv("ANTHROPIC_API_KEY")).
    WithBaseURL("https://api.anthropic.com").
    WithTimeout(45000).    // 45 seconds (Claude can be slower)
    WithRateLimit(200).
    Build()

builder.AddAnthropicCompatibleProvider("anthropic", anthropicProvider)
```

**API Compatibility:** Anthropic-compatible (uses `/v1/messages`)

**Supported Models:**
- `claude-3-7-sonnet-20250219` - Latest Claude 3.7 Sonnet
- `claude-3-5-sonnet-20241022` - Claude 3.5 Sonnet (Oct 2024)
- `claude-3-5-haiku-20241022` - Claude 3.5 Haiku (fast)
- `claude-3-opus-20240229` - Claude 3 Opus (powerful)

**Environment Variable:**
```bash
export ANTHROPIC_API_KEY="sk-ant-..."
```

**Note:** Anthropic supports native prompt caching - see [Caching Guide](./caching.md).

### Google Gemini

```go
geminiProvider := config.NewProviderBuilder(os.Getenv("GEMINI_API_KEY")).
    WithBaseURL("https://generativelanguage.googleapis.com").
    WithTimeout(30000).
    WithRateLimit(100).
    Build()

builder.AddGeminiCompatibleProvider("gemini", geminiProvider)
```

**API Compatibility:** Gemini-compatible (uses `/v1/generateContent` and `/v1/countTokens`)

**Supported Models:**
- `gemini-2.0-flash-exp` - Latest Gemini 2.0 (experimental)
- `gemini-2.0-flash-thinking-exp` - Gemini 2.0 with thinking mode
- `gemini-1.5-pro` - Gemini 1.5 Pro (stable)
- `gemini-1.5-flash` - Gemini 1.5 Flash (fast)

**Environment Variable:**
```bash
export GEMINI_API_KEY="..."
```

### DeepSeek

```go
deepseekProvider := config.NewProviderBuilder(os.Getenv("DEEPSEEK_API_KEY")).
    WithBaseURL("https://api.deepseek.com/v1").
    WithTimeout(30000).
    WithRateLimit(100).
    Build()

builder.AddOpenAICompatibleProvider("deepseek", deepseekProvider)
```

**API Compatibility:** OpenAI-compatible

**Supported Models:** Any DeepSeek model (we're just a proxy)
- Examples: `deepseek-chat`, `deepseek-coder`, etc.

**Environment Variable:**
```bash
export DEEPSEEK_API_KEY="..."
```

**API Compatibility:** OpenAI-compatible (uses `/v1/chat/completions`)

**Supported Models:**
- `deepseek-chat` - Latest DeepSeek chat model
- `deepseek-reasoner` - DeepSeek R1 reasoning model

**Environment Variable:**
```bash
export DEEPSEEK_API_KEY="..."
```

### Groq

```go
groqProvider := config.NewProviderBuilder(os.Getenv("GROQ_API_KEY")).
    WithBaseURL("https://api.groq.com/openai/v1").
    WithTimeout(10000).    // Groq is very fast
    WithRateLimit(500).
    Build()

builder.AddOpenAICompatibleProvider("groq", groqProvider)
```

**API Compatibility:** OpenAI-compatible

**Supported Models:** Any Groq model (we're just a proxy)
- Examples: `llama-3.3-70b-versatile`, `llama-3.1-8b-instant`, `mixtral-8x7b-32768`, etc.

**Environment Variable:**
```bash
export GROQ_API_KEY="gsk_..."
```

## Custom Providers

### OpenAI-Compatible Providers

Many providers use OpenAI's API format:

```go
customProvider := config.NewProviderBuilder(apiKey).
    WithBaseURL("https://api.custom.com/v1").
    WithTimeout(30000).
    Build()

builder.AddOpenAICompatibleProvider("custom", customProvider)
```

**Examples:**
- Together AI - `https://api.together.xyz/v1`
- Fireworks AI - `https://api.fireworks.ai/inference/v1`
- Perplexity - `https://api.perplexity.ai`
- Mistral AI - `https://api.mistral.ai/v1`
- Anyscale - `https://api.endpoints.anyscale.com/v1`
- OpenRouter - `https://openrouter.ai/api/v1`

### Custom Authentication

#### Bearer Token

```go
provider := config.NewProviderBuilder(token).
    WithAuthType("bearer").
    WithBaseURL("https://api.example.com")
```

Sends: `Authorization: Bearer <token>`

#### API Key Header

```go
provider := config.NewProviderBuilder(apiKey).
    WithAuthType("api_key").
    WithAuthHeader("X-API-Key").
    WithBaseURL("https://api.example.com")
```

Sends: `X-API-Key: <apiKey>`

#### Basic Auth

```go
provider := config.NewProviderBuilder("user:pass").
    WithAuthType("basic").
    WithBaseURL("https://api.example.com")
```

Sends: `Authorization: Basic <base64(user:pass)>`

#### Custom Headers

```go
provider := config.NewProviderBuilder(apiKey).
    WithHeader("X-Custom-Auth", "value").
    WithHeader("X-Client-ID", "adaptive-proxy").
    WithBaseURL("https://api.example.com")
```

## Provider Configuration Options

### Timeout

```go
.WithTimeout(30000)  // milliseconds
```

**Recommendations:**
- **OpenAI**: 30000ms (30s)
- **Anthropic**: 45000ms (Claude can be slower for long responses)
- **Gemini**: 30000ms
- **Groq**: 10000ms (very fast)
- **Custom**: Test and adjust

### Rate Limiting

```go
.WithRateLimit(100)  // requests per minute
```

**Per-provider rate limits** prevent overwhelming a single provider.

**Recommendations:**
- Set based on your subscription tier
- Add buffer (e.g., 90% of actual limit)
- Monitor and adjust

### Base URL

```go
.WithBaseURL("https://api.custom.com/v1")
```

**Default URLs (when not specified):**
- OpenAI: `https://api.openai.com/v1`
- Anthropic: `https://api.anthropic.com/v1`
- Gemini: `https://generativelanguage.googleapis.com`

### Health Endpoint

```go
.WithHealthEndpoint("/health")
```

Custom health check endpoint for monitoring.

## Multi-Provider Setup

### All Major Providers

```go
builder := config.New().
    // OpenAI - Primary
    AddOpenAICompatibleProvider("openai",
        config.NewProviderBuilder(os.Getenv("OPENAI_API_KEY")).
            WithBaseURL("https://api.openai.com/v1").
            WithTimeout(30000).
            Build(),
    ).
    
    // Anthropic - Fallback
    AddAnthropicCompatibleProvider("anthropic",
        config.NewProviderBuilder(os.Getenv("ANTHROPIC_API_KEY")).
            WithBaseURL("https://api.anthropic.com").
            WithTimeout(45000).
            Build(),
    ).
    
    // Gemini - Specific use case
    AddGeminiCompatibleProvider("gemini",
        config.NewProviderBuilder(os.Getenv("GEMINI_API_KEY")).
            WithBaseURL("https://generativelanguage.googleapis.com").
            WithTimeout(30000).
            Build(),
    ).
    
    // DeepSeek - Cost-effective fallback
    AddOpenAICompatibleProvider("deepseek",
        config.NewProviderBuilder(os.Getenv("DEEPSEEK_API_KEY")).
            WithBaseURL("https://api.deepseek.com/v1").
            WithTimeout(30000).
            Build(),
    ).
    
    // Groq - Fast responses
    AddOpenAICompatibleProvider("groq",
        config.NewProviderBuilder(os.Getenv("GROQ_API_KEY")).
            WithBaseURL("https://api.groq.com/openai/v1").
            WithTimeout(10000).
            Build(),
    )
```

## Provider Selection

### Automatic (Fallback)

When multiple providers are configured for the same API compatibility:

```go
builder.
    AddOpenAICompatibleProvider("openai", openaiConfig).
    AddOpenAICompatibleProvider("groq", groqConfig)
```

AdaptiveProxy automatically tries providers in order (or race mode).

### Manual Override

Specify provider in request using `provider:model` format:

```bash
curl http://localhost:8080/v1/chat/completions \
  -d '{
    "model": "groq:llama-3.3-70b-versatile",
    "messages": [...]
  }'
```

### Intelligent Routing

When model router is enabled, leave `model` empty:

```bash
curl http://localhost:8080/v1/chat/completions \
  -d '{
    "model": "",
    "messages": [...],
    "model_router": {
      "cost_bias": 0.3
    }
  }'
```

See [Routing Guide](./routing.md) for details.

## Provider-Specific Features

### OpenAI

**Function Calling:**
```json
{
  "model": "gpt-4",
  "messages": [...],
  "tools": [...],
  "tool_choice": "auto"
}
```

**Vision:**
```json
{
  "model": "gpt-4o",
  "messages": [{
    "role": "user",
    "content": [
      {"type": "text", "text": "What's in this image?"},
      {"type": "image_url", "image_url": {"url": "https://..."}}
    ]
  }]
}
```

### Anthropic

**Prompt Caching:**
```json
{
  "model": "claude-3-5-sonnet-20241022",
  "messages": [...],
  "system": [{
    "type": "text",
    "text": "Large system prompt...",
    "cache_control": {"type": "ephemeral"}
  }]
}
```

**Vision:**
```json
{
  "model": "claude-3-5-sonnet-20241022",
  "messages": [{
    "role": "user",
    "content": [
      {"type": "text", "text": "What's in this image?"},
      {"type": "image", "source": {"type": "url", "url": "https://..."}}
    ]
  }]
}
```

### Gemini

**Safety Settings:**
```json
{
  "model": "gemini-2.0-flash-exp",
  "contents": [...],
  "safetySettings": [...]
}
```

## Environment Variables

Create `.env`:

```bash
# OpenAI
OPENAI_API_KEY=sk-...

# Anthropic
ANTHROPIC_API_KEY=sk-ant-...

# Gemini
GEMINI_API_KEY=...

# DeepSeek
DEEPSEEK_API_KEY=...

# Groq
GROQ_API_KEY=gsk_...
```

Load in code:

```go
import "adaptive-backend/internal/config"

config.LoadEnvFiles([]string{".env"})
```

## Testing Providers

### Test Single Provider

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "openai:gpt-4",
    "messages": [{"role": "user", "content": "Say hello"}]
  }'
```

### Test All Providers

```bash
for provider in openai anthropic gemini; do
  echo "Testing $provider..."
  curl -s http://localhost:8080/v1/chat/completions \
    -d "{\"model\":\"$provider:gpt-4\",\"messages\":[{\"role\":\"user\",\"content\":\"Hi\"}]}" | jq
done
```

## Troubleshooting

### "invalid API key"

1. Check environment variable is set: `echo $OPENAI_API_KEY`
2. Verify key format (OpenAI: `sk-...`, Anthropic: `sk-ant-...`)
3. Test key directly with provider's API

### "timeout"

Increase timeout:
```go
.WithTimeout(60000)  // 60 seconds
```

### "rate limit exceeded"

1. Check provider rate limit: `WithRateLimit(100)`
2. Verify your actual tier limit with provider
3. Add delay between requests
4. Enable caching to reduce requests

### "provider not found"

Ensure provider is added with the correct API compatibility method:
```go
builder.AddOpenAICompatibleProvider("openai", providerConfig)
```

## Next Steps

- [Caching](./caching.md) - Enable Redis caching
- [Routing](./routing.md) - Intelligent model selection
- [Fallback](./fallback.md) - Multi-provider resilience
- [Middleware](./middleware.md) - Custom middleware and rate limiting
