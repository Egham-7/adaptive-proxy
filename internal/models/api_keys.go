package models

import "time"

type APIKey struct {
	ID              uint `gorm:"primaryKey;autoIncrement"`
	Name            string
	KeyHash         string `gorm:"uniqueIndex"`
	KeyPrefix       string `gorm:"index"`
	OrganizationID  string `gorm:"index"`
	UserID          string `gorm:"index"`
	ProjectID       uint   `gorm:"index"`
	Metadata        string
	Scopes          string
	RateLimitRpm    int
	BudgetLimit     float64
	BudgetUsed      float64
	BudgetCurrency  string
	BudgetResetType string
	BudgetResetAt   time.Time
	IsActive        bool `gorm:"index"`
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
	Name            string     `json:"name" validate:"required,min=1,max=255"`
	OrganizationID  string     `json:"organization_id,omitzero"`
	UserID          string     `json:"user_id,omitzero"`
	ProjectID       uint       `json:"project_id,omitzero"`
	Metadata        string     `json:"metadata,omitzero"`
	Scopes          []string   `json:"scopes,omitzero"`
	RateLimitRpm    *int       `json:"rate_limit_rpm,omitzero"`
	BudgetLimit     *float64   `json:"budget_limit,omitzero"`
	BudgetCurrency  string     `json:"budget_currency,omitzero"`
	BudgetResetType string     `json:"budget_reset_type,omitzero"`
	ExpiresAt       *time.Time `json:"expires_at,omitzero"`
}

type APIKeyResponse struct {
	ID              uint      `json:"id"`
	Name            string    `json:"name"`
	Key             string    `json:"key,omitzero"`
	KeyPrefix       string    `json:"key_prefix"`
	OrganizationID  string    `json:"organization_id,omitzero"`
	UserID          string    `json:"user_id,omitzero"`
	ProjectID       uint      `json:"project_id,omitzero"`
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
	ID           uint   `gorm:"primaryKey;autoIncrement"`
	APIKeyID     uint   `gorm:"index"`
	Endpoint     string `gorm:"index"`
	Provider     string
	Model        string
	TokensInput  int
	TokensOutput int
	TokensTotal  int
	Cost         float64
	Currency     string
	StatusCode   int
	LatencyMs    int
	Metadata     string
	RequestID    string `gorm:"index"`
	UserAgent    string
	IPAddress    string
	ErrorMessage string
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
