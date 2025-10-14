package projects

import (
	"context"
	"errors"
	"fmt"

	"github.com/Egham-7/adaptive-proxy/internal/models"
	"github.com/Egham-7/adaptive-proxy/internal/services/auth"
	"gorm.io/gorm"
)

var (
	ErrProjectNotFound    = errors.New("project not found")
	ErrUnauthorized       = errors.New("unauthorized access")
	ErrDuplicateProjectID = errors.New("project with this ID already exists")
	ErrMemberNotFound     = errors.New("project member not found")
	ErrCannotRemoveOwner  = errors.New("cannot remove project owner")
	ErrCannotChangeOwner  = errors.New("cannot change owner role")
	ErrInvalidRole        = errors.New("invalid role specified")
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

func (s *Service) CreateProject(ctx context.Context, userID string, req *models.ProjectCreateRequest) (*models.Project, error) {
	hasAccess, err := s.authProvider.ValidateOrganizationAccess(ctx, userID, req.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate organization access: %w", err)
	}
	if !hasAccess {
		return nil, ErrUnauthorized
	}

	if req.Status == "" {
		req.Status = models.ProjectStatusActive
	}

	project := &models.Project{
		Name:           req.Name,
		Description:    req.Description,
		Status:         req.Status,
		Progress:       0,
		OrganizationID: req.OrganizationID,
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(project).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return ErrDuplicateProjectID
			}
			return fmt.Errorf("failed to create project: %w", err)
		}

		owner := &models.ProjectMember{
			UserID:    userID,
			ProjectID: project.ID,
			Role:      models.ProjectMemberRoleOwner,
		}
		if err := tx.Create(owner).Error; err != nil {
			return fmt.Errorf("failed to create project owner: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Reload the project with members
	err = s.db.WithContext(ctx).Preload("Members").First(project, project.ID).Error
	if err != nil {
		return nil, fmt.Errorf("failed to reload project: %w", err)
	}

	return project, nil
}

func (s *Service) GetProject(ctx context.Context, userID string, projectID uint) (*models.Project, error) {
	hasAccess, err := s.authProvider.ValidateProjectAccess(ctx, userID, projectID, auth.RoleMember)
	if err != nil {
		return nil, fmt.Errorf("failed to validate project access: %w", err)
	}
	if !hasAccess {
		return nil, ErrUnauthorized
	}

	var project models.Project
	err = s.db.WithContext(ctx).Preload("Members").First(&project, projectID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to fetch project: %w", err)
	}

	return &project, nil
}

func (s *Service) UpdateProject(ctx context.Context, userID string, projectID uint, req *models.ProjectUpdateRequest) (*models.Project, error) {
	hasAccess, err := s.authProvider.ValidateProjectAccess(ctx, userID, projectID, auth.RoleAdmin)
	if err != nil {
		return nil, fmt.Errorf("failed to validate project access: %w", err)
	}
	if !hasAccess {
		return nil, ErrUnauthorized
	}

	var project models.Project
	err = s.db.WithContext(ctx).First(&project, projectID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to fetch project: %w", err)
	}

	updates := make(map[string]any)
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}
	if req.Progress != nil {
		updates["progress"] = *req.Progress
	}

	if len(updates) > 0 {
		err = s.db.WithContext(ctx).Model(&project).Updates(updates).Error
		if err != nil {
			return nil, fmt.Errorf("failed to update project: %w", err)
		}
	}

	return &project, nil
}

func (s *Service) DeleteProject(ctx context.Context, userID string, projectID uint) error {
	hasAccess, err := s.authProvider.ValidateProjectAccess(ctx, userID, projectID, auth.RoleOwner)
	if err != nil {
		return fmt.Errorf("failed to validate project access: %w", err)
	}
	if !hasAccess {
		return ErrUnauthorized
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("project_id = ?", projectID).Delete(&models.ProjectMember{}).Error; err != nil {
			return fmt.Errorf("failed to delete project members: %w", err)
		}

		result := tx.Delete(&models.Project{}, projectID)
		if result.Error != nil {
			return fmt.Errorf("failed to delete project: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return ErrProjectNotFound
		}

		return nil
	})

	return err
}

func (s *Service) AddMember(ctx context.Context, userID string, projectID uint, req *models.AddProjectMemberRequest) (*models.ProjectMember, error) {
	hasAccess, err := s.authProvider.ValidateProjectAccess(ctx, userID, projectID, auth.RoleAdmin)
	if err != nil {
		return nil, fmt.Errorf("failed to validate project access: %w", err)
	}
	if !hasAccess {
		return nil, ErrUnauthorized
	}

	member := &models.ProjectMember{
		UserID:    req.UserID,
		ProjectID: projectID,
		Role:      req.Role,
	}

	err = s.db.WithContext(ctx).Create(member).Error
	if err != nil {
		return nil, fmt.Errorf("failed to add project member: %w", err)
	}

	return member, nil
}

func (s *Service) RemoveMember(ctx context.Context, userID string, projectID uint, targetUserID string) error {
	hasAccess, err := s.authProvider.ValidateProjectAccess(ctx, userID, projectID, auth.RoleAdmin)
	if err != nil {
		return fmt.Errorf("failed to validate project access: %w", err)
	}
	if !hasAccess {
		return ErrUnauthorized
	}

	var member models.ProjectMember
	err = s.db.WithContext(ctx).Where("project_id = ? AND user_id = ?", projectID, targetUserID).First(&member).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrMemberNotFound
		}
		return fmt.Errorf("failed to fetch member: %w", err)
	}

	if member.Role == models.ProjectMemberRoleOwner {
		return ErrCannotRemoveOwner
	}

	err = s.db.WithContext(ctx).Delete(&member).Error
	if err != nil {
		return fmt.Errorf("failed to remove member: %w", err)
	}

	return nil
}

func (s *Service) UpdateMemberRole(ctx context.Context, userID string, projectID uint, targetUserID, role string) (*models.ProjectMember, error) {
	hasAccess, err := s.authProvider.ValidateProjectAccess(ctx, userID, projectID, auth.RoleAdmin)
	if err != nil {
		return nil, fmt.Errorf("failed to validate project access: %w", err)
	}
	if !hasAccess {
		return nil, ErrUnauthorized
	}

	memberRole := models.ProjectMemberRole(role)
	if memberRole != models.ProjectMemberRoleAdmin && memberRole != models.ProjectMemberRoleMember {
		return nil, ErrInvalidRole
	}

	var member models.ProjectMember
	err = s.db.WithContext(ctx).Where("project_id = ? AND user_id = ?", projectID, targetUserID).First(&member).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMemberNotFound
		}
		return nil, fmt.Errorf("failed to fetch member: %w", err)
	}

	if member.Role == models.ProjectMemberRoleOwner {
		return nil, ErrCannotChangeOwner
	}

	member.Role = memberRole
	err = s.db.WithContext(ctx).Save(&member).Error
	if err != nil {
		return nil, fmt.Errorf("failed to update member role: %w", err)
	}

	return &member, nil
}

func (s *Service) ListProjects(ctx context.Context, userID, organizationID string) ([]models.Project, error) {
	hasAccess, err := s.authProvider.ValidateOrganizationAccess(ctx, userID, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate organization access: %w", err)
	}
	if !hasAccess {
		return nil, ErrUnauthorized
	}

	var projects []models.Project
	err = s.db.WithContext(ctx).Preload("Members").Where("organization_id = ?", organizationID).Find(&projects).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	return projects, nil
}
