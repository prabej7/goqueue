package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port string
	Env  string
	DatabaseConfig
}

type DatabaseConfig struct {
	DatabaseURL           string
	MaxOpenConnection     int
	MaxIdleConnections    int
	MaxConnectionIdleTime string
}

func Load() *Config {
	return &Config{
		Port: getEnv("PORT", "8080"),
		Env:  getEnv("ENV", "development"),
		DatabaseConfig: DatabaseConfig{
			DatabaseURL:           getEnv("DATABASE_URL", ""),
			MaxIdleConnections:    getEnvInt("DB_MAX_IDLE_CONNECTIONS", 20),
			MaxConnectionIdleTime: getEnv("DB_MAX_CONNECTION_IDLE_TIME", "15m"),
			MaxOpenConnection:     getEnvInt("DB_MAX_OPEN_CONNECTION", 10),
		},
	}
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
