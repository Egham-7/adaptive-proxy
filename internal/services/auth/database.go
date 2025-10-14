package auth

import (
	"context"
	"fmt"

	"github.com/Egham-7/adaptive-proxy/internal/models"
	"gorm.io/gorm"
)

type DatabaseAuthProvider struct {
	db *gorm.DB
}

func NewDatabaseAuthProvider(db *gorm.DB) *DatabaseAuthProvider {
	return &DatabaseAuthProvider{db: db}
}

func (p *DatabaseAuthProvider) ValidateOrganizationAccess(ctx context.Context, userID, organizationID string) (bool, error) {
	var org models.Organization
	err := p.db.WithContext(ctx).
		Where("id = ? AND owner_id = ?", organizationID, userID).
		First(&org).Error

	if err == nil {
		return true, nil
	}

	if err != gorm.ErrRecordNotFound {
		return false, fmt.Errorf("database error: %w", err)
	}

	var member models.OrganizationMember
	err = p.db.WithContext(ctx).
		Where("organization_id = ? AND user_id = ?", organizationID, userID).
		First(&member).Error

	if err == gorm.ErrRecordNotFound {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("database error: %w", err)
	}

	return true, nil
}

func (p *DatabaseAuthProvider) ValidateProjectAccess(ctx context.Context, userID string, projectID uint, requiredRole Role) (bool, error) {
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

func (p *DatabaseAuthProvider) GetUserOrganizations(ctx context.Context, userID string) ([]string, error) {
	var orgIDs []string

	err := p.db.WithContext(ctx).
		Model(&models.Organization{}).
		Where("owner_id = ?", userID).
		Pluck("id", &orgIDs).Error

	if err != nil {
		return nil, fmt.Errorf("failed to query owner orgs: %w", err)
	}

	var memberOrgIDs []string
	err = p.db.WithContext(ctx).
		Model(&models.OrganizationMember{}).
		Where("user_id = ?", userID).
		Pluck("organization_id", &memberOrgIDs).Error

	if err != nil {
		return nil, fmt.Errorf("failed to query member orgs: %w", err)
	}

	allOrgIDs := append(orgIDs, memberOrgIDs...)
	return allOrgIDs, nil
}
