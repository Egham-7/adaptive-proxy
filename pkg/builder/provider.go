package builder

import "github.com/Egham-7/adaptive-proxy/internal/models"

type ProviderBuilder struct {
	apiKey         string
	baseURL        string
	authType       string
	authHeaderName string
	healthEndpoint string
	rateLimitRpm   *int
	timeoutMs      int
	headers        map[string]string
}

func NewProviderBuilder(apiKey string) *ProviderBuilder {
	return &ProviderBuilder{
		apiKey:  apiKey,
		headers: make(map[string]string),
	}
}

func (pb *ProviderBuilder) WithBaseURL(url string) *ProviderBuilder {
	pb.baseURL = url
	return pb
}

func (pb *ProviderBuilder) WithAuthType(authType string) *ProviderBuilder {
	pb.authType = authType
	return pb
}

func (pb *ProviderBuilder) WithAuthHeader(name string) *ProviderBuilder {
	pb.authHeaderName = name
	return pb
}

func (pb *ProviderBuilder) WithHealthEndpoint(endpoint string) *ProviderBuilder {
	pb.healthEndpoint = endpoint
	return pb
}

func (pb *ProviderBuilder) WithRateLimit(rpm int) *ProviderBuilder {
	pb.rateLimitRpm = &rpm
	return pb
}

func (pb *ProviderBuilder) WithTimeout(ms int) *ProviderBuilder {
	pb.timeoutMs = ms
	return pb
}

func (pb *ProviderBuilder) WithHeader(key, value string) *ProviderBuilder {
	pb.headers[key] = value
	return pb
}

func (pb *ProviderBuilder) Build() models.ProviderConfig {
	return models.ProviderConfig{
		APIKey:         pb.apiKey,
		BaseURL:        pb.baseURL,
		AuthType:       pb.authType,
		AuthHeaderName: pb.authHeaderName,
		HealthEndpoint: pb.healthEndpoint,
		RateLimitRpm:   pb.rateLimitRpm,
		TimeoutMs:      pb.timeoutMs,
		Headers:        pb.headers,
	}
}

func (b *Builder) AddOpenAICompatibleProvider(name string, cfg models.ProviderConfig) *Builder {
	b.cfg.Endpoints.ChatCompletions.Providers[name] = cfg
	b.enabledEndpoints["chat_completions"] = true
	return b
}

func (b *Builder) AddAnthropicCompatibleProvider(name string, cfg models.ProviderConfig) *Builder {
	b.cfg.Endpoints.Messages.Providers[name] = cfg
	b.enabledEndpoints["messages"] = true
	return b
}

func (b *Builder) AddGeminiCompatibleProvider(name string, cfg models.ProviderConfig) *Builder {
	b.cfg.Endpoints.Generate.Providers[name] = cfg
	b.cfg.Endpoints.CountTokens.Providers[name] = cfg
	b.enabledEndpoints["generate"] = true
	b.enabledEndpoints["count_tokens"] = true
	return b
}
