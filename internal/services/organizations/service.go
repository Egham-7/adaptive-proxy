package organizations

import (
	"context"
	"fmt"

	"github.com/Egham-7/adaptive-proxy/internal/models"
	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{
		db: db,
	}
}

func (s *Service) DeleteOrganizationData(ctx context.Context, organizationID string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("organization_id = ?", organizationID).Delete(&models.OrganizationCredit{}).Error; err != nil {
			return fmt.Errorf("failed to delete organization credits: %w", err)
		}

		if err := tx.Where("organization_id = ?", organizationID).Delete(&models.CreditTransaction{}).Error; err != nil {
			return fmt.Errorf("failed to delete credit transactions: %w", err)
		}

		var apiKeys []models.APIKey
		if err := tx.Where("organization_id = ?", organizationID).Find(&apiKeys).Error; err != nil {
			return fmt.Errorf("failed to find api keys: %w", err)
		}

		for _, apiKey := range apiKeys {
			if err := tx.Where("api_key_id = ?", apiKey.ID).Delete(&models.APIKeyUsage{}).Error; err != nil {
				return fmt.Errorf("failed to delete api key usage: %w", err)
			}
		}

		if err := tx.Where("organization_id = ?", organizationID).Delete(&models.APIKey{}).Error; err != nil {
			return fmt.Errorf("failed to delete api keys: %w", err)
		}

		return nil
	})
}
