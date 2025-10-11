package api

import (
	"strconv"
	"time"

	"github.com/Egham-7/adaptive-proxy/internal/models"
	"github.com/Egham-7/adaptive-proxy/internal/services/apikey"
	"github.com/Egham-7/adaptive-proxy/internal/services/budget"
	"github.com/gofiber/fiber/v2"
)

type APIKeyHandler struct {
	service       *apikey.Service
	budgetService *budget.Service
}

func NewAPIKeyHandler(service *apikey.Service, budgetService *budget.Service) *APIKeyHandler {
	return &APIKeyHandler{
		service:       service,
		budgetService: budgetService,
	}
}

func (h *APIKeyHandler) CreateAPIKey(c *fiber.Ctx) error {
	var req models.APIKeyCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	apiKey, err := h.service.CreateAPIKey(c.Context(), &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(apiKey)
}

func (h *APIKeyHandler) ListAPIKeys(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 50)
	offset := c.QueryInt("offset", 0)

	apiKeys, total, err := h.service.ListAPIKeys(c.Context(), limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data":   apiKeys,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *APIKeyHandler) GetAPIKey(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid API key ID",
		})
	}

	apiKey, err := h.service.GetAPIKey(c.Context(), uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(apiKey)
}

func (h *APIKeyHandler) RevokeAPIKey(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid API key ID",
		})
	}

	if err := h.service.RevokeAPIKey(c.Context(), uint(id)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "API key revoked successfully",
	})
}

func (h *APIKeyHandler) DeleteAPIKey(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid API key ID",
		})
	}

	if err := h.service.DeleteAPIKey(c.Context(), uint(id)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

func (h *APIKeyHandler) UpdateAPIKey(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid API key ID",
		})
	}

	var updates map[string]any
	if err := c.BodyParser(&updates); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.service.UpdateAPIKey(c.Context(), uint(id), updates); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "API key updated successfully",
	})
}

func (h *APIKeyHandler) RegisterRoutes(app *fiber.App, prefix string) {
	apiKeys := app.Group(prefix)

	apiKeys.Post("/", h.CreateAPIKey)
	apiKeys.Get("/", h.ListAPIKeys)
	apiKeys.Get("/:id", h.GetAPIKey)
	apiKeys.Patch("/:id", h.UpdateAPIKey)
	apiKeys.Post("/:id/revoke", h.RevokeAPIKey)
	apiKeys.Delete("/:id", h.DeleteAPIKey)

	apiKeys.Get("/:id/usage", h.GetUsage)
	apiKeys.Get("/:id/stats", h.GetStats)
	apiKeys.Post("/:id/reset-budget", h.ResetBudget)
}

func (h *APIKeyHandler) GetUsage(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid API key ID",
		})
	}

	limit := c.QueryInt("limit", 100)
	usage, err := h.budgetService.GetRecentUsage(c.Context(), uint(id), limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data":  usage,
		"count": len(usage),
	})
}

func (h *APIKeyHandler) GetStats(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid API key ID",
		})
	}

	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")

	var startTime, endTime time.Time
	if startTimeStr != "" {
		startTime, err = time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid start_time format, use RFC3339",
			})
		}
	}
	if endTimeStr != "" {
		endTime, err = time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid end_time format, use RFC3339",
			})
		}
	}

	stats, err := h.budgetService.GetUsageStats(c.Context(), uint(id), startTime, endTime)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	byEndpoint, err := h.budgetService.GetUsageByEndpoint(c.Context(), uint(id), startTime, endTime)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"overall":     stats,
		"by_endpoint": byEndpoint,
	})
}

func (h *APIKeyHandler) ResetBudget(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid API key ID",
		})
	}

	if err := h.budgetService.ResetBudget(c.Context(), uint(id)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Budget reset successfully",
	})
}
