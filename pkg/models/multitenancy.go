package models

import "github.com/Egham-7/adaptive-proxy/internal/models"

type AuthProviderType string

const (
	AuthProviderClerk    AuthProviderType = "clerk"
	AuthProviderDatabase AuthProviderType = "database"
)

type MultiTenancyConfig struct {
	AuthProvider   AuthProviderType
	ClerkSecretKey string
	ClerkWebhook   string
	StripeSecret   string
	StripeWebhook  string
	DatabaseConfig models.DatabaseConfig
	APIKeyConfig   models.APIKeyConfig
}
