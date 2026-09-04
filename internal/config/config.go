package config

import (
	"fmt"
	"os"
)

type Config struct {
	Addr   string
	DBPath string
	Env    string
}

func (c Config) IsProd() bool {
	return c.Env == "prod"
}

func Load() (Config, error) {
	cfg := Config{
		Addr:   getenv("MANTENIMIENTO_ADDR", ":8080"),
		DBPath: getenv("MANTENIMIENTO_DB", "mantenimiento.db"),
		Env:    getenv("MANTENIMIENTO_ENV", "dev"),
	}

	if cfg.Env != "dev" && cfg.Env != "prod" {
		return Config{}, fmt.Errorf("MANTENIMIENTO_ENV must be dev or prod, got %q", cfg.Env)
	}

	return cfg, nil
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
