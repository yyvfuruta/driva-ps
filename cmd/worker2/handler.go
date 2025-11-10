package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/yyvfuruta/driva-ps/internal/broker"
	"github.com/yyvfuruta/driva-ps/internal/models"
)

const maxRetries = 3

type handler struct {
	models models.Models
	broker *broker.Broker
	logger *slog.Logger
}

func (h *handler) HandleMessage(ctx context.Context, msg amqp.Delivery) error {
	var input models.Order
	if err := json.Unmarshal(msg.Body, &input); err != nil {
		return err
	}
	h.logger.Info("Received a message for enrichment", "order_id", input.ID)

	count := retryCount(msg.Headers)
	if count >= maxRetries {
		input.Status = "failed"
		if err := h.models.Order.Update(ctx, input.ID, input.Status); err != nil {
			return fmt.Errorf("Error updating order", "error", err)
		}
		h.logger.Error("Order exceded maximum retries allowed", "order_id", input.ID)
		h.logger.Info("Order status updated", "order_status", input.Status, "order_id", input.ID)

		// Do not send NACK again:
		return nil
	}

	// Failure simulation:
	if strings.Contains(input.CustomerID, "f") {
		return fmt.Errorf("Simulated failure.")
	}

	// Data enrichment simulation:
	h.logger.Info("Starting data enrichment", "order_id", input.ID)
	time.Sleep(5 * time.Second)
	enrichment := &models.OrderEnrichment{
		OrderID: input.ID,
		Data:    []byte(`{"message": "enriched"}`),
	}
	h.logger.Info("Finished data enrichment", "order_id", input.ID)

	if err := h.models.Enrichment.Insert(ctx, enrichment); err != nil {
		return err
	}

	if err := h.broker.Publish(
		ctx,
		broker.OrderEventsExchangeName,
		broker.OrderEnrichedRoutingKey,
		msg.Body,
	); err != nil {
		return err
	}

	return nil
}

// retryCount gets retry count from the header 'x-death'.
func retryCount(headers amqp.Table) int64 {
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
