package builder

import (
	"github.com/Egham-7/adaptive-proxy/internal/config"
	"github.com/Egham-7/adaptive-proxy/internal/models"
	"github.com/gofiber/fiber/v2"
)

type Builder struct {
	cfg              *config.Config
	middlewares      []fiber.Handler
	enabledEndpoints map[string]bool
	rateLimitConfig  *models.RateLimitConfig
	timeoutConfig    *models.TimeoutConfig
}

func New() *Builder {
	return &Builder{
		cfg: &config.Config{
			Server: models.ServerConfig{
				Port:           "8080",
				AllowedOrigins: "*",
				Environment:    "development",
				LogLevel:       "info",
			},
			Fallback: models.FallbackConfig{
				Mode:       "race",
				TimeoutMs:  30000,
				MaxRetries: 3,
				CircuitBreaker: &models.CircuitBreakerConfig{
					FailureThreshold: 5,
					SuccessThreshold: 3,
					TimeoutMs:        15000,
					ResetAfterMs:     60000,
				},
			},
			ModelRouter: nil,
			Endpoints: models.EndpointsConfig{
				ChatCompletions: models.EndpointConfig{Providers: make(map[string]models.ProviderConfig)},
				Messages:        models.EndpointConfig{Providers: make(map[string]models.ProviderConfig)},
				SelectModel:     models.EndpointConfig{Providers: make(map[string]models.ProviderConfig)},
				Generate:        models.EndpointConfig{Providers: make(map[string]models.ProviderConfig)},
				CountTokens:     models.EndpointConfig{Providers: make(map[string]models.ProviderConfig)},
			},
		},
		middlewares:      []fiber.Handler{},
		enabledEndpoints: make(map[string]bool),
	}
}

func (b *Builder) Build() *config.Config {
	return b.cfg
}

func (b *Builder) GetMiddlewares() []fiber.Handler {
	return b.middlewares
}

func (b *Builder) GetEnabledEndpoints() map[string]bool {
	return b.enabledEndpoints
}

func (b *Builder) GetRateLimitConfig() *models.RateLimitConfig {
	return b.rateLimitConfig
}

func (b *Builder) GetTimeoutConfig() *models.TimeoutConfig {
	return b.timeoutConfig
}
