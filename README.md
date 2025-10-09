# AdaptiveProxy

**Intelligent multi-provider LLM routing library for Go applications**

A high-performance Go library that routes requests across multiple LLM providers (OpenAI, Anthropic, Gemini, DeepSeek, Groq) with intelligent fallback, caching, and cost optimization.

## 🚀 Quick Start

### Installation

```bash
go get github.com/Egham-7/adaptive-proxy
```

### Basic Usage

```go
package main

import (
    "log"
    "os"

    "github.com/Egham-7/adaptive-proxy/pkg/config"
)

func main() {
    // Configure proxy with OpenAI and Anthropic fallback
    builder := config.New().
        Port("8080").
        AddOpenAICompatibleProvider("openai",
            config.NewProviderBuilder(os.Getenv("OPENAI_API_KEY")).Build(),
        ).
        AddAnthropicCompatibleProvider("anthropic",
            config.NewProviderBuilder(os.Getenv("ANTHROPIC_API_KEY")).Build(),
        )

    // Start server
    srv := config.NewProxyWithBuilder(builder)
    if err := srv.Run(); err != nil {
        log.Fatal(err)
    }
}
```

Your proxy is now running on `http://localhost:8080` with automatic fallback!

## ⭐ Key Features

- **Zero Vendor Lock-in** - Switch providers without code changes
- **Multi-Provider Support** - OpenAI, Anthropic, Groq, DeepSeek, Gemini
- **Production-Ready** - Redis caching, rate limiting, graceful shutdown
- **Type-Safe** - Fluent builder API with full Go type safety

## 📚 Documentation

**Full documentation available in [docs/](./docs/)**

- [Quick Start](./docs/quickstart.md) - Get running in 5 minutes
- [Installation](./docs/installation.md) - Installation and setup
- [Basic Usage](./docs/basic-usage.md) - Core concepts and patterns
- [Providers](./docs/providers.md) - Configure OpenAI, Anthropic, Gemini, and custom providers
- [Caching](./docs/caching.md) - Redis-backed prompt caching and semantic search
- [Routing](./docs/routing.md) - Intelligent model selection and cost optimization
- [Fallback](./docs/fallback.md) - Multi-provider fallback strategies
- [Middleware](./docs/middleware.md) - Rate limiting, timeouts, and custom middleware

## 💡 API Usage

Once your proxy is running, make requests using the OpenAI-compatible API:

### Chat Completions

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

### Intelligent Routing with Cost Optimization

```json
{
  "model": "",
  "messages": [{ "role": "user", "content": "Complex analysis task" }],
  "model_router": {
    "cost_bias": 0.3,
    "models": [
      { "provider": "openai" },
      { "provider": "anthropic", "model_name": "claude-3-5-sonnet-20241022" }
    ]
  },
  "cache": {
    "enabled": true,
    "semantic_threshold": 0.85
  },
  "fallback": {
    "mode": "sequential"
  }
}
```

### Other Endpoints

- `GET /v1/models` - List available models
- `GET /health` - Health check
- `POST /v1/messages` - Anthropic-compatible messages endpoint

## 🛠️ Development

### Running from Source

```bash
git clone https://github.com/Egham-7/adaptive-proxy.git
cd adaptive-proxy
cp .env.example .env.local  # Add your API keys
go run cmd/api/main.go
```

### Code Quality (Required Before Commit)

```bash
gofmt -w . && golangci-lint run && go mod tidy && go vet ./... && go test ./...
```

### Install Development Tools

```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

## 🏗️ Architecture

See [CLAUDE.md](./CLAUDE.md) for detailed architecture documentation and [AGENTS.md](./AGENTS.md) for AI agent development guidelines.

## 🤝 Contributing

Contributions welcome! Please ensure all pre-commit checks pass:

```bash
gofmt -w . && golangci-lint run && go mod tidy && go vet ./... && go test ./...
```

## 📝 License

See [LICENSE](./LICENSE) for details.

## 🆘 Support

- **Documentation**: [docs/](./docs/)
- **Issues**: [GitHub Issues](https://github.com/Egham-7/adaptive-proxy/issues)
- **Discussions**: [GitHub Discussions](https://github.com/Egham-7/adaptive-proxy/discussions)

