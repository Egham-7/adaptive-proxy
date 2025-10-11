package middleware

import (
	"github.com/Egham-7/adaptive-proxy/internal/models"
	usageSvc "github.com/Egham-7/adaptive-proxy/internal/services/usage"
	"github.com/gofiber/fiber/v2"
	fiberlog "github.com/gofiber/fiber/v2/log"
)

type UsageTracker struct {
	budgetService  *usageSvc.Service
	creditsService *usageSvc.CreditsService
	creditsEnabled bool
}

func NewUsageTracker(budgetService *usageSvc.Service, creditsService *usageSvc.CreditsService, creditsEnabled bool) *UsageTracker {
	return &UsageTracker{
		budgetService:  budgetService,
		creditsService: creditsService,
		creditsEnabled: creditsEnabled,
	}
}

func (u *UsageTracker) TrackUsage() fiber.Handler {
	return func(c *fiber.Ctx) error {
		apiKey, ok := c.Locals("api_key").(*models.APIKey)
		if u.creditsEnabled && ok && apiKey != nil && apiKey.OrganizationID != "" {
			credit, err := u.creditsService.GetOrganizationCredit(c.Context(), apiKey.OrganizationID)
			if err != nil {
				fiberlog.Errorf("Failed to check credit balance: %v", err)
				return fiber.NewError(fiber.StatusInternalServerError, "Failed to verify credit balance")
			}

			if credit.Balance <= 0 {
				return fiber.NewError(fiber.StatusPaymentRequired, "Insufficient credits. Please add credits to continue.")
			}
		}

		err := c.Next()

		return err
	}
}
