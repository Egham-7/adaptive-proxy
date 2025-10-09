package api

import (
	"context"
	"fmt"

	"adaptive-backend/internal/config"
	"adaptive-backend/internal/models"
	"adaptive-backend/internal/services/anthropic/messages"
	"adaptive-backend/internal/services/cache"
	"adaptive-backend/internal/services/circuitbreaker"
	"adaptive-backend/internal/services/fallback"
	"adaptive-backend/internal/services/model_router"
	"adaptive-backend/internal/services/stream/stream_simulator"
	"adaptive-backend/internal/utils"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/gofiber/fiber/v2"
	fiberlog "github.com/gofiber/fiber/v2/log"
)

// MessagesHandler handles Anthropic Messages API requests using dedicated Anthropic services
type MessagesHandler struct {
	cfg             *config.Config
	requestSvc      *messages.RequestService
	messagesSvc     *messages.MessagesService
	responseSvc     *messages.ResponseService
	modelRouter     *model_router.ModelRouter
	promptCache     *cache.AnthropicPromptCache
	circuitBreakers map[string]*circuitbreaker.CircuitBreaker
	fallbackService *fallback.FallbackService
}

// NewMessagesHandler creates a new MessagesHandler with Anthropic-specific services
func NewMessagesHandler(
	cfg *config.Config,
	modelRouter *model_router.ModelRouter,
	promptCache *cache.AnthropicPromptCache,
	circuitBreakers map[string]*circuitbreaker.CircuitBreaker,
) *MessagesHandler {
	return &MessagesHandler{
		cfg:             cfg,
		requestSvc:      messages.NewRequestService(),
		messagesSvc:     messages.NewMessagesService(),
		responseSvc:     messages.NewResponseService(modelRouter),
		modelRouter:     modelRouter,
		promptCache:     promptCache,
		circuitBreakers: circuitBreakers,
		fallbackService: fallback.NewFallbackService(cfg),
	}
}

