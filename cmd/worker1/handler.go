package main

import (
	"context"
	"encoding/json"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/yyvfuruta/driva-ps/internal/broker"
	"github.com/yyvfuruta/driva-ps/internal/models"
)

type handler struct {
	models models.Models
	logger *slog.Logger
	broker *broker.Broker
}

func (h *handler) HandleMessage(ctx context.Context, msg amqp.Delivery) error {
	var order models.Order

	if err := json.Unmarshal(msg.Body, &order); err != nil {
		return err
	}

	order.Status = "processing"
	if err := h.models.Order.Update(ctx, order.ID, order.Status); err != nil {
		return err
	}
	h.logger.Info("Order status updated", "order_status", order.Status, "order_id", order.ID)

	// Marshal because order.Status changed:
	body, err := json.Marshal(order)
	if err != nil {
		return err
	}

	return h.broker.Publish(
		ctx,
		broker.OrderEventsExchangeName,
		broker.EnrichmentRequestRoutingKey,
		body,
	)
}
