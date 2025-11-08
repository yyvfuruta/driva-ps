package main

import (
	"context"
	"encoding/json"
	"flag"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/yyvfuruta/driva-ps/internal/database"
	"github.com/yyvfuruta/driva-ps/internal/logger"
	"github.com/yyvfuruta/driva-ps/internal/models"
	"github.com/yyvfuruta/driva-ps/internal/queue"
)

const (
	orderEventsExchangeName = "order.events"

	enrichmentRequestQueueName      = "order.enrichment.requested"
	enrichmentRequestRoutingKeyName = "order.enrichment.requested"

	maxRetries          = 3
	retryTTLMiliseconds = 10000

	orderEnrichedRoutingKeyName = "order.enriched"
)

func main() {
	var dev bool
	flag.BoolVar(&dev, "dev", false, "Enable godotenv")
	flag.Parse()

	logger := logger.New()

	if dev {
		if err := godotenv.Load(); err != nil {
			logger.Error("Error loading .env file", "error", err)
			os.Exit(1)
		}
	}

	rabbit, err := queue.NewConnection()
	if err != nil {
		logger.Error("Failed to connect to RabbitMQ", "error", err)
		os.Exit(1)
	}
	defer rabbit.Close()

	ch, err := rabbit.Channel()
	if err != nil {
		logger.Error("Failed to open a channel", "error", err)
		os.Exit(1)
	}
	defer ch.Close()

	err = ch.ExchangeDeclare(
		orderEventsExchangeName, // name
		"direct",                // type
		true,                    // durable
		false,                   // auto-deleted
		false,                   // internal
		false,                   // no-wait
		nil,                     // arguments
	)
	if err != nil {
		logger.Error("Failed to declare an exchange", "error", err)
		os.Exit(1)
	}

	enrichmentRequestQueue, err := ch.QueueDeclare(
		enrichmentRequestQueueName, // name
		true,                       // durable
		false,                      // delete when unused
		false,                      // exclusive
		false,                      // no-wait
		amqp.Table{
			"x-message-ttl":             retryTTLMiliseconds,
			"x-dead-letter-routing-key": enrichmentRequestRoutingKeyName,
			"x-dead-letter-exchange":    orderEventsExchangeName,
		}, // args
	)
	if err != nil {
		logger.Error("Failed to declare a queue", "error", err)
		os.Exit(1)
	}

	err = ch.QueueBind(
		enrichmentRequestQueue.Name,     // queue name
		enrichmentRequestRoutingKeyName, // routing key
		orderEventsExchangeName,         // exchange
		false,                           // no-wait
		nil,                             // args
	)
	if err != nil {
		logger.Error("Failed to bind a queue", "error", err)
		os.Exit(1)
	}

	msgs, err := ch.Consume(
		enrichmentRequestQueue.Name, // queue
		"",                          // consumer
		false,                       // auto-ack
		false,                       // exclusive
		false,                       // no-local
		false,                       // no-wait
		nil,                         // args
	)
	if err != nil {
		logger.Error("Failed to register a consumer", "error", err)
		os.Exit(1)
	}

	db, err := database.NewConnection()
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	appModels := models.NewModels(db)

	go func() {
		for msg := range msgs {
			var input models.Order
			if err := json.Unmarshal(msg.Body, &input); err != nil {
				logger.Error("Error decoding message", "error", err)
				msg.Nack(false, false)
				continue
			}
			logger.Info("Received a message for enrichment", "order_id", input.ID)

			retryCount := getRetryCount(msg.Headers)

			if retryCount >= maxRetries {
				if err := appModels.Order.Update(context.Background(), input.ID, "failed"); err != nil {
					logger.Error("Error updating order", "error", err)
				} else {
					logger.Info("Order exceded maximum retries allowed", "order_id", input.ID)
				}
				msg.Ack(false)
				continue
			}

			// Failure simulation:
			if strings.Contains(input.CustomerID, "f") {
				logger.Info("Simulated failure", "order_id", input.ID.String(), "retry_count", retryCount)
				msg.Nack(false, false)
				continue
			}

			logger.Info("Starting data enrichment", "order_id", input.ID)
			time.Sleep(5 * time.Second)
			logger.Info("Finished data enrichment", "order_id", input.ID)

			enrichment := &models.OrderEnrichment{
				OrderID: input.ID,
				Data:    []byte(`{"message": "enriched"}`),
			}

			if err := appModels.Enrichment.Insert(context.Background(), enrichment); err != nil {
				logger.Error("Error creating enrichment", "error", err)
				msg.Nack(false, false)
				continue
			}

			body, err := json.Marshal(input)
			if err != nil {
				logger.Error("Error marshalling order", "error", err)
				msg.Nack(false, false)
				continue
			}

			err = ch.Publish(
				orderEventsExchangeName,     // exchange
				orderEnrichedRoutingKeyName, // routing key
				false,                       // mandatory
				false,                       // immediate
				amqp.Publishing{
					ContentType: "application/json",
					Body:        body,
				},
			)
			if err != nil {
				logger.Error("Error publishing message", "error", err)
				msg.Nack(false, false)
				continue
			}

			msg.Ack(false)
		}
	}()

	forever := make(chan bool)
	logger.Info(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

// getRetryCount checks header 'x-death' to verify how much times the message
// was in retry queue.
func getRetryCount(headers amqp.Table) int64 {
	if headers == nil {
		return 0
	}

	xDeath, ok := headers["x-death"]
	if !ok {
		return 0
	}

	xDeathSlice, ok := xDeath.([]any)
	if !ok {
		return 0
	}

	for _, h := range xDeathSlice {
		table, ok := h.(amqp.Table)
		if !ok {
			continue
		}

		count, ok := table["count"].(int64)
		if !ok {
			return 0
		}
		return count
	}

	return 0
}
