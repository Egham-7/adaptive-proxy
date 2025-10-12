package builder

import "github.com/Egham-7/adaptive-proxy/internal/models"

func (b *Builder) EnableCredits() *Builder {
	if b.cfg.Server.APIKeyConfig == nil {
		b.cfg.Server.APIKeyConfig = &models.APIKeyConfig{
			Enabled:        true,
			HeaderNames:    []string{"X-API-Key", "X-Stainless-API-Key"},
			RequireForAll:  false,
			AllowAnonymous: true,
		}
	}
	b.cfg.Server.APIKeyConfig.CreditsEnabled = true
	return b
}

func (b *Builder) WithStripe(secretKey, webhookSecret string) *Builder {
	b.cfg.Server.StripeConfig = &models.StripeConfig{
		SecretKey:     secretKey,
		WebhookSecret: webhookSecret,
	}

	b.EnableCredits()

	return b
}

func (b *Builder) IsCreditsEnabled() bool {
	return b.cfg.Server.APIKeyConfig != nil && b.cfg.Server.APIKeyConfig.CreditsEnabled
}

func (b *Builder) GetStripeConfig() (secretKey, webhookSecret string, configured bool) {
	if b.cfg.Server.StripeConfig != nil {
		return b.cfg.Server.StripeConfig.SecretKey, b.cfg.Server.StripeConfig.WebhookSecret, true
	}
	return "", "", false
}
