package builder

import (
	"github.com/Egham-7/adaptive-proxy/internal/models"
	"github.com/Egham-7/adaptive-proxy/internal/services/usage"
)

func (b *Builder) WithDatabase(cfg models.DatabaseConfig) *Builder {
	b.cfg.Database = &cfg
	return b
}

func (b *Builder) WithAPIKeyManagement(cfg models.APIKeyConfig) *Builder {
	if len(cfg.HeaderNames) == 0 {
		cfg.HeaderNames = []string{"X-API-Key"}
	}
	b.cfg.Server.APIKeyConfig = &cfg
	return b
}

func (b *Builder) EnableAPIKeyAuth() *Builder {
	cfg := usage.DefaultAPIKeyConfig()
	cfg.Enabled = true
	b.cfg.Server.APIKeyConfig = &cfg
	return b
}
