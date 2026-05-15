// Package config loads application configuration from environment variables
// with production-safe defaults. Never hard-code secrets — use env vars.
package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all runtime configuration for the auth service.
type Config struct {
	Host string
	Port string

	// JWT signing secret — MUST be overridden in production.
	JWTSecret          string
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration

	// DataDir is the directory for persistent storage files.
	DataDir string

	// Security knobs.
	MaxLoginAttempts    int
	LockoutBaseDuration time.Duration
	RateLimitRequests   int
	RateLimitWindow     time.Duration
}

// Load reads configuration from environment variables with safe defaults.
func Load() *Config {
	return &Config{
		Host:                getEnv("HOST", "0.0.0.0"),
		Port:                getEnv("PORT", "8080"),
		JWTSecret:           getEnv("JWT_SECRET", "CHANGE_ME_use_32+_random_bytes_in_prod"),
		AccessTokenExpiry:   getDuration("ACCESS_TOKEN_EXPIRY", 15*time.Minute),
		RefreshTokenExpiry:  getDuration("REFRESH_TOKEN_EXPIRY", 7*24*time.Hour),
		DataDir:             getEnv("DATA_DIR", "./data"),
		MaxLoginAttempts:    getInt("MAX_LOGIN_ATTEMPTS", 5),
		LockoutBaseDuration: getDuration("LOCKOUT_BASE_DURATION", 30*time.Second),
		RateLimitRequests:   getInt("RATE_LIMIT_REQUESTS", 20),
		RateLimitWindow:     getDuration("RATE_LIMIT_WINDOW", time.Minute),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

func getDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
