# Database Configuration

AdaptiveProxy supports PostgreSQL, MySQL, SQLite, and ClickHouse databases for internal use. The database is configured through the SDK but all operations are handled internally by the proxy.

## Installation

Install GORM and the required database drivers:

```bash
go get -u gorm.io/gorm
go get -u gorm.io/driver/postgres
go get -u gorm.io/driver/mysql
go get -u gorm.io/driver/sqlite
go get -u gorm.io/driver/clickhouse
```

## Usage

### Using Config Builder

```go
package main

import (
    "log"
    
    "github.com/Egham-7/adaptive-proxy/internal/models"
    "github.com/Egham-7/adaptive-proxy/pkg/config"
)

func main() {
    // Option 1: Using DSN/Connection URL (recommended)
    builder := config.New().
        Port("8080").
        WithDatabase(models.DatabaseConfig{
            Type: models.PostgreSQL,
            DSN:  "postgres://user:pass@localhost:5432/mydb?sslmode=disable",
        })

    // Option 2: Using individual fields
    builder := config.New().
        Port("8080").
        WithDatabase(models.DatabaseConfig{
            Type:            models.PostgreSQL,
            Host:            "localhost",
            Port:            5432,
            Username:        "postgres",
            Password:        "password",
            Database:        "mydb",
            SSLMode:         "disable",
            MaxOpenConns:    25,
            MaxIdleConns:    5,
            ConnMaxLifetime: 300,
        })

    // MySQL with DSN
    builder := config.New().
        WithDatabase(models.DatabaseConfig{
            Type: models.MySQL,
            DSN:  "user:pass@tcp(localhost:3306)/mydb?parseTime=true",
        })

    // SQLite (DSN not supported, use FilePath)
    builder := config.New().
        WithDatabase(models.DatabaseConfig{
            Type:     models.SQLite,
            FilePath: "./data.db",
        })

    // ClickHouse with DSN
    builder := config.New().
        WithDatabase(models.DatabaseConfig{
            Type: models.ClickHouse,
            DSN:  "clickhouse://default:@localhost:9000/default",
        })

    proxy := config.NewProxyWithBuilder(builder)

    if err := proxy.Run(); err != nil {
        log.Fatal(err)
    }
}
```

### Using YAML Configuration

```yaml
server:
  port: "8080"
  environment: "development"

# Option 1: Using DSN (recommended)
database:
  type: postgresql
  dsn: "${DATABASE_URL:-postgres://user:pass@localhost:5432/adaptive_db?sslmode=disable}"
  max_open_conns: 25
  max_idle_conns: 5

# Option 2: Using individual fields (DSN takes precedence if both provided)
# database:
#   type: postgresql
#   host: localhost
#   port: 5432
#   username: ${DB_USER}
#   password: ${DB_PASSWORD}
#   database: adaptive_db
#   ssl_mode: disable
#   max_open_conns: 25
#   max_idle_conns: 5
#   conn_max_lifetime: 300

endpoints:
  chat_completions:
    providers:
      openai:
        api_key: ${OPENAI_API_KEY}
```

```go
package main

import (
    "log"
    
    "github.com/Egham-7/adaptive-proxy/pkg/config"
)

func main() {
    builder, err := config.FromYAML("config.yaml", []string{".env.local", ".env"})
    if err != nil {
        log.Fatal(err)
    }

    proxy := config.NewProxyWithBuilder(builder)

    if err := proxy.Run(); err != nil {
        log.Fatal(err)
    }
}
```

## Supported Database Types

| Database   | Constant              | DSN Format Example                                   | Required Fields (if not using DSN)        |
|------------|-----------------------|------------------------------------------------------|-------------------------------------------|
| PostgreSQL | `models.PostgreSQL`   | `postgres://user:pass@host:5432/db?sslmode=disable` | Host, Port, Username, Password, Database  |
| MySQL      | `models.MySQL`        | `user:pass@tcp(host:3306)/db?parseTime=true`        | Host, Port, Username, Password, Database  |
| SQLite     | `models.SQLite`       | N/A (use FilePath)                                   | FilePath                                  |
| ClickHouse | `models.ClickHouse`   | `clickhouse://user:pass@host:9000/db`               | Host, Port, Username, Password, Database  |

## Configuration Options

### Common Options

- `type` (DatabaseType): Database type (postgresql, mysql, sqlite, clickhouse)
- `dsn` (string): Full connection URL/DSN (recommended, takes precedence over individual fields)
- `database` (string): Database name (required if not using DSN)
- `max_open_conns` (int): Maximum number of open connections
- `max_idle_conns` (int): Maximum number of idle connections
- `conn_max_lifetime` (int): Maximum lifetime of connections in seconds

### PostgreSQL/MySQL/ClickHouse Specific (when not using DSN)

- `host` (string): Database host
- `port` (int): Database port
- `username` (string): Database username
- `password` (string): Database password

### PostgreSQL Specific (when not using DSN)

- `ssl_mode` (string): SSL mode (disable, require, verify-ca, verify-full)

### SQLite Specific

- `file_path` (string): Path to SQLite database file

## DSN vs Individual Fields

**Using DSN (recommended):**
- Simpler configuration
- Easier to use with environment variables
- Standard connection string format
- Takes precedence if both DSN and individual fields are provided

**Using Individual Fields:**
- More explicit configuration
- Useful when fields come from different sources
- Required for SQLite (FilePath)

## Internal Usage

The database connection is managed internally by AdaptiveProxy. SDK users can configure the database but cannot directly access it for operations. All database interactions are handled by the proxy's internal services.
