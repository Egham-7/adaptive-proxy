package projects

import (
	"context"
	"fmt"

	"github.com/Egham-7/adaptive-proxy/internal/models"
	"github.com/Egham-7/adaptive-proxy/internal/services/auth"
	"github.com/clerk/clerk-sdk-go/v2/organizationmembership"
	"gorm.io/gorm"
)

// AddOrgAdminsToProject adds all organization admins to a project as admin members
// excludeUserID is optional and allows skipping a user (e.g., the project creator)
func (s *Service) AddOrgAdminsToProject(ctx context.Context, projectID uint, organizationID, excludeUserID string) error {
	// Get all members of the organization from Clerk
	listParams := &organizationmembership.ListParams{
		OrganizationID: organizationID,
	}

	memberships, err := organizationmembership.List(ctx, listParams)
	if err != nil {
		return fmt.Errorf("failed to list organization members: %w", err)
	}

	// Filter for admins and prepare batch insert
	var adminsToAdd []models.ProjectMember
	for _, membership := range memberships.OrganizationMemberships {
		// Skip if not an admin
		if membership.Role != "org:admin" {
			continue
		}

		userID := membership.PublicUserData.UserID

		// Skip excluded user (e.g., project creator who's already owner)
		if userID == excludeUserID {
			continue
		}

		adminsToAdd = append(adminsToAdd, models.ProjectMember{
			UserID:    userID,
			ProjectID: projectID,
			Role:      models.ProjectMemberRoleAdmin,
		})
	}

	// Batch insert all admins (ignore duplicates)
	if len(adminsToAdd) > 0 {
		// Use Create which will fail on duplicate unique constraints
		// We handle this gracefully by continuing on constraint errors
		for _, admin := range adminsToAdd {
			err := s.db.WithContext(ctx).Create(&admin).Error
			if err != nil {
				// Ignore duplicate key errors, log others
				if !isDuplicateError(err) {
					// Log but don't fail - best effort sync
					fmt.Printf("Warning: failed to add org admin %s to project %d: %v\n", admin.UserID, projectID, err)
				}
			}
		}
	}

	return nil
}

// AddUserToAllOrgProjects adds a user to all projects in an organization
// This is called when a user is added as an org admin via webhook
func (s *Service) AddUserToAllOrgProjects(ctx context.Context, userID, organizationID string, role models.ProjectMemberRole) error {
	// Get all projects in the organization
	var projects []models.Project
	err := s.db.WithContext(ctx).
		Where("organization_id = ?", organizationID).
		Find(&projects).Error
	if err != nil {
		return fmt.Errorf("failed to list organization projects: %w", err)
	}

	// Add user to each project
	for _, project := range projects {
		member := &models.ProjectMember{
			UserID:    userID,
			ProjectID: project.ID,
			Role:      role,
		}

		err := s.db.WithContext(ctx).Create(member).Error
		if err != nil {
			// Ignore duplicate key errors, log others
			if !isDuplicateError(err) {
				// Log but don't fail - best effort sync
				fmt.Printf("Warning: failed to add user %s to project %d: %v\n", userID, project.ID, err)
			}
		}
	}

	return nil
}

// GetOrganizationAdmins retrieves all admin user IDs for an organization from Clerk
func (s *Service) GetOrganizationAdmins(ctx context.Context, organizationID string) ([]string, error) {
	listParams := &organizationmembership.ListParams{
		OrganizationID: organizationID,
	}

	memberships, err := organizationmembership.List(ctx, listParams)
	if err != nil {
		return nil, fmt.Errorf("failed to list organization members: %w", err)
	}

	var adminIDs []string
	for _, membership := range memberships.OrganizationMemberships {
		if membership.Role == "org:admin" {
			adminIDs = append(adminIDs, membership.PublicUserData.UserID)
		}
	}

	return adminIDs, nil
}

