package models

import (
	"time"
)

type APIKey struct {
	ID              uint      `gorm:"primaryKey;autoIncrement"`
	Name            string    `gorm:"size:255"`
	KeyHash         string    `gorm:"uniqueIndex;size:64"`
	KeyPrefix       string    `gorm:"index;size:12"`
	OrganizationID  string    `gorm:"size:255;index"`
	UserID          string    `gorm:"size:255;index"`
	ProjectID       string    `gorm:"size:255;index"`
	Metadata        string    `gorm:"type:String"`
	Scopes          string    `gorm:"type:String"`
	RateLimitRpm    int       `gorm:"type:Int32"`
	BudgetLimit     float64   `gorm:"type:Float64"`
	BudgetUsed      float64   `gorm:"type:Float64"`
	BudgetCurrency  string    `gorm:"size:3"`
	BudgetResetType string    `gorm:"size:20"`
	BudgetResetAt   time.Time
	IsActive        bool      `gorm:"index"`
	ExpiresAt       time.Time
	LastUsedAt      time.Time
	CreatedAt       time.Time `gorm:"autoCreateTime"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime"`
}

type APIKeyConfig struct {
	Enabled        bool     `yaml:"enabled" json:"enabled"`
	HeaderNames    []string `yaml:"header_names,omitempty" json:"header_names,omitzero"`
	RequireForAll  bool     `yaml:"require_for_all,omitempty" json:"require_for_all,omitzero"`
	AllowAnonymous bool     `yaml:"allow_anonymous,omitempty" json:"allow_anonymous,omitzero"`
}

type APIKeyCreateRequest struct {
	Name            string    `json:"name" validate:"required,min=1,max=255"`
	OrganizationID  string    `json:"organization_id,omitempty"`
	UserID          string    `json:"user_id,omitempty"`
	ProjectID       string    `json:"project_id,omitempty"`
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
	ProjectID       string    `json:"project_id,omitempty"`
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

type APIKeyUsage struct {
	ID           uint      `gorm:"primaryKey;autoIncrement"`
	APIKeyID     uint      `gorm:"index"`
	Endpoint     string    `gorm:"size:100;index"`
	Provider     string    `gorm:"size:50"`
	Model        string    `gorm:"size:100"`
	TokensInput  int       `gorm:"type:Int32"`
	TokensOutput int       `gorm:"type:Int32"`
	TokensTotal  int       `gorm:"type:Int32"`
	Cost         float64   `gorm:"type:Float64"`
	Currency     string    `gorm:"size:3"`
	StatusCode   int       `gorm:"type:Int32"`
	LatencyMs    int       `gorm:"type:Int32"`
	Metadata     string    `gorm:"type:String"`
	RequestID    string    `gorm:"size:100;index"`
	UserAgent    string    `gorm:"size:255"`
	IPAddress    string    `gorm:"size:45"`
	ErrorMessage string    `gorm:"type:String"`
	CreatedAt    time.Time `gorm:"autoCreateTime;index"`
}

func (APIKeyUsage) TableName() string {
	return "api_key_usages"
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
	ID             uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	OrganizationID string    `gorm:"uniqueIndex;not null;size:255" json:"organization_id"`
	Balance        float64   `gorm:"not null;default:0" json:"balance"`
	TotalPurchased float64   `gorm:"not null;default:0" json:"total_purchased"`
	TotalUsed      float64   `gorm:"not null;default:0" json:"total_used"`
	CreatedAt      time.Time `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"not null;autoUpdateTime" json:"updated_at"`
}

type CreditTransaction struct {
	ID                    uint                  `gorm:"primaryKey;autoIncrement"`
	OrganizationID        string                `gorm:"index;size:255"`
	UserID                string                `gorm:"index;size:255"`
	Type                  CreditTransactionType `gorm:"index;size:20"`
	Amount                float64               `gorm:"type:Float64"`
	BalanceAfter          float64               `gorm:"type:Float64"`
	Description           string                `gorm:"type:String"`
	Metadata              string                `gorm:"type:String"`
	StripePaymentIntentID string                `gorm:"index;size:100"`
	StripeSessionID       string                `gorm:"size:100"`
	APIKeyID              uint                  `gorm:"index;type:UInt32"`
	APIUsageID            uint                  `gorm:"index;type:UInt32"`
	CreatedAt             time.Time             `gorm:"autoCreateTime;index"`
}

type CreditPackage struct {
	ID            uint      `gorm:"primaryKey;autoIncrement"`
	Name          string    `gorm:"size:100"`
	Description   string    `gorm:"type:String"`
	CreditAmount  float64   `gorm:"type:Float64"`
	Price         float64   `gorm:"type:Float64"`
	StripePriceID string    `gorm:"uniqueIndex;size:100"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime"`
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
	Metadata       string
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
	Metadata       string
	APIKeyID       uint
	APIUsageID     uint
}

type AddCreditsParams struct {
	OrganizationID        string
	UserID                string
	Amount                float64
	Type                  CreditTransactionType
	Description           string
	Metadata              string
	StripePaymentIntentID string
	StripeSessionID       string
}
