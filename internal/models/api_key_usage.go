package models

import "time"

type APIKeyUsage struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	APIKeyID     uint      `gorm:"not null;index" json:"api_key_id"`
	Endpoint     string    `gorm:"not null;size:100;index;default:''" json:"endpoint"`
	Provider     string    `gorm:"not null;size:50;default:''" json:"provider"`
	Model        string    `gorm:"not null;size:100;default:''" json:"model"`
	TokensInput  int       `gorm:"not null;default:0" json:"tokens_input"`
	TokensOutput int       `gorm:"not null;default:0" json:"tokens_output"`
	TokensTotal  int       `gorm:"not null;default:0" json:"tokens_total"`
	Cost         float64   `gorm:"not null;type:decimal(10,6);default:0" json:"cost"`
	Currency     string    `gorm:"not null;size:3;default:'USD'" json:"currency"`
	StatusCode   int       `gorm:"not null;default:0" json:"status_code"`
	LatencyMs    int       `gorm:"not null;default:0" json:"latency_ms"`
	RequestID    string    `gorm:"not null;size:100;index;default:''" json:"request_id,omitzero"`
	UserAgent    string    `gorm:"not null;size:255;default:''" json:"user_agent,omitzero"`
	IPAddress    string    `gorm:"not null;size:45;default:''" json:"ip_address,omitzero"`
	ErrorMessage string    `gorm:"not null;type:text;default:''" json:"error_message,omitzero"`
	CreatedAt    time.Time `gorm:"not null;autoCreateTime;index" json:"created_at"`
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
