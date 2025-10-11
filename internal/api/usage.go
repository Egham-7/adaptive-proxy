package api

import (
	"strconv"
	"time"

	"github.com/Egham-7/adaptive-proxy/internal/models"
	"github.com/Egham-7/adaptive-proxy/internal/services/usage"
	"github.com/gofiber/fiber/v2"
)

type UsageHandler struct {
	usageService *usage.Service
}

func NewUsageHandler(usageService *usage.Service) *UsageHandler {
	return &UsageHandler{
		usageService: usageService,
	}
}

func (h *UsageHandler) RegisterRoutes(app *fiber.App, basePath string) {
	group := app.Group(basePath)
	group.Get("/:apiKeyId", h.GetUsageByAPIKey)
	group.Get("/:apiKeyId/stats", h.GetUsageStats)
	group.Get("/:apiKeyId/by-period", h.GetUsageByPeriod)
	group.Post("/", h.RecordUsage)
}

func (h *UsageHandler) GetUsageByAPIKey(c *fiber.Ctx) error {
	apiKeyIDStr := c.Params("apiKeyId")
	apiKeyID, err := strconv.ParseUint(apiKeyIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid api key id",
		})
	}

	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	usageRecords, err := h.usageService.GetUsageByAPIKey(c.Context(), uint(apiKeyID), limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get usage records",
		})
	}

	return c.JSON(usageRecords)
}

func (h *UsageHandler) GetUsageStats(c *fiber.Ctx) error {
	apiKeyIDStr := c.Params("apiKeyId")
	apiKeyID, err := strconv.ParseUint(apiKeyIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid api key id",
		})
	}

	startDateStr := c.Query("startDate")
	endDateStr := c.Query("endDate")

	var startDate, endDate time.Time
	if startDateStr != "" {
		startDate, err = time.Parse(time.RFC3339, startDateStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid start date format",
			})
		}
	}
	if endDateStr != "" {
		endDate, err = time.Parse(time.RFC3339, endDateStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid end date format",
			})
		}
	}

	stats, err := h.usageService.GetUsageStats(c.Context(), uint(apiKeyID), startDate, endDate)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get usage stats",
		})
	}

	return c.JSON(stats)
}

func (h *UsageHandler) GetUsageByPeriod(c *fiber.Ctx) error {
	apiKeyIDStr := c.Params("apiKeyId")
	apiKeyID, err := strconv.ParseUint(apiKeyIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid api key id",
		})
	}

	startDateStr := c.Query("startDate")
	endDateStr := c.Query("endDate")
	groupBy := c.Query("groupBy", "day")

	var startDate, endDate time.Time
	if startDateStr != "" {
		startDate, err = time.Parse(time.RFC3339, startDateStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid start date format",
			})
		}
	}
	if endDateStr != "" {
		endDate, err = time.Parse(time.RFC3339, endDateStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid end date format",
			})
		}
	}

	usageByPeriod, err := h.usageService.GetUsageByPeriod(c.Context(), uint(apiKeyID), startDate, endDate, groupBy)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get usage by period",
		})
	}

	return c.JSON(usageByPeriod)
}

func (h *UsageHandler) RecordUsage(c *fiber.Ctx) error {
	var params models.RecordUsageParams
	if err := c.BodyParser(&params); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if params.Metadata == nil {
		params.Metadata = make(models.Metadata)
	}

	usageRecord, err := h.usageService.RecordUsage(c.Context(), params)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(usageRecord)
}
