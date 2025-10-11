package config

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"time"

	"github.com/Egham-7/adaptive-proxy/internal/api"
	geminiapi "github.com/Egham-7/adaptive-proxy/internal/api/gemini"
	"github.com/Egham-7/adaptive-proxy/internal/config"
	"github.com/Egham-7/adaptive-proxy/internal/models"
	"github.com/Egham-7/adaptive-proxy/internal/services/apikey"
	"github.com/Egham-7/adaptive-proxy/internal/services/budget"
	"github.com/Egham-7/adaptive-proxy/internal/services/circuitbreaker"
	"github.com/Egham-7/adaptive-proxy/internal/services/database"
	apikeyMiddleware "github.com/Egham-7/adaptive-proxy/internal/services/middleware"
	"github.com/Egham-7/adaptive-proxy/internal/services/model_router"
	"github.com/Egham-7/adaptive-proxy/internal/services/openai/chat/completions"
	"github.com/Egham-7/adaptive-proxy/internal/services/select_model"

	"github.com/gofiber/fiber/v2"
	fiberlog "github.com/gofiber/fiber/v2/log"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/pprof"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/timeout"
	"github.com/redis/go-redis/v9"
)

// Proxy represents an AdaptiveProxy server instance.
type Proxy struct {
	config           *config.Config
	app              *fiber.App
	redis            *redis.Client
	db               *database.DB
	builder          *Builder
	enabledEndpoints map[string]bool
}

// NewProxy creates a new Proxy instance with the given configuration.
// The cfg parameter is required and must not be nil.
// For full middleware and endpoint control, use NewProxyWithBuilder.
func NewProxy(cfg *config.Config) *Proxy {
	if cfg == nil {
		panic("config cannot be nil - use config.LoadFromFile() or config builder to create config")
	}

	return &Proxy{
		config:           cfg,
		enabledEndpoints: make(map[string]bool),
	}
}

// NewProxyWithBuilder creates a new Proxy instance with a configuration builder.
// This allows full control over middlewares and endpoint configuration.
func NewProxyWithBuilder(builder *Builder) *Proxy {
	return &Proxy{
		config:           builder.Build(),
		builder:          builder,
		enabledEndpoints: builder.GetEnabledEndpoints(),
	}
}

