package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBName     string
	DBUser     string
	DBPassword string

	RedisHost string
	RedisPort string

	ServerPort string
	ServerHost string

	Env      string
	LogLevel string
}

func LoadConfig() (*Config, error) {
	// Load .env file if it exists, but don't fail if it doesn't (could be env vars)
	_ = godotenv.Load()

	cfg := &Config{
		DBHost:     os.Getenv("DB_HOST"),
		DBPort:     os.Getenv("DB_PORT"),
		DBName:     os.Getenv("DB_NAME"),
		DBUser:     os.Getenv("DB_USER"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		RedisHost:  os.Getenv("REDIS_HOST"),
		RedisPort:  os.Getenv("REDIS_PORT"),
		ServerPort: os.Getenv("SERVER_PORT"),
		ServerHost: os.Getenv("SERVER_HOST"),
		Env:        os.Getenv("ENV"),
		LogLevel:   os.Getenv("LOG_LEVEL"),
	}

	if cfg.DBHost == "" || cfg.DBPort == "" || cfg.DBName == "" || cfg.DBUser == "" {
		return nil, fmt.Errorf("missing required database environment variables")
	}

	if cfg.ServerPort == "" {
		cfg.ServerPort = "8080"
	}

	return cfg, nil
}
