package builder

import (
	"github.com/Egham-7/adaptive-proxy/internal/models"
	pkgmodels "github.com/Egham-7/adaptive-proxy/pkg/models"
)

func (b *Builder) WithMultiTenancy(cfg pkgmodels.MultiTenancyConfig) *Builder {
	b.enabledEndpoints["projects"] = true

	if cfg.AuthProvider == pkgmodels.AuthProviderClerk && (cfg.ClerkSecretKey != "" || cfg.ClerkWebhook != "") {
		b.WithClerkAuth(cfg.ClerkSecretKey, cfg.ClerkWebhook)
	} else {
		b.WithDatabaseAuth()
	}

	if cfg.StripeSecret != "" && cfg.StripeWebhook != "" {
		b.WithStripe(cfg.StripeSecret, cfg.StripeWebhook)
	}

	b.WithDatabase(cfg.DatabaseConfig)

	if cfg.APIKeyConfig.Enabled {
		b.WithAPIKeyManagement(cfg.APIKeyConfig)
	} else {
		b.EnableAPIKeyAuth()
	}

	return b
}

func (b *Builder) EnableMultiTenancy() *Builder {
	b.enabledEndpoints["projects"] = true

	return b
}

func (b *Builder) WithClerkAuth(secretKey, webhookSecret string) *Builder {
	b.cfg.Auth.Provider = "clerk"
	b.cfg.Auth.ClerkConfig = &models.ClerkAuthConfig{
		SecretKey:     secretKey,
		WebhookSecret: webhookSecret,
	}

	return b
}

func (b *Builder) WithDatabaseAuth() *Builder {
	b.cfg.Auth.Provider = "database"
	b.cfg.Auth.ClerkConfig = nil

	return b
}

func (b *Builder) GetAuthProviderType() pkgmodels.AuthProviderType {
	if b.cfg.Auth.Provider == "clerk" && b.cfg.Auth.ClerkConfig != nil && (b.cfg.Auth.ClerkConfig.SecretKey != "" || b.cfg.Auth.ClerkConfig.WebhookSecret != "") {
		return pkgmodels.AuthProviderClerk
	}
	return pkgmodels.AuthProviderDatabase
}

func (b *Builder) GetClerkWebhookSecret() (string, bool) {
	if b.cfg.Auth.ClerkConfig != nil {
		return b.cfg.Auth.ClerkConfig.WebhookSecret, true
	}
	return "", false
}

func (b *Builder) IsMultiTenancyEnabled() bool {
	return b.enabledEndpoints["projects"]
}
