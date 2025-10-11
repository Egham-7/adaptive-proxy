package apikey

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Egham-7/adaptive-proxy/internal/models"
	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) AutoMigrate() error {
	return s.db.AutoMigrate(&models.APIKey{})
}

func (s *Service) CreateAPIKey(ctx context.Context, req *models.APIKeyCreateRequest) (*models.APIKeyResponse, error) {
	key, err := models.GenerateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate API key: %w", err)
	}

	budgetCurrency := req.BudgetCurrency
	if budgetCurrency == "" {
		budgetCurrency = "USD"
	}

	apiKey := &models.APIKey{
		Name:            req.Name,
		KeyHash:         models.HashAPIKey(key),
		KeyPrefix:       models.ExtractKeyPrefix(key),
		Metadata:        req.Metadata,
		Scopes:          strings.Join(req.Scopes, ","),
		RateLimitRpm:    req.RateLimitRpm,
		BudgetLimit:     req.BudgetLimit,
		BudgetUsed:      0,
		BudgetCurrency:  budgetCurrency,
		BudgetResetType: req.BudgetResetType,
		IsActive:        true,
		ExpiresAt:       req.ExpiresAt,
	}

	if req.BudgetResetType != "" && req.BudgetResetType != models.BudgetResetNone {
		now := time.Now()
		apiKey.BudgetResetAt = calculateNextReset(now, req.BudgetResetType)
	}

	if err := s.db.WithContext(ctx).Create(apiKey).Error; err != nil {
		return nil, fmt.Errorf("failed to create API key: %w", err)
	}

	return &models.APIKeyResponse{
		ID:              apiKey.ID,
		Name:            apiKey.Name,
		Key:             key,
		KeyPrefix:       apiKey.KeyPrefix,
		Metadata:        apiKey.Metadata,
		Scopes:          apiKey.Scopes,
		RateLimitRpm:    apiKey.RateLimitRpm,
		BudgetLimit:     apiKey.BudgetLimit,
		BudgetUsed:      apiKey.BudgetUsed,
		BudgetRemaining: models.CalculateBudgetRemaining(apiKey.BudgetLimit, apiKey.BudgetUsed),
		BudgetCurrency:  apiKey.BudgetCurrency,
		BudgetResetType: apiKey.BudgetResetType,
		BudgetResetAt:   apiKey.BudgetResetAt,
		IsActive:        apiKey.IsActive,
		ExpiresAt:       apiKey.ExpiresAt,
		CreatedAt:       apiKey.CreatedAt,
		UpdatedAt:       apiKey.UpdatedAt,
	}, nil
}

func (s *Service) ValidateAPIKey(ctx context.Context, key string) (*models.APIKey, error) {
	if key == "" {
		return nil, fmt.Errorf("API key is required")
	}

	keyHash := models.HashAPIKey(key)
	var apiKey models.APIKey

	if err := s.db.WithContext(ctx).Where("key_hash = ? AND is_active = ?", keyHash, true).First(&apiKey).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("invalid API key")
		}
		return nil, fmt.Errorf("failed to validate API key: %w", err)
	}

	if !apiKey.ExpiresAt.IsZero() && apiKey.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("API key has expired")
	}

	now := time.Now()
	s.db.Model(&models.APIKey{}).Where("id = ?", apiKey.ID).Update("last_used_at", now)

	return &apiKey, nil
}

func (s *Service) ListAPIKeys(ctx context.Context, limit, offset int) ([]models.APIKeyResponse, int64, error) {
	var apiKeys []models.APIKey
	var total int64

	if err := s.db.WithContext(ctx).Model(&models.APIKey{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count API keys: %w", err)
	}

	query := s.db.WithContext(ctx).Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	if err := query.Find(&apiKeys).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list API keys: %w", err)
	}

	responses := make([]models.APIKeyResponse, len(apiKeys))
	for i, k := range apiKeys {
		responses[i] = models.APIKeyResponse{
			ID:              k.ID,
			Name:            k.Name,
			KeyPrefix:       k.KeyPrefix,
			Metadata:        k.Metadata,
			Scopes:          k.Scopes,
			RateLimitRpm:    k.RateLimitRpm,
			BudgetLimit:     k.BudgetLimit,
			BudgetUsed:      k.BudgetUsed,
			BudgetRemaining: models.CalculateBudgetRemaining(k.BudgetLimit, k.BudgetUsed),
			BudgetCurrency:  k.BudgetCurrency,
			BudgetResetType: k.BudgetResetType,
			BudgetResetAt:   k.BudgetResetAt,
			IsActive:        k.IsActive,
			ExpiresAt:       k.ExpiresAt,
			LastUsedAt:      k.LastUsedAt,
			CreatedAt:       k.CreatedAt,
			UpdatedAt:       k.UpdatedAt,
		}
	}

	return responses, total, nil
}