// Run starts the proxy server and blocks until shutdown.
func (p *Proxy) Run() error {
	// Validate configuration
	if err := p.config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Set log level
	setupLogLevel(p.config)

	port := p.config.Server.Port
	if port == "" {
		port = "8080"
	}
	listenAddr := ":" + port

	// Create Fiber app
	p.app = createFiberApp(p.config)

	// Setup middleware
	setupMiddleware(p.app, p.config, p)

	// Create Redis client (optional)
	var err error
	p.redis, err = createRedisClient(p.config)
	if err != nil {
		return fmt.Errorf("failed to create Redis client: %w", err)
	}
	if p.redis != nil {
		defer func() {
			if err := p.redis.Close(); err != nil {
				fiberlog.Errorf("Failed to close Redis client: %v", err)
			}
		}()
		fiberlog.Info("Redis client initialized successfully")
	} else {
		fiberlog.Info("Redis not configured - caching disabled")
	}

	// Create Database client (optional)
	if p.config.Database != nil {
		p.db, err = database.New(*p.config.Database)
		if err != nil {
			return fmt.Errorf("failed to create database connection: %w", err)
		}
		defer func() {
			if err := p.db.Close(); err != nil {
				fiberlog.Errorf("Failed to close database connection: %v", err)
			}
		}()
		fiberlog.Infof("Database (%s) initialized successfully", p.db.DriverName())

		if err := runDatabaseMigrations(p.db); err != nil {
			return fmt.Errorf("failed to run database migrations: %w", err)
		}
		fiberlog.Info("Database migrations completed successfully")
	} else {
		fiberlog.Info("Database not configured")
	}

	// Setup routes
	if err := setupRoutes(p.app, p.config, p.redis, p.db, p.enabledEndpoints); err != nil {
		return fmt.Errorf("failed to setup routes: %w", err)
	}

	// Welcome endpoint
	p.app.Get("/", welcomeHandler())

	// Print startup info
	fmt.Printf("🚀 AdaptiveProxy starting on %s\n", listenAddr)
	fmt.Printf("   Environment: %s\n", p.config.Server.Environment)
	fmt.Printf("   Go version: %s\n", runtime.Version())
	fmt.Printf("   GOMAXPROCS: %d\n", runtime.GOMAXPROCS(0))

	// Graceful shutdown handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	serverErrChan := make(chan error, 1)
	go func() {
		if err := p.app.Listen(listenAddr); err != nil {
			serverErrChan <- err
		}
	}()

	// Wait for shutdown signal
	select {
	case sig := <-sigChan:
		fiberlog.Infof("Received signal: %v. Starting graceful shutdown...", sig)
	case err := <-serverErrChan:
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
		fiberlog.Info("Context cancelled, starting shutdown...")
	}

	// Graceful shutdown
	fiberlog.Info("Server shutting down gracefully...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	shutdownErrChan := make(chan error, 1)
	go func() {
		shutdownErrChan <- p.app.ShutdownWithTimeout(30 * time.Second)
	}()

	select {
	case err := <-shutdownErrChan:
		if err != nil {
			return fmt.Errorf("shutdown error: %w", err)
		}
		fiberlog.Info("Server shutdown completed successfully")
	case <-shutdownCtx.Done():
		return fmt.Errorf("shutdown timeout exceeded")
	}

	return nil
}

func createFiberApp(cfg *config.Config) *fiber.App {
	isProd := cfg.IsProduction()

	return fiber.New(fiber.Config{
		AppName:              "AdaptiveProxy v1.0",
		EnablePrintRoutes:    !isProd,
		ReadTimeout:          2 * time.Minute,
		WriteTimeout:         2 * time.Minute,
		IdleTimeout:          5 * time.Minute,
		ReadBufferSize:       8192,
		WriteBufferSize:      8192,
		CompressedFileSuffix: ".gz",
		Prefork:              false,
		CaseSensitive:        true,
		StrictRouting:        false,
		Network:              "tcp",
		ServerHeader:         "AdaptiveProxy",
		ErrorHandler:         createErrorHandler(isProd),
	})
}

func createErrorHandler(isProd bool) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		// Sanitize error for external consumption
		sanitized := models.SanitizeError(err)
		statusCode := sanitized.GetStatusCode()

		// Log internal error details (but don't expose them)
		if isProd {
			fiberlog.Errorf("Request error: path=%s, type=%s, retryable=%v",
				c.Path(), sanitized.Type, sanitized.Retryable)
		} else {
			fiberlog.Errorf("Request error: %v (status: %d, path: %s)", err, statusCode, c.Path())
		}

		// Return sanitized error response
		response := fiber.Map{
			"error": sanitized.Message,
			"type":  sanitized.Type,
			"code":  statusCode,
		}

		// Add retry info for retryable errors
		if sanitized.Retryable {
			response["retryable"] = true
			if sanitized.Type == models.ErrorTypeRateLimit {
				response["retry_after"] = "60s"
			}
		}

		// Add error code if available
		if sanitized.Code != "" {
			response["error_code"] = sanitized.Code
		}

		return c.Status(statusCode).JSON(response)
	}
}

