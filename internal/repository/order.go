package repository

import (
	"context"
	"database/sql"
	"log"

	"github.com/tm-acme-shop/acme-shop-orders-service/internal/models"
)

type OrderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) CreateOrder(ctx context.Context, userID int64, total float64, currency string) (*models.Order, error) {
	log.Printf("Inserting order for user %d into database", userID)

	var id int64
	err := r.db.QueryRowContext(
		ctx,
		`INSERT INTO orders (user_id, status, total, currency, created_at)
		 VALUES ($1, 'pending', $2, $3, NOW())
		 RETURNING id`,
		userID, total, currency,
	).Scan(&id)
	if err != nil {
		return nil, err
	}

	return &models.Order{
		ID:       id,
		UserID:   userID,
		Status:   "pending",
		Total:    total,
		Currency: currency,
	}, nil
}

func (r *OrderRepository) GetOrderByID(ctx context.Context, orderID int64) (*models.Order, error) {
	log.Printf("Fetching order %d from database", orderID)

	var order models.Order
	err := r.db.QueryRowContext(
		ctx,
		`SELECT id, user_id, status, total, currency FROM orders WHERE id = $1`,
		orderID,
	).Scan(&order.ID, &order.UserID, &order.Status, &order.Total, &order.Currency)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &order, nil
}

func (r *OrderRepository) GetOrdersByUserID(ctx context.Context, userID int64) ([]*models.Order, error) {
	log.Printf("Fetching orders for user %d from database", userID)

	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, user_id, status, total, currency FROM orders WHERE user_id = $1 ORDER BY id DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []*models.Order
	for rows.Next() {
		var order models.Order
		if err := rows.Scan(&order.ID, &order.UserID, &order.Status, &order.Total, &order.Currency); err != nil {
			return nil, err
		}
		orders = append(orders, &order)
	}

	return orders, nil
}

func (r *OrderRepository) UpdateOrderStatus(ctx context.Context, orderID int64, status string) (*models.Order, error) {
	log.Printf("Updating order %d status to %s in database", orderID, status)

	var order models.Order
	err := r.db.QueryRowContext(
		ctx,
		`UPDATE orders SET status = $1, updated_at = NOW() WHERE id = $2
		 RETURNING id, user_id, status, total, currency`,
		status, orderID,
	).Scan(&order.ID, &order.UserID, &order.Status, &order.Total, &order.Currency)
	if err != nil {
		return nil, err
	}

	return &order, nil
}
