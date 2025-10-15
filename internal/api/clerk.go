package api

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/Egham-7/adaptive-proxy/internal/models"
	"github.com/Egham-7/adaptive-proxy/internal/services/organizations"
	"github.com/Egham-7/adaptive-proxy/internal/services/usage"
	"github.com/gofiber/fiber/v2"
	svix "github.com/svix/svix-webhooks/go"
)

type ClerkWebhookHandler struct {
	webhookSecret        string
	creditsService       *usage.CreditsService
	organizationsService *organizations.Service
}

func NewClerkWebhookHandler(webhookSecret string, creditsService *usage.CreditsService, organizationsService *organizations.Service) *ClerkWebhookHandler {
	return &ClerkWebhookHandler{
		webhookSecret:        webhookSecret,
		creditsService:       creditsService,
		organizationsService: organizationsService,
	}
}

type ClerkWebhookEvent struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type ClerkOrganizationData struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedBy string `json:"created_by"`
}

func (h *ClerkWebhookHandler) HandleWebhook(c *fiber.Ctx) error {
	payload, err := io.ReadAll(c.Context().RequestBodyStream())
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to read request body",
		})
	}

	headers := make(map[string][]string)
	for key, value := range c.Request().Header.All() {
		headers[string(key)] = []string{string(value)}
	}

	wh, err := svix.NewWebhook(h.webhookSecret)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to initialize webhook verifier",
		})
	}

	var event ClerkWebhookEvent
	err = wh.Verify(payload, headers)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid webhook signature",
		})
	}

	if err := json.Unmarshal(payload, &event); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid JSON payload",
		})
	}

	switch event.Type {
	case "organization.created":
		if err := h.handleOrganizationCreated(c, event.Data); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to process organization.created event: %v", err),
			})
		}
	case "organization.deleted":
		if err := h.handleOrganizationDeleted(c, event.Data); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to process organization.deleted event: %v", err),
			})
		}
	}

	return c.JSON(fiber.Map{
		"received": true,
	})
}

func (h *ClerkWebhookHandler) handleOrganizationCreated(c *fiber.Ctx, data json.RawMessage) error {
	var orgData ClerkOrganizationData
	if err := json.Unmarshal(data, &orgData); err != nil {
		return fmt.Errorf("failed to unmarshal organization data: %w", err)
	}

	_, err := h.creditsService.AddCredits(c.Context(), models.AddCreditsParams{
		OrganizationID: orgData.ID,
		UserID:         orgData.CreatedBy,
		Amount:         3.14,
		Type:           models.CreditTransactionPromotional,
		Description:    "Welcome bonus for new organization",
		Metadata:       fmt.Sprintf(`{"organization_name":"%s"}`, orgData.Name),
	})
	if err != nil {
		return fmt.Errorf("failed to award signup credits: %w", err)
	}

	return nil
}

func (h *ClerkWebhookHandler) handleOrganizationDeleted(c *fiber.Ctx, data json.RawMessage) error {
	var orgData ClerkOrganizationData
	if err := json.Unmarshal(data, &orgData); err != nil {
		return fmt.Errorf("failed to unmarshal organization data: %w", err)
	}

	if err := h.organizationsService.DeleteOrganizationData(c.Context(), orgData.ID); err != nil {
		return fmt.Errorf("failed to delete organization data: %w", err)
	}

	return nil
}

func (h *ClerkWebhookHandler) DeleteOrganizationData(c *fiber.Ctx) error {
	organizationID := c.Params("id")
	if organizationID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "organization_id is required",
		})
	}

	if err := h.organizationsService.DeleteOrganizationData(c.Context(), organizationID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete organization data",
		})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}
