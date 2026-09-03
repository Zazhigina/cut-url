package config

import (
	"flag"
	"fmt"
	"os"
)

type Config struct {
	StorageType string
	Port        string
	// PostgreSQL
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
}

func Load() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.StorageType, "storage", getEnv("STORAGE_TYPE", "memory"), "Storage type: memory or postgres")
	flag.StringVar(&cfg.Port, "port", getEnv("PORT", "8080"), "Server port")
	flag.Parse()

	cfg.DBHost = getEnv("DB_HOST", "localhost")
	cfg.DBPort = getEnv("DB_PORT", "5432")
	cfg.DBUser = getEnv("DB_USER", "postgres")
	cfg.DBPassword = getEnv("DB_PASSWORD", "postgres")
	cfg.DBName = getEnv("DB_NAME", "cut_url")
	cfg.DBSSLMode = getEnv("DB_SSLMODE", "disable")

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (c *Config) GetDBConnString() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode,
	)
}
