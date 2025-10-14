package models

import "time"

type User struct {
	ID        string    `gorm:"primaryKey;type:varchar(255)" json:"id"`
	Email     string    `gorm:"uniqueIndex;not null;type:varchar(255)" json:"email"`
	Name      string    `gorm:"not null;type:varchar(255)" json:"name"`
	CreatedAt time.Time `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;autoUpdateTime" json:"updated_at"`
}

func (User) TableName() string {
	return "users"
}

type ProjectStatus string

const (
	ProjectStatusActive   ProjectStatus = "active"
	ProjectStatusInactive ProjectStatus = "inactive"
	ProjectStatusPaused   ProjectStatus = "paused"
)

type Project struct {
	ID             uint          `gorm:"primaryKey;autoIncrement" json:"id"`
	Name           string        `gorm:"not null;type:varchar(255)" json:"name"`
	Description    string        `gorm:"type:text" json:"description"`
	Status         ProjectStatus `gorm:"not null;default:'active';type:varchar(50)" json:"status"`
	Progress       int           `gorm:"not null;default:0" json:"progress"`
	OrganizationID string        `gorm:"not null;index;type:varchar(255)" json:"organization_id"`
	CreatedAt      time.Time     `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time     `gorm:"not null;autoUpdateTime" json:"updated_at"`

	Members []ProjectMember `gorm:"foreignKey:ProjectID" json:"members"`
}

func (Project) TableName() string {
	return "projects"
}

type ProjectMemberRole string

const (
	ProjectMemberRoleOwner  ProjectMemberRole = "owner"
	ProjectMemberRoleAdmin  ProjectMemberRole = "admin"
	ProjectMemberRoleMember ProjectMemberRole = "member"
)

type ProjectMember struct {
	ID        uint              `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    string            `gorm:"not null;index;type:varchar(255)" json:"user_id"`
	ProjectID uint              `gorm:"not null;index" json:"project_id"`
	Role      ProjectMemberRole `gorm:"not null;type:varchar(50)" json:"role"`
	CreatedAt time.Time         `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time         `gorm:"not null;autoUpdateTime" json:"updated_at"`

	Project *Project `gorm:"foreignKey:ProjectID" json:"-"`
}

func (ProjectMember) TableName() string {
	return "project_members"
}

type ProjectCreateRequest struct {
	Name           string        `json:"name" validate:"required,min=1,max=255"`
	Description    string        `json:"description,omitempty"`
	OrganizationID string        `json:"organization_id" validate:"required"`
	Status         ProjectStatus `json:"status,omitempty"`
}

type ProjectUpdateRequest struct {
	Name        string        `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Description *string       `json:"description,omitempty"`
	Status      ProjectStatus `json:"status,omitempty" validate:"omitempty,oneof=active inactive paused"`
	Progress    *int          `json:"progress,omitempty" validate:"omitempty,min=0,max=100"`
}

type AddProjectMemberRequest struct {
	UserID string            `json:"user_id" validate:"required"`
	Role   ProjectMemberRole `json:"role" validate:"required,oneof=owner admin member"`
}

type UpdateProjectMemberRoleRequest struct {
	Role string `json:"role" validate:"required,oneof=admin member"`
}

type ProjectResponse struct {
	ID             uint            `json:"id"`
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	Status         ProjectStatus   `json:"status"`
	Progress       int             `json:"progress"`
	OrganizationID string          `json:"organization_id"`
	Members        []ProjectMember `json:"members"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

func (p *Project) ToResponse() *ProjectResponse {
	return &ProjectResponse{
		ID:             p.ID,
		Name:           p.Name,
		Description:    p.Description,
		Status:         p.Status,
		Progress:       p.Progress,
		OrganizationID: p.OrganizationID,
		Members:        p.Members,
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}
}

type Organization struct {
	ID        string    `gorm:"primaryKey;type:varchar(255)" json:"id"`
	Name      string    `gorm:"not null;type:varchar(255)" json:"name"`
	OwnerID   string    `gorm:"not null;index;type:varchar(255)" json:"owner_id"`
	CreatedAt time.Time `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;autoUpdateTime" json:"updated_at"`
}

func (Organization) TableName() string {
	return "organizations"
}

type OrganizationMember struct {
	ID             uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID         string    `gorm:"not null;index;type:varchar(255)" json:"user_id"`
	OrganizationID string    `gorm:"not null;index;type:varchar(255)" json:"organization_id"`
	Role           string    `gorm:"not null;type:varchar(50)" json:"role"`
	CreatedAt      time.Time `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"not null;autoUpdateTime" json:"updated_at"`
}

func (OrganizationMember) TableName() string {
	return "organization_members"
}

type OrganizationCreateRequest struct {
	ID   string `json:"id" validate:"required"`
	Name string `json:"name" validate:"required,min=1,max=255"`
}

type OrganizationUpdateRequest struct {
	Name string `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
}

type AddOrganizationMemberRequest struct {
	UserID string `json:"user_id" validate:"required"`
	Role   string `json:"role" validate:"required,oneof=owner admin member"`
}

type UserCreateRequest struct {
	ID    string `json:"id" validate:"required"`
	Email string `json:"email" validate:"required,email"`
	Name  string `json:"name" validate:"required,min=1,max=255"`
}

type UserUpdateRequest struct {
	Email string `json:"email,omitempty" validate:"omitempty,email"`
	Name  string `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
}
