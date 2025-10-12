package main

import (
	"log"

	"github.com/Egham-7/adaptive-proxy/pkg/builder"
	pkgconfig "github.com/Egham-7/adaptive-proxy/pkg/config"

	fiberlog "github.com/gofiber/fiber/v2/log"
)

func main() {
	b, err := builder.FromYAML("config.yaml", []string{".env.local"})
	if err != nil {
		fiberlog.Fatalf("Failed to load configuration: %v", err)
	}

	proxy := pkgconfig.NewProxyWithBuilder(b)

	log.Println("Starting AdaptiveProxy server...")
	if err := proxy.Run(); err != nil {
		fiberlog.Fatalf("Server failed: %v", err)
	}
}
