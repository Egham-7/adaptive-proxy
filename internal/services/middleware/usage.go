package middleware

import (
	"time"

	"github.com/Egham-7/adaptive-proxy/internal/models"
	"github.com/Egham-7/adaptive-proxy/internal/services/budget"
	"github.com/gofiber/fiber/v2"
)

type UsageTracker struct {
	budgetService *budget.Service
}

func NewUsageTracker(budgetService *budget.Service) *UsageTracker {
	return &UsageTracker{
		budgetService: budgetService,
	}
}

func (u *UsageTracker) TrackUsage() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		err := c.Next()

		apiKey, ok := c.Locals("api_key").(*models.APIKey)
		if !ok || apiKey == nil {
			return err
		}

		latency := time.Since(start).Milliseconds()

		usage := &models.APIKeyUsage{
			APIKeyID:   apiKey.ID,
			Endpoint:   c.Path(),
			StatusCode: c.Response().StatusCode(),
			LatencyMs:  int(latency),
			Currency:   "USD",
		}

		if provider, ok := c.Locals("provider").(string); ok {
			usage.Provider = provider
		}
		if model, ok := c.Locals("model").(string); ok {
			usage.Model = model
		}
		if tokensInput, ok := c.Locals("tokens_input").(int); ok {
			usage.TokensInput = tokensInput
		}
		if tokensOutput, ok := c.Locals("tokens_output").(int); ok {
			usage.TokensOutput = tokensOutput
		}
		if tokensTotal, ok := c.Locals("tokens_total").(int); ok {
			usage.TokensTotal = tokensTotal
		}
		if cost, ok := c.Locals("cost").(float64); ok {
			usage.Cost = cost
		}
		if errorMsg, ok := c.Locals("error_message").(string); ok {
			usage.ErrorMessage = errorMsg
		}
		if requestID, ok := c.Locals("request_id").(string); ok {
			usage.RequestID = requestID
		}
		usage.UserAgent = c.Get("User-Agent")
		usage.IPAddress = c.IP()

		if trackErr := u.budgetService.TrackUsage(c.Context(), usage); trackErr != nil {
			return err
		}

		return err
	}
}
