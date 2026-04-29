package main

// @title Multi-Currency E-Wallet API
// @version 1.0
// @description This is a simplified multi-currency E-Wallet backend system implemented in Go using the Repository pattern, Gin framework, and PostgreSQL.
// @contact.name API Support
// @contact.url https://github.com/voca-test
// @host localhost:8080
// @BasePath /api/v1

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"voca-test/internal/db"
	"voca-test/internal/handler"
	"voca-test/internal/repository"
	"voca-test/internal/routes"
	"voca-test/internal/service"
	"voca-test/internal/util"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. Load configuration
	cfg := util.LoadConfig()

	// 2. Setup Logger
	util.SetupLogger(cfg.AppEnv)
	slog.Info("Starting application", "env", cfg.AppEnv, "port", cfg.Port)

	// 3. Database Connection
	database, err := db.ConnectPostgres(cfg)
	if err != nil {
		slog.Error("Could not connect to database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	// 4. Initialize Repositories
	userRepo := repository.NewUserRepository(database.Pool)
	walletRepo := repository.NewWalletRepository(database.Pool)
	ledgerRepo := repository.NewLedgerRepository(database.Pool)
	idempotencyRepo := repository.NewIdempotencyRepository(database.Pool)

	// 5. Initialize Services
	userService := service.NewUserService(userRepo)
	walletService := service.NewWalletService(
		database.Pool,
		userRepo,
		walletRepo,
		ledgerRepo,
		idempotencyRepo,
	)
	ledgerService := service.NewLedgerService(walletRepo, ledgerRepo)

	// 6. Initialize Handlers
	userHandler := handler.NewUserHandler(userService)
	walletHandler := handler.NewWalletHandler(walletService)
	ledgerHandler := handler.NewLedgerHandler(ledgerService)
	workerService := service.NewWorkerService(idempotencyRepo)

	// 7. Start Background Workers
	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()
	go workerService.StartCleanupWorker(
		workerCtx,
		time.Duration(cfg.IdempotencyInterval)*time.Hour,
		time.Duration(cfg.IdempotencyRetention)*time.Hour,
	)

	// 8. Setup Router & Routes
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Use gin.New() to have full control over middlewares
	r := gin.New()
	routes.SetupRoutes(r, cfg, userHandler, walletHandler, ledgerHandler)

	// 9. Graceful Shutdown
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		slog.Info("Server listening", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("listen error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("Server exiting")
}
