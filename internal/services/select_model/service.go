package select_model

import (
	"context"
	"fmt"

	"github.com/Egham-7/adaptive-proxy/internal/models"
	"github.com/Egham-7/adaptive-proxy/internal/services/circuitbreaker"
	"github.com/Egham-7/adaptive-proxy/internal/services/model_router"

	fiberlog "github.com/gofiber/fiber/v2/log"
)

// Service handles model selection logic
type Service struct {
	modelRouter *model_router.ModelRouter
}

// NewService creates a new select model service
func NewService(modelRouter *model_router.ModelRouter) *Service {
	return &Service{
		modelRouter: modelRouter,
	}
}

// SelectModel performs model selection based on the request
func (s *Service) SelectModel(
	ctx context.Context,
	req *models.SelectModelRequest,
	userID, requestID string,
	circuitBreakers map[string]*circuitbreaker.CircuitBreaker,
	mergedConfig *models.ModelRouterConfig,
) (*models.SelectModelResponse, error) {
	fiberlog.Infof("[%s] Starting model selection for user: %s", requestID, userID)

	if mergedConfig != nil {
		fiberlog.Debugf("[%s] Built model config from select model request - cost bias: %.2f", requestID, mergedConfig.CostBias)
	} else {
		fiberlog.Debugf("[%s] Built model config from select model request - using default config (mergedConfig is nil)", requestID)
	}

	// Perform model selection directly with prompt
	// Pass through tool context for function-calling-aware routing
	resp, cacheSource, err := s.modelRouter.SelectModelWithCache(
		ctx,
		req.Prompt, userID, requestID, mergedConfig, circuitBreakers,
		req.Tools, req.ToolCall, // Pass tool context for intelligent routing
	)
	if err != nil {
		fiberlog.Errorf("[%s] Model selection error: %v", requestID, err)
		return nil, fmt.Errorf("model selection failed: %w", err)
	}

	fiberlog.Infof("[%s] model selection completed - provider: %s, model: %s", requestID, resp.Provider, resp.Model)

	return &models.SelectModelResponse{
		Provider:     resp.Provider,
		Model:        resp.Model,
		Alternatives: resp.Alternatives,
		CacheTier:    cacheSource,
	}, nil
}
