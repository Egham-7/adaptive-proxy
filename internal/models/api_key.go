package models

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"
)

type APIKey struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	Name            string     `gorm:"not null;size:255" json:"name"`
	KeyHash         string     `gorm:"uniqueIndex;not null;size:64" json:"-"`
	KeyPrefix       string     `gorm:"index;size:12" json:"key_prefix"`
	Metadata        string     `gorm:"type:text" json:"metadata,omitempty"`
	Scopes          string     `gorm:"type:text" json:"scopes,omitempty"`
	RateLimitRpm    *int       `json:"rate_limit_rpm,omitempty"`
	BudgetLimit     *float64   `gorm:"type:decimal(10,2)" json:"budget_limit,omitempty"`
	BudgetUsed      float64    `gorm:"type:decimal(10,2);default:0" json:"budget_used"`
	BudgetCurrency  string     `gorm:"size:3;default:'USD'" json:"budget_currency"`
	BudgetResetType string     `gorm:"size:20" json:"budget_reset_type,omitempty"`
	BudgetResetAt   *time.Time `json:"budget_reset_at,omitempty"`
	IsActive        bool       `gorm:"default:true;index" json:"is_active"`
	ExpiresAt       *time.Time `gorm:"index" json:"expires_at,omitempty"`
	LastUsedAt      *time.Time `json:"last_used_at,omitempty"`
	CreatedAt       time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (APIKey) TableName() string {
	return "api_keys"
}

type APIKeyConfig struct {
	Enabled        bool   `yaml:"enabled" json:"enabled"`
	HeaderName     string `yaml:"header_name,omitempty" json:"header_name,omitempty"`
	RequireForAll  bool   `yaml:"require_for_all,omitempty" json:"require_for_all,omitempty"`
	AllowAnonymous bool   `yaml:"allow_anonymous,omitempty" json:"allow_anonymous,omitempty"`
}

func DefaultAPIKeyConfig() APIKeyConfig {
	return APIKeyConfig{
		Enabled:        false,
		HeaderName:     "X-API-Key",
		RequireForAll:  false,
		AllowAnonymous: true,
	}
}

type APIKeyCreateRequest struct {
	Name            string     `json:"name" validate:"required,min=1,max=255"`
	Metadata        string     `json:"metadata,omitempty"`
	Scopes          []string   `json:"scopes,omitempty"`
	RateLimitRpm    *int       `json:"rate_limit_rpm,omitempty"`
	BudgetLimit     *float64   `json:"budget_limit,omitempty"`
	BudgetCurrency  string     `json:"budget_currency,omitempty"`
	BudgetResetType string     `json:"budget_reset_type,omitempty"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
}

type APIKeyResponse struct {
	ID              uint       `json:"id"`
	Name            string     `json:"name"`
	Key             string     `json:"key,omitempty"`
	KeyPrefix       string     `json:"key_prefix"`
	Metadata        string     `json:"metadata,omitempty"`
	Scopes          string     `json:"scopes,omitempty"`
	RateLimitRpm    *int       `json:"rate_limit_rpm,omitempty"`
	BudgetLimit     *float64   `json:"budget_limit,omitempty"`
	BudgetUsed      float64    `json:"budget_used"`
	BudgetRemaining *float64   `json:"budget_remaining,omitempty"`
	BudgetCurrency  string     `json:"budget_currency,omitempty"`
	BudgetResetType string     `json:"budget_reset_type,omitempty"`
	BudgetResetAt   *time.Time `json:"budget_reset_at,omitempty"`
	IsActive        bool       `json:"is_active"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	LastUsedAt      *time.Time `json:"last_used_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func GenerateAPIKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return "apk_" + base64.URLEncoding.EncodeToString(b)[:43], nil
}

func HashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%x", hash)
}

func ExtractKeyPrefix(key string) string {
	if len(key) < 12 {
		return key
	}
	return key[:12]
}

const (
	BudgetResetNone    = ""
	BudgetResetDaily   = "daily"
	BudgetResetWeekly  = "weekly"
	BudgetResetMonthly = "monthly"
)

func CalculateBudgetRemaining(budgetLimit *float64, budgetUsed float64) *float64 {
	if budgetLimit == nil {
		return nil
	}
	remaining := *budgetLimit - budgetUsed
	return &remaining
}
