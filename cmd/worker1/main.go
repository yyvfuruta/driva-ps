// Worker1 is responsible for:
// 1. Consumes the messages published from the API.
// 2. Update message status to "processing".
// 3. Publish the message to the next worker ordering the data enrichment.
package main

import (
	"flag"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/yyvfuruta/driva-ps/internal/broker"
	"github.com/yyvfuruta/driva-ps/internal/database"
	"github.com/yyvfuruta/driva-ps/internal/logger"
	"github.com/yyvfuruta/driva-ps/internal/models"
	"github.com/yyvfuruta/driva-ps/internal/worker"
)

func main() {
	logger := logger.New()

	var dev bool

	flag.BoolVar(&dev, "dev", false, "Enable godotenv")
	flag.Parse()

	if dev {
		if err := godotenv.Load(); err != nil {
			logger.Error("Error loading .env file", "error", err)
			os.Exit(1)
		}
	}

	b, err := broker.New()
	if err != nil {
		logger.Error("Failed to create new broker", "error", err)
		os.Exit(1)
	}
	defer b.Channel.Close()

	if err := b.Setup(
		broker.OrderEventsExchangeName,
		broker.OrderEventsExchangeType,
		broker.OrderCreatedQueue,
		broker.OrderCreatedRoutingKey,
		nil, // queueArgs
	); err != nil {
		logger.Error("Failed to create new broker", "error", err)
		os.Exit(1)
	}

	db, err := database.NewConnection()
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	h := &handler{
		models: models.New(db),
		logger: logger,
		broker: b,
	}

	w := worker.New(
		broker.OrderCreatedQueue,
		b,
	)

	w.Run(h)
}
