package config

import (
	"fmt"
	"os"
)

type Config struct {
	AppEnv      string
	HTTPPort    string
	DatabaseURL string
}

func Load() (Config, error) {
	cfg := Config{
		AppEnv:      getenv("APP_ENV", "local"),
		HTTPPort:    getenv("HTTP_PORT", "8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, nil
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
