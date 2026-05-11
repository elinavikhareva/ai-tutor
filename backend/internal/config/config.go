package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	GeminiAPIKey   string
	AppPassword    string
	JWTSecret      string
	Port           string
	MetricsAddr    string
	DatabaseURL    string
	DBMaxConns     int32
	AllowedOrigins []string
	LogFormat      string
	LogLevel       string
}

func Load() (*Config, error) {
	var missing []string
	get := func(key string) string {
		v := os.Getenv(key)
		if v == "" {
			missing = append(missing, key)
		}
		return v
	}
	cfg := &Config{
		GeminiAPIKey: get("GEMINI_API_KEY"),
		AppPassword:  get("APP_PASSWORD"),
		JWTSecret:    get("JWT_SECRET"),
		Port:         getOrDefault("PORT", "8080"),
		MetricsAddr:  getOrDefault("METRICS_ADDR", ":9090"),
		DatabaseURL:  get("DATABASE_URL"),
		LogFormat:    getOrDefault("LOG_FORMAT", "text"),
		LogLevel:     getOrDefault("LOG_LEVEL", "info"),
	}
	rawOrigins := get("ALLOWED_ORIGINS")
	if rawOrigins != "" {
		for _, o := range strings.Split(rawOrigins, ",") {
			o = strings.TrimSpace(o)
			if o != "" {
				cfg.AllowedOrigins = append(cfg.AllowedOrigins, o)
			}
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required env variables: %s", strings.Join(missing, ", "))
	}

	maxConns, err := strconv.ParseInt(getOrDefault("DB_MAX_CONNS", "10"), 10, 32)
	if err != nil || maxConns < 1 {
		return nil, errors.New("DB_MAX_CONNS must be a positive integer")
	}
	cfg.DBMaxConns = int32(maxConns)

	if len(cfg.AllowedOrigins) == 0 {
		return nil, errors.New("ALLOWED_ORIGINS must contain at least one origin")
	}
	return cfg, nil
}

func getOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
