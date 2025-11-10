// Package broker provides a wrapper around the amqp client.
package broker

import (
	"context"
	"fmt"
	"os"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Broker is a wrapper around the amqp client.
type Broker struct {
	Channel *amqp.Channel
}

func New() (*Broker, error) {
	conn, err := NewConnection()
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	return &Broker{
		Channel: ch,
	}, nil
}

func NewConnection() (*amqp.Connection, error) {
	userName := os.Getenv("RABBITMQ_USER_NAME")
	userPass := os.Getenv("RABBITMQ_USER_PASS")
	host := os.Getenv("RABBITMQ_HOST")
	port := os.Getenv("RABBITMQ_PORT")

	envVars := map[string]string{
		"RABBITMQ_HOST":      host,
		"RABBITMQ_PORT":      port,
		"RABBITMQ_USER_NAME": userName,
		"RABBITMQ_USER_PASS": userPass,
	}

	for key, value := range envVars {
		if value == "" {
			return nil, fmt.Errorf("%s environment variable not set", key)
		}
	}

	url := fmt.Sprintf("amqp://%s:%s@%s:%s", userName, userPass, host, port)
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// Setup declares an exchange, a queue, and binds the queue to the exchange.
func (b *Broker) Setup(
	exchangeName,
	exchangeType,
	queueName,
	routingKey string,
	queueArgs amqp.Table,
) error {
	err := b.Channel.ExchangeDeclare(
		exchangeName, // name
		exchangeType, // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare an exchange: %w", err)
	}

	q, err := b.Channel.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		queueArgs, // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare a queue: %w", err)
	}

	err = b.Channel.QueueBind(
		q.Name,       // queue name
		routingKey,   // routing key
		exchangeName, // exchange
		false,        // no-wait
		nil,          // args
	)
	if err != nil {
		return fmt.Errorf("failed to bind a queue: %w", err)
	}

	return nil
}

// Publish publishes a message to an exchange.
func (b *Broker) Publish(ctx context.Context, exchange, routingKey string, body []byte) error {
	return b.Channel.PublishWithContext(ctx,
		exchange,   // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
}

// Consume consumes messages from a queue.
func (b *Broker) Consume(queueName string) (<-chan amqp.Delivery, error) {
	d, err := b.Channel.Consume(
		queueName, // queue
		"",        // consumer
		false,     // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		return nil, fmt.Errorf("failed to register a consumer: %w", err)
	}

	return d, nil
}
