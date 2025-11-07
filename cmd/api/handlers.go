package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/yyvfuruta/driva-ps/internal/models"
	"github.com/yyvfuruta/driva-ps/internal/validator"
)

func (app *application) createOrderHandler(w http.ResponseWriter, r *http.Request) {
	var input models.Order
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	idempotencyKey := r.Header.Get("X-Idempotency-Key")
	if idempotencyKey == "" {
		http.Error(w, "X-Idempotency-Key empty", http.StatusBadRequest)
		return
	}

	// If idempotencyKey already exists.
	key, err := app.models.IdempotencyKey.Get(idempotencyKey)
	if err == nil {
		response := struct {
			OrderID uuid.UUID `json:"order_id"`
			Message string    `json:"message"`
		}{
			OrderID: key.OrderID,
			Message: "Order already exists. Use GET /orders/{id}",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return
	}

	order := &models.Order{
		ID:          uuid.New(),
		CustomerID:  input.CustomerID,
		Status:      "pending",
		TotalAmount: input.TotalAmount,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Items:       input.Items,
	}

	v := validator.New()
	models.ValidateOrder(v, order)
	if !v.Valid() {
		http.Error(w, fmt.Sprintf("Failed validation: %v", v.Errors), http.StatusBadRequest)
		return
	}

	if err := app.models.Order.Insert(order); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := app.models.IdempotencyKey.Insert(idempotencyKey, order.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	body, err := json.Marshal(order)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ch, err := app.rabbit.Channel()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = ch.Publish(
		"order.events",  // exchange
		"order.created", // routing key
		false,           // mandatory
		false,           // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(order)
}

func (app *application) getOrderHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/orders/"):]
	orderID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	order, err := app.models.Order.Get(orderID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	var orderEnriched *models.OrderEnrichment

	if order.Status == "completed" {
		orderEnriched, err = app.models.Enrichment.Get(orderID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	response := struct {
		Order         *models.Order           `json:"order"`
		OrderEnriched *models.OrderEnrichment `json:"enriched_data"`
	}{
		Order:         order,
		OrderEnriched: orderEnriched,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
