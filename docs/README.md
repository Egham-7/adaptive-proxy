# AdaptiveProxy Documentation

**Intelligent multi-provider LLM routing for Go applications**

AdaptiveProxy is a high-performance Go library that routes requests across multiple LLM providers (OpenAI, Anthropic, Gemini, DeepSeek, Groq) with intelligent fallback, caching, and cost optimization.

## 🚀 Why AdaptiveProxy?

- **60-80% cost reduction** through intelligent provider selection and caching
- **99.9% uptime** with automatic fallback and circuit breakers
- **Zero vendor lock-in** - switch providers without code changes
- **Production-ready** - Redis caching, rate limiting, graceful shutdown
- **Type-safe** - Fluent builder API with full Go type safety

## 📚 Documentation

### Getting Started
- [Quick Start](./quickstart.md) - Get running in 5 minutes
- [Installation](./installation.md) - Installation and setup
- [Basic Usage](./basic-usage.md) - Core concepts and patterns

### Configuration
- [Providers](./providers.md) - Configure OpenAI, Anthropic, Gemini, and custom providers
- [Caching](./caching.md) - Redis-backed prompt caching and semantic search
- [Routing](./routing.md) - Intelligent model selection and cost optimization
- [Fallback](./fallback.md) - Multi-provider fallback strategies
- [Middleware](./middleware.md) - Rate limiting, timeouts, and custom middleware

### Advanced
- [YAML Configuration](../config.example.yml) - Using YAML files for configuration

### Examples
- [Examples](../examples/) - Real-world usage examples

### Contributing
- [Architecture](../CLAUDE.md) - System architecture and design patterns
- [Development Guide](../AGENTS.md) - Contributing guidelines

## 🎯 Quick Example

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

That's it! Your proxy is now running on `http://localhost:8080` with automatic fallback.

## 🆘 Need Help?

- **Bugs?** [Open an issue](https://github.com/Egham-7/adaptive-proxy/issues)

## 📖 Next Steps

1. **New to AdaptiveProxy?** Start with [Quick Start](./quickstart.md)
