package admin

import (
	"context"
	"errors"
	"fmt"

	"github.com/Egham-7/adaptive-proxy/internal/models"
	"github.com/Egham-7/adaptive-proxy/internal/services/auth"
	"gorm.io/gorm"
)

var (
	ErrUnauthorized         = errors.New("unauthorized")
	ErrOrganizationExists   = errors.New("organization already exists")
	ErrOrganizationNotFound = errors.New("organization not found")
	ErrUserExists           = errors.New("user already exists")
	ErrUserNotFound         = errors.New("user not found")
	ErrMemberNotFound       = errors.New("member not found")
	ErrCannotRemoveOwner    = errors.New("cannot remove organization owner")
)

type Service struct {
	db           *gorm.DB
	authProvider auth.AuthProvider
}

func NewService(db *gorm.DB, authProvider auth.AuthProvider) *Service {
	return &Service{
		db:           db,
		authProvider: authProvider,
	}
}

func (s *Service) CreateOrganization(ctx context.Context, userID string, req *models.OrganizationCreateRequest) (*models.Organization, error) {
	var existing models.Organization
	err := s.db.WithContext(ctx).Where("id = ?", req.ID).First(&existing).Error
	if err == nil {
		return nil, ErrOrganizationExists
	}
	if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("database error: %w", err)
	}

	org := &models.Organization{
		ID:      req.ID,
		Name:    req.Name,
		OwnerID: userID,
	}

	if err := s.db.WithContext(ctx).Create(org).Error; err != nil {
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}

	member := &models.OrganizationMember{
		UserID:         userID,
		OrganizationID: org.ID,
		Role:           string(auth.RoleOwner),
	}

	if err := s.db.WithContext(ctx).Create(member).Error; err != nil {
		return nil, fmt.Errorf("failed to add owner as member: %w", err)
	}

	return org, nil
}

func (s *Service) GetOrganization(ctx context.Context, userID, organizationID string) (*models.Organization, error) {
	hasAccess, err := s.authProvider.ValidateOrganizationAccess(ctx, userID, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate access: %w", err)
	}
	if !hasAccess {
		return nil, ErrUnauthorized
	}

	var org models.Organization
	if err := s.db.WithContext(ctx).Where("id = ?", organizationID).First(&org).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrOrganizationNotFound
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	return &org, nil
}

func (s *Service) ListOrganizations(ctx context.Context, userID string) ([]*models.Organization, error) {
	organizationIDs, err := s.authProvider.GetUserOrganizations(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user organizations: %w", err)
	}

	if len(organizationIDs) == 0 {
		return []*models.Organization{}, nil
	}

	var organizations []*models.Organization
	if err := s.db.WithContext(ctx).
		Where("id IN ?", organizationIDs).
		Find(&organizations).Error; err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	return organizations, nil
}

func (s *Service) UpdateOrganization(ctx context.Context, userID, organizationID string, req *models.OrganizationUpdateRequest) (*models.Organization, error) {
	var org models.Organization
	if err := s.db.WithContext(ctx).Where("id = ?", organizationID).First(&org).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrOrganizationNotFound
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	if org.OwnerID != userID {
		return nil, ErrUnauthorized
	}

	if req.Name != "" {
		org.Name = req.Name
	}

	if err := s.db.WithContext(ctx).Save(&org).Error; err != nil {
		return nil, fmt.Errorf("failed to update organization: %w", err)
	}

	return &org, nil
}

func (s *Service) DeleteOrganization(ctx context.Context, userID, organizationID string) error {
	var org models.Organization
	if err := s.db.WithContext(ctx).Where("id = ?", organizationID).First(&org).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrOrganizationNotFound
		}
		return fmt.Errorf("database error: %w", err)
	}

	if org.OwnerID != userID {
		return ErrUnauthorized
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("organization_id = ?", organizationID).Delete(&models.OrganizationMember{}).Error; err != nil {
			return fmt.Errorf("failed to delete members: %w", err)
		}

		if err := tx.Delete(&org).Error; err != nil {
			return fmt.Errorf("failed to delete organization: %w", err)
		}

		return nil
	})
}

func (s *Service) AddOrganizationMember(ctx context.Context, userID, organizationID string, req *models.AddOrganizationMemberRequest) (*models.OrganizationMember, error) {
	var org models.Organization
	if err := s.db.WithContext(ctx).Where("id = ?", organizationID).First(&org).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrOrganizationNotFound
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	if org.OwnerID != userID {
		return nil, ErrUnauthorized
	}

	member := &models.OrganizationMember{
		UserID:         req.UserID,
		OrganizationID: organizationID,
		Role:           req.Role,
	}

	if err := s.db.WithContext(ctx).Create(member).Error; err != nil {
		return nil, fmt.Errorf("failed to add member: %w", err)
	}

	return member, nil
}

func (s *Service) RemoveOrganizationMember(ctx context.Context, userID, organizationID, targetUserID string) error {
	var org models.Organization
	if err := s.db.WithContext(ctx).Where("id = ?", organizationID).First(&org).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrOrganizationNotFound
		}
		return fmt.Errorf("database error: %w", err)
	}

	if org.OwnerID != userID {
		return ErrUnauthorized
	}

	if org.OwnerID == targetUserID {
		return ErrCannotRemoveOwner
	}

	result := s.db.WithContext(ctx).
		Where("organization_id = ? AND user_id = ?", organizationID, targetUserID).
		Delete(&models.OrganizationMember{})

	if result.Error != nil {
		return fmt.Errorf("failed to remove member: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrMemberNotFound
	}

	return nil
}

func (s *Service) ListOrganizationMembers(ctx context.Context, userID, organizationID string) ([]*models.OrganizationMember, error) {
	hasAccess, err := s.authProvider.ValidateOrganizationAccess(ctx, userID, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate access: %w", err)
	}
	if !hasAccess {
		return nil, ErrUnauthorized
	}

	var members []*models.OrganizationMember
	if err := s.db.WithContext(ctx).
		Where("organization_id = ?", organizationID).
		Find(&members).Error; err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	return members, nil
}

func (s *Service) CreateUser(ctx context.Context, req *models.UserCreateRequest) (*models.User, error) {
	var existing models.User
	err := s.db.WithContext(ctx).Where("id = ?", req.ID).First(&existing).Error
	if err == nil {
		return nil, ErrUserExists
	}
	if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("database error: %w", err)
	}

	user := &models.User{
		ID:    req.ID,
		Email: req.Email,
		Name:  req.Name,
	}

	if err := s.db.WithContext(ctx).Create(user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

func (s *Service) GetUser(ctx context.Context, userID string) (*models.User, error) {
	var user models.User
	if err := s.db.WithContext(ctx).Where("id = ?", userID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	return &user, nil
}

func (s *Service) UpdateUser(ctx context.Context, userID string, req *models.UserUpdateRequest) (*models.User, error) {
	var user models.User
	if err := s.db.WithContext(ctx).Where("id = ?", userID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	if req.Email != "" {
		user.Email = req.Email
	}
	if req.Name != "" {
		user.Name = req.Name
	}

	if err := s.db.WithContext(ctx).Save(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return &user, nil
}

func (s *Service) DeleteUser(ctx context.Context, userID string) error {
	result := s.db.WithContext(ctx).Where("id = ?", userID).Delete(&models.User{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete user: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}