// Messages handles the Anthropic Messages API HTTP request
func (h *MessagesHandler) Messages(c *fiber.Ctx) error {
	requestID := h.requestSvc.GetRequestID(c)
	fiberlog.Infof("[%s] Starting Anthropic Messages API request from %s", requestID, c.IP())

	// Parse Anthropic Messages request
	req, err := h.requestSvc.ParseRequest(c)
	if err != nil {
		fiberlog.Warnf("[%s] Request parsing failed: %v", requestID, err)
		return h.responseSvc.HandleBadRequest(c, err.Error(), requestID)
	}
	fiberlog.Debugf("[%s] Request parsed successfully - model: %s, messages: %d", requestID, req.Model, len(req.Messages))

	// Resolve config by merging YAML config with request overrides (single source of truth)
	resolvedConfig, err := h.cfg.ResolveConfigFromAnthropicRequest(req)
	if err != nil {
		fiberlog.Errorf("[%s] Config resolution failed: %v", requestID, err)
		return h.responseSvc.HandleError(c, fmt.Errorf("failed to resolve config: %w", err), requestID)
	}
	fiberlog.Debugf("[%s] Configuration resolved successfully", requestID)

	// Check if streaming is requested
	isStreaming := req.Stream != nil && *req.Stream
	fiberlog.Debugf("[%s] Request type: streaming=%t", requestID, isStreaming)

	// Check prompt cache first
	if cachedResponse, cacheSource, found := h.checkPromptCache(c.UserContext(), req, resolvedConfig.PromptCache, requestID); found {
		fiberlog.Infof("[%s] prompt cache hit (%s) - returning cached response", requestID, cacheSource)
		if isStreaming {
			// Convert cached response to streaming format
			return stream_simulator.StreamAnthropicCachedResponse(c, cachedResponse, requestID)
		}
		return c.JSON(cachedResponse)
	}

	// If a model is specified, try to directly route to the appropriate provider
	if req.Model != "" {
		modelStr := string(req.Model)
		fiberlog.Debugf("[%s] Model specified: %s, attempting direct routing", requestID, modelStr)

		// Parse provider and model from the model specification (expecting "provider:model" format)
		provider, parsedModel, err := utils.ParseProviderModel(modelStr)
		if err != nil {
			fiberlog.Debugf("[%s] Failed to parse model specification %s: %v, falling back to intelligent routing", requestID, modelStr, err)
			// Fall through to intelligent routing below instead of returning error
		} else {
			// Update the request with the parsed model name
			req.Model = anthropic.Model(parsedModel)

			fiberlog.Infof("[%s] User-specified model %s routed to provider %s", requestID, modelStr, provider)

			// Get provider configuration
			providers := resolvedConfig.GetProviders("messages")
			providerConfig, exists := providers[provider]
			if !exists {
				return h.responseSvc.HandleProviderNotConfigured(c, provider, requestID)
			}

			// Direct execution - no fallback for user-specified models
			err = h.messagesSvc.HandleAnthropicProvider(c, req, providerConfig, isStreaming, requestID, h.responseSvc, provider, "")
			if err != nil {
				return err
			}

			// Store successful response in semantic cache for user-specified models
			modelResp := &models.ModelSelectionResponse{
				Provider: provider,
				Model:    parsedModel,
			}
			h.responseSvc.StoreSuccessfulSemanticCache(c.UserContext(), req, modelResp, requestID)

			return nil
		}
	}

	// If no model is specified, use model router for selection WITH fallback
	fiberlog.Debugf("[%s] No model specified, using model router for selection with fallback", requestID)

	// Extract prompt for routing
	prompt, err := utils.ExtractPromptFromAnthropicMessages(req.Messages)
	if err != nil {
		fiberlog.Warnf("[%s] Failed to extract prompt for routing: %v", requestID, err)
		return h.responseSvc.HandleBadRequest(c, "failed to extract prompt for routing: "+err.Error(), requestID)
	}

	// Use model router to select best model WITH CIRCUIT BREAKERS
	userID := "anonymous"
	toolCall := utils.ExtractToolCallsFromAnthropicMessages(req.Messages)

	modelResp, cacheSource, err := h.modelRouter.SelectModelWithCache(
		c.UserContext(),
		prompt, userID, requestID, resolvedConfig.ModelRouter, h.circuitBreakers,
		req.Tools, toolCall,
	)
	if err != nil {
		fiberlog.Errorf("[%s] Model router selection failed: %v", requestID, err)
		return h.responseSvc.HandleError(c, err, requestID)
	}

	// Update request with selected model
	req.Model = anthropic.Model(modelResp.Model)
	fiberlog.Infof("[%s] Model router selected - provider: %s, model: %s (with %d alternatives)",
		requestID, modelResp.Provider, modelResp.Model, len(modelResp.Alternatives))

	// Try primary provider first
	primary := models.Alternative{
		Provider: modelResp.Provider,
		Model:    modelResp.Model,
	}
	executeFunc := h.createExecuteFunc(req, isStreaming, cacheSource)

	fiberlog.Infof("[%s] Trying primary provider: %s/%s", requestID, primary.Provider, primary.Model)
	err = executeFunc(c, primary, requestID)

	if err == nil {
		// Primary succeeded
		fiberlog.Infof("[%s] ✅ Primary provider succeeded: %s/%s", requestID, primary.Provider, primary.Model)
		return nil
	}

	// Primary failed - check if we have alternatives
	if len(modelResp.Alternatives) == 0 {
		fiberlog.Errorf("[%s] ❌ Primary provider failed and no alternatives available: %v", requestID, err)
		return err
	}

	// Use fallback service with alternatives only
	fiberlog.Warnf("[%s] ⚠️  Primary provider failed: %v", requestID, err)
	fiberlog.Infof("[%s] Using fallback with %d alternatives", requestID, len(modelResp.Alternatives))

	fallbackConfig := h.fallbackService.GetFallbackConfig(req.Fallback)
	return h.fallbackService.Execute(c, modelResp.Alternatives, fallbackConfig, executeFunc, requestID, isStreaming)
}

