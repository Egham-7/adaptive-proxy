package models

import (
	"time"
)

type APIKey struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	Name            string    `gorm:"not null;size:255" json:"name"`
	KeyHash         string    `gorm:"uniqueIndex;not null;size:64" json:"-"`
	KeyPrefix       string    `gorm:"not null;index;size:12" json:"key_prefix"`
	OrganizationID  string    `gorm:"size:255;index" json:"organization_id,omitempty"`
	UserID          string    `gorm:"size:255;index" json:"user_id,omitempty"`
	ProjectID       *string   `gorm:"size:255;index" json:"project_id,omitempty"`
	Metadata        string    `gorm:"not null;type:text;default:''" json:"metadata,omitzero"`
	Scopes          string    `gorm:"not null;type:text;default:''" json:"scopes,omitzero"`
	RateLimitRpm    int       `gorm:"not null;default:0" json:"rate_limit_rpm,omitzero"`
	BudgetLimit     float64   `gorm:"not null;default:0" json:"budget_limit,omitzero"`
	BudgetUsed      float64   `gorm:"not null;default:0" json:"budget_used"`
	BudgetCurrency  string    `gorm:"not null;size:3;default:'USD'" json:"budget_currency"`
	BudgetResetType string    `gorm:"not null;size:20;default:''" json:"budget_reset_type,omitzero"`
	BudgetResetAt   time.Time `gorm:"not null" json:"budget_reset_at,omitzero"`
	IsActive        bool      `gorm:"not null;default:true;index" json:"is_active"`
	ExpiresAt       time.Time `gorm:"not null" json:"expires_at,omitzero"`
	LastUsedAt      time.Time `gorm:"not null" json:"last_used_at,omitzero"`
	CreatedAt       time.Time `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"not null;autoUpdateTime" json:"updated_at"`
}

type APIKeyConfig struct {
	Enabled        bool     `yaml:"enabled" json:"enabled"`
	HeaderNames    []string `yaml:"header_names,omitempty" json:"header_names,omitzero"`
	RequireForAll  bool     `yaml:"require_for_all,omitempty" json:"require_for_all,omitzero"`
	AllowAnonymous bool     `yaml:"allow_anonymous,omitempty" json:"allow_anonymous,omitzero"`
	CreditsEnabled bool     `yaml:"credits_enabled,omitempty" json:"credits_enabled,omitzero"`
}

type APIKeyCreateRequest struct {
	Name            string    `json:"name" validate:"required,min=1,max=255"`
	OrganizationID  string    `json:"organization_id,omitempty"`
	UserID          string    `json:"user_id,omitempty"`
	ProjectID       *string   `json:"project_id,omitempty"`
	Metadata        string    `json:"metadata,omitzero"`
	Scopes          []string  `json:"scopes,omitzero"`
	RateLimitRpm    int       `json:"rate_limit_rpm,omitzero"`
	BudgetLimit     float64   `json:"budget_limit,omitzero"`
	BudgetCurrency  string    `json:"budget_currency,omitzero"`
	BudgetResetType string    `json:"budget_reset_type,omitzero"`
	ExpiresAt       time.Time `json:"expires_at,omitzero"`
}