func setupMiddleware(app *fiber.App, cfg *config.Config, p *Proxy) {
	isProd := cfg.IsProduction()
	allowedOrigins := cfg.Server.AllowedOrigins

	// Recover middleware (must be first)
	app.Use(recover.New(recover.Config{
		EnableStackTrace: !isProd,
	}))

	// Rate limiter (use builder config if available, otherwise use defaults)
	if p.builder != nil && p.builder.GetRateLimitConfig() != nil {
		rlCfg := p.builder.GetRateLimitConfig()
		keyFunc := rlCfg.KeyFunc
		if keyFunc == nil {
			keyFunc = func(c *fiber.Ctx) string {
				apiKey := c.Get("X-Stainless-API-Key")
				if apiKey != "" {
					return apiKey
				}
				return c.IP()
			}
		}
		app.Use(limiter.New(limiter.Config{
			Max:               rlCfg.Max,
			Expiration:        rlCfg.Expiration,
			LimiterMiddleware: limiter.SlidingWindow{},
			KeyGenerator:      keyFunc,
			LimitReached: func(c *fiber.Ctx) error {
				return models.NewRateLimitError(fmt.Sprintf("%d requests per %v", rlCfg.Max, rlCfg.Expiration))
			},
		}))
	} else {
		// Default rate limiter
		app.Use(limiter.New(limiter.Config{
			Max:               1000,
			Expiration:        1 * time.Minute,
			LimiterMiddleware: limiter.SlidingWindow{},
			KeyGenerator: func(c *fiber.Ctx) string {
				apiKey := c.Get("X-Stainless-API-Key")
				if apiKey != "" {
					return apiKey
				}
				return c.IP()
			},
			LimitReached: func(c *fiber.Ctx) error {
				return models.NewRateLimitError("1000 requests per minute")
			},
		}))
	}

	// Request timeout middleware (use builder config if available)
	if p.builder != nil && p.builder.GetTimeoutConfig() != nil {
		timeoutDuration := p.builder.GetTimeoutConfig().Timeout
		app.Use(func(c *fiber.Ctx) error {
			handler := func(c *fiber.Ctx) error {
				return c.Next()
			}
			return timeout.NewWithContext(handler, timeoutDuration)(c)
		})
	} else {
		// Default timeout middleware
		app.Use(func(c *fiber.Ctx) error {
			const (
				defaultTimeout = 30 * time.Second
				maxTimeout     = 2 * time.Minute
			)

			timeout := defaultTimeout
			if customTimeout := c.Get("X-Request-Timeout"); customTimeout != "" {
				if d, err := time.ParseDuration(customTimeout); err == nil && d > 0 {
					timeout = min(d, maxTimeout)
				}
			}

			ctx, cancel := context.WithTimeout(c.UserContext(), timeout)
			defer cancel()
			c.SetUserContext(ctx)

			return c.Next()
		})
	}

	// Compression
	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))

	// Logging
	if isProd {
		app.Use(logger.New(logger.Config{
			Format: "${time} ${status} ${method} ${path} ${latency} ${bytesSent}b\n",
			Output: os.Stdout,
		}))
	} else {
		app.Use(logger.New(logger.Config{
			Format: "[${time}] ${status} - ${latency} ${method} ${path} ${error}\n",
			Output: os.Stdout,
		}))
	}

	// CORS
	allAllowedHeaders := []string{
		"Origin", "Content-Type", "Accept", "Authorization", "User-Agent",
		"X-Stainless-API-Key", "X-Stainless-Arch", "X-Stainless-OS",
		"X-Stainless-Runtime", "X-Stainless-Runtime-Version",
		"X-Stainless-Package-Version", "X-Stainless-Lang",
		"X-Stainless-Retry-Count", "X-Stainless-Read-Timeout",
		"X-Stainless-Async", "X-Stainless-Raw-Response",
		"X-Stainless-Helper-Method", "X-Stainless-Timeout",
	}

	app.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowHeaders:     strings.Join(allAllowedHeaders, ", "),
		AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS",
		AllowCredentials: true,
		MaxAge:           86400,
		ExposeHeaders:    "Content-Length, Content-Type, X-Request-ID",
	}))

	// Custom middlewares from builder
	if p.builder != nil {
		for _, middleware := range p.builder.GetMiddlewares() {
			app.Use(middleware)
		}
	}

	// Profiler (dev only)
	if !isProd {
		app.Use(pprof.New())
	}
}

func setupLogLevel(cfg *config.Config) {
	logLevel := cfg.GetNormalizedLogLevel()

	switch logLevel {
	case "trace":
		fiberlog.SetLevel(fiberlog.LevelTrace)
	case "debug":
		fiberlog.SetLevel(fiberlog.LevelDebug)
	case "info":
		fiberlog.SetLevel(fiberlog.LevelInfo)
	case "warn", "warning":
		fiberlog.SetLevel(fiberlog.LevelWarn)
	case "error":
		fiberlog.SetLevel(fiberlog.LevelError)
	case "fatal":
		fiberlog.SetLevel(fiberlog.LevelFatal)
	case "panic":
		fiberlog.SetLevel(fiberlog.LevelPanic)
	default:
		fiberlog.SetLevel(fiberlog.LevelInfo)
		fiberlog.Warnf("Unknown log level '%s', defaulting to 'info'", logLevel)
	}

	fiberlog.Infof("Log level set to: %s", logLevel)
}