// createExecuteFunc creates an execution function for the fallback service
func (h *MessagesHandler) createExecuteFunc(
	req *models.AnthropicMessageRequest,
	isStreaming bool,
	cacheSource string,
) models.ExecutionFunc {
	return func(c *fiber.Ctx, provider models.Alternative, reqID string) error {
		// Get provider configuration from resolved config
		resolvedConfig, err := h.cfg.ResolveConfigFromAnthropicRequest(req)
		if err != nil {
			return fmt.Errorf("failed to resolve config: %w", err)
		}

		providers := resolvedConfig.GetProviders("messages")
		providerConfig, exists := providers[provider.Provider]
		if !exists {
			return fmt.Errorf("provider %s not configured", provider.Provider)
		}

		// Create a copy to avoid race conditions when mutating req.Model
		reqCopy := *req
		reqCopy.Model = anthropic.Model(provider.Model)

		// Call the messages service and handle retryable errors
		err = h.messagesSvc.HandleAnthropicProvider(c, &reqCopy, providerConfig, isStreaming, reqID, h.responseSvc, provider.Provider, cacheSource)
		// Check if the error is a retryable provider error that should trigger fallback
		if err != nil {
			// Record failure in circuit breaker
			if cb := h.circuitBreakers[provider.Provider]; cb != nil {
				cb.RecordFailure()
				streamType := "non-streaming"
				if isStreaming {
					streamType = "streaming"
				}
				fiberlog.Warnf("[%s] 🔴 Circuit breaker recorded FAILURE for provider %s (%s)", reqID, provider.Provider, streamType)
			}

			if appErr, ok := err.(*models.AppError); ok && appErr.Type == models.ErrorTypeProvider && appErr.Retryable {
				// Return the provider error to trigger fallback
				return err
			}
			// For non-retryable errors, return original AppError or create one with Retryable=false
			if appErr, ok := err.(*models.AppError); ok {
				// Return the original AppError to preserve the concrete type and Retryable=false signal
				return appErr
			}
			// For non-AppError types, create a non-retryable AppError
			return &models.AppError{
				Type:      models.ErrorTypeProvider,
				Message:   fmt.Sprintf("non-retryable error: %v", err),
				Retryable: false,
			}
		}

		// Record success in circuit breaker
		if cb := h.circuitBreakers[provider.Provider]; cb != nil {
			cb.RecordSuccess()
			streamType := "non-streaming"
			if isStreaming {
				streamType = "streaming"
			}
			fiberlog.Infof("[%s] 🟢 Circuit breaker recorded SUCCESS for provider %s (%s)", reqID, provider.Provider, streamType)
		}

		// Store successful response in semantic cache
		modelResp := &models.ModelSelectionResponse{
			Provider: provider.Provider,
			Model:    provider.Model,
		}
		h.responseSvc.StoreSuccessfulSemanticCache(c.UserContext(), &reqCopy, modelResp, reqID)

		return nil
	}
}

// checkPromptCache checks if prompt cache is enabled and returns cached response if found
func (h *MessagesHandler) checkPromptCache(ctx context.Context, req *models.AnthropicMessageRequest, promptCacheConfig *models.CacheConfig, requestID string) (*models.AnthropicMessage, string, bool) {
	if !promptCacheConfig.Enabled {
		fiberlog.Debugf("[%s] prompt cache disabled", requestID)
		return nil, "", false
	}

	if h.promptCache == nil {
		fiberlog.Debugf("[%s] prompt cache service not available", requestID)
		return nil, "", false
	}

	return h.promptCache.Get(ctx, req, requestID)
}