type APIKeyResponse struct {
	ID              uint      `json:"id"`
	Name            string    `json:"name"`
	Key             string    `json:"key,omitzero"`
	KeyPrefix       string    `json:"key_prefix"`
	OrganizationID  string    `json:"organization_id,omitempty"`
	UserID          string    `json:"user_id,omitempty"`
	ProjectID       *string   `json:"project_id,omitempty"`
	Metadata        string    `json:"metadata,omitzero"`
	Scopes          string    `json:"scopes,omitzero"`
	RateLimitRpm    int       `json:"rate_limit_rpm,omitzero"`
	BudgetLimit     float64   `json:"budget_limit,omitzero"`
	BudgetUsed      float64   `json:"budget_used"`
	BudgetRemaining float64   `json:"budget_remaining,omitzero"`
	BudgetCurrency  string    `json:"budget_currency,omitzero"`
	BudgetResetType string    `json:"budget_reset_type,omitzero"`
	BudgetResetAt   time.Time `json:"budget_reset_at,omitzero"`
	IsActive        bool      `json:"is_active"`
	ExpiresAt       time.Time `json:"expires_at,omitzero"`
	LastUsedAt      time.Time `json:"last_used_at,omitzero"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

const (
	BudgetResetNone    = ""
	BudgetResetDaily   = "daily"
	BudgetResetWeekly  = "weekly"
	BudgetResetMonthly = "monthly"
)

type Metadata map[string]any

type APIKeyUsage struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	APIKeyID     uint      `gorm:"not null;index" json:"api_key_id"`
	Endpoint     string    `gorm:"not null;size:100;index;default:''" json:"endpoint"`
	Provider     string    `gorm:"not null;size:50;default:''" json:"provider"`
	Model        string    `gorm:"not null;size:100;default:''" json:"model"`
	TokensInput  int       `gorm:"not null;default:0" json:"tokens_input"`
	TokensOutput int       `gorm:"not null;default:0" json:"tokens_output"`
	TokensTotal  int       `gorm:"not null;default:0" json:"tokens_total"`
	Cost         float64   `gorm:"not null;default:0" json:"cost"`
	Currency     string    `gorm:"not null;size:3;default:'USD'" json:"currency"`
	StatusCode   int       `gorm:"not null;default:0" json:"status_code"`
	LatencyMs    int       `gorm:"not null;default:0" json:"latency_ms"`
	Metadata     Metadata  `gorm:"type:jsonb" json:"metadata"`
	RequestID    string    `gorm:"not null;size:100;index;default:''" json:"request_id,omitzero"`
	UserAgent    string    `gorm:"not null;size:255;default:''" json:"user_agent,omitzero"`
	IPAddress    string    `gorm:"not null;size:45;default:''" json:"ip_address,omitzero"`
	ErrorMessage string    `gorm:"not null;type:text;default:''" json:"error_message,omitzero"`
	CreatedAt    time.Time `gorm:"not null;autoCreateTime;index" json:"created_at"`
}

type UsageStats struct {
	TotalRequests   int64   `json:"total_requests"`
	TotalCost       float64 `json:"total_cost"`
	TotalTokens     int64   `json:"total_tokens"`
	SuccessRequests int64   `json:"success_requests"`
	FailedRequests  int64   `json:"failed_requests"`
	AvgLatencyMs    float64 `json:"avg_latency_ms"`
}

type UsageByPeriod struct {
	Period string     `json:"period"`
	Stats  UsageStats `json:"stats"`
}

type CreditTransactionType string

const (
	CreditTransactionPurchase    CreditTransactionType = "purchase"
	CreditTransactionUsage       CreditTransactionType = "usage"
	CreditTransactionRefund      CreditTransactionType = "refund"
	CreditTransactionPromotional CreditTransactionType = "promotional"
)

type OrganizationCredit struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	OrganizationID string    `gorm:"uniqueIndex;not null;type:varchar(30)" json:"organization_id"`
	Balance        float64   `gorm:"type:decimal(12,6);default:0" json:"balance"`
	TotalPurchased float64   `gorm:"type:decimal(12,6);default:0" json:"total_purchased"`
	TotalUsed      float64   `gorm:"type:decimal(12,6);default:0" json:"total_used"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type CreditTransaction struct {
	ID                    uint                  `gorm:"primaryKey" json:"id"`
	OrganizationID        string                `gorm:"not null;index;type:varchar(30)" json:"organization_id"`
	UserID                string                `gorm:"not null;index;type:varchar(100)" json:"user_id"`
	Type                  CreditTransactionType `gorm:"not null;index;type:varchar(20)" json:"type"`
	Amount                float64               `gorm:"type:decimal(12,6);not null" json:"amount"`
	BalanceAfter          float64               `gorm:"type:decimal(12,6);not null" json:"balance_after"`
	Description           string                `gorm:"type:text" json:"description,omitempty"`
	Metadata              Metadata              `gorm:"type:jsonb" json:"metadata"`
	StripePaymentIntentID string                `gorm:"index;type:varchar(100)" json:"stripe_payment_intent_id,omitempty"`
	StripeSessionID       string                `gorm:"type:varchar(100)" json:"stripe_session_id,omitempty"`
	APIKeyID              uint                  `gorm:"index" json:"api_key_id,omitempty"`
	APIUsageID            uint                  `gorm:"index" json:"api_usage_id,omitempty"`
	CreatedAt             time.Time             `gorm:"autoCreateTime;index" json:"created_at"`
}

type CreditPackage struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Name          string    `gorm:"not null;type:varchar(100)" json:"name"`
	Description   string    `gorm:"type:text" json:"description,omitempty"`
	CreditAmount  float64   `gorm:"type:decimal(12,6);not null" json:"credit_amount"`
	Price         float64   `gorm:"type:decimal(12,6);not null" json:"price"`
	StripePriceID string    `gorm:"uniqueIndex;not null;type:varchar(100)" json:"stripe_price_id"`
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type RecordUsageParams struct {
	APIKeyID       uint
	OrganizationID string
	UserID         string
	Endpoint       string
	Provider       string
	Model          string
	TokensInput    int
	TokensOutput   int
	Cost           float64
	Currency       string
	StatusCode     int
	LatencyMs      int
	Metadata       Metadata
	RequestID      string
	UserAgent      string
	IPAddress      string
	ErrorMessage   string
}

type PeriodStats struct {
	TotalRequests   int
	TotalCost       float64
	TotalTokens     int
	SuccessRequests int
	FailedRequests  int
}

type DeductCreditsParams struct {
	OrganizationID string
	UserID         string
	Amount         float64
	Description    string
	Metadata       Metadata
	APIKeyID       uint
	APIUsageID     uint
}

type AddCreditsParams struct {
	OrganizationID        string
	UserID                string
	Amount                float64
	Type                  CreditTransactionType
	Description           string
	Metadata              Metadata
	StripePaymentIntentID string
	StripeSessionID       string
}
