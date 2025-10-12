package messages

import (
	"context"
	"fmt"

	"github.com/Egham-7/adaptive-proxy/internal/models"
	"github.com/Egham-7/adaptive-proxy/internal/services/format_adapter"
	"github.com/Egham-7/adaptive-proxy/internal/services/model_router"
	"github.com/Egham-7/adaptive-proxy/internal/services/stream/handlers"
	"github.com/Egham-7/adaptive-proxy/internal/services/usage"
	"github.com/Egham-7/adaptive-proxy/internal/utils"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/packages/ssestream"
	"github.com/gofiber/fiber/v2"
	fiberlog "github.com/gofiber/fiber/v2/log"
)

// ResponseService handles Anthropic Messages response processing and formatting
type ResponseService struct {
	modelRouter  *model_router.ModelRouter
	usageService *usage.Service
}

// NewResponseService creates a new ResponseService
func NewResponseService(modelRouter *model_router.ModelRouter, usageService *usage.Service) *ResponseService {
	return &ResponseService{
		modelRouter:  modelRouter,
		usageService: usageService,
	}
}

// HandleNonStreamingResponse processes a non-streaming Anthropic response
func (rs *ResponseService) HandleNonStreamingResponse(
	c *fiber.Ctx,
	message *anthropic.Message,
	requestID string,
	provider string,
	cacheSource string,
) error {
	fiberlog.Debugf("[%s] Converting Anthropic response to Adaptive format", requestID)
	// Convert response using format adapter
	adaptiveResponse, err := format_adapter.AnthropicToAdaptive.ConvertResponse(message, provider, cacheSource)
	if err != nil {
		fiberlog.Errorf("[%s] Failed to convert Anthropic response: %v", requestID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"type":    "internal_server_error",
				"message": fmt.Sprintf("Response conversion error: %v", err),
			},
		})
	}

	// Record usage if usage service is available
	if rs.usageService != nil {
		apiKeyInterface := c.Locals("api_key")
		if apiKey, ok := apiKeyInterface.(*models.APIKey); ok && apiKey != nil {
			inputTokens := int(adaptiveResponse.Usage.InputTokens)
			outputTokens := int(adaptiveResponse.Usage.OutputTokens)

			endpoint := "/v1/messages"
			model := string(message.Model)
			usageParams := models.RecordUsageParams{
				APIKeyID:       apiKey.ID,
				OrganizationID: apiKey.OrganizationID,
				UserID:         apiKey.UserID,
				Endpoint:       endpoint,
				Provider:       provider,
				Model:          model,
				TokensInput:    inputTokens,
				TokensOutput:   outputTokens,
				Cost:           usage.CalculateCost(provider, string(message.Model), inputTokens, outputTokens),
				StatusCode:     200,
				RequestID:      requestID,
			}

			_, err := rs.usageService.RecordUsage(c.UserContext(), usageParams)
			if err != nil {
				fiberlog.Errorf("[%s] Failed to record usage: %v", requestID, err)
			}
		}
	}

	fiberlog.Infof("[%s] Response converted successfully, sending to client", requestID)
	return c.JSON(adaptiveResponse)
}

// HandleStreamingResponse processes a streaming Anthropic response using the optimized stream handler
func (rs *ResponseService) HandleStreamingResponse(
	c *fiber.Ctx,
	anthropicStream *ssestream.Stream[anthropic.MessageStreamEventUnion],
	requestID string,
	provider string,
	cacheSource string,
	model string,
	endpoint string,
	usageService *usage.Service,
	apiKey *models.APIKey,
) error {
	fiberlog.Infof("[%s] Starting Anthropic streaming response handling", requestID)

	// Use the optimized stream handler that properly handles native Anthropic streams
	return handlers.HandleAnthropicNative(c, anthropicStream, requestID, provider, cacheSource, model, endpoint, usageService, apiKey)
}

// HandleError handles error responses for Anthropic Messages API
func (rs *ResponseService) HandleError(c *fiber.Ctx, err error, requestID string) error {
	fiberlog.Errorf("[%s] anthropic messages error: %v", requestID, err)

	response := fiber.Map{
		"error": fiber.Map{
			"message":    err.Error(),
			"request_id": requestID,
		},
	}

	return c.Status(fiber.StatusInternalServerError).JSON(response)
}

// StoreSuccessfulSemanticCache stores the model response in semantic cache after successful completion
func (rs *ResponseService) StoreSuccessfulSemanticCache(
	ctx context.Context,
	req *models.AnthropicMessageRequest,
	resp *models.ModelSelectionResponse,
	requestID string,
) {
	if rs.modelRouter == nil {
		fiberlog.Debugf("[%s] Model router not available for semantic cache storage", requestID)
		return
	}

	// Extract prompt for cache storage from Anthropic messages
	prompt, err := utils.ExtractPromptFromAnthropicMessages(req.Messages)
	if err != nil {
		fiberlog.Errorf("[%s] Failed to extract prompt for semantic cache: %v", requestID, err)
		return
	}

	// Store in semantic cache
	if err := rs.modelRouter.StoreSuccessfulModel(ctx, prompt, *resp, requestID, nil); err != nil {
		fiberlog.Warnf("[%s] Failed to store successful response in semantic cache: %v", requestID, err)
	}
}

// HandleBadRequest handles validation and request parsing errors
func (rs *ResponseService) HandleBadRequest(c *fiber.Ctx, message, requestID string) error {
	fiberlog.Warnf("[%s] bad request: %s", requestID, message)
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"error": fiber.Map{
			"type":    "invalid_request_error",
			"message": message,
		},
	})
}

// HandleProviderNotConfigured handles cases where the provider is not available
func (rs *ResponseService) HandleProviderNotConfigured(c *fiber.Ctx, provider, requestID string) error {
	message := fmt.Sprintf("Provider '%s' is not configured for messages endpoint", provider)
	fiberlog.Warnf("[%s] %s", requestID, message)
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"error": fiber.Map{
			"type":    "invalid_request_error",
			"message": message,
		},
	})
}
