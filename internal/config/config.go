package config

import (
	"fmt"
	"log"
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
	// Получаем порт с обработкой ошибки
	port, err := strconv.Atoi(getEnv("DB_PORT", "5432"))
	if err != nil {
		log.Printf("Invalid DB_PORT, using default 5432. Error: %v", err)
		port = 5432
	}

	// Конфигурация по умолчанию для Docker
	dbConfig := DatabaseConfig{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     port,
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", "5558465Ab"), // Используем ваш пароль по умолчанию
		DBName:   getEnv("DB_NAME", "people_crud"),
		SSLMode:  getEnv("DB_SSL_MODE", "disable"),
	}

	// Если есть DATABASE_URL (для Heroku/Back4App), переопределяем конфигурацию
	if databaseURL := os.Getenv("DATABASE_URL"); databaseURL != "" {
		dbConfig.URL = databaseURL
	} else {
		// Формируем DSN строку из отдельных параметров
		dbConfig.URL = fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
			dbConfig.User,
			dbConfig.Password,
			dbConfig.Host,
			dbConfig.Port,
			dbConfig.DBName,
			dbConfig.SSLMode)
	}

	return &Config{
		Database: dbConfig,
		Server: ServerConfig{
			Port: getEnv("PORT", getEnv("SERVER_PORT", "8080")),
		},
		Environment: getEnv("ENVIRONMENT", "development"),
	}
}

func (c *Config) DatabaseURL() string {
	// Если DATABASE_URL явно задан (например, вручную в настройках Koyeb), используем его
	if c.Database.URL != "" {
		return c.Database.URL
	}

	// Собираем URL из переменных Koyeb
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=require",
		c.Database.User,
		c.Database.Password,
		c.Database.Host,
		c.Database.Port,
		c.Database.DBName,
	)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
