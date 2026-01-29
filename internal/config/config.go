package config

import "os"

// Config holds all application configuration
type Config struct {
	DatabaseURL string
	Port        string
}

// Load reads configuration from environment variables
func Load() *Config {
	return &Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/splitwise?sslmode=disable"),
		Port:        getEnv("PORT", "8080"),
	}
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
