package main

import (
	"context"
	"encoding/json"
	"flag"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/yyvfuruta/driva-ps/internal/database"
	"github.com/yyvfuruta/driva-ps/internal/logger"
	"github.com/yyvfuruta/driva-ps/internal/models"
	"github.com/yyvfuruta/driva-ps/internal/queue"
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
		"order.events", // name
		"direct",       // type
		true,           // durable
		false,          // auto-deleted
		false,          // internal
		false,          // no-wait
		nil,            // arguments
	)
	if err != nil {
		logger.Error("Failed to declare an exchange", "error", err)
		os.Exit(1)
	}

	q, err := ch.QueueDeclare(
		"order.enriched", // name
		true,             //durable
		false,            // delete when unused
		false,            // exclusive
		false,            // no-wait
		nil,              // arguments
	)
	if err != nil {
		logger.Error("Failed to declare a queue", "error", err)
		os.Exit(1)
	}

	err = ch.QueueBind(
		q.Name,           // queue name
		"order.enriched", // routing key
		"order.events",   // exchange
		false,            // no-wait
		nil,              // args
	)
	if err != nil {
		logger.Error("Failed to bind a queue", "error", err)
		os.Exit(1)
	}

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
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

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				logger.Info("Shutting down worker...")
				return
			case msg, ok := <-msgs:
				if !ok {
					logger.Info("Channel closed, shutting down.")
					return
				}
				var input models.Order
				if err := json.Unmarshal(msg.Body, &input); err != nil {
					logger.Error("Error decoding message", "error", err)
					msg.Nack(false, false)
					continue
				}
				logger.Info("Received a message enriched", "order_id", input.ID)

				if err := appModels.Order.Update(ctx, input.ID, "completed"); err != nil {
					logger.Error("Error updating order status to completed", "error", err)
					msg.Nack(false, false)
					continue
				}
				msg.Ack(false)
			}
		}
	}()

	logger.Info(" [*] Waiting for messages. To exit press CTRL+C")
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	cancel()
	wg.Wait()
	logger.Info("Worker shutdown complete.")
}
