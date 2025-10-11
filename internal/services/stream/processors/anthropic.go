package processors

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Egham-7/adaptive-proxy/internal/models"
	"github.com/Egham-7/adaptive-proxy/internal/services/format_adapter"
	"github.com/Egham-7/adaptive-proxy/internal/services/usage"

	"github.com/anthropics/anthropic-sdk-go"
	fiberlog "github.com/gofiber/fiber/v2/log"
)

// AnthropicChunkProcessor handles Anthropic-specific format conversion
type AnthropicChunkProcessor struct {
	provider     string
	cacheSource  string
	requestID    string
	usageService *usage.Service
	apiKey       *models.APIKey
	model        string
	endpoint     string
}

// NewAnthropicChunkProcessor creates a new Anthropic chunk processor
func NewAnthropicChunkProcessor(provider, cacheSource, requestID, model, endpoint string, usageService *usage.Service, apiKey *models.APIKey) *AnthropicChunkProcessor {
	return &AnthropicChunkProcessor{
		provider:     provider,
		cacheSource:  cacheSource,
		requestID:    requestID,
		usageService: usageService,
		apiKey:       apiKey,
		model:        model,
		endpoint:     endpoint,
	}
}

// Process converts Anthropic chunk data using existing format adapter
func (p *AnthropicChunkProcessor) Process(ctx context.Context, data []byte) ([]byte, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Skip empty content
	if len(data) == 0 {
		return []byte{}, nil
	}

	// Try to parse as Anthropic chunk format
	var anthropicChunk anthropic.MessageStreamEventUnion
	if err := json.Unmarshal(data, &anthropicChunk); err != nil {
		// If not JSON, treat as raw text and pass through
		return data, nil
	}

	// Use existing format adapter to convert to adaptive format
	if format_adapter.AnthropicToAdaptive == nil {
		return nil, fmt.Errorf("AnthropicToAdaptive converter is not initialized")
	}

	adaptiveChunk, err := format_adapter.AnthropicToAdaptive.ConvertStreamingChunk(&anthropicChunk, p.provider, p.cacheSource)
	if err != nil {
		return nil, fmt.Errorf("failed to convert Anthropic chunk: %w", err)
	}

	// Check if this is a message_delta event with usage data and record it
	if adaptiveChunk.Type == "message_delta" && adaptiveChunk.Usage != nil && p.usageService != nil && p.apiKey != nil {
		usageParams := models.RecordUsageParams{
			APIKeyID:       p.apiKey.ID,
			OrganizationID: p.apiKey.OrganizationID,
			UserID:         p.apiKey.UserID,
			Endpoint:       p.endpoint,
			Provider:       p.provider,
			Model:          p.model,
			TokensInput:    int(adaptiveChunk.Usage.InputTokens),
			TokensOutput:   int(adaptiveChunk.Usage.OutputTokens),
			StatusCode:     200,
			RequestID:      p.requestID,
		}

		go func(params models.RecordUsageParams, reqID string) {
			_, err := p.usageService.RecordUsage(context.Background(), params)
			if err != nil {
				fiberlog.Errorf("[%s] Failed to record streaming usage: %v", reqID, err)
			}
		}(usageParams, p.requestID)
	}

	// Marshal to JSON
	chunkJSON, err := json.Marshal(adaptiveChunk)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal adaptive chunk: %w", err)
	}

	// Format as SSE event
	sseData := fmt.Sprintf("event: %s\ndata: %s\n\n", adaptiveChunk.Type, string(chunkJSON))
	return []byte(sseData), nil
}

// Provider returns the provider name
func (p *AnthropicChunkProcessor) Provider() string {
	return p.provider
}
