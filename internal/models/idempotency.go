package models

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type IdempotencyKey struct {
	Key       string    `json:"key"`
	OrderID   uuid.UUID `json:"order_id"`
	CreatedAt time.Time `json:"created_at"`
}

type IdempotencyKeyModel struct {
	DB *sql.DB
}

func (i IdempotencyKeyModel) Get(ctx context.Context, key string) (*IdempotencyKey, error) {
	idempotencyKey := &IdempotencyKey{}
	row := i.DB.QueryRowContext(ctx, `SELECT key, order_id, created_at FROM idempotency_keys WHERE key = $1`, key)
	err := row.Scan(&idempotencyKey.Key, &idempotencyKey.OrderID, &idempotencyKey.CreatedAt)
	if err != nil {
		return nil, err
	}
	return idempotencyKey, nil
}

func (i IdempotencyKeyModel) Insert(ctx context.Context, key string, orderID uuid.UUID) error {
	_, err := i.DB.ExecContext(
		ctx,
		`INSERT INTO idempotency_keys (key, order_id, created_at) VALUES ($1, $2, NOW())`,
		key,
		orderID,
	)
	return err
}
