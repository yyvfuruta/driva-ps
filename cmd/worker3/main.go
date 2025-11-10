// Worker3 is responsible for:
// 1. Consume messages from order.enriched queue.
// 2. Update order status to "completed".
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
		logger.Error("Failed to open a channel", "error", err)
		os.Exit(1)
	}
	defer b.Channel.Close()

	if err := b.Setup(
		broker.OrderEventsExchangeName,
		broker.OrderEventsExchangeType,
		broker.OrderEnrichedQueue,
		broker.OrderEnrichedRoutingKey,
		nil,
	); err != nil {
		logger.Error("Failed to setup broker", "error", err)
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
	}

	w := worker.New(
		broker.OrderEnrichedQueue,
		b,
	)

	w.Run(h)
}
