package models

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

type RateLimitConfig struct {
	Max        int
	Expiration time.Duration
	KeyFunc    func(*fiber.Ctx) string
}

type TimeoutConfig struct {
	Timeout time.Duration
}
