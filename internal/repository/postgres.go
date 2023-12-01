package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/tm-acme-shop/acme-shop-shared-go/errors"
	"github.com/tm-acme-shop/acme-shop-shared-go/logging"
	"github.com/tm-acme-shop/acme-shop-shared-go/models"
)

// PostgresOrderRepository implements interfaces.OrderRepository using PostgreSQL.
type PostgresOrderRepository struct {
	db     *sql.DB
	logger *logging.LoggerV2
}

// NewPostgresOrderRepository creates a new PostgreSQL order repository.
func NewPostgresOrderRepository(db *sql.DB, logger *logging.LoggerV2) *PostgresOrderRepository {
	return &PostgresOrderRepository{
		db:     db,
		logger: logger,
	}
}

// GetByID retrieves an order by its unique identifier.
func (r *PostgresOrderRepository) GetByID(ctx context.Context, id string) (*models.Order, error) {
	r.logger.Debug("Fetching order by ID", logging.Fields{"order_id": id})

	query := `
		SELECT id, user_id, status, items, shipping_address, billing_address,
		       subtotal_amount, subtotal_currency, tax_amount, tax_currency,
		       shipping_amount, shipping_currency, total_amount, total_currency,
		       payment_id, notes, created_at, updated_at, shipped_at, delivered_at
		FROM orders
		WHERE id = $1 AND deleted_at IS NULL
	`

	var order models.Order
	var itemsJSON, shippingJSON, billingJSON []byte
	var shippedAt, deliveredAt sql.NullTime
	var paymentID, notes sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&order.ID,
		&order.UserID,
		&order.Status,
		&itemsJSON,
		&shippingJSON,
		&billingJSON,
		&order.Subtotal.Amount,
		&order.Subtotal.Currency,
		&order.Tax.Amount,
		&order.Tax.Currency,
		&order.ShippingCost.Amount,
		&order.ShippingCost.Currency,
		&order.Total.Amount,
		&order.Total.Currency,
		&paymentID,
		&notes,
		&order.CreatedAt,
		&order.UpdatedAt,
		&shippedAt,
		&deliveredAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.ErrNotFound
	}
	if err != nil {
		r.logger.Error("Failed to fetch order", logging.Fields{
			"order_id": id,
			"error":    err.Error(),
		})
		return nil, err
	}

	if err := json.Unmarshal(itemsJSON, &order.Items); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(shippingJSON, &order.ShippingAddress); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(billingJSON, &order.BillingAddress); err != nil {
		return nil, err
	}

	if paymentID.Valid {
		order.PaymentID = paymentID.String
	}
	if notes.Valid {
		order.Notes = notes.String
	}
	if shippedAt.Valid {
		order.ShippedAt = &shippedAt.Time
	}
	if deliveredAt.Valid {
		order.DeliveredAt = &deliveredAt.Time
	}

	r.logger.Info("Order fetched successfully", logging.Fields{
		"order_id": order.ID,
		"status":   order.Status,
	})

	return &order, nil
}

