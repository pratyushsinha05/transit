package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	// Database settings
	DBHost     string
	DBPort     string
	DBName     string
	DBUser     string
	DBPassword string
	DBMaxConns int
	DBMinConns int

	// Redis settings
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisPoolSize int

	// Server settings
	ServerPort string
	ServerHost string

	// H3 Geospatial settings
	H3Resolution int // Default: 9 (~175m edge length)

	// Environment
	Env      string
	LogLevel string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	// Load .env file if it exists, but don't fail if it doesn't (could be env vars)
	_ = godotenv.Load()

	cfg := &Config{
		// Database
		DBHost:     getEnvOrDefault("DB_HOST", "localhost"),
		DBPort:     getEnvOrDefault("DB_PORT", "5432"),
		DBName:     os.Getenv("DB_NAME"),
		DBUser:     os.Getenv("DB_USER"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBMaxConns: getEnvAsInt("DB_MAX_CONNS", 50),
		DBMinConns: getEnvAsInt("DB_MIN_CONNS", 10),

		// Redis
		RedisHost:     getEnvOrDefault("REDIS_HOST", "localhost"),
		RedisPort:     getEnvOrDefault("REDIS_PORT", "6379"),
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
		RedisPoolSize: getEnvAsInt("REDIS_POOL_SIZE", 50),

		// Server
		ServerPort: getEnvOrDefault("SERVER_PORT", "8080"),
		ServerHost: getEnvOrDefault("SERVER_HOST", "0.0.0.0"),

		// H3 Geospatial
		H3Resolution: getEnvAsInt("H3_RESOLUTION", 9),

		// Environment
		Env:      getEnvOrDefault("ENV", "development"),
		LogLevel: getEnvOrDefault("LOG_LEVEL", "debug"),
	}

	// Validate required fields
	if cfg.DBName == "" {
		return nil, fmt.Errorf("DB_NAME environment variable is required")
	}
	if cfg.DBUser == "" {
		return nil, fmt.Errorf("DB_USER environment variable is required")
	}

	// Validate H3 resolution (0-15 are valid)
	if cfg.H3Resolution < 0 || cfg.H3Resolution > 15 {
		return nil, fmt.Errorf("H3_RESOLUTION must be between 0 and 15, got %d", cfg.H3Resolution)
	}

	return cfg, nil
}

// getEnvOrDefault returns the environment variable value or a default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt returns the environment variable as an integer or a default
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
