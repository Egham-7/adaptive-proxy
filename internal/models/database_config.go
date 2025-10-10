package models

type DatabaseType string

const (
	PostgreSQL DatabaseType = "postgresql"
	MySQL      DatabaseType = "mysql"
	SQLite     DatabaseType = "sqlite"
	ClickHouse DatabaseType = "clickhouse"
)

type DatabaseConfig struct {
	Type     DatabaseType `yaml:"type" json:"type"`
	DSN      string       `yaml:"dsn,omitempty" json:"dsn,omitempty"`
	Host     string       `yaml:"host,omitempty" json:"host,omitempty"`
	Port     int          `yaml:"port,omitempty" json:"port,omitempty"`
	Username string       `yaml:"username,omitempty" json:"username,omitempty"`
	Password string       `yaml:"password,omitempty" json:"password,omitempty"`
	Database string       `yaml:"database" json:"database"`
	SSLMode  string       `yaml:"ssl_mode,omitempty" json:"ssl_mode,omitempty"`
	FilePath string       `yaml:"file_path,omitempty" json:"file_path,omitempty"`

	MaxOpenConns    int `yaml:"max_open_conns,omitempty" json:"max_open_conns,omitempty"`
	MaxIdleConns    int `yaml:"max_idle_conns,omitempty" json:"max_idle_conns,omitempty"`
	ConnMaxLifetime int `yaml:"conn_max_lifetime,omitempty" json:"conn_max_lifetime,omitempty"`
}
