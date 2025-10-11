package usage

import (
	"context"
	"log"
	"time"
)

type BudgetResetScheduler struct {
	usageService *Service
	interval     time.Duration
	stopChan     chan struct{}
}

func NewBudgetResetScheduler(usageService *Service, interval time.Duration) *BudgetResetScheduler {
	if interval == 0 {
		interval = 1 * time.Hour
	}
	return &BudgetResetScheduler{
		usageService: usageService,
		interval:     interval,
		stopChan:     make(chan struct{}),
	}
}

func (s *BudgetResetScheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	log.Printf("Budget reset scheduler started, running every %s", s.interval)

	for {
		select {
		case <-ticker.C:
			if err := s.usageService.ProcessScheduledBudgetResets(ctx); err != nil {
				log.Printf("Error processing scheduled budget resets: %v", err)
			} else {
				log.Println("Successfully processed scheduled budget resets")
			}
		case <-s.stopChan:
			log.Println("Budget reset scheduler stopped")
			return
		case <-ctx.Done():
			log.Println("Budget reset scheduler stopped due to context cancellation")
			return
		}
	}
}

func (s *BudgetResetScheduler) Stop() {
	close(s.stopChan)
}