func createRedisClient(cfg *config.Config) (*redis.Client, error) {
	redisURL := ""

	if cfg.ModelRouter != nil && cfg.ModelRouter.Cache.RedisURL != "" {
		redisURL = cfg.ModelRouter.Cache.RedisURL
	}

	if redisURL == "" {
		fiberlog.Info("Redis not configured - circuit breakers and semantic cache disabled")
		return nil, nil
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	opt.PoolSize = 50
	opt.MinIdleConns = 10
	opt.PoolTimeout = 4 * time.Second
	opt.ConnMaxIdleTime = 5 * time.Minute
	opt.ConnMaxLifetime = 30 * time.Minute
	opt.DialTimeout = 10 * time.Second
	opt.ReadTimeout = 3 * time.Second
	opt.WriteTimeout = 3 * time.Second
	opt.MaxRetries = 3
	opt.MinRetryBackoff = 8 * time.Millisecond
	opt.MaxRetryBackoff = 512 * time.Millisecond

	fiberlog.Debugf("Redis client configuration: PoolSize=%d, MinIdle=%d, MaxRetries=%d",
		opt.PoolSize, opt.MinIdleConns, opt.MaxRetries)

	client := redis.NewClient(opt)

	// Test connection
	return testRedisConnectionWithRetry(client)
}

func testRedisConnectionWithRetry(client *redis.Client) (*redis.Client, error) {
	const maxAttempts = 3
	const baseDelay = 1 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := client.Ping(ctx).Err()
		cancel()

		if err == nil {
			fiberlog.Infof("Redis connection established successfully (attempt %d/%d)", attempt, maxAttempts)
			stats := client.PoolStats()
			fiberlog.Debugf("Redis pool initialized: Hits=%d, Misses=%d, Timeouts=%d, TotalConns=%d, IdleConns=%d",
				stats.Hits, stats.Misses, stats.Timeouts, stats.TotalConns, stats.IdleConns)
			return client, nil
		}

		fiberlog.Warnf("Redis connection failed (attempt %d/%d): %v", attempt, maxAttempts, err)

		if attempt < maxAttempts {
			delay := time.Duration(attempt) * baseDelay
			fiberlog.Infof("Retrying Redis connection in %v...", delay)
			time.Sleep(delay)
		}
	}

	if err := client.Close(); err != nil {
		fiberlog.Errorf("Failed to close Redis client after connection failures: %v", err)
	}

	return nil, fmt.Errorf("failed to connect to Redis after %d attempts", maxAttempts)
}