// isDuplicateError checks if an error is a duplicate key constraint violation
func isDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	// GORM and PostgreSQL duplicate key errors
	return err == gorm.ErrDuplicatedKey ||
		// Check for PostgreSQL unique violation error code
		containsString(err.Error(), "duplicate key") ||
		containsString(err.Error(), "unique constraint") ||
		containsString(err.Error(), "UNIQUE constraint failed")
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// RemoveUserFromAllOrgProjects removes a user from all projects in an organization
// Handles ownership transition: if user is owner, promotes another admin to owner
// This is called when a user is removed from org or demoted from admin role
func (s *Service) RemoveUserFromAllOrgProjects(ctx context.Context, userID, organizationID string) error {
	// Get all projects in the organization where this user is a member
	var members []models.ProjectMember
	err := s.db.WithContext(ctx).
		Joins("JOIN projects ON projects.id = project_members.project_id").
		Where("project_members.user_id = ? AND projects.organization_id = ?", userID, organizationID).
		Preload("Project").
		Find(&members).Error

	if err != nil {
		return fmt.Errorf("failed to find user's project memberships: %w", err)
	}

	// Process each project membership
	for _, member := range members {
		if member.Role == models.ProjectMemberRoleOwner {
			// User is owner - need to transfer ownership before removal
			if err := s.transferOwnership(ctx, member.Project, userID); err != nil {
				// Log error but continue with other projects
				fmt.Printf("Warning: failed to transfer ownership of project %d: %v\n", member.Project.ID, err)
				continue
			}
		}

		// Remove the user from this project
		err := s.db.WithContext(ctx).
			Where("project_id = ? AND user_id = ?", member.ProjectID, userID).
			Delete(&models.ProjectMember{}).Error

		if err != nil {
			// Log error but continue with other projects
			fmt.Printf("Warning: failed to remove user %s from project %d: %v\n", userID, member.ProjectID, err)
		}
	}

	return nil
}

// transferOwnership transfers project ownership from current owner to another admin
// If no other admin exists, the first member becomes admin and owner
func (s *Service) transferOwnership(ctx context.Context, project *models.Project, currentOwnerID string) error {
	// Find all other members of the project (excluding current owner)
	var members []models.ProjectMember
	err := s.db.WithContext(ctx).
		Where("project_id = ? AND user_id != ?", project.ID, currentOwnerID).
		Order("CASE WHEN role = 'admin' THEN 1 WHEN role = 'member' THEN 2 END, created_at ASC").
		Find(&members).Error

	if err != nil {
		return fmt.Errorf("failed to find project members: %w", err)
	}

	if len(members) == 0 {
		// No other members - we'll delete the owner, effectively making project ownerless
		// The project will remain in DB for potential recovery, but inaccessible
		fmt.Printf("Warning: Project %d has no other members after owner removal\n", project.ID)
		return nil
	}

	// Find the best candidate for new owner:
	// 1. Prefer existing admins (already sorted first by SQL)
	// 2. Otherwise use oldest member
	newOwner := members[0]

	// Promote to owner
	newOwner.Role = models.ProjectMemberRoleOwner
	err = s.db.WithContext(ctx).Save(&newOwner).Error
	if err != nil {
		return fmt.Errorf("failed to promote new owner: %w", err)
	}

	fmt.Printf("Transferred ownership of project %d from %s to %s\n",
		project.ID, currentOwnerID, newOwner.UserID)

	return nil
}

// ValidateUserIsOrgAdmin checks if a user is an admin of an organization
// This is a helper that wraps the auth provider's GetOrganizationRole
func (s *Service) ValidateUserIsOrgAdmin(ctx context.Context, userID, organizationID string) (bool, error) {
	// Type assert to get the Clerk auth provider
	clerkProvider, ok := s.authProvider.(*auth.ClerkAuthProvider)
	if !ok {
		return false, fmt.Errorf("auth provider is not ClerkAuthProvider")
	}

	role, err := clerkProvider.GetOrganizationRole(ctx, userID, organizationID)
	if err != nil {
		return false, nil // Not a member
	}

	return role == "org:admin", nil
}
