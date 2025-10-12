package processors

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Egham-7/adaptive-proxy/internal/models"
	"github.com/Egham-7/adaptive-proxy/internal/services/format_adapter"
	"github.com/Egham-7/adaptive-proxy/internal/services/usage"

	fiberlog "github.com/gofiber/fiber/v2/log"
	"google.golang.org/genai"
)

// GeminiChunkProcessor handles processing of Gemini stream chunks
// Converts between formats and adds provider metadata
type GeminiChunkProcessor struct {
	provider     string
	cacheSource  string
	requestID    string
	usageService *usage.Service
	apiKey       *models.APIKey
	model        string
	endpoint     string
}

// NewGeminiChunkProcessor creates a new Gemini chunk processor
func NewGeminiChunkProcessor(provider, cacheSource, requestID, model, endpoint string, usageService *usage.Service, apiKey *models.APIKey) *GeminiChunkProcessor {
	return &GeminiChunkProcessor{
		provider:     provider,
		cacheSource:  cacheSource,
		requestID:    requestID,
		usageService: usageService,
		apiKey:       apiKey,
		model:        model,
		endpoint:     endpoint,
	}
}

// Process processes raw Gemini chunk data and converts it to our adaptive format
func (p *GeminiChunkProcessor) Process(ctx context.Context, data []byte) ([]byte, error) {
	// Skip empty data
	if len(data) == 0 {
		return nil, nil
	}

	// Parse the Gemini response chunk
	var geminiChunk genai.GenerateContentResponse
	if err := json.Unmarshal(data, &geminiChunk); err != nil {
		fiberlog.Errorf("[%s] Failed to unmarshal Gemini chunk: %v", p.requestID, err)
		return nil, fmt.Errorf("failed to parse Gemini chunk: %w", err)
	}

	// Convert to our adaptive format using the format adapter
	adaptiveResponse, err := format_adapter.GeminiToAdaptive.ConvertResponse(&geminiChunk, p.provider, p.cacheSource)
	if err != nil {
		fiberlog.Errorf("[%s] Failed to convert Gemini response: %v", p.requestID, err)
		return nil, fmt.Errorf("failed to convert Gemini response: %w", err)
	}

	// Add cache source metadata if available
	if p.cacheSource != "" {
		fiberlog.Debugf("[%s] Chunk served from cache: %s", p.requestID, p.cacheSource)
		// Note: Cache metadata would be added here if needed in the response structure
	}

	if adaptiveResponse.UsageMetadata != nil && p.usageService != nil && p.apiKey != nil {
		inputTokens := int(adaptiveResponse.UsageMetadata.PromptTokenCount)
		outputTokens := int(adaptiveResponse.UsageMetadata.CandidatesTokenCount)

		usageParams := models.RecordUsageParams{
			APIKeyID:       p.apiKey.ID,
			OrganizationID: p.apiKey.OrganizationID,
			UserID:         p.apiKey.UserID,
			Endpoint:       &p.endpoint,
			Provider:       &p.provider,
			Model:          &p.model,
			TokensInput:    inputTokens,
			TokensOutput:   outputTokens,
			Cost:           usage.CalculateCost(p.provider, p.model, inputTokens, outputTokens),
			StatusCode:     200,
			RequestID:      &p.requestID,
		}

		go func(params models.RecordUsageParams, reqID string) {
			_, err := p.usageService.RecordUsage(context.Background(), params)
			if err != nil {
				fiberlog.Errorf("[%s] Failed to record streaming usage: %v", reqID, err)
			}
		}(usageParams, p.requestID)
	}

	// Marshal back to JSON for output
	chunkJSON, err := json.Marshal(adaptiveResponse)
	if err != nil {
		fiberlog.Errorf("[%s] Failed to marshal adaptive response: %v", p.requestID, err)
		return nil, fmt.Errorf("failed to marshal adaptive response: %w", err)
	}

	// Format as SSE event - this matches the responseLineRE regex: /^\s*data: (.*)(?:\n\n|\r\r|\r\n\r\n)/
	sseData := fmt.Sprintf("data: %s\n\n", string(chunkJSON))
	return []byte(sseData), nil
}

// Provider returns the provider name
func (p *GeminiChunkProcessor) Provider() string {
	return p.provider
}
