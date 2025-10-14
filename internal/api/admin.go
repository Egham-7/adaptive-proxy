package api

import (
	"errors"

	"github.com/Egham-7/adaptive-proxy/internal/models"
	"github.com/Egham-7/adaptive-proxy/internal/services/admin"
	"github.com/Egham-7/adaptive-proxy/internal/services/auth"
	"github.com/gofiber/fiber/v2"
)

type AdminHandler struct {
	adminService *admin.Service
}

func NewAdminHandler(adminService *admin.Service) *AdminHandler {
	return &AdminHandler{
		adminService: adminService,
	}
}

func (h *AdminHandler) CreateOrganization(c *fiber.Ctx) error {
	userID, ok := auth.GetUserID(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}

	var req models.OrganizationCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	org, err := h.adminService.CreateOrganization(c.Context(), userID, &req)
	if err != nil {
		if errors.Is(err, admin.ErrOrganizationExists) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "Organization with this ID already exists",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create organization",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(org)
}

func (h *AdminHandler) GetOrganization(c *fiber.Ctx) error {
	userID, ok := auth.GetUserID(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}
	organizationID := c.Params("id")

	if organizationID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "organization_id is required",
		})
	}

	org, err := h.adminService.GetOrganization(c.Context(), userID, organizationID)
	if err != nil {
		if errors.Is(err, admin.ErrUnauthorized) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "You don't have access to this organization",
			})
		}
		if errors.Is(err, admin.ErrOrganizationNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Organization not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get organization",
		})
	}

	return c.JSON(org)
}

func (h *AdminHandler) ListOrganizations(c *fiber.Ctx) error {
	userID, ok := auth.GetUserID(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}

	organizations, err := h.adminService.ListOrganizations(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list organizations",
		})
	}

	return c.JSON(fiber.Map{
		"organizations": organizations,
		"total":         len(organizations),
	})
}

func (h *AdminHandler) UpdateOrganization(c *fiber.Ctx) error {
	userID, ok := auth.GetUserID(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}
	organizationID := c.Params("id")

	if organizationID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "organization_id is required",
		})
	}

	var req models.OrganizationUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	org, err := h.adminService.UpdateOrganization(c.Context(), userID, organizationID, &req)
	if err != nil {
		if errors.Is(err, admin.ErrUnauthorized) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Only organization owners can update organizations",
			})
		}
		if errors.Is(err, admin.ErrOrganizationNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Organization not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update organization",
		})
	}

	return c.JSON(org)
}

func (h *AdminHandler) DeleteOrganization(c *fiber.Ctx) error {
	userID, ok := auth.GetUserID(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}
	organizationID := c.Params("id")

	if organizationID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "organization_id is required",
		})
	}

	err := h.adminService.DeleteOrganization(c.Context(), userID, organizationID)
	if err != nil {
		if errors.Is(err, admin.ErrUnauthorized) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Only organization owners can delete organizations",
			})
		}
		if errors.Is(err, admin.ErrOrganizationNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Organization not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete organization",
		})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

func (h *AdminHandler) AddOrganizationMember(c *fiber.Ctx) error {
	userID, ok := auth.GetUserID(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}
	organizationID := c.Params("id")

	if organizationID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "organization_id is required",
		})
	}

	var req models.AddOrganizationMemberRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	member, err := h.adminService.AddOrganizationMember(c.Context(), userID, organizationID, &req)
	if err != nil {
		if errors.Is(err, admin.ErrUnauthorized) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Only organization owners can add members",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to add member",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(member)
}

func (h *AdminHandler) RemoveOrganizationMember(c *fiber.Ctx) error {
	userID, ok := auth.GetUserID(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}
	organizationID := c.Params("id")
	targetUserID := c.Params("user_id")

	if organizationID == "" || targetUserID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "organization_id and user_id are required",
		})
	}

	err := h.adminService.RemoveOrganizationMember(c.Context(), userID, organizationID, targetUserID)
	if err != nil {
		if errors.Is(err, admin.ErrUnauthorized) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Only organization owners can remove members",
			})
		}
		if errors.Is(err, admin.ErrMemberNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Member not found",
			})
		}
		if errors.Is(err, admin.ErrCannotRemoveOwner) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Cannot remove organization owner",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to remove member",
		})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

func (h *AdminHandler) ListOrganizationMembers(c *fiber.Ctx) error {
	userID, ok := auth.GetUserID(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}
	organizationID := c.Params("id")

	if organizationID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "organization_id is required",
		})
	}

	members, err := h.adminService.ListOrganizationMembers(c.Context(), userID, organizationID)
	if err != nil {
		if errors.Is(err, admin.ErrUnauthorized) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "You don't have access to this organization",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list members",
		})
	}

	return c.JSON(fiber.Map{
		"members": members,
		"total":   len(members),
	})
}

func (h *AdminHandler) CreateUser(c *fiber.Ctx) error {
	var req models.UserCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	user, err := h.adminService.CreateUser(c.Context(), &req)
	if err != nil {
		if errors.Is(err, admin.ErrUserExists) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "User with this ID already exists",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create user",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(user)
}

func (h *AdminHandler) GetUser(c *fiber.Ctx) error {
	userID := c.Params("id")

	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "user_id is required",
		})
	}

	user, err := h.adminService.GetUser(c.Context(), userID)
	if err != nil {
		if errors.Is(err, admin.ErrUserNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "User not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get user",
		})
	}

	return c.JSON(user)
}

func (h *AdminHandler) UpdateUser(c *fiber.Ctx) error {
	userID := c.Params("id")

	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "user_id is required",
		})
	}

	var req models.UserUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	user, err := h.adminService.UpdateUser(c.Context(), userID, &req)
	if err != nil {
		if errors.Is(err, admin.ErrUserNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "User not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update user",
		})
	}

	return c.JSON(user)
}

func (h *AdminHandler) DeleteUser(c *fiber.Ctx) error {
	userID := c.Params("id")

	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "user_id is required",
		})
	}

	err := h.adminService.DeleteUser(c.Context(), userID)
	if err != nil {
		if errors.Is(err, admin.ErrUserNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "User not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete user",
		})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}
