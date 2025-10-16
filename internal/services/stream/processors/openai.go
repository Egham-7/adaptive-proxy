package processors

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Egham-7/adaptive-proxy/internal/models"
	"github.com/Egham-7/adaptive-proxy/internal/services/format_adapter"
	"github.com/Egham-7/adaptive-proxy/internal/services/usage"

	"github.com/openai/openai-go/v2"
)

// OpenAIChunkProcessor handles OpenAI-specific format conversion
type OpenAIChunkProcessor struct {
	provider     string
	cacheSource  string
	requestID    string
	usageService *usage.Service
	apiKey       *models.APIKey
	model        string
	endpoint     string
	usageWorker  *usage.Worker
}

// NewOpenAIChunkProcessor creates a new OpenAI chunk processor
func NewOpenAIChunkProcessor(provider, cacheSource, requestID, model, endpoint string, usageService *usage.Service, apiKey *models.APIKey, usageWorker *usage.Worker) *OpenAIChunkProcessor {
	return &OpenAIChunkProcessor{
		provider:     provider,
		cacheSource:  cacheSource,
		requestID:    requestID,
		usageService: usageService,
		apiKey:       apiKey,
		model:        model,
		endpoint:     endpoint,
		usageWorker:  usageWorker,
	}
}

// Process converts OpenAI chunk data using existing format adapter
func (p *OpenAIChunkProcessor) Process(ctx context.Context, data []byte) ([]byte, error) {
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

	// Deserialize OpenAI chunk from data
	var openaiChunk openai.ChatCompletionChunk
	if err := json.Unmarshal(data, &openaiChunk); err != nil {
		return nil, fmt.Errorf("failed to unmarshal OpenAI chunk: %w", err)
	}

	// Use existing format adapter to convert to adaptive format
	adaptiveChunk, err := format_adapter.OpenAIToAdaptive.ConvertStreamingChunk(&openaiChunk, p.provider, p.cacheSource)
	if err != nil {
		return nil, fmt.Errorf("failed to convert OpenAI chunk: %w", err)
	}

	// Check if this chunk contains usage data (final chunk) and record it
	if adaptiveChunk.Usage.TotalTokens > 0 && p.usageWorker != nil && p.apiKey != nil {
		inputTokens := int(adaptiveChunk.Usage.PromptTokens)
		outputTokens := int(adaptiveChunk.Usage.CompletionTokens)

		usageParams := models.RecordUsageParams{
			APIKeyID:       p.apiKey.ID,
			OrganizationID: p.apiKey.OrganizationID,
			UserID:         p.apiKey.UserID,
			Endpoint:       p.endpoint,
			Provider:       p.provider,
			Model:          p.model,
			TokensInput:    inputTokens,
			TokensOutput:   outputTokens,
			Cost:           usage.CalculateCost(p.provider, p.model, inputTokens, outputTokens),
			StatusCode:     200,
			RequestID:      p.requestID,
		}

		p.usageWorker.Submit(usageParams, p.requestID)
	}

	// Marshal to JSON
	chunkJSON, err := json.Marshal(adaptiveChunk)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal adaptive chunk: %w", err)
	}

	// Format as SSE event - optimized with pre-allocated buffer
	dataPrefix := []byte("data: ")
	suffix := []byte("\n\n")

	result := make([]byte, 0, len(dataPrefix)+len(chunkJSON)+len(suffix))
	result = append(result, dataPrefix...)
	result = append(result, chunkJSON...)
	result = append(result, suffix...)

	return result, nil
}

// Provider returns the provider name
func (p *OpenAIChunkProcessor) Provider() string {
	return p.provider
}
