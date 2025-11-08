package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"github.com/yyvfuruta/driva-ps/internal/cache"
	"github.com/yyvfuruta/driva-ps/internal/database"
	"github.com/yyvfuruta/driva-ps/internal/models"
	"github.com/yyvfuruta/driva-ps/internal/queue"
)

type application struct {
	db     *sql.DB
	rabbit *amqp.Connection
	models models.Models
	redis  *redis.Client
}

func main() {
	var dev bool
	flag.BoolVar(&dev, "dev", false, "Enable godotenv")
	flag.Parse()

	if dev {
		err := godotenv.Load()
		if err != nil {
			log.Fatal(err)
		}
	}

	db, err := database.NewConnection()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	rabbitConn, err := queue.NewConnection()
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer rabbitConn.Close()

	rdb, err := cache.NewConnection()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	app := &application{
		db:     db,
		rabbit: rabbitConn,
		models: models.NewModels(db),
		redis:  rdb,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /orders", authMiddleware(app.createOrderHandler))
	mux.HandleFunc("GET /orders/{id}", app.getOrderHandler)
	mux.HandleFunc("GET /healthz", app.healthzHandler)
	mux.HandleFunc("GET /readyz", app.readyzHandler)

	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("API_PORT empty")
	}
	log.Printf("API starting on port %s\n", port)
	log.Fatal(http.ListenAndServe(
		fmt.Sprintf(":%s", port),
		mux,
	))
}
