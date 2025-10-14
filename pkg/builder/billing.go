package builder

import "github.com/Egham-7/adaptive-proxy/internal/models"

func (b *Builder) EnableCredits() *Builder {
	if b.cfg.APIKey == nil {
		b.cfg.APIKey = &models.APIKeyConfig{
			Enabled:        true,
			HeaderNames:    []string{"X-API-Key", "X-Stainless-API-Key"},
			RequireForAll:  false,
			AllowAnonymous: true,
		}
	}
	return b
}

func (b *Builder) WithStripe(secretKey, webhookSecret string) *Builder {
	b.cfg.Billing = &models.StripeConfig{
		SecretKey:     secretKey,
		WebhookSecret: webhookSecret,
	}

	b.EnableCredits()

	return b
}

func (b *Builder) GetStripeConfig() (secretKey, webhookSecret string, configured bool) {
	if b.cfg.Billing != nil {
		return b.cfg.Billing.SecretKey, b.cfg.Billing.WebhookSecret, true
	}
	return "", "", false
}
