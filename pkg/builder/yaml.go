package builder

import (
	"github.com/Egham-7/adaptive-proxy/internal/config"
	"github.com/gofiber/fiber/v2"
)

func FromYAML(path string, envFiles []string) (*Builder, error) {
	if len(envFiles) > 0 {
		config.LoadEnvFiles(envFiles)
	}

	cfg, err := config.LoadFromFile(path)
	if err != nil {
		return nil, err
	}

	return builderFromConfig(cfg), nil
}

func builderFromConfig(cfg *config.Config) *Builder {
	builder := &Builder{
		cfg:              cfg,
		middlewares:      []fiber.Handler{},
		enabledEndpoints: make(map[string]bool),
	}

	if len(cfg.Endpoints.ChatCompletions.Providers) > 0 {
		builder.enabledEndpoints["chat_completions"] = true
	}
	if len(cfg.Endpoints.Messages.Providers) > 0 {
		builder.enabledEndpoints["messages"] = true
	}
	if len(cfg.Endpoints.SelectModel.Providers) > 0 {
		builder.enabledEndpoints["select_model"] = true
	}
	if len(cfg.Endpoints.Generate.Providers) > 0 {
		builder.enabledEndpoints["generate"] = true
	}
	if len(cfg.Endpoints.CountTokens.Providers) > 0 {
		builder.enabledEndpoints["count_tokens"] = true
	}

	return builder
}
