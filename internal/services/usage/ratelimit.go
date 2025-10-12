package usage

import (
	"context"
	"sync"
	"time"
)

type RateLimiter struct {
	mu       sync.RWMutex
	requests map[uint][]time.Time
}

func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[uint][]time.Time),
	}
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) CheckRateLimit(ctx context.Context, apiKeyID uint, limitRpm int) (bool, error) {
	if limitRpm <= 0 {
		return true, nil
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	oneMinuteAgo := now.Add(-1 * time.Minute)

	if rl.requests[apiKeyID] == nil {
		rl.requests[apiKeyID] = []time.Time{}
	}

	filtered := []time.Time{}
	for _, reqTime := range rl.requests[apiKeyID] {
		if reqTime.After(oneMinuteAgo) {
			filtered = append(filtered, reqTime)
		}
	}
	rl.requests[apiKeyID] = filtered

	if len(rl.requests[apiKeyID]) >= limitRpm {
		return false, nil
	}

	rl.requests[apiKeyID] = append(rl.requests[apiKeyID], now)
	return true, nil
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		oneMinuteAgo := now.Add(-1 * time.Minute)

		for apiKeyID, requests := range rl.requests {
			filtered := []time.Time{}
			for _, reqTime := range requests {
				if reqTime.After(oneMinuteAgo) {
					filtered = append(filtered, reqTime)
				}
			}
			if len(filtered) == 0 {
				delete(rl.requests, apiKeyID)
			} else {
				rl.requests[apiKeyID] = filtered
			}
		}
		rl.mu.Unlock()
	}
}
