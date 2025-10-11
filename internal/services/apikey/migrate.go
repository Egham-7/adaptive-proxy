package apikey

import (
	"fmt"

	"github.com/Egham-7/adaptive-proxy/internal/models"
	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&models.APIKey{}); err != nil {
		return fmt.Errorf("failed to migrate api_keys table: %w", err)
	}

	if err := createIndexes(db); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	return nil
}

func createIndexes(db *gorm.DB) error {
	indexes := []struct {
		name   string
		table  string
		column string
		unique bool
	}{
		{name: "idx_api_keys_key_hash", table: "api_keys", column: "key_hash", unique: true},
		{name: "idx_api_keys_key_prefix", table: "api_keys", column: "key_prefix", unique: false},
		{name: "idx_api_keys_is_active", table: "api_keys", column: "is_active", unique: false},
		{name: "idx_api_keys_expires_at", table: "api_keys", column: "expires_at", unique: false},
	}

	for _, idx := range indexes {
		var exists bool
		query := `SELECT EXISTS (
			SELECT 1 FROM information_schema.statistics 
			WHERE table_schema = DATABASE() 
			AND table_name = ? 
			AND index_name = ?
		)`

		if err := db.Raw(query, idx.table, idx.name).Scan(&exists).Error; err != nil {
			query = `SELECT EXISTS (
				SELECT 1 FROM pg_indexes 
				WHERE tablename = ? 
				AND indexname = ?
			)`
			if err := db.Raw(query, idx.table, idx.name).Scan(&exists).Error; err != nil {
				continue
			}
		}

		if !exists {
			uniqueClause := ""
			if idx.unique {
				uniqueClause = "UNIQUE"
			}
			sql := fmt.Sprintf("CREATE %s INDEX %s ON %s (%s)", uniqueClause, idx.name, idx.table, idx.column)
			if err := db.Exec(sql).Error; err != nil {
				return fmt.Errorf("failed to create index %s: %w", idx.name, err)
			}
		}
	}

	return nil
}
