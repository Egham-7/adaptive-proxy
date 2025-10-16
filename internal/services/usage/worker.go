package usage

import (
	"context"
	"sync"

	"github.com/Egham-7/adaptive-proxy/internal/models"
	fiberlog "github.com/gofiber/fiber/v2/log"
)

// Worker represents a usage recording worker that processes recording tasks
type Worker struct {
	service  *Service
	tasks    chan RecordTask
	wg       sync.WaitGroup
	stopOnce sync.Once
	stopped  chan struct{}
}

// RecordTask represents a usage recording task
type RecordTask struct {
	Params    models.RecordUsageParams
	RequestID string
}

// NewWorker creates a new usage recording worker with the specified pool size
func NewWorker(service *Service, poolSize, bufferSize int) *Worker {
	w := &Worker{
		service: service,
		tasks:   make(chan RecordTask, bufferSize),
		stopped: make(chan struct{}),
	}

	// Start worker goroutines
	for range poolSize {
		w.wg.Add(1)
		go w.run()
	}

	return w
}

// Submit submits a usage recording task to the worker pool
func (w *Worker) Submit(params models.RecordUsageParams, requestID string) {
	select {
	case <-w.stopped:
		// Worker stopped, log directly
		fiberlog.Warnf("[%s] Worker stopped, cannot submit usage recording task", requestID)
		return
	case w.tasks <- RecordTask{Params: params, RequestID: requestID}:
		// Task submitted successfully
	default:
		// Buffer full, log warning and drop task
		fiberlog.Warnf("[%s] Usage recording buffer full, dropping task", requestID)
	}
}

// run processes tasks from the queue
func (w *Worker) run() {
	defer w.wg.Done()

	for {
		select {
		case <-w.stopped:
			return
		case task := <-w.tasks:
			_, err := w.service.RecordUsage(context.Background(), task.Params)
			if err != nil {
				fiberlog.Errorf("[%s] Failed to record streaming usage: %v", task.RequestID, err)
			}
		}
	}
}

// Stop gracefully stops the worker pool
func (w *Worker) Stop() {
	w.stopOnce.Do(func() {
		close(w.stopped)
		close(w.tasks)
		w.wg.Wait()
	})
}
