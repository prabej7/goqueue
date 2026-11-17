package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port string
	Env  string
	DatabaseConfig
	WorkerConfig
}

type DatabaseConfig struct {
	DatabaseURL           string
	MaxOpenConnections    int
	MaxIdleConnections    int
	MaxConnectionIdleTime time.Duration
}

type WorkerConfig struct {
	LeaseDuration     time.Duration
	HeartbeatInterval time.Duration
	RenewTimeout      time.Duration
	JobTimeout        time.Duration
}

func Load() (*Config, error) {
	cfg := &Config{
		Port: getEnv("PORT", "8080"),
		Env:  getEnv("ENV", "development"),
		DatabaseConfig: DatabaseConfig{
			DatabaseURL:           getEnv("DATABASE_URL", ""),
			MaxIdleConnections:    getEnvInt("DB_MAX_IDLE_CONNECTIONS", 20),
			MaxConnectionIdleTime: getEnvDuration("DB_MAX_CONNECTION_IDLE_TIME", time.Minute*15),
			MaxOpenConnections:    getEnvInt("DB_MAX_OPEN_CONNECTIONS", 10),
		},
		WorkerConfig: WorkerConfig{
			LeaseDuration:     getEnvDuration("LEASE_DURATION", 5*time.Minute),
			HeartbeatInterval: getEnvDuration("HEARTBEAT_INTERVAL", 30*time.Second),
			RenewTimeout:      getEnvDuration("RENEWAL_TIMEOUT", 5*time.Second),
			JobTimeout:        getEnvDuration("JOB_TIMEOUT", 2*time.Minute),
		},
	}

	err := cfg.Validate()

	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)

	if value == "" {
		return fallback
	}

	return value
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)

	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)

	if err != nil {
		return fallback
	}

	return parsed
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	duration := os.Getenv(key)

	if duration == "" {
		return fallback
	}

	parsedDuration, err := time.ParseDuration(duration)

	if err != nil {
		return fallback
	}

	return parsedDuration
}
