package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	AppEnv           string
	Port             int
	CORSOrigins      []string
	DatabaseURL      string
	AuthURL          string
	AuthCacheTTL     time.Duration
	ReservationSweep time.Duration
}

func Load() (Config, error) {
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")
	viper.AutomaticEnv()
	viper.SetDefault("APP_ENV", "development")
	viper.SetDefault("PORT", 3003)
	viper.SetDefault("CORS_ORIGIN", "*")
	viper.SetDefault("AUTH_CACHE_TTL_SECONDS", 30)
	viper.SetDefault("RESERVATION_SWEEP_INTERVAL_SECONDS", 30)
	_ = viper.ReadInConfig()

	databaseURL := strings.TrimSpace(viper.GetString("DATABASE_URL"))
	authURL := strings.TrimRight(strings.TrimSpace(viper.GetString("AUTH_SERVICE_URL")), "/")
	if databaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if authURL == "" {
		return Config{}, fmt.Errorf("AUTH_SERVICE_URL is required")
	}

	return Config{
		AppEnv:           viper.GetString("APP_ENV"),
		Port:             viper.GetInt("PORT"),
		CORSOrigins:      splitCSV(viper.GetString("CORS_ORIGIN")),
		DatabaseURL:      databaseURL,
		AuthURL:          authURL,
		AuthCacheTTL:     time.Duration(viper.GetInt("AUTH_CACHE_TTL_SECONDS")) * time.Second,
		ReservationSweep: time.Duration(viper.GetInt("RESERVATION_SWEEP_INTERVAL_SECONDS")) * time.Second,
	}, nil
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "*" {
		return []string{"*"}
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
