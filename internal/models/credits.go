package models

import "time"

type CreditTransactionType string

const (
	CreditTransactionPurchase    CreditTransactionType = "purchase"
	CreditTransactionUsage       CreditTransactionType = "usage"
	CreditTransactionRefund      CreditTransactionType = "refund"
	CreditTransactionPromotional CreditTransactionType = "promotional"
)

type OrganizationCredit struct {
	ID             uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	OrganizationID string    `gorm:"uniqueIndex;not null" json:"organization_id"`
	Balance        float64   `gorm:"not null;default:0" json:"balance"`
	TotalPurchased float64   `gorm:"not null;default:0" json:"total_purchased"`
	TotalUsed      float64   `gorm:"not null;default:0" json:"total_used"`
	CreatedAt      time.Time `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"not null;autoUpdateTime" json:"updated_at"`
}

type CreditTransaction struct {
	ID                    uint                  `gorm:"primaryKey;autoIncrement"`
	OrganizationID        string                `gorm:"index"`
	UserID                string                `gorm:"index"`
	Type                  CreditTransactionType `gorm:"index"`
	Amount                float64
	BalanceAfter          float64
	Description           string
	Metadata              string
	StripePaymentIntentID string `gorm:"index"`
	StripeSessionID       string
	APIKeyID              uint      `gorm:"index"`
	APIUsageID            uint      `gorm:"index"`
	CreatedAt             time.Time `gorm:"autoCreateTime;index"`
}

type CreditPackage struct {
	ID            uint `gorm:"primaryKey;autoIncrement"`
	Name          string
	Description   string
	CreditAmount  float64
	Price         float64
	StripePriceID string    `gorm:"uniqueIndex"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime"`
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
