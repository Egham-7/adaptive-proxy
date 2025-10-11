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

### 1. Initialize API Key Service

```go
package main

import (
    "github.com/Egham-7/adaptive-proxy/internal/services/database"
    "github.com/Egham-7/adaptive-proxy/internal/services/apikey"
    "github.com/Egham-7/adaptive-proxy/internal/services/middleware"
    "github.com/Egham-7/adaptive-proxy/internal/api"
    "github.com/gofiber/fiber/v2"
)

func main() {
    // Load config
    cfg := config.New().
        WithDatabase(dbConfig).
        EnableAPIKeyAuth().
        Build()

    // Initialize database
    db, err := database.New(*cfg.Database)
    if err != nil {
        panic(err)
    }

    // Run migrations
    if err := apikey.Migrate(db.DB); err != nil {
        panic(err)
    }

    // Create API key service
    apikeyService := apikey.NewService(db.DB)

    // Create middleware
    apikeyMiddleware := middleware.NewAPIKeyMiddleware(
        apikeyService,
        cfg.Server.APIKeyConfig,
    )

    // Create API handler
    apikeyHandler := api.NewAPIKeyHandler(apikeyService)

    // Setup Fiber app
    app := fiber.New()

    // Apply authentication middleware globally
    app.Use(apikeyMiddleware.Authenticate())

    // Register API key management routes
    apikeyHandler.RegisterRoutes(app, "/admin/api-keys")

    // Protected route example
    app.Get("/protected", 
        apikeyMiddleware.RequireAPIKey(),
        func(c *fiber.Ctx) error {
            return c.JSON(fiber.Map{"message": "Access granted"})
        },
    )

    // Scope-protected route
    app.Post("/admin/users",
        apikeyMiddleware.RequireAPIKey(),
        apikeyMiddleware.RequireScope("admin", "users:write"),
        func(c *fiber.Ctx) error {
            return c.JSON(fiber.Map{"message": "User created"})
        },
    )

    app.Listen(":8080")
}
```

### 2. Create API Keys

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

### 3. Use API Keys

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

### 4. List API Keys

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

### 5. Revoke API Key

```bash
curl -X POST http://localhost:8080/admin/api-keys/1/revoke
```

### 6. Update API Key

```bash
curl -X PATCH http://localhost:8080/admin/api-keys/1 \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Updated Key Name",
    "rate_limit_rpm": 2000
  }'
```

### 7. Delete API Key

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
```go
// Require specific scopes
app.Post("/admin/users",
    apikeyMiddleware.RequireAPIKey(),
    apikeyMiddleware.RequireScope("admin", "users:write"),
    handler,
)
```

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

## Middleware Options

### 1. Optional Authentication
```go
// Authenticate if key provided, allow anonymous otherwise
app.Use(apikeyMiddleware.Authenticate())
```

### 2. Required Authentication
```go
// Always require API key
app.Use(apikeyMiddleware.RequireAPIKey())
```

### 3. Scope Validation
```go
// Require specific scopes
app.Use(apikeyMiddleware.RequireScope("admin", "users:write"))
```

### 4. Access Key Information in Handlers
```go
app.Get("/profile", func(c *fiber.Ctx) error {
    // Get API key object
    apiKey := c.Locals("api_key").(*models.APIKey)
    
    // Get API key ID
    keyID := c.Locals("api_key_id").(uint)
    
    // Get scopes
    scopes := c.Locals("api_key_scopes").([]string)
    
    // Get rate limit
    rateLimit := c.Locals("api_key_rate_limit").(int)
    
    return c.JSON(fiber.Map{
        "key_id": keyID,
        "scopes": scopes,
    })
})
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
