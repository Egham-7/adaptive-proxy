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
	"github.com/Egham-7/adaptive-proxy/internal/services/admin"
	"github.com/Egham-7/adaptive-proxy/internal/services/auth"
	"github.com/Egham-7/adaptive-proxy/internal/services/circuitbreaker"
	"github.com/Egham-7/adaptive-proxy/internal/services/database"
	"github.com/Egham-7/adaptive-proxy/internal/services/middleware"
	"github.com/Egham-7/adaptive-proxy/internal/services/model_router"
	"github.com/Egham-7/adaptive-proxy/internal/services/openai/chat/completions"
	"github.com/Egham-7/adaptive-proxy/internal/services/organizations"
	"github.com/Egham-7/adaptive-proxy/internal/services/projects"
	"github.com/Egham-7/adaptive-proxy/internal/services/select_model"
	"github.com/Egham-7/adaptive-proxy/internal/services/usage"
	"github.com/Egham-7/adaptive-proxy/pkg/builder"

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
	builder          *builder.Builder
	enabledEndpoints map[string]bool
	usageTracker     *middleware.UsageTracker
}

type proxyServices struct {
	usageService   *usage.Service
	creditsService *usage.CreditsService
	apiKeyService  *usage.APIKeyService
	usageTracker   *middleware.UsageTracker
}

