package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Database    DatabaseConfig
	Server      ServerConfig
	Environment string
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
	URL      string // Добавляем поддержку DATABASE_URL
}

type ServerConfig struct {
	Port string
}

func Load() *Config {
	// Приоритет DATABASE_URL (для Back4App)
	databaseURL := os.Getenv("DATABASE_URL")

	port, _ := strconv.Atoi(getEnv("DB_PORT", "5432"))

	return &Config{
		Database: DatabaseConfig{
			URL:      databaseURL,
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     port,
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "password"),
			DBName:   getEnv("DB_NAME", "people_crud"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
		},
		Server: ServerConfig{
			Port: getEnv("PORT", getEnv("SERVER_PORT", "8080")), // Back4App использует PORT
		},
		Environment: getEnv("ENVIRONMENT", "development"),
	}
}

func (c *Config) DatabaseURL() string {
	// Если есть DATABASE_URL, используем его (приоритет для Back4App)
	if c.Database.URL != "" {
		return c.Database.URL
	}

	// Иначе собираем из частей
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.Database.User,
		c.Database.Password,
		c.Database.Host,
		c.Database.Port,
		c.Database.DBName,
		c.Database.SSLMode,
	)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
