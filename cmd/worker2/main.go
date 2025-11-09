// Worker2 is responsible for:
// 1. Consume data from order.enrichment queue.
// 2. Simulate the process to enrich the data.
// 3. If succesfull, save the data in the database and publish a message in order.enriched queue.
// 4. In case of failure, try to requeue the order.
package main

import (
	"flag"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/yyvfuruta/driva-ps/internal/broker"
	"github.com/yyvfuruta/driva-ps/internal/database"
	"github.com/yyvfuruta/driva-ps/internal/logger"
	"github.com/yyvfuruta/driva-ps/internal/models"
	"github.com/yyvfuruta/driva-ps/internal/worker"
)

const retryTTLMiliseconds = 10000

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
		logger.Error("Failed to create a broker", "error", err)
		os.Exit(1)
	}
	defer b.Channel.Close()

	if err := b.Setup(
		broker.OrderEventsExchangeName,
		broker.OrderEventsExchangeType,
		broker.EnrichmentRequestQueue,
		broker.EnrichmentRequestRoutingKey,
		amqp.Table{
			"x-message-ttl":             retryTTLMiliseconds,
			"x-dead-letter-routing-key": broker.EnrichmentRequestRoutingKey,
			"x-dead-letter-exchange":    broker.OrderEventsExchangeName,
		},
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
		broker: b,
		logger: logger,
	}

	w := worker.New(
		broker.EnrichmentRequestQueue,
		b,
	)

	w.Run(h)
}
