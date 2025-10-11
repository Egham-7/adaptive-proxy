package database

import (
	"fmt"

	"github.com/Egham-7/adaptive-proxy/internal/models"
	"gorm.io/driver/clickhouse"
	"gorm.io/gorm"
)

func newClickHouse(config models.DatabaseConfig) (*DB, error) {
	var dsn string
	if config.DSN != "" {
		dsn = config.DSN
	} else {
		dsn = fmt.Sprintf(
			"clickhouse://%s:%s@%s:%d/%s",
			config.Username,
			config.Password,
			config.Host,
			config.Port,
			config.Database,
		)
	}

	gormDB, err := gorm.Open(clickhouse.New(clickhouse.Config{
		DSN:                          dsn,
		DisableDatetimePrecision:     false,
		DontSupportRenameColumn:      false,
		DontSupportEmptyDefaultValue: false,
		SkipInitializeWithVersion:    false,
		DefaultGranularity:           3,
		DefaultCompression:           "LZ4",
		DefaultIndexType:             "minmax",
		DefaultTableEngineOpts:       "ENGINE=MergeTree() ORDER BY id",
	}), &gorm.Config{
		// Disable prepared statements for ClickHouse - the driver has incomplete support
		// This prevents panics and errors with SELECT queries and column introspection
		// See: https://github.com/go-gorm/gorm/issues/7493
		PrepareStmt: false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open ClickHouse connection: %w", err)
	}

	db := &DB{
		DB:         gormDB,
		config:     config,
		driverName: "clickhouse",
	}

	db.setConnectionPool()

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping ClickHouse: %w", err)
	}

	return db, nil
}