type proxyInfrastructure struct {
	redis *redis.Client
	db    *database.DB
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
func NewProxyWithBuilder(b *builder.Builder) *Proxy {
	return &Proxy{
		config:           b.Build(),
		builder:          b,
		enabledEndpoints: b.GetEnabledEndpoints(),
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

	// === Infrastructure Setup ===
	infra, err := initializeInfrastructure(p.config)
	if err != nil {
		return err
	}
	p.redis = infra.redis
	p.db = infra.db

	// Setup cleanup handlers
	if p.redis != nil {
		defer func() {
			if err := p.redis.Close(); err != nil {
				fiberlog.Errorf("Failed to close Redis client: %v", err)
			}
		}()
	}
	if p.db != nil {
		defer func() {
			if err := p.db.Close(); err != nil {
				fiberlog.Errorf("Failed to close database connection: %v", err)
			}
		}()
	}

	// === Services Initialization ===
	services := initializeServices(p.db, p.config)
	if services != nil {
		p.usageTracker = services.usageTracker
	}

	// === Middleware Setup ===
	setupMiddleware(p.app, p.config, p)

	// === Routes Setup ===
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
		// Use default Fiber error handler - simpler and more standard
	})
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
				if apiKey, ok := c.Locals("api_key_raw").(string); ok && apiKey != "" {
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
				return fmt.Errorf("%d requests per %v", rlCfg.Max, rlCfg.Expiration)
			},
		}))
	} else {
		// Default rate limiter
		app.Use(limiter.New(limiter.Config{
			Max:               1000,
			Expiration:        1 * time.Minute,
			LimiterMiddleware: limiter.SlidingWindow{},
			KeyGenerator: func(c *fiber.Ctx) string {
				if apiKey, ok := c.Locals("api_key_raw").(string); ok && apiKey != "" {
					return apiKey
				}
				return c.IP()
			},
			LimitReached: func(c *fiber.Ctx) error {
				return fmt.Errorf("1000 requests per minute")
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

	// Usage tracking middleware (if services are initialized)
	if p.usageTracker != nil {
		app.Use(p.usageTracker.EnforceUsageLimits())
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
	var usageSvc *usage.Service

	var creditsSvc *usage.CreditsService
	var stripeSvc *usage.StripeService
	var authMiddleware *middleware.AuthMiddleware
	if db != nil && cfg.APIKey != nil && cfg.APIKey.Enabled {
		apiKeySvc := usage.NewAPIKeyService(db.DB)

		creditsEnabled := cfg.Billing != nil

		if creditsEnabled {
			creditsSvc = usage.NewCreditsService(db.DB)

			if cfg.Billing.SecretKey != "" {
				stripeSvc = usage.NewStripeService(usage.StripeConfig{
					SecretKey:     cfg.Billing.SecretKey,
					WebhookSecret: cfg.Billing.WebhookSecret,
				}, creditsSvc)
			}
		}

		usageSvc = usage.NewService(db.DB, creditsSvc)

		var authProvider auth.AuthProvider
		var projectsSvc *projects.Service

		if cfg.Auth != nil {
			if cfg.Auth.ClerkConfig != nil && (cfg.Auth.ClerkConfig.SecretKey != "" || cfg.Auth.ClerkConfig.WebhookSecret != "") {
				authProvider = auth.NewClerkAuthProvider(cfg.Auth.ClerkConfig.SecretKey, db.DB)

				authMiddleware = middleware.NewAuthMiddleware(authProvider, apiKeySvc, usageSvc, &middleware.AuthMiddlewareConfig{
					Enabled:        true,
					AllowAnonymous: false,
					ClerkSecretKey: cfg.Auth.ClerkConfig.SecretKey,
					HeaderNames:    []string{"Authorization"},
					SkipPaths:      []string{"/health", "/webhooks"},
					EnableAPIKeys:  true,
				})

				app.Use("/admin/*", authMiddleware.RequireAuth())

				// Initialize projects service for webhook handler and routes
				projectsSvc = projects.NewService(db.DB, authProvider)

				if creditsSvc != nil {
					organizationsSvc := organizations.NewService(db.DB)
					clerkWebhookHandler := api.NewClerkWebhookHandler(cfg.Auth.ClerkConfig.WebhookSecret, creditsSvc, organizationsSvc, projectsSvc)
					app.Post("/webhooks/clerk", clerkWebhookHandler.HandleWebhook)

					orgGroup := app.Group("/admin/organizations")
					orgGroup.Delete("/:id", clerkWebhookHandler.DeleteOrganizationData)
				}

				if err := db.AutoMigrate(&models.Project{}, &models.ProjectMember{}); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to migrate project tables: %v\n", err)
				}
			} else if cfg.Auth.DatabaseConfig != nil {
				authProvider = auth.NewDatabaseAuthProvider(db.DB)

				authMiddleware = middleware.NewAuthMiddleware(authProvider, apiKeySvc, usageSvc, &middleware.AuthMiddlewareConfig{
					Enabled:        true,
					AllowAnonymous: false,
					HeaderNames:    []string{"Authorization"},
					SkipPaths:      []string{"/health", "/webhooks"},
					EnableAPIKeys:  true,
				})

				app.Use("/admin/*", authMiddleware.RequireAuth())

				if err := db.AutoMigrate(&models.User{}, &models.Organization{}, &models.OrganizationMember{}, &models.Project{}, &models.ProjectMember{}); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to migrate multi-tenancy tables: %v\n", err)
				}
			}

			// Initialize projects routes (projectsSvc already initialized above for webhook handler)
			if projectsSvc != nil {
				projectsHandler := api.NewProjectsHandler(projectsSvc)

				projectsGroup := app.Group("/admin/projects")
				projectsGroup.Post("/", projectsHandler.CreateProject)
				projectsGroup.Get("/:id", projectsHandler.GetProject)
				projectsGroup.Get("/organization/:org_id", projectsHandler.ListProjects)
				projectsGroup.Patch("/:id", projectsHandler.UpdateProject)
				projectsGroup.Delete("/:id", projectsHandler.DeleteProject)
				projectsGroup.Post("/:id/members", projectsHandler.AddMember)
				projectsGroup.Delete("/:id/members/:user_id", projectsHandler.RemoveMember)
				projectsGroup.Get("/:id/members", projectsHandler.ListMembers)
				projectsGroup.Patch("/:id/members/:user_id", projectsHandler.UpdateMemberRole)

				if cfg.Auth.DatabaseConfig != nil {
					adminSvc := admin.NewService(db.DB, authProvider)
					adminHandler := api.NewAdminHandler(adminSvc)

					orgGroup := app.Group("/admin/organizations")
					orgGroup.Post("/", adminHandler.CreateOrganization)
					orgGroup.Get("/:id", adminHandler.GetOrganization)
					orgGroup.Get("/", adminHandler.ListOrganizations)
					orgGroup.Patch("/:id", adminHandler.UpdateOrganization)
					orgGroup.Delete("/:id", adminHandler.DeleteOrganization)
					orgGroup.Post("/:id/members", adminHandler.AddOrganizationMember)
					orgGroup.Delete("/:id/members/:user_id", adminHandler.RemoveOrganizationMember)
					orgGroup.Get("/:id/members", adminHandler.ListOrganizationMembers)

					userGroup := app.Group("/admin/users")
					userGroup.Post("/", adminHandler.CreateUser)
					userGroup.Get("/:id", adminHandler.GetUser)
					userGroup.Patch("/:id", adminHandler.UpdateUser)
					userGroup.Delete("/:id", adminHandler.DeleteUser)
				}
			}
		}

		apiKeyHandler := api.NewAPIKeyHandler(apiKeySvc, usageSvc, creditsEnabled, authProvider)
		apiKeyHandler.RegisterRoutes(app, "/admin/api-keys")

		if creditsEnabled && creditsSvc != nil {
			creditsHandler := api.NewCreditsHandler(creditsSvc, authProvider)
			creditsGroup := app.Group("/admin/credits")
			creditsGroup.Get("/balance/:organization_id", creditsHandler.GetBalance)
			creditsGroup.Post("/check", creditsHandler.CheckCredits)
			creditsGroup.Get("/transactions/:organization_id", creditsHandler.GetTransactionHistory)

			if stripeSvc != nil {
				stripeHandler := api.NewStripeHandler(stripeSvc)
				app.Post("/webhooks/stripe", stripeHandler.HandleWebhook)

				stripeGroup := app.Group("/admin/stripe")
				stripeGroup.Post("/checkout-session", stripeHandler.CreateCheckoutSession)
			}
		}
	}

	completionSvc := completions.NewCompletionService(cfg, respSvc, circuitBreakers, usageSvc)

	// Create select model services
	selectModelReqSvc := select_model.NewRequestService()
	selectModelSvc := select_model.NewService(modelRouter)
	selectModelRespSvc := select_model.NewResponseService()

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
		messagesHandler = api.NewMessagesHandler(cfg, modelRouter, circuitBreakers, usageSvc)
	}

	if isEnabled("generate") {
		generateHandler = geminiapi.NewGenerateHandler(cfg, modelRouter, circuitBreakers, usageSvc)
	}

	if isEnabled("count_tokens") {
		countTokensHandler = geminiapi.NewCountTokensHandler(cfg, modelRouter, circuitBreakers)
	}

	healthHandler := api.NewHealthHandler(cfg, redisClient, db)

	// Health check endpoint (always enabled)
	app.Get("/health", healthHandler.HealthCheck)

	// v1 routes (only register enabled endpoints)
	v1Group := app.Group("/v1")
	if authMiddleware != nil {
		v1Group.Use(authMiddleware.RequireAuth())
	}

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

		if authMiddleware != nil {
			v1betaGroup.Use(authMiddleware.RequireAuth())
		}
		v1betaGroup.Post(`/models/:model\:generateContent`, generateHandler.Generate)
		v1betaGroup.Post(`/models/:model\:streamGenerateContent`, generateHandler.StreamGenerate)
	}

	if countTokensHandler != nil {
		// Add to v1beta if not already created
		v1betaGroup := app.Group("/v1beta")
		if authMiddleware != nil {
			v1betaGroup.Use(authMiddleware.RequireAuth())
		}
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
	apiKeySvc := usage.NewAPIKeyService(db.DB)
	if err := apiKeySvc.AutoMigrate(); err != nil {
		return fmt.Errorf("failed to migrate api_keys table: %w", err)
	}

	creditsSvc := usage.NewCreditsService(db.DB)
	if err := creditsSvc.AutoMigrate(); err != nil {
		return fmt.Errorf("failed to migrate credits tables: %w", err)
	}

	usageSvc := usage.NewService(db.DB, creditsSvc)
	if err := usageSvc.AutoMigrate(); err != nil {
		return fmt.Errorf("failed to migrate usage table: %w", err)
	}

	return nil
}

func initializeServices(db *database.DB, cfg *config.Config) *proxyServices {
	if db == nil || cfg.APIKey == nil || !cfg.APIKey.Enabled {
		return nil
	}

	apiKeySvc := usage.NewAPIKeyService(db.DB)

	var creditsSvc *usage.CreditsService
	creditsEnabled := cfg.Billing != nil
	if creditsEnabled {
		creditsSvc = usage.NewCreditsService(db.DB)
	}

	usageSvc := usage.NewService(db.DB, creditsSvc)

	var usageTracker *middleware.UsageTracker
	if usageSvc != nil {
		usageTracker = middleware.NewUsageTracker(usageSvc, creditsSvc, creditsEnabled)
	}

	return &proxyServices{
		usageService:   usageSvc,
		creditsService: creditsSvc,
		apiKeyService:  apiKeySvc,
		usageTracker:   usageTracker,
	}
}

func initializeInfrastructure(cfg *config.Config) (*proxyInfrastructure, error) {
	infra := &proxyInfrastructure{}

	redisClient, err := createRedisClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Redis client: %w", err)
	}
	infra.redis = redisClient

	if redisClient != nil {
		fiberlog.Info("Redis client initialized successfully")
	} else {
		fiberlog.Info("Redis not configured - caching disabled")
	}

	if cfg.Database != nil {
		db, err := database.New(*cfg.Database)
		if err != nil {
			return nil, fmt.Errorf("failed to create database connection: %w", err)
		}
		infra.db = db

		fiberlog.Infof("Database (%s) initialized successfully", db.DriverName())

		if err := runDatabaseMigrations(db); err != nil {
			return nil, fmt.Errorf("failed to run database migrations: %w", err)
		}
		fiberlog.Info("Database migrations completed successfully")
	} else {
		fiberlog.Info("Database not configured")
	}

	return infra, nil
}
