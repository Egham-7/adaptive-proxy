package budget

import (
	"context"
	"fmt"
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
	return s.db.AutoMigrate(&models.APIKeyUsage{})
}

func (s *Service) TrackUsage(ctx context.Context, usage *models.APIKeyUsage) error {
	if err := s.db.WithContext(ctx).Create(usage).Error; err != nil {
		return fmt.Errorf("failed to track usage: %w", err)
	}

	if usage.Cost > 0 && usage.APIKeyID > 0 {
		if err := s.IncrementBudgetUsed(ctx, usage.APIKeyID, usage.Cost); err != nil {
			return fmt.Errorf("failed to increment budget: %w", err)
		}
	}

	return nil
}

func (s *Service) IncrementBudgetUsed(ctx context.Context, apiKeyID uint, cost float64) error {
	result := s.db.WithContext(ctx).
		Model(&models.APIKey{}).
		Where("id = ?", apiKeyID).
		Update("budget_used", gorm.Expr("budget_used + ?", cost))

	if result.Error != nil {
		return fmt.Errorf("failed to increment budget: %w", result.Error)
	}

	return nil
}

func (s *Service) CheckBudgetLimit(ctx context.Context, apiKeyID uint) (bool, *models.APIKey, error) {
	var apiKey models.APIKey
	if err := s.db.WithContext(ctx).First(&apiKey, apiKeyID).Error; err != nil {
		return false, nil, fmt.Errorf("failed to get API key: %w", err)
	}

	if apiKey.BudgetLimit == nil {
		return true, &apiKey, nil
	}

	if apiKey.BudgetUsed >= *apiKey.BudgetLimit {
		return false, &apiKey, nil
	}

	return true, &apiKey, nil
}

func (s *Service) ResetBudget(ctx context.Context, apiKeyID uint) error {
	now := time.Now()
	nextReset := s.calculateNextReset(now, "")

	result := s.db.WithContext(ctx).
		Model(&models.APIKey{}).
		Where("id = ?", apiKeyID).
		Updates(map[string]any{
			"budget_used":     0,
			"budget_reset_at": nextReset,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to reset budget: %w", result.Error)
	}

	return nil
}

func (s *Service) ProcessScheduledBudgetResets(ctx context.Context) error {
	now := time.Now()

	var apiKeys []models.APIKey
	if err := s.db.WithContext(ctx).
		Where("budget_reset_type != ? AND budget_reset_at IS NOT NULL AND budget_reset_at <= ?", models.BudgetResetNone, now).
		Find(&apiKeys).Error; err != nil {
		return fmt.Errorf("failed to find API keys for budget reset: %w", err)
	}

	for _, apiKey := range apiKeys {
		nextReset := s.calculateNextReset(now, apiKey.BudgetResetType)

		if err := s.db.WithContext(ctx).
			Model(&models.APIKey{}).
			Where("id = ?", apiKey.ID).
			Updates(map[string]any{
				"budget_used":     0,
				"budget_reset_at": nextReset,
			}).Error; err != nil {
			return fmt.Errorf("failed to reset budget for key %d: %w", apiKey.ID, err)
		}
	}

	return nil
}

func (s *Service) calculateNextReset(from time.Time, resetType string) *time.Time {
	var next time.Time

	switch resetType {
	case models.BudgetResetDaily:
		next = from.AddDate(0, 0, 1)
	case models.BudgetResetWeekly:
		next = from.AddDate(0, 0, 7)
	case models.BudgetResetMonthly:
		next = from.AddDate(0, 1, 0)
	default:
		return nil
	}

	return &next
}

func (s *Service) GetUsageStats(ctx context.Context, apiKeyID uint, startTime, endTime time.Time) (*models.UsageStats, error) {
	var stats models.UsageStats

	query := s.db.WithContext(ctx).
		Model(&models.APIKeyUsage{}).
		Where("api_key_id = ?", apiKeyID)

	if !startTime.IsZero() {
		query = query.Where("created_at >= ?", startTime)
	}
	if !endTime.IsZero() {
		query = query.Where("created_at <= ?", endTime)
	}

	if err := query.
		Select(`
			COUNT(*) as total_requests,
			COALESCE(SUM(cost), 0) as total_cost,
			COALESCE(SUM(tokens_total), 0) as total_tokens,
			COALESCE(SUM(CASE WHEN status_code >= 200 AND status_code < 300 THEN 1 ELSE 0 END), 0) as success_requests,
			COALESCE(SUM(CASE WHEN status_code >= 400 THEN 1 ELSE 0 END), 0) as failed_requests,
			COALESCE(AVG(latency_ms), 0) as avg_latency_ms
		`).
		Scan(&stats).Error; err != nil {
		return nil, fmt.Errorf("failed to get usage stats: %w", err)
	}

	return &stats, nil
}

func (s *Service) GetUsageByEndpoint(ctx context.Context, apiKeyID uint, startTime, endTime time.Time) (map[string]*models.UsageStats, error) {
	type EndpointStats struct {
		Endpoint string
		models.UsageStats
	}

	query := s.db.WithContext(ctx).
		Model(&models.APIKeyUsage{}).
		Where("api_key_id = ?", apiKeyID)

	if !startTime.IsZero() {
		query = query.Where("created_at >= ?", startTime)
	}
	if !endTime.IsZero() {
		query = query.Where("created_at <= ?", endTime)
	}

	var results []EndpointStats
	if err := query.
		Select(`
			endpoint,
			COUNT(*) as total_requests,
			COALESCE(SUM(cost), 0) as total_cost,
			COALESCE(SUM(tokens_total), 0) as total_tokens,
			COALESCE(SUM(CASE WHEN status_code >= 200 AND status_code < 300 THEN 1 ELSE 0 END), 0) as success_requests,
			COALESCE(SUM(CASE WHEN status_code >= 400 THEN 1 ELSE 0 END), 0) as failed_requests,
			COALESCE(AVG(latency_ms), 0) as avg_latency_ms
		`).
		Group("endpoint").
		Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to get usage by endpoint: %w", err)
	}

	statsMap := make(map[string]*models.UsageStats)
	for _, result := range results {
		statsMap[result.Endpoint] = &models.UsageStats{
			TotalRequests:   result.TotalRequests,
			TotalCost:       result.TotalCost,
			TotalTokens:     result.TotalTokens,
			SuccessRequests: result.SuccessRequests,
			FailedRequests:  result.FailedRequests,
			AvgLatencyMs:    result.AvgLatencyMs,
		}
	}

	return statsMap, nil
}

func (s *Service) GetRecentUsage(ctx context.Context, apiKeyID uint, limit int) ([]models.APIKeyUsage, error) {
	var usage []models.APIKeyUsage

	query := s.db.WithContext(ctx).
		Where("api_key_id = ?", apiKeyID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&usage).Error; err != nil {
		return nil, fmt.Errorf("failed to get recent usage: %w", err)
	}

	return usage, nil
}
