package middleware

import (
	"github.com/Egham-7/adaptive-proxy/internal/services/auth"
	usageSvc "github.com/Egham-7/adaptive-proxy/internal/services/usage"
	"github.com/gofiber/fiber/v2"
	fiberlog "github.com/gofiber/fiber/v2/log"
)

type UsageTracker struct {
	budgetService  *usageSvc.Service
	creditsService *usageSvc.CreditsService
	rateLimiter    *usageSvc.RateLimiter
	creditsEnabled bool
}

func NewUsageTracker(budgetService *usageSvc.Service, creditsService *usageSvc.CreditsService, creditsEnabled bool) *UsageTracker {
	return &UsageTracker{
		budgetService:  budgetService,
		creditsService: creditsService,
		rateLimiter:    usageSvc.NewRateLimiter(),
		creditsEnabled: creditsEnabled,
	}
}

func (u *UsageTracker) EnforceUsageLimits() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authCtx := auth.GetAuthContext(c)
		if authCtx == nil || !authCtx.IsAPIKey() || authCtx.APIKey == nil {
			return c.Next()
		}

		apiKey := authCtx.APIKey.Key

		if apiKey.RateLimitRpm > 0 {
			allowed, err := u.rateLimiter.CheckRateLimit(c.Context(), apiKey.ID, apiKey.RateLimitRpm)
			if err != nil {
				fiberlog.Errorf("Failed to check rate limit: %v", err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to check rate limit",
				})
			}
			if !allowed {
				return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
					"error": "Rate limit exceeded",
				})
			}
		}

		if u.budgetService != nil {
			withinLimit, _, err := u.budgetService.CheckBudgetLimit(c.Context(), apiKey.ID)
			if err != nil {
				fiberlog.Errorf("Failed to check budget limit: %v", err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to check budget limit",
				})
			}
			if !withinLimit {
				return c.Status(fiber.StatusPaymentRequired).JSON(fiber.Map{
					"error": "Budget limit exceeded",
				})
			}
		}

		if u.creditsEnabled && authCtx.APIKey.OrganizationID != "" {
			credit, err := u.creditsService.GetOrganizationCredit(c.Context(), authCtx.APIKey.OrganizationID)
			if err != nil {
				fiberlog.Errorf("Failed to check credit balance: %v", err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to verify credit balance",
				})
			}

			if credit.Balance <= 0 {
				return c.Status(fiber.StatusPaymentRequired).JSON(fiber.Map{
					"error": "Insufficient credits. Please add credits to continue.",
				})
			}
		}

		return c.Next()
	}
}
