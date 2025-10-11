package database

import (
	"gorm.io/gorm"
)

// RunClickHouseMigrations performs safe migrations for ClickHouse by creating tables
// directly without using GORM's AutoMigrate (which has issues with ClickHouse driver)
func RunClickHouseMigrations(db *gorm.DB) error {
	// Create api_keys table
	apiKeysSQL := `
	CREATE TABLE IF NOT EXISTS api_keys (
		id UInt64,
		name String NOT NULL,
		key_hash String NOT NULL,
		key_prefix String NOT NULL,
		metadata String NOT NULL DEFAULT '',
		scopes String NOT NULL DEFAULT '',
		rate_limit_rpm Int32 NOT NULL DEFAULT 0,
		budget_limit Float64 NOT NULL DEFAULT 0,
		budget_used Float64 NOT NULL DEFAULT 0,
		budget_currency String NOT NULL DEFAULT 'USD',
		budget_reset_type String NOT NULL DEFAULT '',
		budget_reset_at DateTime NOT NULL DEFAULT '1970-01-01 00:00:00',
		is_active UInt8 NOT NULL DEFAULT 1,
		expires_at DateTime NOT NULL DEFAULT '1970-01-01 00:00:00',
		last_used_at DateTime NOT NULL DEFAULT '1970-01-01 00:00:00',
		created_at DateTime NOT NULL DEFAULT now(),
		updated_at DateTime NOT NULL DEFAULT now()
	) ENGINE = MergeTree()
	ORDER BY id
	SETTINGS index_granularity = 8192;
	`

	if err := db.Exec(apiKeysSQL).Error; err != nil {
		return err
	}

	// Create indexes for api_keys
	indexSQL := []string{
		`CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys (key_hash) TYPE minmax GRANULARITY 3`,
		`CREATE INDEX IF NOT EXISTS idx_api_keys_key_prefix ON api_keys (key_prefix) TYPE minmax GRANULARITY 3`,
		`CREATE INDEX IF NOT EXISTS idx_api_keys_is_active ON api_keys (is_active) TYPE minmax GRANULARITY 3`,
		`CREATE INDEX IF NOT EXISTS idx_api_keys_expires_at ON api_keys (expires_at) TYPE minmax GRANULARITY 3`,
	}

	for _, sql := range indexSQL {
		if err := db.Exec(sql).Error; err != nil {
			// Indexes might already exist, continue
			continue
		}
	}

	// Create api_key_usage table
	usageSQL := `
	CREATE TABLE IF NOT EXISTS api_key_usage (
		id UInt64,
		api_key_id UInt64 NOT NULL,
		endpoint String NOT NULL DEFAULT '',
		provider String NOT NULL DEFAULT '',
		model String NOT NULL DEFAULT '',
		tokens_input Int32 NOT NULL DEFAULT 0,
		tokens_output Int32 NOT NULL DEFAULT 0,
		tokens_total Int32 NOT NULL DEFAULT 0,
		cost Float64 NOT NULL DEFAULT 0,
		currency String NOT NULL DEFAULT 'USD',
		status_code Int32 NOT NULL DEFAULT 0,
		latency_ms Int32 NOT NULL DEFAULT 0,
		request_id String NOT NULL DEFAULT '',
		user_agent String NOT NULL DEFAULT '',
		ip_address String NOT NULL DEFAULT '',
		error_message String NOT NULL DEFAULT '',
		created_at DateTime NOT NULL DEFAULT now()
	) ENGINE = MergeTree()
	ORDER BY (api_key_id, created_at)
	SETTINGS index_granularity = 8192;
	`

	if err := db.Exec(usageSQL).Error; err != nil {
		return err
	}

	// Create indexes for api_key_usage
	usageIndexSQL := []string{
		`CREATE INDEX IF NOT EXISTS idx_api_key_usage_api_key_id ON api_key_usage (api_key_id) TYPE minmax GRANULARITY 3`,
		`CREATE INDEX IF NOT EXISTS idx_api_key_usage_endpoint ON api_key_usage (endpoint) TYPE minmax GRANULARITY 3`,
		`CREATE INDEX IF NOT EXISTS idx_api_key_usage_request_id ON api_key_usage (request_id) TYPE minmax GRANULARITY 3`,
		`CREATE INDEX IF NOT EXISTS idx_api_key_usage_created_at ON api_key_usage (created_at) TYPE minmax GRANULARITY 3`,
	}

	for _, sql := range usageIndexSQL {
		if err := db.Exec(sql).Error; err != nil {
			// Indexes might already exist, continue
			continue
		}
	}

	return nil
}
