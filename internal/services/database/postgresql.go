package database

import (
	"fmt"

	"github.com/Egham-7/adaptive-proxy/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func newPostgreSQL(config models.DatabaseConfig) (*DB, error) {
	var dsn string
	if config.DSN != "" {
		dsn = config.DSN
	} else {
		dsn = fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			config.Host,
			config.Port,
			config.Username,
			config.Password,
			config.Database,
			getSSLMode(config.SSLMode),
		)
	}

	gormDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open PostgreSQL connection: %w", err)
	}

	db := &DB{
		DB:         gormDB,
		config:     config,
		driverName: "postgres",
	}

	db.setConnectionPool()

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	return db, nil
}

func getSSLMode(mode string) string {
	if mode == "" {
		return "disable"
	}
	return mode
}
