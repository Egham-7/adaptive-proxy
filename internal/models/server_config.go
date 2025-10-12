package models

type ServerConfig struct {
	Port           string        `json:"port,omitzero" yaml:"port"`
	AllowedOrigins string        `json:"allowed_origins,omitzero" yaml:"allowed_origins"`
	Environment    string        `json:"environment,omitzero" yaml:"environment"`
	LogLevel       string        `json:"log_level,omitzero" yaml:"log_level"`
	APIKeyConfig   *APIKeyConfig `json:"api_key,omitzero" yaml:"api_key,omitempty"`
	StripeConfig   *StripeConfig `json:"stripe,omitzero" yaml:"stripe,omitempty"`
}

type StripeConfig struct {
	SecretKey     string `json:"secret_key" yaml:"secret_key"`
	WebhookSecret string `json:"webhook_secret" yaml:"webhook_secret"`
}
