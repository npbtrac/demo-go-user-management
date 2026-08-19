package config

import (
	"fmt"
	"os"
	"strings"
)

// Config holds process listen addresses and the Postgres DSN.
type Config struct {
	DatabaseURL      string
	PublicHTTPAddr   string
	InternalHTTPAddr string
	GRPCAddr         string
}

// FromEnv loads config from environment variables with local-dev defaults.
func FromEnv() (Config, error) {
	cfg := Config{
		DatabaseURL:      strings.TrimSpace(os.Getenv("DATABASE_URL")),
		PublicHTTPAddr:   envOr("PUBLIC_HTTP_ADDR", ":10100"),
		InternalHTTPAddr: envOr("INTERNAL_HTTP_ADDR", ":10101"),
		GRPCAddr:         envOr("GRPC_ADDR", ":10102"),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	return cfg, nil
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
