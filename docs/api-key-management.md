# API Key Management

Enterprise-ready API key management for adaptive-proxy with support for authentication, authorization, rate limiting, and key lifecycle management.

## Features

- **Secure Key Generation**: Cryptographically secure API key generation with SHA-256 hashing
- **Flexible Configuration**: Disabled by default, enable via builder or YAML config
- **Database-Backed**: Stores API keys in your configured database (PostgreSQL, MySQL, SQLite, ClickHouse)
- **Scope-Based Permissions**: Fine-grained access control with scopes
- **Rate Limiting**: Per-key rate limiting support
- **Key Expiration**: Optional expiration dates for temporary access
- **Key Lifecycle**: Create, list, revoke, update, and delete API keys
- **Last Used Tracking**: Monitor API key usage
- **Multiple Auth Methods**: Support for `X-API-Key` header and `Authorization: Bearer` header

## Configuration

### Via Builder (Programmatic)

```go
package main

import (
    "github.com/Egham-7/adaptive-proxy/pkg/config"
    "github.com/Egham-7/adaptive-proxy/internal/models"
)

func main() {
    builder := config.New().
        Port("8080").
        // Configure database (required for API key management)
        WithDatabase(models.DatabaseConfig{
            Type: models.PostgreSQL,
            Host: "localhost",
            Port: "5432",
            User: "postgres",
            Password: "password",
            Database: "adaptive_proxy",
        }).
        // Enable API key management with default settings
        EnableAPIKeyAuth()

    // Or configure with custom settings
    builder.WithAPIKeyManagement(models.APIKeyConfig{
        Enabled:        true,
        HeaderName:     "X-API-Key",
        RequireForAll:  false,  // Optional: require for all endpoints
        AllowAnonymous: true,   // Allow requests without API key
    })
}
```

### Via YAML Config

```yaml
server:
  port: "8080"
  allowed_origins: "*"
  environment: "production"
  log_level: "info"
  api_key:
    enabled: true
    header_name: "X-API-Key"
    require_for_all: false
    allow_anonymous: true

database:
  type: "postgresql"
  host: "${DB_HOST:-localhost}"
  port: "${DB_PORT:-5432}"
  user: "${DB_USER:-postgres}"
  password: "${DB_PASSWORD}"
  database: "${DB_NAME:-adaptive_proxy}"
```

## Usage

### Basic Setup

API key management is automatically configured when you enable it via the builder. The proxy handles all the setup including database migrations, middleware registration, and route setup.

```go
package main

import (
    "github.com/Egham-7/adaptive-proxy/pkg/config"
    "github.com/Egham-7/adaptive-proxy/internal/models"
)

func main() {
    // Create builder with API key management enabled
    builder := config.New().
        Port("8080").
        WithDatabase(models.DatabaseConfig{
            Type:     models.PostgreSQL,
            Host:     "localhost",
            Port:     "5432",
            User:     "postgres",
            Password: "password",
            Database: "adaptive_proxy",
        }).
        EnableAPIKeyAuth()

    // Create and run proxy - API key management is automatically set up
    proxy := config.NewProxyWithBuilder(builder)
    if err := proxy.Run(); err != nil {
        panic(err)
    }
}
```

That's it! The proxy automatically:
- Runs database migrations for API key tables
- Sets up authentication middleware
- Registers API key management endpoints at `/admin/api-keys`
- Configures rate limiting and usage tracking

### Create API Keys

**Request:**
```bash
curl -X POST http://localhost:8080/admin/api-keys \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Production API Key",
    "scopes": ["chat:read", "chat:write"],
    "rate_limit_rpm": 1000,
    "metadata": "Production environment key",
    "expires_at": "2025-12-31T23:59:59Z"
  }'
```

**Response:**
```json
{
  "id": 1,
  "name": "Production API Key",
  "key": "apk_Ab3dEfGhI1jKlMnOpQrStUvWxYz0123456789ABC",
  "key_prefix": "apk_Ab3dEfGh",
  "scopes": "chat:read,chat:write",
  "rate_limit_rpm": 1000,
  "metadata": "Production environment key",
  "is_active": true,
  "expires_at": "2025-12-31T23:59:59Z",
  "created_at": "2025-01-15T10:30:00Z",
  "updated_at": "2025-01-15T10:30:00Z"
}
```

**⚠️ Important:** Save the `key` value immediately - it's only shown once during creation.

### Use API Keys

**Using X-API-Key header:**
```bash
curl http://localhost:8080/v1/chat/completions \
  -H "X-API-Key: apk_Ab3dEfGhI1jKlMnOpQrStUvWxYz0123456789ABC" \
  -H "Content-Type: application/json" \
  -d '{"model": "gpt-4", "messages": [...]}'
```

**Using Authorization header:**
```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer apk_Ab3dEfGhI1jKlMnOpQrStUvWxYz0123456789ABC" \
  -H "Content-Type: application/json" \
  -d '{"model": "gpt-4", "messages": [...]}'
```

### List API Keys

```bash
curl http://localhost:8080/admin/api-keys?limit=10&offset=0
```

**Response:**
```json
{
  "data": [
    {
      "id": 1,
      "name": "Production API Key",
      "key_prefix": "apk_Ab3dEfGh",
      "scopes": "chat:read,chat:write",
      "is_active": true,
      "last_used_at": "2025-01-15T12:45:30Z",
      "created_at": "2025-01-15T10:30:00Z"
    }
  ],
  "total": 1,
  "limit": 10,
  "offset": 0
}
```

