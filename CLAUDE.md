# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**AdaptiveProxy** is a high-performance Go API server providing OpenAI-compatible endpoints with intelligent multi-provider LLM routing. It acts as a unified proxy that routes requests to OpenAI, Anthropic, Gemini, DeepSeek, Groq, and other LLM providers while optimizing for cost (60-80% reduction) and reliability.

**Module**: `adaptive-backend` (Go 1.25+)
**Framework**: Fiber v2 (Express-like HTTP framework)
**Key Dependencies**: OpenAI Go SDK v2, Anthropic SDK, Google GenAI, Redis

## Essential Commands

### Development
```bash
# Start server (listens on port 8080 by default)
go run cmd/api/main.go

# Build binary
go build -o main cmd/api/main.go

# Run tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run specific test package
go test ./internal/services/providers/...
```

### Required Before Every Commit
```bash
# 1. Format code (MANDATORY)
gofmt -w .

# 2. Run linter (MANDATORY - must pass)
golangci-lint run

# 3. Clean dependencies (MANDATORY)
go mod tidy

# 4. Verify
go vet ./...
go test ./...
```

### Install Development Tools
```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install mvdan.cc/gofumpt@latest
go install golang.org/x/tools/cmd/goimports@latest
```

## Architecture Overview

### Core Request Flow

```
HTTP Request → Handler (api/) → Config Resolution → Model Selection → Provider Execution → Response
                                        ↓                    ↓
                                  Merge YAML +        Cache Check
                                Request Overrides    (Redis/Semantic)
                                        ↓                    ↓
                                Single Source       Circuit Breaker
                                  of Truth           Evaluation
```

### Key Architectural Patterns

#### 1. Config Resolution Pattern
The system uses a two-layer configuration approach:
- **Base Layer**: YAML config (`config.yaml`) with environment variable substitution
- **Override Layer**: Request-specific overrides passed in API calls

**Critical**: Always use `ResolveConfig()` methods to merge YAML + request overrides into a single source of truth before processing:
```go
// Example from completions.go:82-86
resolvedConfig, err := h.cfg.ResolveConfig(req)
// Now use resolvedConfig everywhere - never use h.cfg directly after this point
```

This pattern prevents configuration inconsistencies and ensures request-level overrides (provider configs, fallback settings, cost bias) take precedence over YAML defaults.

#### 2. Model Selection Strategy
Two modes supported:
- **Intelligent Routing**: Empty or unrecognized `model` field → calls external AI service for cost-optimized selection
- **Manual Override**: `model` format `"provider:model_name"` (e.g., `"openai:gpt-4"`) → direct routing

**Implementation** (`api/completions.go:selectModel`):
1. Check if model explicitly provided (`req.Model != ""`): Try manual override
2. Parse `provider:model` format using `utils.ParseProviderModel()`
3. If parsing fails, fall through to intelligent routing via `modelRouter.SelectModelWithCache()`

#### 3. Multi-Level Caching
Three cache layers in order of check:
1. **Prompt-Response Cache** (Redis): Exact prompt match → cached completion response
2. **Model Router Cache** (Redis + Semantic): Similar prompt → cached model selection
3. **Provider-level Cache**: Native provider caching (Anthropic prompt caching, etc.)

Cache keys track source: `"manual_override"`, `"redis_cache"`, `"semantic_cache"`, `"ml_inference"`

#### 4. Circuit Breaker Pattern
Shared circuit breakers per provider across all endpoints prevent cascading failures:
- States: `Closed` (normal) → `Open` (failing) → `Half-Open` (testing recovery)
- Created once at startup in `main.go:65-74` and shared across handlers
- Redis-backed state allows distributed tracking

#### 5. Format Adapters (Singleton Pattern)
Provider-agnostic request/response translation:
- `AdaptiveToOpenAI`: Converts internal format → OpenAI SDK parameters
- `AdaptiveToAnthropic`: Converts internal → Anthropic messages
- `AdaptiveToGemini`: Converts internal → Gemini generate requests
- Reverse adapters: `OpenAIToAdaptive`, `AnthropicToAdaptive`, `GeminiToAdaptive`

**Critical**: Always initialize singletons (see `internal/services/format_adapter/adapters.go`)

### Service Layer Organization

