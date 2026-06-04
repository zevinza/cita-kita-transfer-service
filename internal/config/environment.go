package config

import "time"

type Environment struct {
	Port             int
	MaxRetries       int
	RetryBaseBackoff time.Duration
	BatchConcurrent  bool
	CacheTTL         time.Duration
	Redis            RedisConfig
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}
