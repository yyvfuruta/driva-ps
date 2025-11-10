package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/yyvfuruta/driva-ps/internal/broker"
	"github.com/yyvfuruta/driva-ps/internal/cache"
	"github.com/yyvfuruta/driva-ps/internal/database"
	"github.com/yyvfuruta/driva-ps/internal/logger"
	"github.com/yyvfuruta/driva-ps/internal/models"
)

type application struct {
	db     *sql.DB
	broker *broker.Broker
	models models.Models
	cache  *cache.Cache
	logger *slog.Logger
}

func main() {
	logger := logger.New()

	var dev bool
	flag.BoolVar(&dev, "dev", false, "Enable godotenv")
	flag.Parse()

	if dev {
		err := godotenv.Load()
		if err != nil {
			logger.Error("Error loading .env file", "error", err)
			os.Exit(1)
		}
	}

	db, err := database.NewConnection()
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	b, err := broker.New()
	if err != nil {
		logger.Error("Failed to connect to broker", "error", err)
		os.Exit(1)
	}

	if err := b.Setup(
		broker.OrderEventsExchangeName,
		broker.OrderEventsExchangeType,
		broker.OrderCreatedQueue,
		broker.OrderCreatedRoutingKey,
		nil,
	); err != nil {
		logger.Error("Failed to setup broker", "error", err)
		os.Exit(1)
	}

	c, err := cache.New()
	if err != nil {
		logger.Error("Failed to connect to cache", "error", err)
		os.Exit(1)
	}

	app := &application{
		db:     db,
		broker: b,
		models: models.New(db),
		cache:  c,
		logger: logger,
	}

	port := os.Getenv("API_PORT")
	if port == "" {
		logger.Error("API_PORT empty")
		os.Exit(1)
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: app.routes(),
	}

	go func() {
		logger.Info("API starting", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Could not listen on", "port", port, "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	logger.Info("Server exiting")
}
