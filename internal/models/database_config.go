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
	DSN      string       `yaml:"dsn,omitzero" json:"dsn,omitzero"`
	Host     string       `yaml:"host,omitzero" json:"host,omitzero"`
	Port     int          `yaml:"port,omitzero" json:"port,omitzero"`
	Username string       `yaml:"username,omitzero" json:"username,omitzero"`
	Password string       `yaml:"password,omitzero" json:"password,omitzero"`
	Database string       `yaml:"database" json:"database"`
	SSLMode  string       `yaml:"ssl_mode,omitzero" json:"ssl_mode,omitzero"`
	FilePath string       `yaml:"file_path,omitzero" json:"file_path,omitzero"`

	MaxOpenConns    int `yaml:"max_open_conns,omitzero" json:"max_open_conns,omitzero"`
	MaxIdleConns    int `yaml:"max_idle_conns,omitzero" json:"max_idle_conns,omitzero"`
	ConnMaxLifetime int `yaml:"conn_max_lifetime,omitzero" json:"conn_max_lifetime,omitzero"`
}