```
internal/
├── api/              # HTTP handlers (CompletionHandler, MessagesHandler, etc.)
├── config/           # YAML config loading with env var substitution
├── models/           # Request/response types and domain models
├── services/
│   ├── cache/        # Provider-specific prompt caching (Redis-backed)
│   ├── chat/         # Chat completion orchestration
│   ├── circuitbreaker/ # Circuit breaker implementation (Redis state)
│   ├── fallback/     # Sequential/race fallback strategies
│   ├── format_adapter/ # Provider format translation (singletons)
│   ├── model_router/ # Intelligent model selection with caching
│   ├── stream/       # SSE streaming response handling
│   └── providers/    # Direct provider API integrations
└── utils/           # Shared utilities (parsing, extraction, buffer pools)
```

## Configuration System

AdaptiveProxy supports two configuration approaches:

### 1. Programmatic Configuration (Recommended for Library Usage)

Use the fluent builder API from `pkg/config`:

```go
import (
    "adaptive-backend/pkg/config"
)

builder := config.New().
    Port("8080").
    Environment("production").
    
    // Add providers with type-safe configuration
    AddProvider("openai",
        config.NewProviderBuilder(apiKey).
            WithTimeout(30000).
            WithRateLimit(100).
            Build(),
        "chat_completions",
    ).
    
    // Configure middlewares
    WithRateLimit(500, 1*time.Minute).
    WithTimeout(60 * time.Second).
    WithMiddleware(customMiddleware)

srv := config.NewProxyWithBuilder(builder)
srv.Run()
```

**Benefits:**
- Type safety at compile time
- IDE autocomplete and documentation
- Dynamic configuration
- Endpoint control (only enable needed endpoints)
- Custom middleware support

### 2. YAML Configuration (Traditional Approach)

**IMPORTANT**: Configuration loading is now fully explicit. You must:
1. Explicitly load environment files
2. Explicitly load the YAML config file
3. Explicitly pass config to the proxy

**Example in `main.go`:**
```go
import (
    "adaptive-backend/internal/config"
    pkgconfig "adaptive-backend/pkg/config"
)

func main() {
    // 1. Load environment files explicitly (in order of priority)
    envFiles := []string{".env.local", ".env.development", ".env"}
    config.LoadEnvFiles(envFiles)

    // 2. Load YAML configuration
    cfg, err := config.LoadFromFile("config.yaml")
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // 3. Create proxy with explicit config
    proxy := pkgconfig.NewProxy(cfg)
    proxy.Run()
}
```

**Alternative: Using Builder with YAML:**
```go
// Load env files and YAML in one call
builder, err := config.FromYAML("config.yaml", []string{".env.local", ".env"})
if err != nil {
    log.Fatalf("Failed to load config: %v", err)
}

proxy := config.NewProxyWithBuilder(builder)
proxy.Run()
```

YAML Structure (`config.yaml`):
```yaml
server:              # Server settings (port, origins, environment)
endpoints:           # Per-endpoint provider configs
  chat_completions:
    providers:
      openai: {...}
      anthropic: {...}
  messages:
    providers: {...}
services:
  model_router:      # Intelligent routing settings
    cost_bias: 0.9   # 0.0=cheapest, 1.0=best performance
  redis:
    url: "redis://localhost:6379"
fallback:
  mode: "race"       # "race" or "sequential"
prompt_cache:
  enabled: false
```

### Environment Variable Substitution
Pattern: `${VAR_NAME}` or `${VAR_NAME:-default_value}`

Example:
```yaml
server:
  port: "${PORT:-8080}"  # Uses PORT env var, falls back to 8080
```

### Config Loading Principles
1. **No implicit defaults** - All paths and env files must be explicit
2. **Load order matters** - First env file has highest priority
3. **Env files loaded before YAML** - Ensures proper variable substitution
4. **Config cannot be nil** - NewProxy() requires explicit config (no nil defaults)

## Credits and Billing System (Optional)

AdaptiveProxy includes an **optional** credit-based billing system with Stripe integration. This is **disabled by default** and must be explicitly enabled via the builder API.

### Architecture

**Usage Tracking** (Always On):
- All API requests are tracked in `api_key_usage` table
- Metadata includes: projectId, organizationId, clusterId, cacheTier, userId
- Stored via GORM to PostgreSQL/MySQL/SQLite/ClickHouse

**Credits System** (Opt-in):
- Tracks organization credit balances in `organization_credits` table
- Deducts credits automatically after each API request
- Pre-flight check: blocks requests if balance <= 0
- Post-flight deduction: allows slight overdraft (request already processed)
- All transactions logged in `credit_transactions` table

**Stripe Integration** (Opt-in):
- Webhook endpoint: `/webhooks/stripe`
- Handles: `checkout.session.completed`, `payment_intent.succeeded`
- Automatically adds credits on successful payment

