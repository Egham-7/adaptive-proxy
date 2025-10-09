package readers

import (
	"encoding/json"
	"io"
	"iter"
	"sync"

	"github.com/Egham-7/adaptive-proxy/internal/utils"

	"github.com/valyala/bytebufferpool"
	"google.golang.org/genai"
)

// GeminiStreamReader provides pure I/O reading from Gemini streams
// This reader ONLY reads raw chunk data - no format conversion
type GeminiStreamReader struct {
	iterator   iter.Seq2[*genai.GenerateContentResponse, error]
	buffer     *bytebufferpool.ByteBuffer
	done       bool
	doneMux    sync.RWMutex
	requestID  string
	closeOnce  sync.Once
	next       func() (*genai.GenerateContentResponse, error, bool)
	stop       func()
	firstChunk *genai.GenerateContentResponse // Cached first chunk to replay
	firstRead  bool                           // Track if first chunk has been read
}

// NewGeminiStreamReader creates a new Gemini stream reader
// Validates stream by reading first chunk
func NewGeminiStreamReader(
	streamIter iter.Seq2[*genai.GenerateContentResponse, error],
	requestID string,
) (*GeminiStreamReader, error) {
	reader := &GeminiStreamReader{
		iterator:  streamIter,
		buffer:    utils.Get(), // Get buffer from pool
		requestID: requestID,
	}

	// Set up stateful iterator using iter.Pull2
	reader.setupIterator()

	// Validate stream by trying to get first chunk
	firstChunk, err, hasNext := reader.next()
	if !hasNext || err != nil {
		if reader.stop != nil {
			reader.stop()
		}
		if err != nil && err != io.EOF {
			return nil, err
		}
		return nil, io.EOF
	}

	// Store first chunk to replay it
	reader.firstChunk = firstChunk
	reader.firstRead = false

	return reader, nil
}

// setupIterator sets up the stateful iterator using iter.Pull2
func (r *GeminiStreamReader) setupIterator() {
	// Use iter.Pull2 to create a stateful next function and stop function
	nextFunc, stopFunc := iter.Pull2(r.iterator)

	// Store the stop function to release resources when needed
	r.stop = stopFunc

	// Create our next function that wraps the stateful iterator
	r.next = func() (*genai.GenerateContentResponse, error, bool) {
		resp, err, more := nextFunc()
		if !more {
			// Iterator is exhausted
			return nil, io.EOF, false
		}
		if err != nil {
			// Error occurred
			return nil, err, false
		}
		// Valid response
		return resp, nil, true
	}
}

// Read implements io.Reader - pure I/O operation
func (r *GeminiStreamReader) Read(p []byte) (n int, err error) {
	// Fast path: return buffered data first
	if len(r.buffer.B) > 0 {
		n = copy(p, r.buffer.B)
		r.buffer.B = r.buffer.B[n:]
		return n, nil
	}

	// Check if stream is done
	r.doneMux.RLock()
	done := r.done
	r.doneMux.RUnlock()

	if done {
		return 0, io.EOF
	}

	var chunk *genai.GenerateContentResponse
	var hasNext bool

	// If we have a first chunk and haven't read it yet, use it
	if r.firstChunk != nil && !r.firstRead {
		chunk = r.firstChunk
		err = nil
		hasNext = true
		r.firstRead = true
	} else {
		// Read next chunk from Gemini stream
		chunk, err, hasNext = r.next()
	}
	if !hasNext || err != nil {
		r.doneMux.Lock()
		r.done = true
		r.doneMux.Unlock()

		// Release iterator resources
		if r.stop != nil {
			r.stop()
		}

		if err != nil && err != io.EOF {
			return 0, err
		}
		return 0, io.EOF
	}

	// Marshal chunk to JSON (writer will handle array formatting)
	chunkData, err := json.Marshal(chunk)
	if err != nil {
		return 0, err
	}

	// Copy to output buffer (no additional formatting needed)
	n = copy(p, chunkData)
	if n < len(chunkData) {
		// Buffer remaining data for next read
		r.buffer.B = append(r.buffer.B, chunkData[n:]...)
	}

	return n, nil
}

// Close implements io.Closer
func (r *GeminiStreamReader) Close() error {
	var closeErr error
	r.closeOnce.Do(func() {
		r.doneMux.Lock()
		r.done = true
		r.doneMux.Unlock()

		if r.stop != nil {
			r.stop()
		}

		if r.buffer != nil {
			utils.Put(r.buffer) // Return buffer to pool
			r.buffer = nil
		}
	})
	return closeErr
}
