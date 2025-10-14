package auth

import (
	"github.com/Egham-7/adaptive-proxy/internal/models"
	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/gofiber/fiber/v2"
)

type AuthType string

const (
	AuthTypeClerk  AuthType = "clerk"
	AuthTypeAPIKey AuthType = "api_key"
)

type AuthContext struct {
	Type   AuthType
	Clerk  *ClerkAuthContext
	APIKey *APIKeyAuthContext
}

type ClerkAuthContext struct {
	UserID string
	Claims *clerk.SessionClaims
}

type APIKeyAuthContext struct {
	Key            *models.APIKey
	UserID         string
	OrganizationID string
	ProjectID      uint
	Scopes         []string
}

func (a *AuthContext) GetUserID() (string, bool) {
	switch a.Type {
	case AuthTypeClerk:
		if a.Clerk != nil {
			return a.Clerk.UserID, a.Clerk.UserID != ""
		}
	case AuthTypeAPIKey:
		if a.APIKey != nil {
			return a.APIKey.UserID, a.APIKey.UserID != ""
		}
	}
	return "", false
}

func (a *AuthContext) GetOrganizationID() (string, bool) {
	if a.Type == AuthTypeAPIKey && a.APIKey != nil {
		return a.APIKey.OrganizationID, a.APIKey.OrganizationID != ""
	}
	return "", false
}

func (a *AuthContext) GetProjectID() (uint, bool) {
	if a.Type == AuthTypeAPIKey && a.APIKey != nil {
		return a.APIKey.ProjectID, a.APIKey.ProjectID != 0
	}
	return 0, false
}

func (a *AuthContext) IsClerk() bool {
	return a.Type == AuthTypeClerk
}

func (a *AuthContext) IsAPIKey() bool {
	return a.Type == AuthTypeAPIKey
}

func GetAuthContext(c *fiber.Ctx) *AuthContext {
	authCtx, ok := c.Locals("auth_context").(*AuthContext)
	if !ok {
		return nil
	}
	return authCtx
}

func GetUserID(c *fiber.Ctx) (string, bool) {
	authCtx := GetAuthContext(c)
	if authCtx == nil {
		return "", false
	}
	return authCtx.GetUserID()
}

func GetAuthType(c *fiber.Ctx) string {
	authCtx := GetAuthContext(c)
	if authCtx == nil {
		return ""
	}
	return string(authCtx.Type)
}

func IsClerkAuth(c *fiber.Ctx) bool {
	authCtx := GetAuthContext(c)
	return authCtx != nil && authCtx.IsClerk()
}

func IsAPIKeyAuth(c *fiber.Ctx) bool {
	authCtx := GetAuthContext(c)
	return authCtx != nil && authCtx.IsAPIKey()
}

func GetClerkClaims(c *fiber.Ctx) (*clerk.SessionClaims, bool) {
	authCtx := GetAuthContext(c)
	if authCtx == nil || authCtx.Clerk == nil {
		return nil, false
	}
	return authCtx.Clerk.Claims, authCtx.Clerk.Claims != nil
}

func GetAPIKey(c *fiber.Ctx) (*models.APIKey, bool) {
	authCtx := GetAuthContext(c)
	if authCtx == nil || authCtx.APIKey == nil {
		return nil, false
	}
	return authCtx.APIKey.Key, authCtx.APIKey.Key != nil
}

func GetOrganizationID(c *fiber.Ctx) (string, bool) {
	authCtx := GetAuthContext(c)
	if authCtx == nil {
		return "", false
	}
	return authCtx.GetOrganizationID()
}

func GetProjectID(c *fiber.Ctx) (uint, bool) {
	authCtx := GetAuthContext(c)
	if authCtx == nil {
		return 0, false
	}
	return authCtx.GetProjectID()
}

func RequireClerkAuth(c *fiber.Ctx) error {
	if !IsClerkAuth(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "This operation requires user authentication (Clerk token)",
		})
	}
	return nil
}

func RequireAPIKeyAuth(c *fiber.Ctx) error {
	if !IsAPIKeyAuth(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "This operation requires API key authentication",
		})
	}
	return nil
}
