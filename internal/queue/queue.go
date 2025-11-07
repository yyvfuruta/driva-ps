package queue

import (
	"fmt"
	"os"

	amqp "github.com/rabbitmq/amqp091-go"
)

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
