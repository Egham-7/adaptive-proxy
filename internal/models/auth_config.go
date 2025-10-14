package models

type AuthConfig struct {
	Provider       string              `json:"provider" yaml:"provider"`
	ClerkConfig    *ClerkAuthConfig    `json:"clerk,omitempty" yaml:"clerk,omitempty"`
	DatabaseConfig *DatabaseAuthConfig `json:"database,omitempty" yaml:"database,omitempty"`
}

type ClerkAuthConfig struct {
	SecretKey     string `json:"secret_key" yaml:"secret_key"`
	WebhookSecret string `json:"webhook_secret" yaml:"webhook_secret"`
}

type DatabaseAuthConfig struct {
	DatabaseURL string `json:"database_url,omitempty" yaml:"database_url,omitempty"`
}
