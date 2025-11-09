package main

import (
	"context"
	"encoding/json"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/yyvfuruta/driva-ps/internal/models"
)

type handler struct {
	models models.Models
	logger *slog.Logger
}

func (h *handler) HandleMessage(ctx context.Context, msg amqp.Delivery) error {
	var input models.Order

	if err := json.Unmarshal(msg.Body, &input); err != nil {
		return err
	}

	h.logger.Info("Received a message enriched", "order_id", input.ID)

	if err := h.models.Order.Update(ctx, input.ID, "completed"); err != nil {
		return err
	}

	return nil
}
