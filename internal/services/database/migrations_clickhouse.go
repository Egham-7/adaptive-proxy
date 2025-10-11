package database

import (
	"fmt"

	"gorm.io/gorm"
)

func RunClickHouseMigrations(db *gorm.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS api_keys (
			id UInt32,
			name String,
			key_hash String,
			key_prefix String,
			organization_id String,
			user_id String,
			project_id Nullable(String),
			metadata String,
			scopes String,
			rate_limit_rpm Int32,
			budget_limit Float64,
			budget_used Float64,
			budget_currency String,
			budget_reset_type String,
			budget_reset_at DateTime,
			is_active UInt8,
			expires_at DateTime,
			last_used_at DateTime,
			created_at DateTime DEFAULT now(),
			updated_at DateTime DEFAULT now()
		) ENGINE = MergeTree()
		ORDER BY id`,

		`ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS project_id Nullable(String)`,

		`CREATE TABLE IF NOT EXISTS api_key_usages (
			id UInt32,
			api_key_id UInt32,
			endpoint String,
			provider String,
			model String,
			tokens_input Int32,
			tokens_output Int32,
			tokens_total Int32,
			cost Float64,
			currency String,
			status_code Int32,
			latency_ms Int32,
			metadata String,
			request_id String,
			user_agent String,
			ip_address String,
			error_message String,
			created_at DateTime DEFAULT now()
		) ENGINE = MergeTree()
		ORDER BY (api_key_id, created_at)`,

		`CREATE TABLE IF NOT EXISTS organization_credits (
			id UInt32,
			organization_id String,
			balance Decimal(12, 6),
			total_purchased Decimal(12, 6),
			total_used Decimal(12, 6),
			created_at DateTime DEFAULT now(),
			updated_at DateTime DEFAULT now()
		) ENGINE = MergeTree()
		ORDER BY id`,

		`CREATE TABLE IF NOT EXISTS credit_transactions (
			id UInt32,
			organization_id String,
			user_id String,
			type String,
			amount Decimal(12, 6),
			balance_after Decimal(12, 6),
			description String,
			metadata String,
			stripe_payment_intent_id String,
			stripe_session_id String,
			api_key_id UInt32,
			api_usage_id UInt32,
			created_at DateTime DEFAULT now()
		) ENGINE = MergeTree()
		ORDER BY (organization_id, created_at)`,

		`CREATE TABLE IF NOT EXISTS credit_packages (
			id UInt32,
			name String,
			description String,
			credit_amount Decimal(12, 6),
			price Decimal(12, 6),
			stripe_price_id String,
			created_at DateTime DEFAULT now(),
			updated_at DateTime DEFAULT now()
		) ENGINE = MergeTree()
		ORDER BY id`,
	}

	for _, query := range queries {
		if err := db.Exec(query).Error; err != nil {
			return fmt.Errorf("failed to execute migration: %w", err)
		}
	}

	return nil
}
