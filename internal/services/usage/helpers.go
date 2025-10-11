package usage

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"github.com/Egham-7/adaptive-proxy/internal/models"
)

func GenerateAPIKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return "apk_" + base64.URLEncoding.EncodeToString(b)[:43], nil
}

func HashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%x", hash)
}

func ExtractKeyPrefix(key string) string {
	if len(key) < 12 {
		return key
	}
	return key[:12]
}

func CalculateBudgetRemaining(budgetLimit, budgetUsed float64) float64 {
	if budgetLimit == 0 {
		return 0
	}
	remaining := budgetLimit - budgetUsed
	return remaining
}

func DefaultAPIKeyConfig() models.APIKeyConfig {
	return models.APIKeyConfig{
		Enabled:        false,
		HeaderNames:    []string{"X-API-Key", "X-Stainless-API-Key"},
		RequireForAll:  false,
		AllowAnonymous: true,
		CreditsEnabled: false,
	}
}
