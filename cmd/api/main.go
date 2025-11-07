package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/yyvfuruta/driva-ps/internal/database"
	"github.com/yyvfuruta/driva-ps/internal/models"
	"github.com/yyvfuruta/driva-ps/internal/queue"
)

type application struct {
	rabbit *amqp.Connection
	models models.Models
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

	app := &application{
		rabbit: rabbitConn,
		models: models.NewModels(db),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /orders", authMiddleware(app.createOrderHandler))
	mux.HandleFunc("GET /orders/{id}", app.getOrderHandler)

	log.Println("API server starting on port 8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
