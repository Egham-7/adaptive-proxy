package main

import (
	"log"

	"github.com/Egham-7/adaptive-proxy/internal/models"
	"github.com/Egham-7/adaptive-proxy/pkg/config"
)

func main() {
	builder := config.New().
		Port("8080").
		Environment("development").
		LogLevel("info").
		AddOpenAICompatibleProvider("openai", config.NewProviderBuilder("sk-...").Build()).
		WithDatabase(models.DatabaseConfig{
			Type:         models.PostgreSQL,
			Host:         "localhost",
			Port:         5432,
			Username:     "user",
			Password:     "password",
			Database:     "adaptive_db",
			SSLMode:      "disable",
			MaxOpenConns: 25,
			MaxIdleConns: 5,
		})

	proxy := config.NewProxyWithBuilder(builder)

	if err := proxy.Run(); err != nil {
		log.Fatal(err)
	}
}
