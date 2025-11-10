// Package models provides the data models and database access functions.
package models

import "database/sql"

// Models is a wrapper for all the models.
type Models struct {
	Order          OrderModel
	IdempotencyKey IdempotencyKeyModel
	Enrichment     OrderEnrichmentModel
}

func New(db *sql.DB) Models {
	return Models{
		Order:          OrderModel{DB: db},
		IdempotencyKey: IdempotencyKeyModel{DB: db},
		Enrichment:     OrderEnrichmentModel{DB: db},
	}
}
