package main

import (
	"log"

	"github.com/Egham-7/adaptive-proxy/internal/config"
	pkgconfig "github.com/Egham-7/adaptive-proxy/pkg/config"

	fiberlog "github.com/gofiber/fiber/v2/log"
)

func main() {
	// Load environment files explicitly
	envFiles := []string{".env.local", ".env.development", ".env"}
	config.LoadEnvFiles(envFiles)

	// Load configuration from YAML
	cfg, err := config.LoadFromFile("config.yaml")
	if err != nil {
		fiberlog.Fatalf("Failed to load config: %v", err)
	}

	// Create proxy with explicit config
	proxy := pkgconfig.NewProxy(cfg)

	// Start the server
	log.Println("Starting AdaptiveProxy server...")
	if err := proxy.Run(); err != nil {
		fiberlog.Fatalf("Server failed: %v", err)
	}
}
