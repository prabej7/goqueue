package config

import (
	"errors"
)

func (c *Config) Validate() error {
	if c.DatabaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}

	if c.MaxOpenConnections <= 0 {
		return errors.New("DB_MAX_OPEN_CONNECTION must be greater than zero")
	}

	if c.MaxIdleConnections > c.MaxOpenConnections {
		return errors.New("DB_MAX_IDLE_CONNECTIONS cannot exceed DB_MAX_OPEN_CONNECTION")
	}

	if c.MaxConnectionIdleTime <= 0 {
		return errors.New("DB_MAX_CONNECTION_IDLE_TIME must be positive")
	}

	return nil
}