### Enabling Credits

```go
builder := config.New().
    Port("8080").

    // 1. Database required for credits
    WithDatabase(models.DatabaseConfig{
        Type:     models.PostgreSQL,
        DSN:      os.Getenv("DATABASE_URL"),
    }).

    // 2. Enable API key management (required)
    EnableAPIKeyAuth().

    // 3. Enable credits system
    EnableCredits().

    // 4. Optional: Enable Stripe for purchases
    WithStripe(
        os.Getenv("STRIPE_SECRET_KEY"),
        os.Getenv("STRIPE_WEBHOOK_SECRET"),
    ).

    AddOpenAICompatibleProvider("openai", ...)

proxy := config.NewProxyWithBuilder(builder)
proxy.Run()
```

### Credit Flow

1. **Pre-Request Check**: Middleware checks if organization balance > 0
   - If balance <= 0: Returns `402 Payment Required`
   - If balance > 0: Allow request to proceed

2. **Process Request**: Normal LLM provider call

3. **Post-Request Deduction**:
   - Calculate actual cost from provider response
   - Deduct from organization balance (can go negative)
   - Create transaction record with metadata
   - Link to `api_usage_id` for audit trail

4. **Future Requests**: Blocked if balance <= 0 after deduction

### Database Schema

**api_key_usage** (usage tracking):
- `metadata` (JSONB): Flexible map[string]any for consumer-specific data
  - `project_id`, `organization_id`, `cluster_id`, `cache_tier`, `user_id`
  - No hardcoded schema - general purpose proxy design

**organization_credits** (opt-in):
- `balance`: Current credit balance (USD)
- `total_purchased`: Lifetime purchases
- `total_used`: Lifetime usage

**credit_transactions** (opt-in):
- Links to: `organization_id`, `user_id`, `api_key_id`, `api_usage_id`
- Types: `purchase`, `usage`, `refund`, `promotional`
- Stripe fields: `stripe_payment_intent_id`, `stripe_session_id`

### Stripe Webhook Setup

Configure Stripe CLI for local testing:
```bash
stripe listen --forward-to localhost:8080/webhooks/stripe
```

Production webhook events to handle:
- `checkout.session.completed` - Add credits after successful purchase
- `payment_intent.succeeded` - Confirm payment processed

### Credits Service API

Located in `internal/services/credits/service.go`:

```go
// Check balance before processing
credit, err := creditsService.GetOrganizationCredit(ctx, organizationID)
if credit.Balance <= 0 {
    return ErrInsufficientCredits
}

// Deduct after processing (allows overdraft)
transaction, err := creditsService.DeductCredits(ctx, DeductCreditsParams{
    OrganizationID: "org_123",
    UserID:         "user_456",
    Amount:         0.05,
    Description:    "API usage",
    Metadata:       map[string]any{
        "provider": "openai",
        "model":    "gpt-4",
        "tokens_input": 100,
        "tokens_output": 50,
    },
    APIKeyID:   "key_prefix",
    APIUsageID: usageID,
})

// Add credits (purchase, refund, promotional)
transaction, err := creditsService.AddCredits(ctx, AddCreditsParams{
    OrganizationID:        "org_123",
    UserID:                "user_456",
    Amount:                10.00,
    Type:                  CreditTransactionPurchase,
    Description:           "Credit purchase via Stripe",
    StripePaymentIntentID: "pi_123",
    StripeSessionID:       "cs_123",
})
```

### Design Principles

1. **Opt-in**: Credits disabled by default - proxy remains general-purpose
2. **No Domain Coupling**: No hardcoded concepts like "projects" or "clusters"
3. **Flexible Metadata**: Consumers can store any data in JSONB metadata field
4. **Separation of Concerns**: Usage tracking ≠ Credits ≠ Stripe
5. **Performance**: Pre-flight check prevents unnecessary LLM calls

### Migration from Frontend

If migrating from frontend-based credit management:
- Remove tRPC `recordApiUsage` calls in frontend
- Remove credit deduction logic in frontend routes
- Frontend now only displays credit balance (read-only)
- All tracking and deduction happens in Go proxy automatically

## API Endpoints

### Chat Completions - `/v1/chat/completions`
OpenAI-compatible endpoint with extensions:

**Intelligent Routing** (empty model):
```json
{
  "model": "",
  "messages": [{"role": "user", "content": "Hello"}],
  "model_router": {
    "cost_bias": 0.3,
    "models": [{"provider": "openai"}, {"provider": "anthropic"}]
  }
}
```

