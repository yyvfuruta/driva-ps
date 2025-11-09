package worker

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/yyvfuruta/driva-ps/internal/broker"
	"github.com/yyvfuruta/driva-ps/internal/logger"
)

type Handlerer interface {
	HandleMessage(ctx context.Context, msg amqp.Delivery) error
}

type Worker struct {
	queueName string
	broker    *broker.Broker
}

func New(queueName string, broker *broker.Broker) *Worker {
	return &Worker{
		queueName: queueName,
		broker:    broker,
	}
}

func (w *Worker) Run(handler Handlerer) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := logger.New()

	msgs, err := w.broker.Consume(w.queueName)
	if err != nil {
		logger.Error("Failed to register a consumer", "error", err)
		return
	}

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
					msg.Nack(false, false)
					return
				}
				if err := handler.HandleMessage(ctx, msg); err != nil {
					logger.Error("Error handling message", "error", err)
					msg.Nack(false, false)
					return
				}
				msg.Ack(false)
			}
		}
	}()

	logger.Info("Waiting for messages.")
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	cancel()
	wg.Wait()
	logger.Info("Worker shutdown complete.")
}
