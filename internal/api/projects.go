package api

import (
	"errors"
	"strconv"

	"github.com/Egham-7/adaptive-proxy/internal/models"
	"github.com/Egham-7/adaptive-proxy/internal/services/auth"
	"github.com/Egham-7/adaptive-proxy/internal/services/projects"
	"github.com/gofiber/fiber/v2"
)

type ProjectsHandler struct {
	projectsService *projects.Service
}

func NewProjectsHandler(projectsService *projects.Service) *ProjectsHandler {
	return &ProjectsHandler{
		projectsService: projectsService,
	}
}

func (h *ProjectsHandler) CreateProject(c *fiber.Ctx) error {
	userID, ok := auth.GetUserID(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}

	var req models.ProjectCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	project, err := h.projectsService.CreateProject(c.Context(), userID, &req)
	if err != nil {
		if errors.Is(err, projects.ErrUnauthorized) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "You don't have access to this organization",
			})
		}
		if errors.Is(err, projects.ErrDuplicateProjectID) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "Project with this ID already exists",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create project",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(project.ToResponse())
}

func (h *ProjectsHandler) GetProject(c *fiber.Ctx) error {
	userID, ok := auth.GetUserID(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}
	projectIDStr := c.Params("id")

	if projectIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "project_id is required",
		})
	}

	projectID, err := strconv.ParseUint(projectIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid project_id",
		})
	}

	project, err := h.projectsService.GetProject(c.Context(), userID, uint(projectID))
	if err != nil {
		if errors.Is(err, projects.ErrUnauthorized) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "You don't have access to this project",
			})
		}
		if errors.Is(err, projects.ErrProjectNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Project not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get project",
		})
	}

	return c.JSON(project.ToResponse())
}

func (h *ProjectsHandler) ListProjects(c *fiber.Ctx) error {
	userID, ok := auth.GetUserID(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}
	organizationID := c.Params("org_id")

	if organizationID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "organization_id is required",
		})
	}

	projectsList, err := h.projectsService.ListProjects(c.Context(), userID, organizationID)
	if err != nil {
		if errors.Is(err, projects.ErrUnauthorized) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "You don't have access to this organization",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list projects",
		})
	}

	responses := make([]*models.ProjectResponse, len(projectsList))
	for i, p := range projectsList {
		responses[i] = p.ToResponse()
	}

	return c.JSON(fiber.Map{
		"projects": responses,
		"total":    len(responses),
	})
}

func (h *ProjectsHandler) UpdateProject(c *fiber.Ctx) error {
	userID, ok := auth.GetUserID(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}
	projectIDStr := c.Params("id")

	if projectIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "project_id is required",
		})
	}

	projectID, err := strconv.ParseUint(projectIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid project_id",
		})
	}

	var req models.ProjectUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	project, err := h.projectsService.UpdateProject(c.Context(), userID, uint(projectID), &req)
	if err != nil {
		if errors.Is(err, projects.ErrUnauthorized) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "You don't have admin access to this project",
			})
		}
		if errors.Is(err, projects.ErrProjectNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Project not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update project",
		})
	}

	return c.JSON(project.ToResponse())
}

func (h *ProjectsHandler) DeleteProject(c *fiber.Ctx) error {
	userID, ok := auth.GetUserID(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}
	projectIDStr := c.Params("id")

	if projectIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "project_id is required",
		})
	}

	projectID, err := strconv.ParseUint(projectIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid project_id",
		})
	}

	err = h.projectsService.DeleteProject(c.Context(), userID, uint(projectID))
	if err != nil {
		if errors.Is(err, projects.ErrUnauthorized) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Only project owners can delete projects",
			})
		}
		if errors.Is(err, projects.ErrProjectNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Project not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete project",
		})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

func (h *ProjectsHandler) AddMember(c *fiber.Ctx) error {
	userID, ok := auth.GetUserID(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}
	projectIDStr := c.Params("id")

	if projectIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "project_id is required",
		})
	}

	projectID, err := strconv.ParseUint(projectIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid project_id",
		})
	}

	var req models.AddProjectMemberRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	member, err := h.projectsService.AddMember(c.Context(), userID, uint(projectID), &req)
	if err != nil {
		if errors.Is(err, projects.ErrUnauthorized) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "You don't have admin access to this project",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to add member",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(member)
}

func (h *ProjectsHandler) RemoveMember(c *fiber.Ctx) error {
	userID, ok := auth.GetUserID(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}
	projectIDStr := c.Params("id")
	targetUserID := c.Params("user_id")

	if projectIDStr == "" || targetUserID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "project_id and user_id are required",
		})
	}

	projectID, err := strconv.ParseUint(projectIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid project_id",
		})
	}

	err = h.projectsService.RemoveMember(c.Context(), userID, uint(projectID), targetUserID)
	if err != nil {
		if errors.Is(err, projects.ErrUnauthorized) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "You don't have admin access to this project",
			})
		}
		if errors.Is(err, projects.ErrMemberNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Member not found",
			})
		}
		if errors.Is(err, projects.ErrCannotRemoveOwner) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Cannot remove project owner",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to remove member",
		})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

func (h *ProjectsHandler) ListMembers(c *fiber.Ctx) error {
	userID, ok := auth.GetUserID(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}
	projectIDStr := c.Params("id")

	if projectIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "project_id is required",
		})
	}

	projectID, err := strconv.ParseUint(projectIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid project_id",
		})
	}

	project, err := h.projectsService.GetProject(c.Context(), userID, uint(projectID))
	if err != nil {
		if errors.Is(err, projects.ErrUnauthorized) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "You don't have access to this project",
			})
		}
		if errors.Is(err, projects.ErrProjectNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Project not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get project members",
		})
	}

	return c.JSON(fiber.Map{
		"members": project.Members,
		"total":   len(project.Members),
	})
}
