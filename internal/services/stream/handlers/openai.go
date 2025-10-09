package handlers

import (
	"bufio"

	"github.com/Egham-7/adaptive-proxy/internal/services/stream/contracts"
	"github.com/Egham-7/adaptive-proxy/internal/services/stream/writers"

	"github.com/gofiber/fiber/v2"
	fiberlog "github.com/gofiber/fiber/v2/log"
	"github.com/openai/openai-go/v2"
	openai_ssestream "github.com/openai/openai-go/v2/packages/ssestream"
	"github.com/valyala/fasthttp"
)

// HandleOpenAI manages OpenAI streaming response using proper layered architecture
func HandleOpenAI(c *fiber.Ctx, resp *openai_ssestream.Stream[openai.ChatCompletionChunk], requestID, provider, cacheSource string) error {
	fiberlog.Infof("[%s] Starting OpenAI stream handling", requestID)

	// Create streaming pipeline - validates stream internally by reading first chunk
	// If validation fails (429, 500, etc.), error is returned BEFORE HTTP streaming starts
	// This allows fallback to trigger properly
	factory := NewStreamFactory()
	handler, err := factory.CreateOpenAIPipeline(resp, requestID, provider, cacheSource)
	if err != nil {
		fiberlog.Errorf("[%s] Stream validation failed: %v", requestID, err)
		return err
	}

	fiberlog.Infof("[%s] Stream validated successfully, starting HTTP stream", requestID)

	fasthttpCtx := c.Context()
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Access-Control-Allow-Origin", "*")

	fasthttpCtx.SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		// Create connection state tracker
		connState := writers.NewFastHTTPConnectionState(fasthttpCtx)

		// Create HTTP writer (with [DONE] message for OpenAI compatibility)
		httpWriter := writers.NewHTTPStreamWriter(w, connState, requestID, true)

		// Handle the stream
		if err := handler.Handle(fasthttpCtx, httpWriter); err != nil {
			if !contracts.IsExpectedError(err) {
				fiberlog.Errorf("[%s] Stream error: %v", requestID, err)
			} else {
				fiberlog.Infof("[%s] Stream ended: %v", requestID, err)
			}
		}
	}))

	return nil
}
