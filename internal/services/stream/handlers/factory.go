package handlers

import (
	"iter"

	"github.com/Egham-7/adaptive-proxy/internal/models"
	"github.com/Egham-7/adaptive-proxy/internal/services/stream/contracts"
	"github.com/Egham-7/adaptive-proxy/internal/services/stream/processors"
	"github.com/Egham-7/adaptive-proxy/internal/services/stream/readers"
	"github.com/Egham-7/adaptive-proxy/internal/services/usage"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/packages/ssestream"
	"github.com/openai/openai-go/v2"
	openai_ssestream "github.com/openai/openai-go/v2/packages/ssestream"
	"google.golang.org/genai"
)

// StreamFactory creates properly layered streaming pipelines
type StreamFactory struct{}

// NewStreamFactory creates a new factory
func NewStreamFactory() *StreamFactory {
	return &StreamFactory{}
}

// CreateOpenAIPipeline creates a complete OpenAI streaming pipeline
// Returns error if stream validation fails (allows fallback before HTTP streaming starts)
func (f *StreamFactory) CreateOpenAIPipeline(
	stream *openai_ssestream.Stream[openai.ChatCompletionChunk],
	requestID, provider, cacheSource, model, endpoint string,
	usageService *usage.Service,
	apiKey *models.APIKey,
) (contracts.StreamHandler, error) {
	reader, err := readers.NewOpenAIStreamReader(stream, requestID)
	if err != nil {
		return nil, err
	}
	processor := processors.NewOpenAIChunkProcessor(provider, cacheSource, requestID, model, endpoint, usageService, apiKey)
	return NewStreamOrchestrator(reader, processor, requestID), nil
}

// CreateAnthropicNativePipeline creates a complete Anthropic native streaming pipeline
// Returns error if stream validation fails (allows fallback before HTTP streaming starts)
func (f *StreamFactory) CreateAnthropicNativePipeline(
	stream *ssestream.Stream[anthropic.MessageStreamEventUnion],
	requestID, provider, cacheSource, model, endpoint string,
	usageService *usage.Service,
	apiKey *models.APIKey,
) (contracts.StreamHandler, error) {
	reader, err := readers.NewAnthropicNativeStreamReader(stream, requestID)
	if err != nil {
		return nil, err
	}
	processor := processors.NewAnthropicChunkProcessor(provider, cacheSource, requestID, model, endpoint, usageService, apiKey)
	return NewStreamOrchestrator(reader, processor, requestID), nil
}

// CreateGeminiPipeline creates a complete Gemini streaming pipeline
// Returns error if stream validation fails (allows fallback before HTTP streaming starts)
func (f *StreamFactory) CreateGeminiPipeline(
	streamIter iter.Seq2[*genai.GenerateContentResponse, error],
	requestID, provider, cacheSource, model, endpoint string,
	usageService *usage.Service,
	apiKey *models.APIKey,
) (contracts.StreamHandler, error) {
	reader, err := readers.NewGeminiStreamReader(streamIter, requestID)
	if err != nil {
		return nil, err
	}
	// Use Gemini processor to format as SSE events for SDK compatibility
	processor := processors.NewGeminiChunkProcessor(provider, cacheSource, requestID, model, endpoint, usageService, apiKey)
	return NewStreamOrchestrator(reader, processor, requestID), nil
}
