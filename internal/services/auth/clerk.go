package auth

import (
	"context"
	"fmt"

	"github.com/Egham-7/adaptive-proxy/internal/models"
	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/jwt"
	"github.com/clerk/clerk-sdk-go/v2/organizationmembership"
	"gorm.io/gorm"
)

type ClerkAuthProvider struct {
	secretKey string
	db        *gorm.DB
}

func NewClerkAuthProvider(secretKey string, db *gorm.DB) *ClerkAuthProvider {
	clerk.SetKey(secretKey)

	return &ClerkAuthProvider{
		secretKey: secretKey,
		db:        db,
	}
}

func (p *ClerkAuthProvider) ValidateToken(ctx context.Context, token string) (*clerk.SessionClaims, error) {
	claims, err := jwt.Verify(ctx, &jwt.VerifyParams{
		Token: token,
	})
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	return claims, nil
}

func (p *ClerkAuthProvider) ValidateOrganizationAccess(ctx context.Context, userID, organizationID string) (bool, error) {
	listParams := &organizationmembership.ListParams{
		OrganizationID: organizationID,
		UserIDs:        []string{userID},
	}

	memberships, err := organizationmembership.List(ctx, listParams)
	if err != nil {
		return false, fmt.Errorf("failed to check organization membership: %w", err)
	}

	return len(memberships.OrganizationMemberships) > 0, nil
}

func (p *ClerkAuthProvider) ValidateProjectAccess(ctx context.Context, userID string, projectID uint, requiredRole Role) (bool, error) {
	var member models.ProjectMember

	err := p.db.WithContext(ctx).
		Where("user_id = ? AND project_id = ?", userID, projectID).
		First(&member).Error

	if err == gorm.ErrRecordNotFound {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("database error: %w", err)
	}

	memberRole := Role(member.Role)
	return memberRole.HasPermission(requiredRole), nil
}

func (p *ClerkAuthProvider) GetUserOrganizations(ctx context.Context, userID string) ([]string, error) {
	params := &organizationmembership.ListParams{
		UserIDs: []string{userID},
	}

	memberships, err := organizationmembership.List(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get user organizations: %w", err)
	}

	orgIDs := make([]string, 0, len(memberships.OrganizationMemberships))
	for _, membership := range memberships.OrganizationMemberships {
		orgIDs = append(orgIDs, membership.Organization.ID)
	}

	return orgIDs, nil
}

func (p *ClerkAuthProvider) GetOrganizationRole(ctx context.Context, userID, organizationID string) (string, error) {
	listParams := &organizationmembership.ListParams{
		OrganizationID: organizationID,
		UserIDs:        []string{userID},
	}

	memberships, err := organizationmembership.List(ctx, listParams)
	if err != nil {
		return "", fmt.Errorf("failed to get organization membership: %w", err)
	}

	if len(memberships.OrganizationMemberships) == 0 {
		return "", fmt.Errorf("user is not a member of this organization")
	}

	return memberships.OrganizationMemberships[0].Role, nil
}