**Manual Override** (provider:model format):
```json
{
  "model": "anthropic:claude-3-5-sonnet-20241022",
  "messages": [{"role": "user", "content": "Hello"}]
}
```

### Messages - `/v1/messages`
Anthropic-compatible endpoint with intelligent routing.

### Generate - `/v1/generate` and `/v1beta/models/:model\:generateContent`
Gemini-compatible endpoints with format translation.

### Select Model - `/v1/select-model`
Provider-agnostic model selection API:
```json
{
  "models": [
    {"provider": "openai", "model": "gpt-4"},
    {"provider": "anthropic", "model": "claude-3-5-sonnet-20241022"}
  ],
  "prompt": "Write a Python function"
}
```

## Provider Integration

### Adding a New Provider
1. Add provider config to `config.yaml` under relevant endpoints
2. Implement format adapters if needed (see `internal/services/format_adapter/`)
3. Add circuit breaker initialization in `main.go:65-74`
4. Update model router config to include provider in available models

### Provider-Specific Notes
- **OpenAI**: Used for embeddings in semantic cache even if not selected for completion
- **Anthropic**: Native prompt caching support via `AnthropicPromptCache`
- **Gemini**: Requires special URL escaping for `:generateContent` routes (see `main.go:106-108`)

## Testing Strategy

### Test File Organization
- Place tests adjacent to implementation: `foo.go` → `foo_test.go`
- Use table-driven tests for multiple scenarios
- Mock external dependencies (Redis, HTTP clients)

### Running Specific Tests
```bash
# Test specific package
go test ./internal/config/

# Test with race detection
go test -race ./...

# Test with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Common Development Tasks

### Modifying Configuration
1. Update YAML schema in `internal/models/*_config.go`
2. Update config loading in `internal/config/config.go`
3. Update merge logic for request overrides (e.g., `MergeProviderConfig`)
4. Update relevant `ResolveConfig*` methods

### Adding a New Endpoint
1. Create handler in `internal/api/` (follow `CompletionHandler` pattern)
2. Add endpoint config to `models/endpoint_config.go`
3. Register routes in `cmd/api/main.go:SetupRoutes()`
4. Add circuit breaker initialization if needed
5. Implement format adapters for request/response translation

### Debugging Streaming Issues
- Check `internal/services/stream/` for SSE implementation
- Use `stream_simulator` for cached response streaming
- Verify proper buffer pool usage (`bufferpool.Get()` / `bufferpool.Put()`)

## Error Handling

### Error Types (`internal/models/errors.go`)
- `ErrorTypeInvalidRequest`: 400 Bad Request
- `ErrorTypeRateLimit`: 429 Too Many Requests
- `ErrorTypeProvider`: 502 Bad Gateway
- `ErrorTypeInternal`: 500 Internal Server Error

### Circuit Breaker States
When circuit is `Open`, requests fail fast with cached error to prevent overload.

## Performance Considerations

### Redis Connection Pool
- Configured in `main.go:createRedisClient()`
- Pool size: 50 connections, min idle: 10
- Optimized for 1000+ req/s throughput

### Buffer Pools
- Use `bufferpool.Get()` / `bufferpool.Put()` for streaming responses
- Prevents excessive allocations under load

### FastHTTP Usage
- `valyala/fasthttp` used for provider HTTP clients
- Reuses connections across requests

## Go-Specific Patterns

### Error Handling
```go
// ALWAYS handle errors - never ignore
if err != nil {
    return fmt.Errorf("descriptive context: %w", err)
}
```

### Context Propagation
```go
// Pass context through all service layers
ctx := c.UserContext()
resp, err := h.service.DoWork(ctx, req)
```

### Struct Initialization
```go
// Use named fields for clarity
config := &models.Config{
    Enabled: true,
    Timeout: 30 * time.Second,
}
```

## Security Notes

- API keys never logged or exposed in responses
- Path validation prevents directory traversal (`config/config.go:33-44`)
- Input sanitization for all external inputs
- CORS configuration in `main.go:setupMiddleware()`
- Rate limiting per API key (1000 req/min default)

## Deployment

### Docker
Binary built in two-stage Dockerfile (builder + runtime Alpine).

### Environment Requirements
- Go 1.25+
- Redis (for caching and circuit breaker state)
- Network access to LLM provider APIs

### Health Checks
- `/health` endpoint checks Redis connectivity and service status
- Returns 200 OK when healthy, 503 Service Unavailable otherwise