func setupRoutes(app *fiber.App, cfg *config.Config, redisClient *redis.Client, db *database.DB, enabledEndpoints map[string]bool) error {
	// Create shared services
	reqSvc := completions.NewRequestService()

	// Create model router
	modelRouter, err := model_router.NewModelRouter(cfg, redisClient)
	if err != nil {
		return fmt.Errorf("model router initialization failed: %w", err)
	}

	// Create response service
	respSvc := completions.NewResponseService(modelRouter)

	// Create circuit breakers
	circuitBreakers := make(map[string]*circuitbreaker.CircuitBreaker)
	providerTypes := []string{"chat_completions", "messages", "generate", "count_tokens"}

	for _, serviceType := range providerTypes {
		for providerName := range cfg.GetProviders(serviceType) {
			if _, exists := circuitBreakers[providerName]; !exists {
				circuitBreakers[providerName] = circuitbreaker.NewForProvider(redisClient, providerName)
			}
		}
	}

	// Create completion service
	completionSvc := completions.NewCompletionService(cfg, respSvc, circuitBreakers)

	// Create select model services
	selectModelReqSvc := select_model.NewRequestService()
	selectModelSvc := select_model.NewService(modelRouter)
	selectModelRespSvc := select_model.NewResponseService()

	// Initialize API key services if database is available
	var apiKeyMiddleware *apikeyMiddleware.APIKeyMiddleware
	if db != nil && cfg.Server.APIKeyConfig != nil && cfg.Server.APIKeyConfig.Enabled {
		apiKeySvc := apikey.NewService(db.DB)
		budgetSvc := budget.NewService(db.DB)
		apiKeyMiddleware = apikeyMiddleware.NewAPIKeyMiddlewareWithBudget(apiKeySvc, budgetSvc, cfg.Server.APIKeyConfig)

		// Apply API key middleware globally if required
		if cfg.Server.APIKeyConfig.RequireForAll {
			app.Use(apiKeyMiddleware.RequireAPIKey())
		} else if !cfg.Server.APIKeyConfig.AllowAnonymous {
			app.Use(apiKeyMiddleware.Authenticate())
		}

		// Register admin API key routes
		apiKeyHandler := api.NewAPIKeyHandler(apiKeySvc, budgetSvc)
		apiKeyHandler.RegisterRoutes(app, "/admin/api-keys")
	}

	// Initialize handlers (only for enabled endpoints)
	var chatCompletionHandler *api.CompletionHandler
	var selectModelHandler *api.SelectModelHandler
	var messagesHandler *api.MessagesHandler
	var generateHandler *geminiapi.GenerateHandler
	var countTokensHandler *geminiapi.CountTokensHandler

	// Helper function to check if endpoint is enabled (if map is empty, enable all)
	isEnabled := func(endpoint string) bool {
		if len(enabledEndpoints) == 0 {
			return true // No restrictions, enable all
		}
		return enabledEndpoints[endpoint]
	}

	if isEnabled("chat_completions") {
		chatCompletionHandler = api.NewCompletionHandler(cfg, reqSvc, respSvc, completionSvc, modelRouter, circuitBreakers)
	}

	if isEnabled("select_model") {
		selectModelHandler = api.NewSelectModelHandler(cfg, selectModelReqSvc, selectModelSvc, selectModelRespSvc, circuitBreakers)
	}

	if isEnabled("messages") {
		messagesHandler = api.NewMessagesHandler(cfg, modelRouter, circuitBreakers)
	}

	if isEnabled("generate") {
		generateHandler = geminiapi.NewGenerateHandler(cfg, modelRouter, circuitBreakers)
	}

	if isEnabled("count_tokens") {
		countTokensHandler = geminiapi.NewCountTokensHandler(cfg, modelRouter, circuitBreakers)
	}

	healthHandler := api.NewHealthHandler(cfg, redisClient)

	// Health check endpoint (always enabled)
	app.Get("/health", healthHandler.HealthCheck)

	// v1 routes (only register enabled endpoints)
	v1Group := app.Group("/v1")

	if chatCompletionHandler != nil {
		v1Group.Post("/chat/completions", chatCompletionHandler.ChatCompletion)
	}

	if messagesHandler != nil {
		v1Group.Post("/messages", messagesHandler.Messages)
	}

	if selectModelHandler != nil {
		v1Group.Post("/select-model", selectModelHandler.SelectModel)
	}

	if generateHandler != nil {
		v1Group.Post("/generate", generateHandler.Generate)
		v1Group.Post("/generate/stream", generateHandler.StreamGenerate)

		// v1beta routes (Gemini SDK compatibility)
		v1betaGroup := app.Group("/v1beta")
		v1betaGroup.Post(`/models/:model\:generateContent`, generateHandler.Generate)
		v1betaGroup.Post(`/models/:model\:streamGenerateContent`, generateHandler.StreamGenerate)
	}

	if countTokensHandler != nil {
		// Add to v1beta if not already created
		v1betaGroup := app.Group("/v1beta")
		v1betaGroup.Post(`/models/:model\:countTokens`, countTokensHandler.CountTokens)
	}

	return nil
}

func welcomeHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message":    "Welcome to AdaptiveProxy!",
			"version":    "1.0.0",
			"go_version": runtime.Version(),
			"status":     "running",
			"endpoints": fiber.Map{
				"chat":         "/v1/chat/completions",
				"messages":     "/v1/messages",
				"select_model": "/v1/select-model",
				"generate":     "/v1/generate",
				"health":       "/health",
			},
		})
	}
}

func runDatabaseMigrations(db *database.DB) error {
	// AutoMigrate works for all databases now with PrepareStmt disabled for ClickHouse
	apiKeySvc := apikey.NewService(db.DB)
	if err := apiKeySvc.AutoMigrate(); err != nil {
		return fmt.Errorf("failed to migrate api_keys table: %w", err)
	}

	budgetSvc := budget.NewService(db.DB)
	if err := budgetSvc.AutoMigrate(); err != nil {
		return fmt.Errorf("failed to migrate api_key_usage table: %w", err)
	}

	return nil
}
