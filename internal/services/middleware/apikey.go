package middleware

import (
	"strings"

	"github.com/Egham-7/adaptive-proxy/internal/models"
	"github.com/Egham-7/adaptive-proxy/internal/services/apikey"
	"github.com/Egham-7/adaptive-proxy/internal/services/budget"
	"github.com/gofiber/fiber/v2"
)

type APIKeyMiddleware struct {
	service       *apikey.Service
	budgetService *budget.Service
	config        *models.APIKeyConfig
}

func NewAPIKeyMiddleware(service *apikey.Service, config *models.APIKeyConfig) *APIKeyMiddleware {
	if config == nil {
		defaultConfig := models.DefaultAPIKeyConfig()
		config = &defaultConfig
	}
	if len(config.HeaderNames) == 0 {
		config.HeaderNames = []string{"X-API-Key", "X-Stainless-API-Key"}
	}
	return &APIKeyMiddleware{
		service: service,
		config:  config,
	}
}

func NewAPIKeyMiddlewareWithBudget(service *apikey.Service, budgetService *budget.Service, config *models.APIKeyConfig) *APIKeyMiddleware {
	middleware := NewAPIKeyMiddleware(service, config)
	middleware.budgetService = budgetService
	return middleware
}

func (m *APIKeyMiddleware) Authenticate() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !m.config.Enabled {
			return c.Next()
		}

		key := m.extractAPIKey(c)

		if key == "" {
			if m.config.AllowAnonymous {
				return c.Next()
			}
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "API key required",
			})
		}

		apiKey, err := m.service.ValidateAPIKey(c.Context(), key)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired API key",
			})
		}

		if m.budgetService != nil {
			withinLimit, _, err := m.budgetService.CheckBudgetLimit(c.Context(), apiKey.ID)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to check budget limit",
				})
			}
			if !withinLimit {
				return c.Status(fiber.StatusPaymentRequired).JSON(fiber.Map{
					"error": "Budget limit exceeded",
				})
			}
		}

		c.Locals("api_key", apiKey)
		c.Locals("api_key_id", apiKey.ID)

		if apiKey.Scopes != "" {
			c.Locals("api_key_scopes", strings.Split(apiKey.Scopes, ","))
		}

		if apiKey.RateLimitRpm != 0 {
			c.Locals("api_key_rate_limit", apiKey.RateLimitRpm)
		}

		return c.Next()
	}
}

func (m *APIKeyMiddleware) RequireAPIKey() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !m.config.Enabled {
			return c.Next()
		}

		key := m.extractAPIKey(c)
		if key == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "API key required",
			})
		}

		apiKey, err := m.service.ValidateAPIKey(c.Context(), key)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired API key",
			})
		}

		if m.budgetService != nil {
			withinLimit, _, err := m.budgetService.CheckBudgetLimit(c.Context(), apiKey.ID)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to check budget limit",
				})
			}
			if !withinLimit {
				return c.Status(fiber.StatusPaymentRequired).JSON(fiber.Map{
					"error": "Budget limit exceeded",
				})
			}
		}

		c.Locals("api_key", apiKey)
		c.Locals("api_key_id", apiKey.ID)

		if apiKey.Scopes != "" {
			c.Locals("api_key_scopes", strings.Split(apiKey.Scopes, ","))
		}

		if apiKey.RateLimitRpm != 0 {
			c.Locals("api_key_rate_limit", apiKey.RateLimitRpm)
		}

		return c.Next()
	}
}

func (m *APIKeyMiddleware) RequireScope(requiredScopes ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !m.config.Enabled {
			return c.Next()
		}

		scopes, ok := c.Locals("api_key_scopes").([]string)
		if !ok || len(scopes) == 0 {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Insufficient permissions",
			})
		}

		scopeMap := make(map[string]bool)
		for _, scope := range scopes {
			scopeMap[scope] = true
		}

		for _, required := range requiredScopes {
			if !scopeMap[required] && !scopeMap["*"] {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "Insufficient permissions",
				})
			}
		}

		return c.Next()
	}
}

func (m *APIKeyMiddleware) extractAPIKey(c *fiber.Ctx) string {
	for _, headerName := range m.config.HeaderNames {
		key := c.Get(headerName)
		if key != "" {
			return strings.TrimSpace(key)
		}
	}

	authHeader := c.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			return strings.TrimSpace(parts[1])
		}
	}

	return ""
}