func (s *Service) GetAPIKey(ctx context.Context, id uint) (*models.APIKeyResponse, error) {
	var apiKey models.APIKey

	if err := s.db.WithContext(ctx).First(&apiKey, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("API key not found")
		}
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	return &models.APIKeyResponse{
		ID:              apiKey.ID,
		Name:            apiKey.Name,
		KeyPrefix:       apiKey.KeyPrefix,
		Metadata:        apiKey.Metadata,
		Scopes:          apiKey.Scopes,
		RateLimitRpm:    apiKey.RateLimitRpm,
		BudgetLimit:     apiKey.BudgetLimit,
		BudgetUsed:      apiKey.BudgetUsed,
		BudgetRemaining: models.CalculateBudgetRemaining(apiKey.BudgetLimit, apiKey.BudgetUsed),
		BudgetCurrency:  apiKey.BudgetCurrency,
		BudgetResetType: apiKey.BudgetResetType,
		BudgetResetAt:   apiKey.BudgetResetAt,
		IsActive:        apiKey.IsActive,
		ExpiresAt:       apiKey.ExpiresAt,
		LastUsedAt:      apiKey.LastUsedAt,
		CreatedAt:       apiKey.CreatedAt,
		UpdatedAt:       apiKey.UpdatedAt,
	}, nil
}

func (s *Service) RevokeAPIKey(ctx context.Context, id uint) error {
	result := s.db.WithContext(ctx).Model(&models.APIKey{}).Where("id = ?", id).Update("is_active", false)
	if result.Error != nil {
		return fmt.Errorf("failed to revoke API key: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("API key not found")
	}
	return nil
}

func (s *Service) DeleteAPIKey(ctx context.Context, id uint) error {
	result := s.db.WithContext(ctx).Delete(&models.APIKey{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete API key: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("API key not found")
	}
	return nil
}

func calculateNextReset(from time.Time, resetType string) time.Time {
	switch resetType {
	case models.BudgetResetDaily:
		return from.AddDate(0, 0, 1)
	case models.BudgetResetWeekly:
		return from.AddDate(0, 0, 7)
	case models.BudgetResetMonthly:
		return from.AddDate(0, 1, 0)
	default:
		return time.Time{}
	}
}

func (s *Service) UpdateAPIKey(ctx context.Context, id uint, updates map[string]any) error {
	allowedFields := map[string]bool{
		"name":              true,
		"metadata":          true,
		"scopes":            true,
		"rate_limit_rpm":    true,
		"budget_limit":      true,
		"budget_currency":   true,
		"budget_reset_type": true,
		"is_active":         true,
		"expires_at":        true,
	}

	filteredUpdates := make(map[string]any)
	for k, v := range updates {
		if allowedFields[k] {
			filteredUpdates[k] = v
		}
	}

	if resetType, ok := filteredUpdates["budget_reset_type"].(string); ok {
		if resetType != "" && resetType != models.BudgetResetNone {
			now := time.Now()
			filteredUpdates["budget_reset_at"] = calculateNextReset(now, resetType)
		}
	}

	result := s.db.WithContext(ctx).Model(&models.APIKey{}).Where("id = ?", id).Updates(filteredUpdates)
	if result.Error != nil {
		return fmt.Errorf("failed to update API key: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("API key not found")
	}
	return nil
}

func (s *Service) GetByHash(ctx context.Context, keyHash string) (*models.APIKey, error) {
	var apiKey models.APIKey

	if err := s.db.WithContext(ctx).Where("key_hash = ?", keyHash).First(&apiKey).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("API key not found")
		}
		return nil, fmt.Errorf("failed to get API key by hash: %w", err)
	}

	return &apiKey, nil
}

// MigrateAPIKey inserts a pre-existing API key from Prisma into the database.
// Deprecated: This is for migration purposes only and will be removed after migration is complete.
func (s *Service) MigrateAPIKey(ctx context.Context, apiKey *models.APIKey) error {
	if err := s.db.WithContext(ctx).Create(apiKey).Error; err != nil {
		return fmt.Errorf("failed to migrate API key: %w", err)
	}
	return nil
}
