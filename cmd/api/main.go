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

	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/yyvfuruta/driva-ps/internal/cache"
	"github.com/yyvfuruta/driva-ps/internal/database"
	"github.com/yyvfuruta/driva-ps/internal/logger"
	"github.com/yyvfuruta/driva-ps/internal/models"
	"github.com/yyvfuruta/driva-ps/internal/queue"
)

type application struct {
	db     *sql.DB
	rabbit *amqp.Connection
	models models.Models
	redis  *redis.Client
	logger *slog.Logger
}

func main() {
	var dev bool
	flag.BoolVar(&dev, "dev", false, "Enable godotenv")
	flag.Parse()

	logger := logger.New()

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

	rabbitConn, err := queue.NewConnection()
	if err != nil {
		logger.Error("Failed to connect to RabbitMQ", "error", err)
		os.Exit(1)
	}
	defer rabbitConn.Close()

	rdb, err := cache.NewConnection()
	if err != nil {
		logger.Error("Failed to connect to Redis", "error", err)
		os.Exit(1)
	}

	app := &application{
		db:     db,
		rabbit: rabbitConn,
		models: models.NewModels(db),
		redis:  rdb,
		logger: logger,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /orders", authMiddleware(app.createOrderHandler))
	mux.HandleFunc("GET /orders/{id}", app.getOrderHandler)
	mux.HandleFunc("GET /healthz", app.healthzHandler)
	mux.HandleFunc("GET /readyz", app.readyzHandler)

	port := os.Getenv("API_PORT")
	if port == "" {
		logger.Error("API_PORT empty")
		os.Exit(1)
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: mux,
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
