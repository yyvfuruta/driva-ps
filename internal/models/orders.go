package models

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/yyvfuruta/driva-ps/internal/validator"
)

type Order struct {
	ID          uuid.UUID   `json:"id"`
	CustomerID  string      `json:"customer_id"`
	Status      string      `json:"status"`
	TotalAmount float64     `json:"total_amount"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	Items       []OrderItem `json:"items"`
}

func ValidateOrder(v *validator.Validator, order *Order) {
	v.Check(order.CustomerID != "", "customer_id", "must be provided")

	v.Check(order.TotalAmount >= 0.0, "total_amount", "must be zero or positive")

	v.Check(len(order.Items) >= 0, "items", "must be zero or positive")

	for i, item := range order.Items {
		itemKeySKU := fmt.Sprintf("items[%d].sku", i)
		itemKeyQty := fmt.Sprintf("items[%d].qty", i)

		v.Check(item.SKU != "", itemKeySKU, "sku must be provided")
		v.Check(item.Qty > 0, itemKeyQty, "quantity must be greater than zero")
	}
}

type OrderItem struct {
	ID      int       `json:"id"`
	OrderID uuid.UUID `json:"order_id"`
	SKU     string    `json:"sku"`
	Qty     int       `json:"qty"`
}

type OrderModel struct {
	DB *sql.DB
}

func (o OrderModel) Get(ctx context.Context, orderID uuid.UUID) (*Order, error) {
	order := &Order{}
	row := o.DB.QueryRowContext(
		ctx,
		`SELECT id, customer_id, status, total_amount, created_at, updated_at FROM orders WHERE id = $1`,
		orderID,
	)
	err := row.Scan(&order.ID, &order.CustomerID, &order.Status, &order.TotalAmount, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		return nil, err
	}

	rows, err := o.DB.QueryContext(
		ctx,
		`SELECT id, sku, qty FROM order_items WHERE order_id = $1`,
		orderID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		item := OrderItem{}
		err = rows.Scan(&item.ID, &item.SKU, &item.Qty)
		if err != nil {
			return nil, err
		}
		order.Items = append(order.Items, item)
	}

	return order, nil
}

func (o OrderModel) Insert(ctx context.Context, order *Order) error {
	tx, err := o.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO orders (id, customer_id, status, total_amount, created_at, updated_at) 
        VALUES ($1, $2, $3, $4, $5, $6)`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, order.ID, order.CustomerID, order.Status, order.TotalAmount, order.CreatedAt, order.UpdatedAt)
	if err != nil {
		tx.Rollback()
		return err
	}

	stmt, err = tx.PrepareContext(ctx, `INSERT INTO order_items (order_id, sku, qty) VALUES ($1, $2, $3)`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, item := range order.Items {
		_, err = stmt.ExecContext(ctx, order.ID, item.SKU, item.Qty)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (o OrderModel) Update(ctx context.Context, orderID uuid.UUID, status string) error {
	_, err := o.DB.ExecContext(
		ctx,
		`UPDATE orders SET status = $1, updated_at = NOW() WHERE id = $2`,
		status,
		orderID,
	)
	return err
}
