package models

type StripeConfig struct {
	SecretKey     string `json:"secret_key" yaml:"secret_key"`
	WebhookSecret string `json:"webhook_secret" yaml:"webhook_secret"`
}
