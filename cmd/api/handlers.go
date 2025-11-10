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
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	idempotencyKey := r.Header.Get("X-Idempotency-Key")
	if idempotencyKey == "" {
		errorResponse(w, http.StatusBadRequest, "Header X-Idempotency-Key empty")
		return
	}

	ctx := r.Context()

	// If idempotencyKey already exists.
	key, err := app.models.IdempotencyKey.Get(ctx, idempotencyKey)
	if err == nil {
		writeJSON(w, http.StatusOK, envelope{"order_id": key.OrderID, "message": "Order already exists."}, nil)
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
		writeJSON(w, http.StatusOK, envelope{"order_id": order.ID, "message": v.Errors}, nil)
		return
	}

	app.logger.Info("Order created", "order_status", order.Status, "order_id", order.ID)

	if err := app.models.Order.Insert(ctx, order); err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := app.models.IdempotencyKey.Insert(ctx, idempotencyKey, order.ID); err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	body, err := json.Marshal(order)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := app.broker.Publish(
		ctx,
		broker.OrderEventsExchangeName,
		broker.OrderCreatedRoutingKey,
		body,
	); err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, envelope{"data": map[string]any{"order_id": order.ID, "message": "Order created succesfully"}}, nil)
}

func (app *application) getOrderHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/orders/"):]
	orderID, err := uuid.Parse(idStr)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, "Invalid order ID")
		return
	}

	ctx := r.Context()

	// Check cache:
	cacheKey := fmt.Sprintf("order:%s", idStr)
	cachedData, err := app.cache.Get(ctx, cacheKey)
	if err == nil {
		app.logger.Info("Returing cached order", "order_id", idStr)
		headers := make(http.Header)
		headers.Set("X-Cache", "HIT")
		var structuredData json.RawMessage = []byte(cachedData)
		writeJSON(w, http.StatusOK, envelope{"data": structuredData}, headers)
		return
	}
	if err != redis.Nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Cache error: %v", err))
		return
	}

	// Check DB:
	order, err := app.models.Order.Get(ctx, orderID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	var orderEnriched *models.OrderEnrichment

	if order.Status == "completed" {
		orderEnriched, err = app.models.Enrichment.Get(ctx, orderID)
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	response := struct {
		Order         *models.Order           `json:"order"`
		OrderEnriched *models.OrderEnrichment `json:"order_enriched"`
	}{
		Order:         order,
		OrderEnriched: orderEnriched,
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Cache reponse:
	if err := app.cache.Set(ctx, cacheKey, responseJSON, 1*time.Minute); err != nil {
		app.logger.Error("Could not save order to cache", "error", err)
	}

	app.logger.Info("Saving order to cache", "order_id", response.Order.ID)

	writeJSON(w, http.StatusOK, envelope{"data": map[string]any{"order": order, "order_enriched": orderEnriched}}, nil)
}

func (app *application) healthzHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, envelope{"status": "alive"}, nil)
}

func (app *application) readyzHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Check database connection
	if err := app.db.PingContext(ctx); err != nil {
		app.logger.Error("Readiness check failed: database ping error", "error", err)
		errorResponse(w, http.StatusServiceUnavailable, "Database not ready")
		return
	}

	// Check broker connection
	if _, err := broker.NewConnection(); err != nil {
		app.logger.Error("Readiness check failed: broker connection error", "error", err)
		errorResponse(w, http.StatusServiceUnavailable, "Broker not ready")
		return
	}

	// Check cache connection
	if err := app.cache.Ping(ctx); err != nil {
		app.logger.Error("Readiness check failed: cache ping error", "error", err)
		errorResponse(w, http.StatusServiceUnavailable, "Cache not ready")
		return
	}

	writeJSON(w, http.StatusOK, envelope{"status": "ready"}, nil)
}
