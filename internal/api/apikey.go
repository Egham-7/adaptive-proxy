package api

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/Egham-7/adaptive-proxy/internal/models"
	"github.com/Egham-7/adaptive-proxy/internal/services/usage"
	"github.com/gofiber/fiber/v2"
)

type APIKeyHandler struct {
	service        *usage.APIKeyService
	budgetService  *usage.Service
	creditsEnabled bool
}

func NewAPIKeyHandler(service *usage.APIKeyService, budgetService *usage.Service, creditsEnabled bool) *APIKeyHandler {
	return &APIKeyHandler{
		service:        service,
		budgetService:  budgetService,
		creditsEnabled: creditsEnabled,
	}
}

func (h *APIKeyHandler) CreateAPIKey(c *fiber.Ctx) error {
	var req models.APIKeyCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if h.creditsEnabled {
		if req.OrganizationID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "organization_id is required when credits are enabled",
			})
		}
		if req.UserID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "user_id is required when credits are enabled",
			})
		}
		if req.ProjectID == nil || *req.ProjectID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "project_id is required when credits are enabled",
			})
		}
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

	apiKeys.Post("/verify", h.VerifyAPIKey)

	apiKeys.Get("/:id/usage", h.GetUsage)
	apiKeys.Get("/:id/stats", h.GetStats)
	apiKeys.Post("/:id/reset-budget", h.ResetBudget)

	// Deprecated: Use direct ClickHouse migration script instead
	apiKeys.Post("/migrate", h.MigrateFromPrisma)
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

func (h *APIKeyHandler) VerifyAPIKey(c *fiber.Ctx) error {
	var req struct {
		Key string `json:"key"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Key == "" {
		return c.JSON(fiber.Map{
			"valid":  false,
			"reason": "API key is required",
		})
	}

	apiKey, err := h.service.ValidateAPIKey(c.Context(), req.Key)
	if err != nil {
		return c.JSON(fiber.Map{
			"valid":  false,
			"reason": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"valid":        true,
		"api_key_id":   apiKey.ID,
		"metadata":     apiKey.Metadata,
		"expires_at":   apiKey.ExpiresAt,
		"is_active":    apiKey.IsActive,
		"last_used_at": apiKey.LastUsedAt,
	})
}

// MigrateFromPrisma accepts API key data from Prisma and inserts it into ClickHouse.
// Deprecated: This endpoint is for migration purposes only and will be removed after migration is complete.
func (h *APIKeyHandler) MigrateFromPrisma(c *fiber.Ctx) error {
	var req struct {
		ID         string  `json:"id"`
		UserID     string  `json:"userId"`
		ProjectID  *string `json:"projectId"`
		Name       string  `json:"name"`
		KeyPrefix  string  `json:"keyPrefix"`
		KeyHash    string  `json:"keyHash"`
		Status     string  `json:"status"`
		CreatedAt  string  `json:"createdAt"`
		UpdatedAt  string  `json:"updatedAt"`
		ExpiresAt  *string `json:"expiresAt"`
		LastUsedAt *string `json:"lastUsedAt"`
		Project    *struct {
			Name string `json:"name"`
		} `json:"project"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	var expiresAtTime time.Time
	if req.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid expiresAt format",
			})
		}
		expiresAtTime = t
	}

	var lastUsedAtTime time.Time
	if req.LastUsedAt != nil {
		t, err := time.Parse(time.RFC3339, *req.LastUsedAt)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid lastUsedAt format",
			})
		}
		lastUsedAtTime = t
	}

	createdAt, err := time.Parse(time.RFC3339, req.CreatedAt)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid createdAt format",
		})
	}

	updatedAt, err := time.Parse(time.RFC3339, req.UpdatedAt)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid updatedAt format",
		})
	}

	metadata := map[string]string{
		"prisma_id": req.ID,
		"user_id":   req.UserID,
	}
	if req.ProjectID != nil {
		metadata["project_id"] = *req.ProjectID
	}
	if req.Project != nil {
		metadata["project_name"] = req.Project.Name
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to encode metadata",
		})
	}

	apiKey := &models.APIKey{
		Name:       req.Name,
		KeyHash:    req.KeyHash,
		KeyPrefix:  req.KeyPrefix,
		Metadata:   string(metadataJSON),
		IsActive:   req.Status == "active",
		ExpiresAt:  expiresAtTime,
		LastUsedAt: lastUsedAtTime,
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
	}

	if err := h.service.MigrateAPIKey(c.Context(), apiKey); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":    "API key migrated successfully",
		"key_prefix": req.KeyPrefix,
	})
}
