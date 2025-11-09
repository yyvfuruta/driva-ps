package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"

	"github.com/yyvfuruta/driva-ps/internal/broker"
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

	ctx := r.Context()

	// If idempotencyKey already exists.
	key, err := app.models.IdempotencyKey.Get(ctx, idempotencyKey)
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

	if err := app.models.Order.Insert(ctx, order); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := app.models.IdempotencyKey.Insert(ctx, idempotencyKey, order.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	body, err := json.Marshal(order)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := app.broker.Publish(
		ctx,
		broker.OrderEventsExchangeName,
		broker.OrderCreatedRoutingKey,
		body,
	); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := struct {
		ID      uuid.UUID `json:"id"`
		Message string    `json:"msg"`
	}{
		ID:      order.ID,
		Message: "Order created succesfully",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (app *application) getOrderHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/orders/"):]
	orderID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Check cache:
	cacheKey := fmt.Sprintf("order:%s", idStr)
	cachedData, err := app.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		app.logger.Info("Returing cached order", "order_id", idStr)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.Write([]byte(cachedData))
		return
	}
	if err != redis.Nil {
		http.Error(w, "Redis error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Check DB:
	order, err := app.models.Order.Get(ctx, orderID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	var orderEnriched *models.OrderEnrichment

	if order.Status == "completed" {
		orderEnriched, err = app.models.Enrichment.Get(ctx, orderID)
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

	responseJSON, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Cache reponse:
	if err := app.redis.Set(ctx, cacheKey, responseJSON, 1*time.Minute).Err(); err != nil {
		app.logger.Error("Could not save order to cache", "error", err)
	}
	app.logger.Info("Saving order to cache", "order_id", response.Order.ID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (app *application) healthzHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"status": "alive",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (app *application) readyzHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Check database connection
	if err := app.db.PingContext(ctx); err != nil {
		app.logger.Error("Readiness check failed: database ping error", "error", err)
		http.Error(w, "database not ready", http.StatusServiceUnavailable)
		return
	}

	// Check broker connection
	if _, err := broker.NewConnection(); err != nil {
		app.logger.Error("Readiness check failed: broker connection error", "error", err)
		http.Error(w, "Broker not ready", http.StatusServiceUnavailable)
		return
	}

	// Check cache connection
	if err := app.redis.Ping(ctx).Err(); err != nil {
		app.logger.Error("Readiness check failed: cache ping error", "error", err)
		http.Error(w, "cache not ready", http.StatusServiceUnavailable)
		return
	}

	response := map[string]string{
		"status": "ready",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
