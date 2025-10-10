package database

import (
	"fmt"

	"github.com/Egham-7/adaptive-proxy/internal/models"
	"gorm.io/gorm"
)

type DB struct {
	*gorm.DB
	config     models.DatabaseConfig
	driverName string
}

func (db *DB) Close() error {
	if db.DB == nil {
		return nil
	}
	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (db *DB) Ping() error {
	if db.DB == nil {
		return fmt.Errorf("database not connected")
	}
	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

func (db *DB) DriverName() string {
	return db.driverName
}

func (db *DB) setConnectionPool() {
	if db.DB == nil {
		return
	}

	sqlDB, err := db.DB.DB()
	if err != nil {
		return
	}

	if db.config.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(db.config.MaxOpenConns)
	}
	if db.config.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(db.config.MaxIdleConns)
	}
	if db.config.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(0)
	}
}

func New(config models.DatabaseConfig) (*DB, error) {
	switch config.Type {
	case models.PostgreSQL:
		return newPostgreSQL(config)
	case models.MySQL:
		return newMySQL(config)
	case models.SQLite:
		return newSQLite(config)
	case models.ClickHouse:
		return newClickHouse(config)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", config.Type)
	}
}
