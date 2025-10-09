# Installation

## Requirements

- **Go**: 1.25 or later
- **Redis** (optional): For caching and model routing features
- **LLM Provider API Key**: At least one from OpenAI, Anthropic, Gemini, DeepSeek, or Groq

## Install AdaptiveProxy

### As a Library (Recommended)

Add to your Go project:

```bash
go get adaptive-backend
```

Or add to your `go.mod`:

```go
require adaptive-backend latest
```

Then run:

```bash
go mod tidy
```

### From Source

Clone and build:

```bash
git clone https://github.com/Egham-7/adaptive-proxy.git
cd adaptive-proxy
go build -o adaptive-proxy cmd/api/main.go
```

## Install Redis (Optional)

Redis is required for:
- Prompt-response caching
- Semantic caching
- Model router caching
- Circuit breaker state persistence

### macOS

```bash
brew install redis
brew services start redis
```

Verify:
```bash
redis-cli ping  # Should return: PONG
```

### Linux (Ubuntu/Debian)

```bash
sudo apt update
sudo apt install redis-server
sudo systemctl start redis
sudo systemctl enable redis
```

Verify:
```bash
redis-cli ping  # Should return: PONG
```

### Docker

```bash
docker run -d \
  --name redis \
  -p 6379:6379 \
  redis:alpine
```

Verify:
```bash
docker exec redis redis-cli ping  # Should return: PONG
```

### Windows

Download from [Redis for Windows](https://github.com/microsoftarchive/redis/releases) or use WSL2 with the Linux instructions.

## Get API Keys

### OpenAI

1. Go to [platform.openai.com](https://platform.openai.com)
2. Create account or sign in
3. Navigate to API keys
4. Create new secret key
5. Save it: `export OPENAI_API_KEY="sk-..."`

### Anthropic

1. Go to [console.anthropic.com](https://console.anthropic.com)
2. Create account or sign in
3. Navigate to API Keys
4. Create key
5. Save it: `export ANTHROPIC_API_KEY="sk-ant-..."`

### Google Gemini

1. Go to [ai.google.dev](https://ai.google.dev)
2. Get API key
3. Save it: `export GEMINI_API_KEY="..."`

### DeepSeek

1. Go to [platform.deepseek.com](https://platform.deepseek.com)
2. Create account
3. Get API key
4. Save it: `export DEEPSEEK_API_KEY="..."`

### Groq

1. Go to [console.groq.com](https://console.groq.com)
2. Create account
3. Get API key
4. Save it: `export GROQ_API_KEY="..."`

## Environment Setup

### Using .env Files

Create `.env`:

```bash
# Server
PORT=8080
ENVIRONMENT=development

# LLM Providers
OPENAI_API_KEY=sk-...
ANTHROPIC_API_KEY=sk-ant-...
GEMINI_API_KEY=...

# Redis (optional)
REDIS_URL=redis://localhost:6379

# Model Router (optional)
ROUTER_URL=http://localhost:8000
JWT_SECRET=your-secret-here
```

Load in your app:

```go
import "adaptive-backend/internal/config"

config.LoadEnvFiles([]string{".env.local", ".env"})
```

### Using Shell Environment

```bash
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."
export REDIS_URL="redis://localhost:6379"
```

Make permanent (add to `~/.bashrc` or `~/.zshrc`):

```bash
echo 'export OPENAI_API_KEY="sk-..."' >> ~/.bashrc
source ~/.bashrc
```

## Verify Installation

Create `test.go`:

```go
package main

import (
    "fmt"
    "os"
    
    "adaptive-backend/pkg/config"
)

func main() {
    if os.Getenv("OPENAI_API_KEY") == "" {
        fmt.Println("❌ OPENAI_API_KEY not set")
        return
    }
    
    builder := config.New()
    fmt.Println("✅ AdaptiveProxy installed successfully!")
    fmt.Printf("   Go version: %s\n", builder.Build().GoVersion())
}
```

Run:

```bash
go run test.go
```

Expected output:
```
✅ AdaptiveProxy installed successfully!
   Go version: go1.25.0
```

## Next Steps

- [Quick Start](./quickstart.md) - Get running in 5 minutes
- [Basic Usage](./basic-usage.md) - Learn core concepts
- [Providers](./providers.md) - Configure your LLM providers

## Troubleshooting

### "package adaptive-backend is not in GOROOT"

Run:
```bash
go mod init your-project-name
go get adaptive-backend
```

### "cannot find package"

Ensure you're using Go 1.25+:
```bash
go version  # Should be >= go1.25.0
```

Upgrade if needed:
```bash
# macOS
brew upgrade go

# Linux
sudo snap refresh go --classic

# Windows
# Download from golang.org/dl
```

### Redis connection issues

Check Redis is running:
```bash
redis-cli ping
```

If not running:
```bash
# macOS
brew services start redis

# Linux
sudo systemctl start redis

# Docker
docker start redis
```

### Import path issues

If using the library in a different module, ensure your `go.mod` has the correct replace directive or you've pushed the module to a Git repository.

## Development Tools (Optional)

### Linter

```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### Formatter

```bash
go install golang.org/x/tools/cmd/goimports@latest
```

### Testing

```bash
go test ./...
```

All set! Continue to [Quick Start](./quickstart.md) →
