package api

import (
	"context"
	"time"

	"github.com/Egham-7/adaptive-proxy/internal/config"
	"github.com/Egham-7/adaptive-proxy/internal/models"
	"github.com/Egham-7/adaptive-proxy/internal/services/model_router"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	cfg               *config.Config
	redisClient       *redis.Client
	modelRouterClient *model_router.ModelRouterClient
}

// NewHealthHandler creates a new health check handler
func NewHealthHandler(cfg *config.Config, redisClient *redis.Client) *HealthHandler {
	return &HealthHandler{
		cfg:               cfg,
		redisClient:       redisClient,
		modelRouterClient: model_router.NewModelRouterClient(cfg, redisClient),
	}
}

// HealthCheck returns the health status of the service and its dependencies
func (h *HealthHandler) HealthCheck(c *fiber.Ctx) error {
	redisStatus := h.checkRedis()

	aiServiceStatus := h.checkAIService()

	overallStatus := "healthy"
	statusCode := fiber.StatusOK

	if redisStatus != "healthy" || aiServiceStatus != "healthy" {
		overallStatus = "degraded"
		statusCode = fiber.StatusServiceUnavailable
	}

	response := fiber.Map{
		"status":    overallStatus,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"checks": fiber.Map{
			"redis":      redisStatus,
			"ai_service": aiServiceStatus,
		},
	}

	return c.Status(statusCode).JSON(response)
}

// checkRedis verifies Redis connectivity
func (h *HealthHandler) checkRedis() string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := h.redisClient.Ping(ctx).Err(); err != nil {
		return "unhealthy"
	}

	return "healthy"
}

// checkAIService verifies adaptive_ai service connectivity with a warm-up request
func (h *HealthHandler) checkAIService() string {
	if h.modelRouterClient == nil {
		return "unknown"
	}

	// Use a longer timeout to account for Modal cold starts (can take 10-20 seconds)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get configured providers from chat_completions endpoint
	configuredModels := h.cfg.GetModelCapabilitiesFromEndpoint("chat_completions")
	if len(configuredModels) == 0 {
		// Fallback to a default if no providers configured
		configuredModels = []models.ModelCapability{
			{Provider: "openai", ModelName: "gpt-5"},
		}
	}

	// Send a model selection request to warm up the Modal function
	// This uses the same client and authentication as normal model selection
	dummyRequest := models.ModelSelectionRequest{
		Prompt: "health check",
		Models: configuredModels,
	}

	response := h.modelRouterClient.SelectModel(ctx, dummyRequest)

	// Check if we got a valid response (not fallback)
	if response.IsValid() && response.Provider != "" && response.Model != "" {
		return "healthy"
	}

	return "unhealthy"
}
