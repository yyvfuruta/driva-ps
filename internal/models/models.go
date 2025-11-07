package models

import "database/sql"

type Models struct {
	Order          OrderModel
	IdempotencyKey IdempotencyKeyModel
	Enrichment     OrderEnrichmentModel
}

func NewModels(db *sql.DB) Models {
	return Models{
		Order:          OrderModel{DB: db},
		IdempotencyKey: IdempotencyKeyModel{DB: db},
		Enrichment:     OrderEnrichmentModel{DB: db},
	}
}
