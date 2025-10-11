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
	DSN      string       `yaml:"dsn,omitempty" json:"dsn,omitzero"`
	Host     string       `yaml:"host,omitempty" json:"host,omitzero"`
	Port     int          `yaml:"port,omitempty" json:"port,omitzero"`
	Username string       `yaml:"username,omitempty" json:"username,omitzero"`
	Password string       `yaml:"password,omitempty" json:"password,omitzero"`
	Database string       `yaml:"database" json:"database"`
	SSLMode  string       `yaml:"ssl_mode,omitempty" json:"ssl_mode,omitzero"`
	FilePath string       `yaml:"file_path,omitempty" json:"file_path,omitzero"`

	MaxOpenConns    int `yaml:"max_open_conns,omitempty" json:"max_open_conns,omitzero"`
	MaxIdleConns    int `yaml:"max_idle_conns,omitempty" json:"max_idle_conns,omitzero"`
	ConnMaxLifetime int `yaml:"conn_max_lifetime,omitempty" json:"conn_max_lifetime,omitzero"`
}
