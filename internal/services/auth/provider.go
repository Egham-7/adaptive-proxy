package auth

import (
	"context"
	"time"
)

type AuthProvider interface {
	ValidateOrganizationAccess(ctx context.Context, userID, organizationID string) (bool, error)

	ValidateProjectAccess(ctx context.Context, userID string, projectID uint, requiredRole Role) (bool, error)

	GetUserOrganizations(ctx context.Context, userID string) ([]string, error)
}

type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

var RoleHierarchy = map[Role]int{
	RoleOwner:  3,
	RoleAdmin:  2,
	RoleMember: 1,
}

func (r Role) HasPermission(required Role) bool {
	return RoleHierarchy[r] >= RoleHierarchy[required]
}

type Organization struct {
	ID        string
	Name      string
	OwnerID   string
	CreatedAt time.Time
}

type Member struct {
	UserID string
	OrgID  string
	Role   string
}
