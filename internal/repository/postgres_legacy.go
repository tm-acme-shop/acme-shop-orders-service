package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/tm-acme-shop/acme-shop-shared-go/logging"
)

// PostgresOrderRepositoryV1 is the deprecated legacy order repository.
// Deprecated: Use PostgresOrderRepository instead.
// TODO(TEAM-API): Remove after v1 API migration complete
type PostgresOrderRepositoryV1 struct {
	db *sql.DB
}

// NewPostgresOrderRepositoryV1 creates a new legacy order repository.
// Deprecated: Use NewPostgresOrderRepository instead.
func NewPostgresOrderRepositoryV1(db *sql.DB) *PostgresOrderRepositoryV1 {
	// TODO(TEAM-API): Remove this constructor after migration
	logging.Infof("Creating legacy order repository v1")
	return &PostgresOrderRepositoryV1{db: db}
}

// GetOrderByID retrieves an order by ID using the legacy format.
// Deprecated: Use PostgresOrderRepository.GetByID instead.
func (r *PostgresOrderRepositoryV1) GetOrderByID(ctx context.Context, id int64) (*LegacyOrder, error) {
	// TODO(TEAM-API): Migrate callers to new repository
	logging.Infof("Legacy: Fetching order by ID: %d", id)

	query := `
		SELECT id, user_id, status, total_amount, total_currency, created_at, updated_at
		FROM orders_v1
		WHERE id = $1
	`

	var order LegacyOrder
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&order.ID,
		&order.UserID,
		&order.Status,
		&order.TotalPrice,
		&order.Currency,
		&order.CreatedAt,
		&order.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("order not found: %d", id)
	}
	if err != nil {
		logging.Infof("Legacy: Failed to fetch order: %v", err)
		return nil, err
	}

	return &order, nil
}

// CreateOrder creates a new order using legacy format.
// Deprecated: Use PostgresOrderRepository.Create instead.
func (r *PostgresOrderRepositoryV1) CreateOrder(ctx context.Context, req *LegacyCreateOrderRequest) (*LegacyOrder, error) {
	// TODO(TEAM-API): Migrate callers to new repository
	logging.Infof("Legacy: Creating order for user: %d", req.UserID)

	itemsJSON, err := json.Marshal(req.Items)
	if err != nil {
		return nil, err
	}

	query := `
		INSERT INTO orders_v1 (user_id, status, items, total_amount, total_currency, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	now := time.Now().Format(time.RFC3339)
	var id int64

	err = r.db.QueryRowContext(ctx, query,
		req.UserID,
		"pending",
		itemsJSON,
		req.TotalPrice,
		req.Currency,
		now,
		now,
	).Scan(&id)

	if err != nil {
		logging.Infof("Legacy: Failed to create order: %v", err)
		return nil, err
	}

	return &LegacyOrder{
		ID:         id,
		UserID:     req.UserID,
		Status:     "pending",
		TotalPrice: req.TotalPrice,
		Currency:   req.Currency,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

// UpdateOrderStatus updates order status.
// Deprecated: Use PostgresOrderRepository.UpdateStatus instead.
func (r *PostgresOrderRepositoryV1) UpdateOrderStatus(ctx context.Context, id int64, status string) error {
	// TODO(TEAM-API): Migrate callers to new repository
	logging.Infof("Legacy: Updating order %d status to: %s", id, status)

	query := `
		UPDATE orders_v1
		SET status = $2, updated_at = $3
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, status, time.Now().Format(time.RFC3339))
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("order not found: %d", id)
	}

	return nil
}

// GetOrdersByUserID retrieves orders for a user.
// Deprecated: Use PostgresOrderRepository.GetByUserID instead.
func (r *PostgresOrderRepositoryV1) GetOrdersByUserID(ctx context.Context, userID int64) ([]*LegacyOrder, error) {
	// TODO(TEAM-API): Migrate callers to new repository
	logging.Infof("Legacy: Fetching orders for user: %d", userID)

	query := `
		SELECT id, user_id, status, total_amount, total_currency, created_at, updated_at
		FROM orders_v1
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT 100
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]*LegacyOrder, 0)
	for rows.Next() {
		var order LegacyOrder
		err := rows.Scan(
			&order.ID,
			&order.UserID,
			&order.Status,
			&order.TotalPrice,
			&order.Currency,
			&order.CreatedAt,
			&order.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		orders = append(orders, &order)
	}

	return orders, nil
}

// ConvertToLegacyOrder converts a v2 order ID to legacy format.
// Deprecated: Remove after migration.
// TODO(TEAM-API): Remove this helper after migration complete
func ConvertToLegacyOrder(orderID string, userID string, status string, total float64, currency string) *LegacyOrder {
	logging.Infof("Converting order %s to legacy format", orderID)
	now := time.Now().Format(time.RFC3339)
	return &LegacyOrder{
		ID:         0, // Legacy uses int64, new uses string UUIDs
		UserID:     0,
		Status:     status,
		TotalPrice: total,
		Currency:   currency,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}
