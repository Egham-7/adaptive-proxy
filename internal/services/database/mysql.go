package database

import (
	"fmt"

	"github.com/Egham-7/adaptive-proxy/internal/models"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func newMySQL(config models.DatabaseConfig) (*DB, error) {
	var dsn string
	if config.DSN != "" {
		dsn = config.DSN
	} else {
		dsn = fmt.Sprintf(
			"%s:%s@tcp(%s:%d)/%s?parseTime=true",
			config.Username,
			config.Password,
			config.Host,
			config.Port,
			config.Database,
		)
	}

	gormDB, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open MySQL connection: %w", err)
	}

	db := &DB{
		DB:         gormDB,
		config:     config,
		driverName: "mysql",
	}

	db.setConnectionPool()

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping MySQL: %w", err)
	}

	return db, nil
}
