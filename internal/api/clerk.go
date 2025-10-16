package api

import (
	"encoding/json"
	"fmt"

	"github.com/Egham-7/adaptive-proxy/internal/models"
	"github.com/Egham-7/adaptive-proxy/internal/services/organizations"
	"github.com/Egham-7/adaptive-proxy/internal/services/projects"
	"github.com/Egham-7/adaptive-proxy/internal/services/usage"
	"github.com/gofiber/fiber/v2"
	svix "github.com/svix/svix-webhooks/go"
)

type ClerkWebhookHandler struct {
	webhookSecret        string
	creditsService       *usage.CreditsService
	organizationsService *organizations.Service
	projectsService      *projects.Service
}

func NewClerkWebhookHandler(webhookSecret string, creditsService *usage.CreditsService, organizationsService *organizations.Service, projectsService *projects.Service) *ClerkWebhookHandler {
	return &ClerkWebhookHandler{
		webhookSecret:        webhookSecret,
		creditsService:       creditsService,
		organizationsService: organizationsService,
		projectsService:      projectsService,
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

type ClerkOrganizationMembershipData struct {
	ID             string                `json:"id"`
	Organization   ClerkOrganizationInfo `json:"organization"`
	PublicUserData ClerkPublicUserData   `json:"public_user_data"`
	Role           string                `json:"role"`
}

type ClerkOrganizationInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ClerkPublicUserData struct {
	UserID string `json:"user_id"`
}

func (h *ClerkWebhookHandler) HandleWebhook(c *fiber.Ctx) error {
	payload := c.Body()
	if len(payload) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Empty request body",
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
	case "organizationMembership.created":
		if err := h.handleOrganizationMembershipCreated(c, event.Data); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to process organizationMembership.created event: %v", err),
			})
		}
	case "organizationMembership.updated":
		if err := h.handleOrganizationMembershipUpdated(c, event.Data); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to process organizationMembership.updated event: %v", err),
			})
		}
	case "organizationMembership.deleted":
		if err := h.handleOrganizationMembershipDeleted(c, event.Data); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to process organizationMembership.deleted event: %v", err),
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

func (h *ClerkWebhookHandler) handleOrganizationMembershipCreated(c *fiber.Ctx, data json.RawMessage) error {
	var membershipData ClerkOrganizationMembershipData
	if err := json.Unmarshal(data, &membershipData); err != nil {
		return fmt.Errorf("failed to unmarshal organization membership data: %w", err)
	}

	// Only process if the new member is an org admin
	if membershipData.Role != "org:admin" {
		// Not an admin, no action needed
		return nil
	}

	userID := membershipData.PublicUserData.UserID
	organizationID := membershipData.Organization.ID

	// Add the new org admin to all existing projects in the organization
	// This uses the projects service which has access to the DB
	// Note: We pass the projects service through the handler during initialization
	if h.projectsService != nil {
		err := h.projectsService.AddUserToAllOrgProjects(c.Context(), userID, organizationID, models.ProjectMemberRoleAdmin)
		if err != nil {
			// Log error but don't fail the webhook - lazy auth will handle it
			fmt.Printf("Warning: failed to add org admin %s to projects in org %s: %v\n", userID, organizationID, err)
		}
	}

	return nil
}

func (h *ClerkWebhookHandler) handleOrganizationMembershipUpdated(c *fiber.Ctx, data json.RawMessage) error {
	var membershipData ClerkOrganizationMembershipData
	if err := json.Unmarshal(data, &membershipData); err != nil {
		return fmt.Errorf("failed to unmarshal organization membership data: %w", err)
	}

	userID := membershipData.PublicUserData.UserID
	organizationID := membershipData.Organization.ID

	if h.projectsService == nil {
		return nil
	}

	// Check the new role
	if membershipData.Role == "org:admin" {
		// Promoted to admin - add to all projects
		err := h.projectsService.AddUserToAllOrgProjects(c.Context(), userID, organizationID, models.ProjectMemberRoleAdmin)
		if err != nil {
			// Log error but don't fail the webhook - lazy auth will handle it
			fmt.Printf("Warning: failed to add promoted admin %s to projects in org %s: %v\n", userID, organizationID, err)
		}
	} else {
		// Demoted from admin to member - remove from all projects (except as owner)
		err := h.projectsService.RemoveUserFromAllOrgProjects(c.Context(), userID, organizationID)
		if err != nil {
			// Log error but don't fail the webhook
			fmt.Printf("Warning: failed to remove demoted admin %s from projects in org %s: %v\n", userID, organizationID, err)
		}
	}

	return nil
}

func (h *ClerkWebhookHandler) handleOrganizationMembershipDeleted(c *fiber.Ctx, data json.RawMessage) error {
	var membershipData ClerkOrganizationMembershipData
	if err := json.Unmarshal(data, &membershipData); err != nil {
		return fmt.Errorf("failed to unmarshal organization membership data: %w", err)
	}

	userID := membershipData.PublicUserData.UserID
	organizationID := membershipData.Organization.ID

	// Remove user from all projects in the organization (except as owner)
	if h.projectsService != nil {
		err := h.projectsService.RemoveUserFromAllOrgProjects(c.Context(), userID, organizationID)
		if err != nil {
			// Log error but don't fail the webhook
			fmt.Printf("Warning: failed to remove user %s from projects in org %s: %v\n", userID, organizationID, err)
		}
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
