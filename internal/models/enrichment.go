package models

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// OrderEnrichment represents the enriched data for an order.
type OrderEnrichment struct {
	ID        int       `json:"id"`
	OrderID   uuid.UUID `json:"order_id"`
	Data      []byte    `json:"data"`
	CreatedAt time.Time `json:"created_at"`
}

// OrderEnrichmentModel is a wrapper for the database connection.
type OrderEnrichmentModel struct {
	DB *sql.DB
}

// Insert inserts an order enrichment into the database.
func (o *OrderEnrichmentModel) Insert(ctx context.Context, enrichment *OrderEnrichment) error {
	if enrichment == nil {
		return fmt.Errorf("enrichment can't be nil")
	}

	err := uuid.Validate(enrichment.OrderID.String())
	if err != nil {
		return err
	}

	if enrichment.Data == nil {
		return fmt.Errorf("enrichment.Data can't be nil")
	}

	if o.DB == nil {
		return fmt.Errorf("enrichment.DB can't be nil")
	}

	_, err = o.DB.ExecContext(
		ctx,
		`INSERT INTO order_enrichments (order_id, data, created_at) VALUES ($1, $2, NOW())`,
		enrichment.OrderID,
		enrichment.Data,
	)
	return err
}

// Get gets an order enrichment from the database.
func (o *OrderEnrichmentModel) Get(ctx context.Context, orderID uuid.UUID) (*OrderEnrichment, error) {
	if o.DB == nil {
		return nil, fmt.Errorf("enrichment.DB can't be nil")
	}

	// Valide o UUID
	err := uuid.Validate(orderID.String())
	if err != nil {
		return nil, err
	}

	stmt := `SELECT id, order_id, data, created_at FROM order_enrichments WHERE order_id = $1`

	var enrichment OrderEnrichment

	err = o.DB.QueryRowContext(ctx, stmt, orderID).Scan(
		&enrichment.ID,
		&enrichment.OrderID,
		&enrichment.Data,
		&enrichment.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}

	return &enrichment, nil
}
