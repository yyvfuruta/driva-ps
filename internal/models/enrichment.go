package models

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type OrderEnrichment struct {
	ID        int       `json:"id"`
	OrderID   uuid.UUID `json:"order_id"`
	Data      []byte    `json:"data"`
	CreatedAt time.Time `json:"created_at"`
}

type OrderEnrichmentModel struct {
	DB *sql.DB
}

func (o *OrderEnrichmentModel) Insert(enrichment *OrderEnrichment) error {
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

	_, err = o.DB.Exec(
		`INSERT INTO order_enrichments (order_id, data, created_at) VALUES ($1, $2, NOW())`,
		enrichment.OrderID,
		enrichment.Data,
	)
	return err
}

// Get busca um OrderEnrichment pelo order_id.
func (o *OrderEnrichmentModel) Get(orderID uuid.UUID) (*OrderEnrichment, error) {
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

	err = o.DB.QueryRow(stmt, orderID).Scan(
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
