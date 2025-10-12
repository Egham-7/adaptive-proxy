package builder

import "github.com/Egham-7/adaptive-proxy/internal/models"

func (b *Builder) WithFallback(cfg models.FallbackConfig) *Builder {
	if cfg.Mode == "" {
		cfg.Mode = "race"
	}
	if cfg.TimeoutMs == 0 {
		cfg.TimeoutMs = 30000
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}

	b.cfg.Fallback = cfg
	return b
}

func (b *Builder) WithModelRouter(cfg models.ModelRouterConfig) *Builder {
	if cfg.CostBias == 0 {
		cfg.CostBias = 0.9
	}
	if cfg.Client.TimeoutMs == 0 {
		cfg.Client.TimeoutMs = 3000
	}
	if cfg.Cache.SemanticThreshold == 0 {
		cfg.Cache.SemanticThreshold = 0.95
	}
	if cfg.Client.CircuitBreaker.FailureThreshold == 0 {
		cfg.Client.CircuitBreaker.FailureThreshold = 3
	}
	if cfg.Client.CircuitBreaker.SuccessThreshold == 0 {
		cfg.Client.CircuitBreaker.SuccessThreshold = 2
	}
	if cfg.Client.CircuitBreaker.TimeoutMs == 0 {
		cfg.Client.CircuitBreaker.TimeoutMs = 5000
	}
	if cfg.Client.CircuitBreaker.ResetAfterMs == 0 {
		cfg.Client.CircuitBreaker.ResetAfterMs = 30000
	}

	b.cfg.ModelRouter = &cfg
	return b
}
