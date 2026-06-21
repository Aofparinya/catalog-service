package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/saaof/order-platform/catalog-service/internal/adapter/auth"
	commonclient "github.com/saaof/order-platform/catalog-service/internal/adapter/common"
	httpadapter "github.com/saaof/order-platform/catalog-service/internal/adapter/http"
	postgresrepository "github.com/saaof/order-platform/catalog-service/internal/adapter/persistence/postgres"
	storageclient "github.com/saaof/order-platform/catalog-service/internal/adapter/storage"
	"github.com/saaof/order-platform/catalog-service/internal/application"
	"github.com/saaof/order-platform/catalog-service/internal/config"
	"github.com/saaof/order-platform/catalog-service/internal/infrastructure/database"
	"github.com/saaof/order-platform/catalog-service/internal/worker"
)

func main() {
	if err := run(); err != nil {
		slog.Error("catalog service stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	db, err := database.Open(cfg.DatabaseURL, cfg.AppEnv == "production")
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	if err := database.Migrate(db); err != nil {
		return err
	}

	numberClient := commonclient.New(cfg.CommonURL, cfg.AuthURL, cfg.ServiceClientID, cfg.ServiceClientSecret)
	repository := postgresrepository.NewRepository(db, numberClient)
	fileClient := storageclient.New(cfg.StorageURL, numberClient)
	service := application.NewService(repository, fileClient)
	authClient := auth.NewClient(cfg.AuthURL, cfg.AuthCacheTTL)
	handler := httpadapter.NewHandler(service)
	server := httpadapter.NewRouter(handler, authClient, cfg.CORSOrigins)

	workerContext, stopWorker := context.WithCancel(context.Background())
	defer stopWorker()
	go worker.RunReservationExpiration(workerContext, service, cfg.ReservationSweep)

	serverError := make(chan error, 1)
	go func() {
		address := fmt.Sprintf(":%d", cfg.Port)
		slog.Info("catalog service started", "address", address)
		if err := server.Start(address); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverError <- err
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-serverError:
		return err
	case signalValue := <-signals:
		slog.Info("shutdown signal received", "signal", signalValue.String())
	}

	stopWorker()
	shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return server.Shutdown(shutdownContext)
}
