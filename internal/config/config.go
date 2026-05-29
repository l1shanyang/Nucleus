package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	App      AppConfig
	HTTP     HTTPConfig
	Database DatabaseConfig
	Log      LogConfig
}

type AppConfig struct {
	Env string // local, test, production
}

func (a AppConfig) IsLocal() bool      { return a.Env == "local" }
func (a AppConfig) IsTest() bool       { return a.Env == "test" }
func (a AppConfig) IsProduction() bool { return a.Env == "production" }

type HTTPConfig struct {
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	CORSOrigins     []string
}

type DatabaseConfig struct {
	URL          string
	MaxOpenConns int
	MinConns     int
	MaxIdleTime  time.Duration
}

type LogConfig struct {
	Level string // debug, info, warn, error
}

func Load() (Config, error) {
	cfg := Config{
		App: AppConfig{
			Env: getenv("APP_ENV", "local"),
		},
		HTTP: HTTPConfig{
			Port:            getenv("HTTP_PORT", "8080"),
			ReadTimeout:     getDuration("HTTP_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:    getDuration("HTTP_WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:     getDuration("HTTP_IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout: getDuration("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second),
			CORSOrigins:     getCSV("CORS_ALLOWED_ORIGINS", []string{"*"}),
		},
		Database: DatabaseConfig{
			URL:          os.Getenv("DATABASE_URL"),
			MaxOpenConns: getInt("DB_MAX_OPEN_CONNS", 10),
			MinConns:     getInt("DB_MIN_CONNS", 1),
			MaxIdleTime:  getDuration("DB_MAX_IDLE_TIME", 15*time.Minute),
		},
		Log: LogConfig{
			Level: getenv("LOG_LEVEL", "info"),
		},
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) validate() error {
	if c.Database.URL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	if c.App.Env != "local" && c.App.Env != "test" && c.App.Env != "production" {
		return fmt.Errorf("APP_ENV must be one of: local, test, production (got %q)", c.App.Env)
	}

	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[c.Log.Level] {
		return fmt.Errorf("LOG_LEVEL must be one of: debug, info, warn, error (got %q)", c.Log.Level)
	}

	port, err := strconv.Atoi(c.HTTP.Port)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("HTTP_PORT must be a valid port number (got %q)", c.HTTP.Port)
	}

	return nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

func getCSV(key string, fallback []string) []string {
	v := os.Getenv(key)
	if strings.TrimSpace(v) == "" {
		return fallback
	}

	parts := strings.Split(v, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			values = append(values, part)
		}
	}
	if len(values) == 0 {
		return fallback
	}
	return values
}
