package usage

import (
	"context"
	"fmt"

	"github.com/Egham-7/adaptive-proxy/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CreditsService struct {
	db *gorm.DB
}

func NewCreditsService(db *gorm.DB) *CreditsService {
	return &CreditsService{db: db}
}

// AutoMigrate runs database migrations for credit tables
func (s *CreditsService) AutoMigrate() error {
	return s.db.AutoMigrate(
		&models.OrganizationCredit{},
		&models.CreditTransaction{},
		&models.CreditPackage{},
	)
}

// GetOrganizationCredit retrieves the credit record for an organization
// Creates one if it doesn't exist
func (s *CreditsService) GetOrganizationCredit(ctx context.Context, organizationID string) (*models.OrganizationCredit, error) {
	var credit models.OrganizationCredit

	err := s.db.WithContext(ctx).
		Where("organization_id = ?", organizationID).
		First(&credit).Error

	if err == gorm.ErrRecordNotFound {
		// Create new credit record
		credit = models.OrganizationCredit{
			ID:             generateCUID(),
			OrganizationID: organizationID,
			Balance:        0,
			TotalPurchased: 0,
			TotalUsed:      0,
		}

		if err := s.db.WithContext(ctx).Create(&credit).Error; err != nil {
			return nil, fmt.Errorf("failed to create organization credit: %w", err)
		}

		return &credit, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get organization credit: %w", err)
	}

	return &credit, nil
}

// CheckSufficientCredits verifies if an organization has enough credits
func (s *CreditsService) CheckSufficientCredits(ctx context.Context, organizationID string, amount float64) error {
	credit, err := s.GetOrganizationCredit(ctx, organizationID)
	if err != nil {
		return err
	}

	if credit.Balance < amount {
		return fmt.Errorf("insufficient credits: balance=%.6f, required=%.6f", credit.Balance, amount)
	}

	return nil
}

// DeductCredits deducts credits from an organization's balance
// Creates a transaction record and updates the balance atomically
func (s *CreditsService) DeductCredits(ctx context.Context, params models.DeductCreditsParams) (*models.CreditTransaction, error) {
	var transaction models.CreditTransaction

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Get current credit record with lock
		var credit models.OrganizationCredit
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("organization_id = ?", params.OrganizationID).
			First(&credit).Error; err != nil {
			return fmt.Errorf("failed to lock organization credit: %w", err)
		}

		// Allow overdraft - balance can go negative
		// Pre-flight check in middleware prevents starting new requests with balance <= 0

		// Update balance (can go negative)
		newBalance := credit.Balance - params.Amount
		newTotalUsed := credit.TotalUsed + params.Amount

		if err := tx.Model(&credit).Updates(map[string]any{
			"balance":    newBalance,
			"total_used": newTotalUsed,
		}).Error; err != nil {
			return fmt.Errorf("failed to update credit balance: %w", err)
		}

		// Create transaction record
		transaction = models.CreditTransaction{
			ID:             generateCUID(),
			OrganizationID: params.OrganizationID,
			UserID:         params.UserID,
			Type:           models.CreditTransactionUsage,
			Amount:         -params.Amount, // Negative for deduction
			BalanceAfter:   newBalance,
			Description:    params.Description,
			Metadata:       params.Metadata,
			APIKeyID:       params.APIKeyID,
			APIUsageID:     params.APIUsageID,
		}

		if err := tx.Create(&transaction).Error; err != nil {
			return fmt.Errorf("failed to create credit transaction: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &transaction, nil
}

// AddCredits adds credits to an organization's balance
func (s *CreditsService) AddCredits(ctx context.Context, params models.AddCreditsParams) (*models.CreditTransaction, error) {
	var transaction models.CreditTransaction

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Get current credit record with lock
		var credit models.OrganizationCredit
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("organization_id = ?", params.OrganizationID).
			First(&credit).Error; err != nil {
			// Create if doesn't exist
			if err == gorm.ErrRecordNotFound {
				credit = models.OrganizationCredit{
					ID:             generateCUID(),
					OrganizationID: params.OrganizationID,
					Balance:        0,
					TotalPurchased: 0,
					TotalUsed:      0,
				}
				if err := tx.Create(&credit).Error; err != nil {
					return fmt.Errorf("failed to create organization credit: %w", err)
				}
			} else {
				return fmt.Errorf("failed to lock organization credit: %w", err)
			}
		}

		// Update balance
		newBalance := credit.Balance + params.Amount
		updates := map[string]any{
			"balance": newBalance,
		}

		// Update total purchased if this is a purchase transaction
		if params.Type == models.CreditTransactionPurchase {
			updates["total_purchased"] = credit.TotalPurchased + params.Amount
		}

		if err := tx.Model(&credit).Updates(updates).Error; err != nil {
			return fmt.Errorf("failed to update credit balance: %w", err)
		}

		// Create transaction record
		transaction = models.CreditTransaction{
			ID:                    generateCUID(),
			OrganizationID:        params.OrganizationID,
			UserID:                params.UserID,
			Type:                  params.Type,
			Amount:                params.Amount, // Positive for addition
			BalanceAfter:          newBalance,
			Description:           params.Description,
			Metadata:              params.Metadata,
			StripePaymentIntentID: params.StripePaymentIntentID,
			StripeSessionID:       params.StripeSessionID,
		}

		if err := tx.Create(&transaction).Error; err != nil {
			return fmt.Errorf("failed to create credit transaction: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &transaction, nil
}

// GetTransactionHistory retrieves transaction history for an organization
func (s *CreditsService) GetTransactionHistory(ctx context.Context, organizationID string, limit, offset int) ([]models.CreditTransaction, error) {
	var transactions []models.CreditTransaction

	query := s.db.WithContext(ctx).
		Where("organization_id = ?", organizationID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&transactions).Error; err != nil {
		return nil, fmt.Errorf("failed to get transaction history: %w", err)
	}

	return transactions, nil
}

// GetCreditPackages retrieves all available credit packages
func (s *CreditsService) GetCreditPackages(ctx context.Context) ([]models.CreditPackage, error) {
	var packages []models.CreditPackage

	if err := s.db.WithContext(ctx).Find(&packages).Error; err != nil {
		return nil, fmt.Errorf("failed to get credit packages: %w", err)
	}

	return packages, nil
}

// Helper function to generate CUID-like IDs
func generateCUID() string {
	return uuid.New().String()
}
