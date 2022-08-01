package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/tm-acme-shop/acme-shop-orders-service/internal/models"
)

type OrderRepositoryV2 struct {
	db *sql.DB
}

func NewOrderRepositoryV2(db *sql.DB) *OrderRepositoryV2 {
	return &OrderRepositoryV2{db: db}
}

func (r *OrderRepositoryV2) CreateOrder(ctx context.Context, userID string, total float64, currency string) (*models.OrderV2, error) {
	log.Printf("Inserting order for user %s into database", userID)

	id := fmt.Sprintf("ord_%d", time.Now().UnixNano())
	now := time.Now()

	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO orders_v2 (id, user_id, status, total, currency, created_at, updated_at)
		 VALUES ($1, $2, 'pending', $3, $4, $5, $6)`,
		id, userID, total, currency, now, now,
	)
	if err != nil {
		return nil, err
	}

	return &models.OrderV2{
		ID:        id,
		UserID:    userID,
		Status:    "pending",
		Total:     total,
		Currency:  currency,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (r *OrderRepositoryV2) GetOrderByID(ctx context.Context, orderID string) (*models.OrderV2, error) {
	log.Printf("Fetching order %s from database", orderID)

	var order models.OrderV2
	err := r.db.QueryRowContext(
		ctx,
		`SELECT id, user_id, status, total, currency, created_at, updated_at FROM orders_v2 WHERE id = $1`,
		orderID,
	).Scan(&order.ID, &order.UserID, &order.Status, &order.Total, &order.Currency, &order.CreatedAt, &order.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &order, nil
}

func (r *OrderRepositoryV2) GetOrdersByUserID(ctx context.Context, userID string) ([]*models.OrderV2, error) {
	log.Printf("Fetching orders for user %s from database", userID)

	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, user_id, status, total, currency, created_at, updated_at FROM orders_v2 WHERE user_id = $1 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []*models.OrderV2
	for rows.Next() {
		var order models.OrderV2
		if err := rows.Scan(&order.ID, &order.UserID, &order.Status, &order.Total, &order.Currency, &order.CreatedAt, &order.UpdatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, &order)
	}

	return orders, nil
}

func (r *OrderRepositoryV2) UpdateOrderStatus(ctx context.Context, orderID string, status string) (*models.OrderV2, error) {
	log.Printf("Updating order %s status to %s in database", orderID, status)

	now := time.Now()
	var order models.OrderV2
	err := r.db.QueryRowContext(
		ctx,
		`UPDATE orders_v2 SET status = $1, updated_at = $2 WHERE id = $3
		 RETURNING id, user_id, status, total, currency, created_at, updated_at`,
		status, now, orderID,
	).Scan(&order.ID, &order.UserID, &order.Status, &order.Total, &order.Currency, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &order, nil
}