### Revoke API Key

```bash
curl -X POST http://localhost:8080/admin/api-keys/1/revoke
```

### Update API Key

```bash
curl -X PATCH http://localhost:8080/admin/api-keys/1 \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Updated Key Name",
    "rate_limit_rpm": 2000
  }'
```

### Delete API Key

```bash
curl -X DELETE http://localhost:8080/admin/api-keys/1
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/admin/api-keys` | Create new API key |
| `GET` | `/admin/api-keys` | List all API keys |
| `GET` | `/admin/api-keys/:id` | Get API key by ID |
| `PATCH` | `/admin/api-keys/:id` | Update API key |
| `POST` | `/admin/api-keys/:id/revoke` | Revoke API key |
| `DELETE` | `/admin/api-keys/:id` | Delete API key |

## Security Features

### 1. Key Hashing
- API keys are hashed using SHA-256 before storage
- Only key prefix (first 12 characters) is stored in plaintext for identification
- Full key is never retrievable after creation

### 2. Key Format
- Prefix: `apk_` (API Key)
- Length: 46 characters total
- Format: `apk_` + 43 characters base64url-encoded random bytes

### 3. Automatic Expiration
- Keys can have optional expiration dates
- Expired keys are automatically rejected during validation

### 4. Activity Tracking
- `last_used_at` timestamp updated on each successful authentication
- Monitor key usage patterns and detect anomalies

### 5. Scope-Based Access Control

Scopes can be assigned to API keys during creation. While the proxy doesn't enforce scope-based routing by default, you can access and validate scopes in custom middleware if needed. See the [Advanced Usage](#advanced-usage) section for details on implementing custom scope validation.

### 6. Per-Key Rate Limiting
- Set custom rate limits per API key
- Override global rate limits
- Stored in `rate_limit_rpm` field

## Database Schema

```sql
CREATE TABLE api_keys (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    key_hash VARCHAR(64) UNIQUE NOT NULL,
    key_prefix VARCHAR(12),
    metadata TEXT,
    scopes TEXT,
    rate_limit_rpm INTEGER,
    is_active BOOLEAN DEFAULT true,
    expires_at TIMESTAMP,
    last_used_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_key_prefix ON api_keys(key_prefix);
CREATE INDEX idx_api_keys_is_active ON api_keys(is_active);
CREATE INDEX idx_api_keys_expires_at ON api_keys(expires_at);
```

## Advanced Usage

### Custom Middleware and Routes

If you need to extend the proxy with custom routes that access API key information, you can use the Fiber context locals that are automatically set by the authentication middleware:

```go
package main

import (
    "github.com/Egham-7/adaptive-proxy/pkg/config"
    "github.com/Egham-7/adaptive-proxy/internal/models"
    "github.com/gofiber/fiber/v2"
)

func main() {
    builder := config.New().
        Port("8080").
        WithDatabase(dbConfig).
        EnableAPIKeyAuth()

    // Add custom middleware that accesses API key info
    builder.WithMiddleware(func(c *fiber.Ctx) error {
        // Access API key information if authenticated
        if apiKey, ok := c.Locals("api_key").(*models.APIKey); ok {
            // Log API key usage
            fmt.Printf("Request from API key: %s\n", apiKey.Name)
        }
        return c.Next()
    })

    proxy := config.NewProxyWithBuilder(builder)
    
    // You can also add custom routes to the proxy's app after creation
    // Note: This is advanced usage - most users won't need this
    
    if err := proxy.Run(); err != nil {
        panic(err)
    }
}
```

### Accessing API Key Information in Context

When a request is authenticated with an API key, the following information is available in the Fiber context:

```go
// Get API key object
apiKey := c.Locals("api_key").(*models.APIKey)

// Get API key ID
keyID := c.Locals("api_key_id").(uint)

// Get scopes
scopes := c.Locals("api_key_scopes").([]string)

// Get rate limit
rateLimit := c.Locals("api_key_rate_limit").(int)
```

## Best Practices

1. **Enable in Production**: Always enable API key management in production environments
2. **Secure Admin Endpoints**: Protect API key management endpoints with additional authentication
3. **Rotate Keys Regularly**: Implement key rotation policies
4. **Use Scopes**: Leverage scope-based permissions for fine-grained access control
5. **Set Expiration**: Use expiration dates for temporary access
6. **Monitor Usage**: Track `last_used_at` to detect unused or compromised keys
7. **Database Required**: API key management requires a configured database
8. **Environment Variables**: Store sensitive config in environment variables

## Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `false` | Enable/disable API key management |
| `header_name` | string | `X-API-Key` | Custom header name for API key |
| `require_for_all` | bool | `false` | Require API key for all endpoints |
| `allow_anonymous` | bool | `true` | Allow requests without API key |

## Migration Guide

### From No Authentication

```go
// Before
builder := config.New().Port("8080")

// After
builder := config.New().
    Port("8080").
    WithDatabase(dbConfig).
    EnableAPIKeyAuth()
```

### YAML Config Migration

```yaml
# Before
server:
  port: "8080"

# After
server:
  port: "8080"
  api_key:
    enabled: true
    header_name: "X-API-Key"
    allow_anonymous: true

database:
  type: "postgresql"
  host: "localhost"
  # ... database config
```
