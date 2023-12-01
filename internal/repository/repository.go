package repository

import (
	"context"

	"github.com/tm-acme-shop/acme-shop-shared-go/interfaces"
	"github.com/tm-acme-shop/acme-shop-shared-go/models"
)

// Ensure PostgresOrderRepository implements interfaces.OrderRepository
var _ interfaces.OrderRepository = (*PostgresOrderRepository)(nil)

// OrderCache defines caching operations for orders.
type OrderCache interface {
	Get(ctx context.Context, id string) (*models.Order, error)
	Set(ctx context.Context, order *models.Order) error
	Delete(ctx context.Context, id string) error
	GetByUserID(ctx context.Context, userID string) ([]*models.Order, error)
	SetByUserID(ctx context.Context, userID string, orders []*models.Order) error
	InvalidateByUserID(ctx context.Context, userID string) error
}

// OrderRepositoryV1 is the deprecated legacy order repository interface.
// Deprecated: Use interfaces.OrderRepository instead.
// TODO(TEAM-API): Remove after v1 API migration complete
type OrderRepositoryV1 interface {
	// GetOrderByID retrieves an order by ID using the legacy format.
	// Deprecated: Use OrderRepository.GetByID instead.
	GetOrderByID(ctx context.Context, id int64) (*LegacyOrder, error)

	// CreateOrder creates a new order using legacy format.
	// Deprecated: Use OrderRepository.Create instead.
	CreateOrder(ctx context.Context, req *LegacyCreateOrderRequest) (*LegacyOrder, error)

	// UpdateOrderStatus updates order status.
	// Deprecated: Use OrderRepository.UpdateStatus instead.
	UpdateOrderStatus(ctx context.Context, id int64, status string) error

	// GetOrdersByUserID retrieves orders for a user.
	// Deprecated: Use OrderRepository.GetByUserID instead.
	GetOrdersByUserID(ctx context.Context, userID int64) ([]*LegacyOrder, error)
}

// LegacyOrder is the deprecated order format.
// Deprecated: Use models.Order instead.
// TODO(TEAM-API): Remove after migration
type LegacyOrder struct {
	ID         int64   `json:"id"`
	UserID     int64   `json:"user_id"`
	Status     string  `json:"status"`
	TotalPrice float64 `json:"total_price"`
	Currency   string  `json:"currency"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
}

// LegacyCreateOrderRequest is the deprecated order creation request.
// Deprecated: Use models.CreateOrderRequest instead.
type LegacyCreateOrderRequest struct {
	UserID     int64              `json:"user_id"`
	Items      []LegacyOrderItem  `json:"items"`
	TotalPrice float64            `json:"total_price"`
	Currency   string             `json:"currency"`
}

// LegacyOrderItem is the deprecated order item format.
// Deprecated: Use models.OrderItem instead.
type LegacyOrderItem struct {
	ProductID int64   `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}
