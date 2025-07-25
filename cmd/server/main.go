package main

import (
	"PeopleCRUD/internal/api/routes"
	"PeopleCRUD/internal/cache"
	"PeopleCRUD/internal/config"
	"PeopleCRUD/internal/database"
	"PeopleCRUD/internal/repository"
	"PeopleCRUD/internal/service"
	"PeopleCRUD/internal/utils"
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	logger := utils.InitLogger()

	cfg := config.Load()

	db, err := database.Connect(cfg.DatabaseURL())
	if err != nil {
		logger.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	cacheInst := cache.NewMemoryCache()

	// Инициализация слоев
	personRepo := repository.NewPersonRepository(db)
	personService := service.NewPersonService(personRepo, cacheInst, logger)

	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	routes.SetupRoutes(router, personService, logger)

	server := &http.Server{
		Addr:           ":" + cfg.Server.Port,
		Handler:        router,
		ReadTimeout:    time.Second * 10,
		WriteTimeout:   time.Second * 10,
		IdleTimeout:    time.Second * 60,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		logger.Info("Server starting on port ", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server startup failed:", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Server shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown:", err)
	}

	logger.Info("Server exited")
}
