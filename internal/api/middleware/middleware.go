package middleware

import (
	"PeopleCRUD/pkg/errors"
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Logger middleware для логирования HTTP запросов
func Logger(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		// Process request
		c.Next()

		// Log request
		duration := time.Since(startTime)

		entry := logger.WithFields(logrus.Fields{
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"status":     c.Writer.Status(),
			"duration":   duration,
			"client_ip":  c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
		})

		if c.Writer.Status() >= 500 {
			entry.Error("HTTP request completed with server error")
		} else if c.Writer.Status() >= 400 {
			entry.Warn("HTTP request completed with client error")
		} else {
			entry.Info("HTTP request completed")
		}
	}
}

// Recovery middleware для обработки паники
func Recovery(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.WithFields(logrus.Fields{
					"error": err,
					"path":  c.Request.URL.Path,
				}).Error("Panic recovered")

				c.JSON(500, errors.NewInternalServerError("Internal server error"))
				c.Abort()
			}
		}()

		c.Next()
	}
}

// CORS middleware для кросс-доменных запросов
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// Timeout middleware для ограничения времени выполнения запросов
func Timeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Устанавливаем таймаут для контекста
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// RateLimiter middleware для ограничения количества запросов
func RateLimiter() gin.HandlerFunc {
	// Простая реализация с использованием in-memory хранилища
	// В продакшене лучше использовать Redis или подобное
	return func(c *gin.Context) {
		// Здесь можно добавить логику rate limiting
		c.Next()
	}
}
