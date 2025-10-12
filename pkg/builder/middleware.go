package builder

import (
	"time"

	"github.com/Egham-7/adaptive-proxy/internal/models"
	"github.com/gofiber/fiber/v2"
)

func (b *Builder) WithRateLimit(max int, expiration time.Duration, keyFunc ...func(*fiber.Ctx) string) *Builder {
	cfg := &models.RateLimitConfig{
		Max:        max,
		Expiration: expiration,
	}
	if len(keyFunc) > 0 {
		cfg.KeyFunc = keyFunc[0]
	}
	b.rateLimitConfig = cfg
	return b
}

func (b *Builder) WithTimeout(timeout time.Duration) *Builder {
	b.timeoutConfig = &models.TimeoutConfig{
		Timeout: timeout,
	}
	return b
}

func (b *Builder) WithMiddleware(middleware fiber.Handler) *Builder {
	b.middlewares = append(b.middlewares, middleware)
	return b
}
