package middleware

import (
	"fmt"
	"strings"

	"github.com/Egham-7/adaptive-proxy/internal/services/auth"
	"github.com/Egham-7/adaptive-proxy/internal/services/usage"
	"github.com/gofiber/fiber/v2"
)

type AuthMiddleware struct {
	authProvider  auth.AuthProvider
	apiKeyService *usage.APIKeyService
	usageService  *usage.Service
	rateLimiter   *usage.RateLimiter
	config        *AuthMiddlewareConfig
}

type AuthMiddlewareConfig struct {
	Enabled        bool
	AllowAnonymous bool
	ClerkSecretKey string
	HeaderNames    []string
	SkipPaths      []string
	EnableAPIKeys  bool
}

func DefaultAuthMiddlewareConfig() *AuthMiddlewareConfig {
	return &AuthMiddlewareConfig{
		Enabled:        true,
		AllowAnonymous: false,
		HeaderNames:    []string{"Authorization"},
		SkipPaths: []string{
			"/health",
			"/webhooks",
		},
		EnableAPIKeys: true,
	}
}

func NewAuthMiddleware(authProvider auth.AuthProvider, apiKeyService *usage.APIKeyService, usageService *usage.Service, config *AuthMiddlewareConfig) *AuthMiddleware {
	if config == nil {
		config = DefaultAuthMiddlewareConfig()
	}
	if len(config.HeaderNames) == 0 {
		config.HeaderNames = []string{"Authorization"}
	}
	return &AuthMiddleware{
		authProvider:  authProvider,
		apiKeyService: apiKeyService,
		usageService:  usageService,
		rateLimiter:   usage.NewRateLimiter(),
		config:        config,
	}
}

func (m *AuthMiddleware) Authenticate() fiber.Handler {
	return m.authenticate(false)
}

func (m *AuthMiddleware) RequireAuth() fiber.Handler {
	return m.authenticate(true)
}

func (m *AuthMiddleware) RequireClerkAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authCtx := auth.GetAuthContext(c)
		if authCtx == nil || !authCtx.IsClerk() {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "This operation requires user authentication (Clerk token)",
			})
		}
		return c.Next()
	}
}

func (m *AuthMiddleware) RequireAPIKeyAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authCtx := auth.GetAuthContext(c)
		if authCtx == nil || !authCtx.IsAPIKey() {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "This operation requires API key authentication",
			})
		}
		return c.Next()
	}
}

func (m *AuthMiddleware) authenticate(required bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !m.config.Enabled {
			return c.Next()
		}

		if m.shouldSkipPath(c.Path()) {
			return c.Next()
		}

		token := m.extractToken(c)

		if token == "" {
			if !required && m.config.AllowAnonymous {
				return c.Next()
			}
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authentication required",
			})
		}

		authenticated, authType, err := m.validateToken(c, token)
		if err != nil || !authenticated {
			errMsg := "Invalid or expired token"
			if err != nil {
				errMsg = err.Error()
			}
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": errMsg,
			})
		}

		c.Locals("auth_type", authType)
		c.Locals("auth_token", token)

		return c.Next()
	}
}

func (m *AuthMiddleware) extractToken(c *fiber.Ctx) string {
	for _, headerName := range m.config.HeaderNames {
		if header := c.Get(headerName); header != "" {
			if after, ok := strings.CutPrefix(header, "Bearer "); ok {
				return after
			}
			return strings.TrimSpace(header)
		}
	}

	return ""
}

func (m *AuthMiddleware) validateToken(c *fiber.Ctx, token string) (bool, string, error) {
	if m.config.ClerkSecretKey != "" {
		if authCtx, err := m.tryClerkToken(c, token); err == nil && authCtx != nil {
			c.Locals("auth_context", authCtx)
			return true, string(authCtx.Type), nil
		}
	}

	if m.config.EnableAPIKeys && m.apiKeyService != nil {
		authCtx, err := m.tryAPIKey(c, token)
		if err != nil {
			return false, "", fmt.Errorf("API key validation failed: %w", err)
		}
		if authCtx != nil {
			c.Locals("auth_context", authCtx)
			return true, string(authCtx.Type), nil
		}
	}

	return false, "", fmt.Errorf("invalid token")
}

func (m *AuthMiddleware) RequireScope(requiredScopes ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authCtx := auth.GetAuthContext(c)
		if authCtx == nil || !authCtx.IsAPIKey() || authCtx.APIKey == nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Insufficient permissions",
			})
		}

		scopes := authCtx.APIKey.Scopes
		if len(scopes) == 0 {
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

func (m *AuthMiddleware) tryClerkToken(c *fiber.Ctx, token string) (*auth.AuthContext, error) {
	clerkProvider, ok := m.authProvider.(*auth.ClerkAuthProvider)
	if !ok {
		return nil, fmt.Errorf("auth provider is not ClerkAuthProvider")
	}

	claims, err := clerkProvider.ValidateToken(c.Context(), token)
	if err != nil {
		return nil, err
	}

	return &auth.AuthContext{
		Type: auth.AuthTypeClerk,
		Clerk: &auth.ClerkAuthContext{
			UserID: claims.Subject,
			Claims: claims,
		},
	}, nil
}

func (m *AuthMiddleware) tryAPIKey(c *fiber.Ctx, token string) (*auth.AuthContext, error) {
	apiKey, err := m.apiKeyService.ValidateAPIKey(c.Context(), token)
	if err != nil {
		c.Locals("auth_error", err.Error())
		return nil, err
	}

	if apiKey.RateLimitRpm > 0 {
		allowed, err := m.rateLimiter.CheckRateLimit(c.Context(), apiKey.ID, apiKey.RateLimitRpm)
		if err != nil {
			return nil, fmt.Errorf("failed to check rate limit: %w", err)
		}
		if !allowed {
			return nil, fmt.Errorf("rate limit exceeded")
		}
	}

	if m.usageService != nil {
		withinLimit, _, err := m.usageService.CheckBudgetLimit(c.Context(), apiKey.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to check budget limit: %w", err)
		}
		if !withinLimit {
			return nil, fmt.Errorf("budget limit exceeded")
		}
	}

	scopes := []string{}
	if apiKey.Scopes != "" {
		scopes = strings.Split(apiKey.Scopes, ",")
	}

	return &auth.AuthContext{
		Type: auth.AuthTypeAPIKey,
		APIKey: &auth.APIKeyAuthContext{
			Key:            apiKey,
			UserID:         apiKey.UserID,
			OrganizationID: apiKey.OrganizationID,
			ProjectID:      apiKey.ProjectID,
			Scopes:         scopes,
		},
	}, nil
}

func (m *AuthMiddleware) shouldSkipPath(path string) bool {
	for _, skipPath := range m.config.SkipPaths {
		if strings.HasPrefix(path, skipPath) {
			return true
		}
	}
	return false
}