// Create creates a new order.
func (r *PostgresOrderRepository) Create(ctx context.Context, req *models.CreateOrderRequest) (*models.Order, error) {
	r.logger.Debug("Creating new order", logging.Fields{"user_id": req.UserID})

	// TODO(TEAM-API): Add idempotency key support
	order := &models.Order{
		ID:              generateOrderID(),
		UserID:          req.UserID,
		Status:          models.OrderStatusPending,
		Items:           req.Items,
		ShippingAddress: req.ShippingAddress,
		BillingAddress:  req.BillingAddress,
		Notes:           req.Notes,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	order.CalculateTotal()

	itemsJSON, err := json.Marshal(order.Items)
	if err != nil {
		return nil, err
	}

	shippingJSON, err := json.Marshal(order.ShippingAddress)
	if err != nil {
		return nil, err
	}

	billingJSON, err := json.Marshal(order.BillingAddress)
	if err != nil {
		return nil, err
	}

	query := `
		INSERT INTO orders (
			id, user_id, status, items, shipping_address, billing_address,
			subtotal_amount, subtotal_currency, tax_amount, tax_currency,
			shipping_amount, shipping_currency, total_amount, total_currency,
			notes, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
		)
	`

	_, err = r.db.ExecContext(ctx, query,
		order.ID,
		order.UserID,
		order.Status,
		itemsJSON,
		shippingJSON,
		billingJSON,
		order.Subtotal.Amount,
		order.Subtotal.Currency,
		order.Tax.Amount,
		order.Tax.Currency,
		order.ShippingCost.Amount,
		order.ShippingCost.Currency,
		order.Total.Amount,
		order.Total.Currency,
		order.Notes,
		order.CreatedAt,
		order.UpdatedAt,
	)

	if err != nil {
		r.logger.Error("Failed to create order", logging.Fields{
			"user_id": req.UserID,
			"error":   err.Error(),
		})
		return nil, err
	}

	r.logger.Info("Order created successfully", logging.Fields{
		"order_id": order.ID,
		"user_id":  order.UserID,
		"total":    order.Total.Amount,
	})

	return order, nil
}

// UpdateStatus updates the status of an order.
func (r *PostgresOrderRepository) UpdateStatus(ctx context.Context, id string, req *models.UpdateOrderStatusRequest) (*models.Order, error) {
	r.logger.Debug("Updating order status", logging.Fields{
		"order_id":   id,
		"new_status": req.Status,
	})

	now := time.Now()

	var shippedAt, deliveredAt *time.Time
	if req.Status == models.OrderStatusShipped {
		shippedAt = &now
	} else if req.Status == models.OrderStatusDelivered {
		deliveredAt = &now
	}

	query := `
		UPDATE orders
		SET status = $2, notes = COALESCE($3, notes), updated_at = $4,
		    shipped_at = COALESCE($5, shipped_at),
		    delivered_at = COALESCE($6, delivered_at)
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id
	`

	var returnedID string
	err := r.db.QueryRowContext(ctx, query, id, req.Status, req.Notes, now, shippedAt, deliveredAt).Scan(&returnedID)
	if err == sql.ErrNoRows {
		return nil, errors.ErrNotFound
	}
	if err != nil {
		r.logger.Error("Failed to update order status", logging.Fields{
			"order_id": id,
			"error":    err.Error(),
		})
		return nil, err
	}

	r.logger.Info("Order status updated", logging.Fields{
		"order_id":   id,
		"new_status": req.Status,
	})

	return r.GetByID(ctx, id)
}

// List retrieves orders based on filter criteria.
func (r *PostgresOrderRepository) List(ctx context.Context, filter *models.OrderListFilter) ([]*models.Order, int, error) {
	r.logger.Debug("Listing orders", logging.Fields{
		"user_id": filter.UserID,
		"status":  filter.Status,
		"limit":   filter.Limit,
		"offset":  filter.Offset,
	})

	// TODO(TEAM-API): Add proper query builder
	baseQuery := `
		FROM orders
		WHERE deleted_at IS NULL
	`
	args := make([]interface{}, 0)
	argIdx := 1

	if filter.UserID != "" {
		baseQuery += " AND user_id = $" + string(rune('0'+argIdx))
		args = append(args, filter.UserID)
		argIdx++
	}

	if filter.Status != nil {
		baseQuery += " AND status = $" + string(rune('0'+argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}

	// Get total count
	var total int
	countQuery := "SELECT COUNT(*) " + baseQuery
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Get orders
	selectQuery := `
		SELECT id, user_id, status, items, shipping_address, billing_address,
		       subtotal_amount, subtotal_currency, tax_amount, tax_currency,
		       shipping_amount, shipping_currency, total_amount, total_currency,
		       payment_id, notes, created_at, updated_at, shipped_at, delivered_at
	` + baseQuery + " ORDER BY created_at DESC LIMIT $" + string(rune('0'+argIdx)) + " OFFSET $" + string(rune('0'+argIdx+1))

	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.QueryContext(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	orders := make([]*models.Order, 0)
	for rows.Next() {
		order, err := r.scanOrder(rows)
		if err != nil {
			return nil, 0, err
		}
		orders = append(orders, order)
	}

	r.logger.Info("Orders listed", logging.Fields{
		"count": len(orders),
		"total": total,
	})

	return orders, total, nil
}

// GetByUserID retrieves all orders for a specific user.
func (r *PostgresOrderRepository) GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.Order, int, error) {
	// TODO(TEAM-PLATFORM): Optimize with prepared statements
	logging.Infof("Fetching orders for user: %s", userID)

	filter := &models.OrderListFilter{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	}

	return r.List(ctx, filter)
}

// Delete soft-deletes an order.
func (r *PostgresOrderRepository) Delete(ctx context.Context, id string) error {
	r.logger.Debug("Deleting order", logging.Fields{"order_id": id})

	query := `
		UPDATE orders
		SET deleted_at = $2, status = $3, updated_at = $2
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, id, time.Now(), models.OrderStatusCancelled)
	if err != nil {
		r.logger.Error("Failed to delete order", logging.Fields{
			"order_id": id,
			"error":    err.Error(),
		})
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.ErrNotFound
	}

	r.logger.Info("Order deleted", logging.Fields{"order_id": id})
	return nil
}

// SetPaymentID associates a payment with an order.
func (r *PostgresOrderRepository) SetPaymentID(ctx context.Context, orderID, paymentID string) error {
	r.logger.Debug("Setting payment ID", logging.Fields{
		"order_id":   orderID,
		"payment_id": paymentID,
	})

	query := `
		UPDATE orders
		SET payment_id = $2, updated_at = $3
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, orderID, paymentID, time.Now())
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.ErrNotFound
	}

	r.logger.Info("Payment ID set", logging.Fields{
		"order_id":   orderID,
		"payment_id": paymentID,
	})

	return nil
}

func (r *PostgresOrderRepository) scanOrder(rows *sql.Rows) (*models.Order, error) {
	var order models.Order
	var itemsJSON, shippingJSON, billingJSON []byte
	var shippedAt, deliveredAt sql.NullTime
	var paymentID, notes sql.NullString

	err := rows.Scan(
		&order.ID,
		&order.UserID,
		&order.Status,
		&itemsJSON,
		&shippingJSON,
		&billingJSON,
		&order.Subtotal.Amount,
		&order.Subtotal.Currency,
		&order.Tax.Amount,
		&order.Tax.Currency,
		&order.ShippingCost.Amount,
		&order.ShippingCost.Currency,
		&order.Total.Amount,
		&order.Total.Currency,
		&paymentID,
		&notes,
		&order.CreatedAt,
		&order.UpdatedAt,
		&shippedAt,
		&deliveredAt,
	)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(itemsJSON, &order.Items); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(shippingJSON, &order.ShippingAddress); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(billingJSON, &order.BillingAddress); err != nil {
		return nil, err
	}

	if paymentID.Valid {
		order.PaymentID = paymentID.String
	}
	if notes.Valid {
		order.Notes = notes.String
	}
	if shippedAt.Valid {
		order.ShippedAt = &shippedAt.Time
	}
	if deliveredAt.Valid {
		order.DeliveredAt = &deliveredAt.Time
	}

	return &order, nil
}

func generateOrderID() string {
	// TODO(TEAM-API): Use proper UUID or ULID generation
	return "ord_" + time.Now().Format("20060102150405")
}
