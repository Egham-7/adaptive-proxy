package models

import "time"

type APIKeyUsage struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	APIKeyID     uint      `gorm:"not null;index" json:"api_key_id"`
	Endpoint     string    `gorm:"size:100;index" json:"endpoint"`
	Provider     string    `gorm:"size:50" json:"provider"`
	Model        string    `gorm:"size:100" json:"model"`
	TokensInput  int       `json:"tokens_input"`
	TokensOutput int       `json:"tokens_output"`
	TokensTotal  int       `json:"tokens_total"`
	Cost         float64   `gorm:"type:decimal(10,6)" json:"cost"`
	Currency     string    `gorm:"size:3;default:'USD'" json:"currency"`
	StatusCode   int       `json:"status_code"`
	LatencyMs    int       `json:"latency_ms"`
	RequestID    string    `gorm:"size:100;index" json:"request_id,omitzero"`
	UserAgent    string    `gorm:"size:255" json:"user_agent,omitzero"`
	IPAddress    string    `gorm:"size:45" json:"ip_address,omitzero"`
	ErrorMessage string    `gorm:"type:text" json:"error_message,omitzero"`
	CreatedAt    time.Time `gorm:"autoCreateTime;index" json:"created_at"`
}

func (APIKeyUsage) TableName() string {
	return "api_key_usage"
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
