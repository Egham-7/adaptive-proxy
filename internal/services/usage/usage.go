package usage

import (
	"context"
	"fmt"
	"time"

	"github.com/Egham-7/adaptive-proxy/internal/models"
	"gorm.io/gorm"
)

type Service struct {
	db             *gorm.DB
	creditsService *CreditsService
}

func NewService(db *gorm.DB, creditsService *CreditsService) *Service {
	return &Service{
		db:             db,
		creditsService: creditsService,
	}
}

func (s *Service) AutoMigrate() error {
	return s.db.AutoMigrate(&models.APIKeyUsage{})
}

func (s *Service) RecordUsage(ctx context.Context, params models.RecordUsageParams) (*models.APIKeyUsage, error) {
	usage := models.APIKeyUsage{
		APIKeyID:     params.APIKeyID,
		Endpoint:     params.Endpoint,
		Provider:     params.Provider,
		Model:        params.Model,
		TokensInput:  params.TokensInput,
		TokensOutput: params.TokensOutput,
		TokensTotal:  params.TokensInput + params.TokensOutput,
		Cost:         params.Cost,
		Currency:     params.Currency,
		StatusCode:   params.StatusCode,
		LatencyMs:    params.LatencyMs,
		Metadata:     params.Metadata,
		RequestID:    params.RequestID,
		UserAgent:    params.UserAgent,
		IPAddress:    params.IPAddress,
		ErrorMessage: params.ErrorMessage,
	}

	if usage.Currency == "" {
		usage.Currency = "USD"
	}

	if err := s.db.WithContext(ctx).Create(&usage).Error; err != nil {
		return nil, fmt.Errorf("failed to record usage: %w", err)
	}

	if params.OrganizationID != "" && params.Cost > 0 {
		_, err := s.creditsService.DeductCredits(ctx, models.DeductCreditsParams{
			OrganizationID: params.OrganizationID,
			UserID:         params.UserID,
			Amount:         params.Cost,
			Description:    fmt.Sprintf("API usage: %s - %s", params.Provider, params.Model),
			Metadata: models.Metadata{
				"provider": params.Provider,
				"model":    params.Model,
				"endpoint": params.Endpoint,
			},
			APIKeyID:   params.APIKeyID,
			APIUsageID: usage.ID,
		})
		if err != nil {
			return &usage, fmt.Errorf("usage recorded but failed to deduct credits: %w", err)
		}
	}

	return &usage, nil
}

func (s *Service) GetUsageByAPIKey(ctx context.Context, apiKeyID uint, limit, offset int) ([]models.APIKeyUsage, error) {
	var usage []models.APIKeyUsage

	query := s.db.WithContext(ctx).
		Where("api_key_id = ?", apiKeyID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&usage).Error; err != nil {
		return nil, fmt.Errorf("failed to get usage: %w", err)
	}

	return usage, nil
}

func (s *Service) GetUsageStats(ctx context.Context, apiKeyID uint, startDate, endDate time.Time) (*models.UsageStats, error) {
	var stats models.UsageStats

	query := s.db.WithContext(ctx).
		Model(&models.APIKeyUsage{}).
		Where("api_key_id = ?", apiKeyID)

	if !startDate.IsZero() {
		query = query.Where("created_at >= ?", startDate)
	}
	if !endDate.IsZero() {
		query = query.Where("created_at <= ?", endDate)
	}

	err := query.
		Select(
			"COUNT(*) as total_requests",
			"COALESCE(SUM(cost), 0) as total_cost",
			"COALESCE(SUM(tokens_total), 0) as total_tokens",
			"COUNT(CASE WHEN status_code >= 200 AND status_code < 300 THEN 1 END) as success_requests",
			"COUNT(CASE WHEN status_code >= 400 OR status_code = 0 THEN 1 END) as failed_requests",
			"COALESCE(AVG(latency_ms), 0) as avg_latency_ms",
		).
		Scan(&stats).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get usage stats: %w", err)
	}

	return &stats, nil
}

func (s *Service) GetUsageByPeriod(ctx context.Context, apiKeyID uint, startDate, endDate time.Time, groupBy string) ([]map[string]any, error) {
	query := s.db.WithContext(ctx).
		Model(&models.APIKeyUsage{}).
		Where("api_key_id = ?", apiKeyID)

	if !startDate.IsZero() {
		query = query.Where("created_at >= ?", startDate)
	}
	if !endDate.IsZero() {
		query = query.Where("created_at <= ?", endDate)
	}

	var usageRecords []models.APIKeyUsage
	if err := query.Order("created_at ASC").Find(&usageRecords).Error; err != nil {
		return nil, fmt.Errorf("failed to get usage: %w", err)
	}

	periodMap := make(map[string]*models.PeriodStats)
	for _, record := range usageRecords {
		periodKey := formatPeriod(record.CreatedAt, groupBy)

		if periodMap[periodKey] == nil {
			periodMap[periodKey] = &models.PeriodStats{}
		}

		stats := periodMap[periodKey]
		stats.TotalRequests++
		stats.TotalCost += record.Cost
		stats.TotalTokens += record.TokensTotal

		if record.StatusCode >= 200 && record.StatusCode < 300 {
			stats.SuccessRequests++
		}
		if record.StatusCode >= 400 || record.StatusCode == 0 {
			stats.FailedRequests++
		}
	}

	results := make([]map[string]any, 0, len(periodMap))
	for period, stats := range periodMap {
		results = append(results, map[string]any{
			"period":           period,
			"total_requests":   stats.TotalRequests,
			"total_cost":       stats.TotalCost,
			"total_tokens":     stats.TotalTokens,
			"success_requests": stats.SuccessRequests,
			"failed_requests":  stats.FailedRequests,
		})
	}

	return results, nil
}

func formatPeriod(t time.Time, groupBy string) string {
	switch groupBy {
	case "hour":
		return t.Format("2006-01-02 15:00:00")
	case "week":
		year, week := t.ISOWeek()
		return fmt.Sprintf("%d-W%02d", year, week)
	case "month":
		return t.Format("2006-01")
	default:
		return t.Format("2006-01-02")
	}
}
