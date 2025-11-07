package main

import (
	"encoding/json"
	"flag"
	"log"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/yyvfuruta/driva-ps/internal/database"
	"github.com/yyvfuruta/driva-ps/internal/models"
	"github.com/yyvfuruta/driva-ps/internal/queue"
)

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
		"order.events", // name
		"direct",       // type
		true,           // durable
		false,          // auto-deleted
		false,          // internal
		false,          // no-wait
		nil,            // arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare an exchange: %v", err)
	}

	q, err := ch.QueueDeclare(
		"order.created", // name
		true,            // durable
		false,           // delete when unused
		false,           // exclusive
		false,           // no-wait
		nil,             // arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare a queue: %v", err)
	}

	err = ch.QueueBind(
		q.Name,          // queue name
		"order.created", // routing key
		"order.events",  // exchange
		false,           // no-wait
		nil,             // args
	)
	if err != nil {
		log.Fatalf("Failed to bind a queue: %v", err)
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
		for msg := range msgs {
			var order models.Order
			if err := json.Unmarshal(msg.Body, &order); err != nil {
				log.Printf("Error decoding message: %s", err)
				msg.Nack(false, false)
				continue
			}
			log.Printf("[%s] Received order", order.ID)

			body, err := json.Marshal(order)
			if err != nil {
				log.Printf("Error marshalling order: %s", err)
				msg.Nack(false, false)
				continue
			}

			err = appModels.Order.Update(order.ID, "processing")
			if err != nil {
				log.Printf(err.Error())
				msg.Nack(false, false)
				continue
			}

			if err := ch.Publish(
				"order.events",               // exchange
				"order.enrichment.requested", // routing key
				false,                        // mandatory
				false,                        // immediate
				amqp.Publishing{
					ContentType: "application/json",
					Body:        body,
				},
			); err != nil {
				log.Printf("Error publishing message: %s", err)
				msg.Nack(false, false)
				continue
			}

			msg.Ack(false)
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}
