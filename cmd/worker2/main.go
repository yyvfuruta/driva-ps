package main

import (
	"encoding/json"
	"flag"
	"log"
	"strings"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/yyvfuruta/driva-ps/internal/database"
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

	if dev {
		if err := godotenv.Load(); err != nil {
			log.Fatal(err)
		}
	}

	rabbit, err := queue.NewConnection()
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer rabbit.Close()

	ch, err := rabbit.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
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
		log.Fatal(err)
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
		log.Fatalf("Failed to declare a queue: %v", err)
	}

	err = ch.QueueBind(
		enrichmentRequestQueue.Name,     // queue name
		enrichmentRequestRoutingKeyName, // routing key
		orderEventsExchangeName,         // exchange
		false,                           // no-wait
		nil,                             // args
	)
	if err != nil {
		log.Fatalf("Failed to declare a queue: %v", err)
	}

	enrichmentRequests, err := ch.Consume(
		enrichmentRequestQueue.Name, // queue
		"",                          // consumer
		false,                       // auto-ack
		false,                       // exclusive
		false,                       // no-local
		false,                       // no-wait
		nil,                         // args
	)
	if err != nil {
		log.Fatalf("Failed to register a consumer: %v", err)
	}

	forever := make(chan bool)

	db, err := database.NewConnection()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	appModels := models.NewModels(db)

	go func() {
		for d := range enrichmentRequests {
			var input models.Order
			if err := json.Unmarshal(d.Body, &input); err != nil {
				log.Printf("Error decoding message: %s", err)
				d.Nack(false, false)
				continue
			}
			log.Printf("[%s] Received a message for enrichment", input.ID)

			retryCount := getRetryCount(d.Headers)

			if retryCount >= maxRetries {
				if err := appModels.Order.Update(input.ID, "failed"); err != nil {
					log.Println(err)
				} else {
					log.Printf("[%s] Order exceded maximum retries allowed", input.ID)
				}
				d.Ack(false)
				continue
			}

			// Failure simulation:
			if strings.Contains(input.CustomerID, "f") {
				log.Printf("[%s] Simulated failure (count: %d)", input.ID.String(), retryCount)
				d.Nack(false, false)
				continue
			}

			log.Printf("[%s] Starting data enrichment", input.ID)
			time.Sleep(5 * time.Second)
			log.Printf("[%s] Finished data enrichment", input.ID)

			enrichment := &models.OrderEnrichment{
				OrderID: input.ID,
				Data:    []byte(`{"message": "enriched"}`),
			}

			if err := appModels.Enrichment.Insert(enrichment); err != nil {
				log.Printf("Error creating enrichment: %s", err)
				d.Nack(false, false)
				continue
			}

			body, err := json.Marshal(input)
			if err != nil {
				log.Printf("Error marshalling order: %s", err)
				d.Nack(false, false)
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
				log.Printf("Error publishing message: %s", err)
				d.Nack(false, false)
				continue
			}

			d.Ack(false)
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
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
