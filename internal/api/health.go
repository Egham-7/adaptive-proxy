package api

import (
	"context"
	"time"

	"github.com/Egham-7/adaptive-proxy/internal/config"
	"github.com/Egham-7/adaptive-proxy/internal/models"
	"github.com/Egham-7/adaptive-proxy/internal/services/database"
	"github.com/Egham-7/adaptive-proxy/internal/services/model_router"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

type HealthHandler struct {
	cfg               *config.Config
	redisClient       *redis.Client
	db                *database.DB
	modelRouterClient *model_router.ModelRouterClient
}

func NewHealthHandler(cfg *config.Config, redisClient *redis.Client, db *database.DB) *HealthHandler {
	return &HealthHandler{
		cfg:               cfg,
		redisClient:       redisClient,
		db:                db,
		modelRouterClient: model_router.NewModelRouterClient(cfg, redisClient),
	}
}

func (h *HealthHandler) HealthCheck(c *fiber.Ctx) error {
	redisStatus := h.checkRedis()
	dbStatus := h.checkDatabase()
	aiServiceStatus := h.checkAIService()

	overallStatus := "healthy"
	statusCode := fiber.StatusOK

	if redisStatus != "healthy" || dbStatus != "healthy" || aiServiceStatus != "healthy" {
		overallStatus = "degraded"
		statusCode = fiber.StatusServiceUnavailable
	}

	response := fiber.Map{
		"status":    overallStatus,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"checks": fiber.Map{
			"redis":      redisStatus,
			"database":   dbStatus,
			"ai_service": aiServiceStatus,
		},
	}

	return c.Status(statusCode).JSON(response)
}

func (h *HealthHandler) checkRedis() string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := h.redisClient.Ping(ctx).Err(); err != nil {
		return "unhealthy"
	}

	return "healthy"
}

func (h *HealthHandler) checkDatabase() string {
	if h.db == nil {
		return "not_configured"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- h.db.Ping()
	}()

	select {
	case err := <-done:
		if err != nil {
			return "unhealthy"
		}
		return "healthy"
	case <-ctx.Done():
		return "unhealthy"
	}
}

func (h *HealthHandler) checkAIService() string {
	if h.modelRouterClient == nil {
		return "unknown"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	configuredModels := h.cfg.GetModelCapabilitiesFromEndpoint("chat_completions")
	if len(configuredModels) == 0 {
		configuredModels = []models.ModelCapability{
			{Provider: "openai", ModelName: "gpt-5"},
		}
	}

	dummyRequest := models.ModelSelectionRequest{
		Prompt: "health check",
		Models: configuredModels,
	}

	response := h.modelRouterClient.SelectModel(ctx, dummyRequest)

	if response.IsValid() && response.Provider != "" && response.Model != "" {
		return "healthy"
	}

	return "unhealthy"
}
