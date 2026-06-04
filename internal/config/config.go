package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

var environment Environment

func Load() error {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("load .env: %w", err)
	}

	environment = Environment{
		Port:             envInt("PORT", 8000),
		MaxRetries:       envInt("MAX_RETRIES", 3),
		RetryBaseBackoff: envDuration("RETRY_BASE_BACKOFF", time.Second),
		BatchConcurrent:  envBool("BATCH_CONCURRENT", false),
		CacheTTL:         envDuration("CACHE_TTL_SECONDS", 3600*time.Second),
		Redis: RedisConfig{
			Host:     envString("REDIS_HOST", "localhost"),
			Port:     envInt("REDIS_PORT", 6379),
			Password: envString("REDIS_PASSWORD", ""),
			DB:       envInt("REDIS_INDEX", 0),
		},
	}

	return nil
}

func Get() Environment {
	return environment
}

func envString(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func envInt(key string, defaultValue int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultValue
	}
	return n
}

func envBool(key string, defaultValue bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return defaultValue
	}
	return b
}

func envDuration(key string, defaultValue time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		if seconds, err := strconv.Atoi(v); err == nil {
			return time.Duration(seconds) * time.Second
		}
		return defaultValue
	}
	return d
}
