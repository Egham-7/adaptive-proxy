package middleware

import (
	"strings"

	"github.com/Egham-7/adaptive-proxy/internal/models"
	"github.com/Egham-7/adaptive-proxy/internal/services/usage"
	"github.com/gofiber/fiber/v2"
)

type APIKeyMiddleware struct {
	apiKeyService *usage.APIKeyService
	usageService  *usage.Service
	config        *models.APIKeyConfig
}

func NewAPIKeyMiddleware(apiKeyService *usage.APIKeyService, usageService *usage.Service, config *models.APIKeyConfig) *APIKeyMiddleware {
	if config == nil {
		defaultConfig := usage.DefaultAPIKeyConfig()
		config = &defaultConfig
	}
	if len(config.HeaderNames) == 0 {
		config.HeaderNames = []string{"X-API-Key", "X-Stainless-API-Key"}
	}
	return &APIKeyMiddleware{
		apiKeyService: apiKeyService,
		usageService:  usageService,
		config:        config,
	}
}

func (m *APIKeyMiddleware) Authenticate() fiber.Handler {
	return m.authenticate(false)
}

func (m *APIKeyMiddleware) RequireAPIKey() fiber.Handler {
	return m.authenticate(true)
}

func (m *APIKeyMiddleware) authenticate(required bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !m.config.Enabled {
			return c.Next()
		}

		key := m.extractAPIKey(c)

		if key == "" {
			if !required && m.config.AllowAnonymous {
				return c.Next()
			}
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "API key required",
			})
		}

		apiKey, err := m.apiKeyService.ValidateAPIKey(c.Context(), key)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired API key",
			})
		}

		if m.usageService != nil {
			withinLimit, _, err := m.usageService.CheckBudgetLimit(c.Context(), apiKey.ID)
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

		m.setLocals(c, apiKey)
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
		if key := c.Get(headerName); key != "" {
			key = strings.TrimSpace(key)
			c.Locals("api_key_raw", key)
			return key
		}
	}

	if authHeader := c.Get("Authorization"); authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			key := strings.TrimSpace(parts[1])
			c.Locals("api_key_raw", key)
			return key
		}
	}

	return ""
}

func (m *APIKeyMiddleware) setLocals(c *fiber.Ctx, apiKey *models.APIKey) {
	c.Locals("api_key", apiKey)
	c.Locals("api_key_id", apiKey.ID)

	if apiKey.Scopes != nil && *apiKey.Scopes != "" {
		c.Locals("api_key_scopes", strings.Split(*apiKey.Scopes, ","))
	}

	if apiKey.RateLimitRpm != nil && *apiKey.RateLimitRpm != 0 {
		c.Locals("api_key_rate_limit", *apiKey.RateLimitRpm)
	}
}
