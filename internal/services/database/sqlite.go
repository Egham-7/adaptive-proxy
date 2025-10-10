package database

import (
	"fmt"

	"github.com/Egham-7/adaptive-proxy/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newSQLite(config models.DatabaseConfig) (*DB, error) {
	if config.FilePath == "" {
		return nil, fmt.Errorf("file_path is required for SQLite")
	}

	gormDB, err := gorm.Open(sqlite.Open(config.FilePath), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite connection: %w", err)
	}

	db := &DB{
		DB:         gormDB,
		config:     config,
		driverName: "sqlite3",
	}

	db.setConnectionPool()

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping SQLite: %w", err)
	}

	return db, nil
}
