package service

import (
	"context"
	"fmt"

	"github.com/tm-acme-shop/acme-shop-orders-service/internal/clients"
	"github.com/tm-acme-shop/acme-shop-orders-service/internal/config"
	"github.com/tm-acme-shop/acme-shop-orders-service/internal/repository"
	"github.com/tm-acme-shop/acme-shop-shared-go/errors"
	"github.com/tm-acme-shop/acme-shop-shared-go/interfaces"
	"github.com/tm-acme-shop/acme-shop-shared-go/logging"
	"github.com/tm-acme-shop/acme-shop-shared-go/models"
)

// OrderService handles order business logic.
type OrderService struct {
	orderRepo           interfaces.OrderRepository
	orderCache          repository.OrderCache
	legacyRepo          repository.OrderRepositoryV1
	paymentClient       interfaces.PaymentClient
	legacyPaymentClient interfaces.LegacyPaymentClient
	userClient          *clients.HTTPUserClient
	notificationClient  interfaces.NotificationSender
	eventPublisher      interfaces.OrderEventPublisher
	config              *config.Config
	logger              *logging.LoggerV2
}

// NewOrderService creates a new order service.
func NewOrderService(
	orderRepo interfaces.OrderRepository,
	orderCache repository.OrderCache,
	legacyRepo repository.OrderRepositoryV1,
	paymentClient interfaces.PaymentClient,
	legacyPaymentClient interfaces.LegacyPaymentClient,
	userClient *clients.HTTPUserClient,
	notificationClient interfaces.NotificationSender,
	eventPublisher interfaces.OrderEventPublisher,
	cfg *config.Config,
) *OrderService {
	return &OrderService{
		orderRepo:           orderRepo,
		orderCache:          orderCache,
		legacyRepo:          legacyRepo,
		paymentClient:       paymentClient,
		legacyPaymentClient: legacyPaymentClient,
		userClient:          userClient,
		notificationClient:  notificationClient,
		eventPublisher:      eventPublisher,
		config:              cfg,
		logger:              logging.NewLoggerV2("order-service"),
	}
}

// CreateOrder creates a new order.
func (s *OrderService) CreateOrder(ctx context.Context, req *models.CreateOrderRequest) (*models.Order, error) {
	s.logger.Info("Creating order", logging.Fields{
		"user_id":    req.UserID,
		"item_count": len(req.Items),
	})

	// Calculate subtotal from order items
	var subtotal float64
	for _, item := range req.Items {
		subtotal += item.Total.ToFloat()
	}

	// Calculate order totals using configured tax rate
	_ = CalculateOrderTotal(subtotal, s.config.TaxRate)

	// Validate user exists
	valid, err := s.userClient.ValidateUser(ctx, req.UserID)
	if err != nil {
		s.logger.Error("Failed to validate user", logging.Fields{
			"user_id": req.UserID,
			"error":   err.Error(),
		})
		return nil, err
	}
	if !valid {
		return nil, errors.NewValidationError("user_id", "user not found or inactive")
	}

	// Validate order
	if err := ValidateCreateOrderRequest(req); err != nil {
		return nil, err
	}

	// Create order
	order, err := s.orderRepo.Create(ctx, req)
	if err != nil {
		s.logger.Error("Failed to create order", logging.Fields{
			"user_id": req.UserID,
			"error":   err.Error(),
		})
		return nil, err
	}

	// Cache the order
	if s.config.Features.EnableOrderCaching {
		if err := s.orderCache.Set(ctx, order); err != nil {
			// Log but don't fail
			s.logger.Error("Failed to cache order", logging.Fields{
				"order_id": order.ID,
				"error":    err.Error(),
			})
		}
		// Invalidate user's order list cache
		s.orderCache.InvalidateByUserID(ctx, order.UserID)
	}

	// Publish event
	if s.config.Features.EnableOrderEvents {
		if err := s.eventPublisher.PublishOrderCreated(ctx, order); err != nil {
			// Log but don't fail
			s.logger.Error("Failed to publish order created event", logging.Fields{
				"order_id": order.ID,
				"error":    err.Error(),
			})
		}
	}

	// Send notification
	go s.sendOrderConfirmationNotification(context.Background(), order)

	s.logger.Info("Order created successfully", logging.Fields{
		"order_id": order.ID,
		"total":    order.Total.Amount,
	})

	return order, nil
}

// GetOrder retrieves an order by ID.
func (s *OrderService) GetOrder(ctx context.Context, id string) (*models.Order, error) {
	s.logger.Debug("Getting order", logging.Fields{"order_id": id})

	// Check cache first
	if s.config.Features.EnableOrderCaching {
		if order, err := s.orderCache.Get(ctx, id); err == nil && order != nil {
			s.logger.Debug("Order found in cache", logging.Fields{"order_id": id})
			return order, nil
		}
	}

	// Get from database
	order, err := s.orderRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, errors.ErrNotFound
	}

	// Cache for next time
	if s.config.Features.EnableOrderCaching {
		s.orderCache.Set(ctx, order)
	}

	return order, nil
}

// GetOrderV1 retrieves an order using the deprecated v1 format.
// Deprecated: Use GetOrder instead.
// TODO(TEAM-API): Remove after v1 API migration complete
func (s *OrderService) GetOrderV1(ctx context.Context, id int64) (*repository.LegacyOrder, error) {
	// TODO(TEAM-API): Migrate callers to GetOrder
	logging.Infof("Legacy: Getting order v1: %d", id)
	return s.legacyRepo.GetOrderByID(ctx, id)
}

// UpdateOrderStatus updates the status of an order.
func (s *OrderService) UpdateOrderStatus(ctx context.Context, id string, req *models.UpdateOrderStatusRequest) (*models.Order, error) {
	s.logger.Info("Updating order status", logging.Fields{
		"order_id":   id,
		"new_status": req.Status,
	})

	// Get current order to check status transition
	currentOrder, err := s.orderRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if currentOrder == nil {
		return nil, errors.ErrNotFound
	}

	// Validate status transition
	if !isValidStatusTransition(currentOrder.Status, req.Status) {
		return nil, errors.NewValidationError("status", fmt.Sprintf(
			"invalid status transition from %s to %s",
			currentOrder.Status,
			req.Status,
		))
	}

	previousStatus := currentOrder.Status

	// Update status
	order, err := s.orderRepo.UpdateStatus(ctx, id, req)
	if err != nil {
		return nil, err
	}

	// Invalidate cache
	if s.config.Features.EnableOrderCaching {
		s.orderCache.Delete(ctx, id)
		s.orderCache.InvalidateByUserID(ctx, order.UserID)
	}

	// Publish event
	if s.config.Features.EnableOrderEvents {
		if err := s.eventPublisher.PublishOrderStatusChanged(ctx, order, previousStatus); err != nil {
			s.logger.Error("Failed to publish status change event", logging.Fields{
				"order_id": order.ID,
				"error":    err.Error(),
			})
		}
	}

	// Send notification for important status changes
	go s.sendStatusChangeNotification(context.Background(), order, previousStatus)

	return order, nil
}

// CancelOrder cancels an order.
func (s *OrderService) CancelOrder(ctx context.Context, id string, reason string) (*models.Order, error) {
	s.logger.Info("Cancelling order", logging.Fields{
		"order_id": id,
		"reason":   reason,
	})

	// Get current order
	order, err := s.orderRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, errors.ErrNotFound
	}

	// Check if cancellation is allowed
	if !order.CanCancel() {
		return nil, errors.NewValidationError("status", "order cannot be cancelled in current state")
	}

	// Cancel any pending payment
	if order.PaymentID != "" {
		payment, err := s.paymentClient.GetPaymentStatus(ctx, order.PaymentID)
		if err != nil {
			s.logger.Error("Failed to get payment status", logging.Fields{
				"payment_id": order.PaymentID,
				"error":      err.Error(),
			})
		} else if payment != nil && payment.Status == models.PaymentStatusPending {
			if err := s.paymentClient.CancelPayment(ctx, order.PaymentID); err != nil {
				s.logger.Error("Failed to cancel payment", logging.Fields{
					"payment_id": order.PaymentID,
					"error":      err.Error(),
				})
			}
		}
	}

	// Update order status
	previousStatus := order.Status
	req := &models.UpdateOrderStatusRequest{
		Status: models.OrderStatusCancelled,
		Notes:  reason,
	}

	order, err = s.orderRepo.UpdateStatus(ctx, id, req)
	if err != nil {
		return nil, err
	}

	// Invalidate cache
	if s.config.Features.EnableOrderCaching {
		s.orderCache.Delete(ctx, id)
		s.orderCache.InvalidateByUserID(ctx, order.UserID)
	}

	// Publish event
	if s.config.Features.EnableOrderEvents {
		if err := s.eventPublisher.PublishOrderCancelled(ctx, order, reason); err != nil {
			s.logger.Error("Failed to publish order cancelled event", logging.Fields{
				"order_id": order.ID,
				"error":    err.Error(),
			})
		}
	}

	go s.sendCancellationNotification(context.Background(), order, previousStatus)

	return order, nil
}

// ListOrders retrieves orders based on filter criteria.
func (s *OrderService) ListOrders(ctx context.Context, filter *models.OrderListFilter) ([]*models.Order, int, error) {
	s.logger.Debug("Listing orders", logging.Fields{
		"user_id": filter.UserID,
		"status":  filter.Status,
	})

	// Set defaults
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}

	return s.orderRepo.List(ctx, filter)
}

// GetUserOrders retrieves orders for a specific user.
func (s *OrderService) GetUserOrders(ctx context.Context, userID string, limit, offset int) ([]*models.Order, int, error) {
	s.logger.Debug("Getting user orders", logging.Fields{
		"user_id": userID,
		"limit":   limit,
		"offset":  offset,
	})

	// Check cache first
	if s.config.Features.EnableOrderCaching && offset == 0 {
		if orders, err := s.orderCache.GetByUserID(ctx, userID); err == nil && orders != nil {
			s.logger.Debug("User orders found in cache", logging.Fields{"user_id": userID})
			return orders, len(orders), nil
		}
	}

	orders, total, err := s.orderRepo.GetByUserID(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	// Cache if first page
	if s.config.Features.EnableOrderCaching && offset == 0 {
		s.orderCache.SetByUserID(ctx, userID, orders)
	}

	return orders, total, nil
}

// GetUserOrdersV1 retrieves orders using the deprecated v1 format.
// Deprecated: Use GetUserOrders instead.
// TODO(TEAM-API): Remove after v1 API migration complete
func (s *OrderService) GetUserOrdersV1(ctx context.Context, userID int64) ([]*repository.LegacyOrder, error) {
	// TODO(TEAM-API): Migrate callers to GetUserOrders
	logging.Infof("Legacy: Getting orders for user v1: %d", userID)
	return s.legacyRepo.GetOrdersByUserID(ctx, userID)
}

// ProcessOrderPayment processes payment for an order.
func (s *OrderService) ProcessOrderPayment(ctx context.Context, orderID string, paymentReq *models.ProcessPaymentRequest) (*models.ProcessPaymentResponse, error) {
	s.logger.Info("Processing order payment", logging.Fields{
		"order_id": orderID,
		"method":   paymentReq.Method,
	})

	// Get the order
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, errors.ErrNotFound
	}

	// Ensure order is in pending state
	if order.Status != models.OrderStatusPending {
		return nil, errors.NewValidationError("status", "order is not in pending state")
	}

	// Set order ID and amount from order
	paymentReq.OrderID = orderID
	paymentReq.Amount = order.Total

	// Process payment
	var paymentResp *models.ProcessPaymentResponse
	
	// TODO(TEAM-PAYMENTS): Remove legacy payment path after migration
	if s.config.Features.EnableLegacyPayments && paymentReq.Method == models.PaymentMethodBankTransfer {
		// Use legacy payment client for bank transfers (temporary)
		legacyReq := &models.LegacyPaymentRequest{
			OrderID:  orderID,
			Amount:   order.Total.ToFloat(),
			Currency: order.Total.Currency,
		}
		txnID, err := s.legacyPaymentClient.ProcessLegacyPayment(ctx, legacyReq)
		if err != nil {
			return nil, err
		}
		paymentResp = &models.ProcessPaymentResponse{
			PaymentID: txnID,
			Status:    models.PaymentStatusPending,
		}
	} else {
		paymentResp, err = s.paymentClient.ProcessPayment(ctx, paymentReq)
		if err != nil {
			s.logger.Error("Payment processing failed", logging.Fields{
				"order_id": orderID,
				"error":    err.Error(),
			})
			return nil, err
		}
	}

	// Update order with payment ID
	if err := s.orderRepo.SetPaymentID(ctx, orderID, paymentResp.PaymentID); err != nil {
		s.logger.Error("Failed to set payment ID on order", logging.Fields{
			"order_id":   orderID,
			"payment_id": paymentResp.PaymentID,
			"error":      err.Error(),
		})
	}

	// Update order status if payment completed
	if paymentResp.Status == models.PaymentStatusCompleted {
		s.UpdateOrderStatus(ctx, orderID, &models.UpdateOrderStatusRequest{
			Status: models.OrderStatusConfirmed,
			Notes:  "Payment completed",
		})
	}

	// Invalidate cache
	if s.config.Features.EnableOrderCaching {
		s.orderCache.Delete(ctx, orderID)
	}

	return paymentResp, nil
}

// RefundOrder processes a refund for an order.
func (s *OrderService) RefundOrder(ctx context.Context, orderID string, reason string) (*models.RefundResponse, error) {
	s.logger.Info("Processing order refund", logging.Fields{
		"order_id": orderID,
		"reason":   reason,
	})

	// Get the order
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, errors.ErrNotFound
	}

	// Check if refund is allowed
	if !order.CanRefund() {
		return nil, errors.NewValidationError("status", "order cannot be refunded")
	}

	// Process refund
	refundReq := &models.RefundRequest{
		PaymentID: order.PaymentID,
		Amount:    order.Total,
		Reason:    reason,
	}

	refundResp, err := s.paymentClient.Refund(ctx, refundReq)
	if err != nil {
		s.logger.Error("Refund processing failed", logging.Fields{
			"order_id":   orderID,
			"payment_id": order.PaymentID,
			"error":      err.Error(),
		})
		return nil, err
	}

	// Update order status
	if refundResp.Status == models.PaymentStatusRefunded {
		s.UpdateOrderStatus(ctx, orderID, &models.UpdateOrderStatusRequest{
			Status: models.OrderStatusRefunded,
			Notes:  "Refund processed: " + reason,
		})
	}

	return refundResp, nil
}

// HandlePaymentWebhook handles incoming payment webhooks.
func (s *OrderService) HandlePaymentWebhook(ctx context.Context, payload []byte, signature string) error {
	s.logger.Debug("Handling payment webhook", logging.Fields{
		"payload_size": len(payload),
	})

	// Validate webhook
	valid, err := s.paymentClient.ValidateWebhook(ctx, payload, signature)
	if err != nil {
		return err
	}
	if !valid {
		return errors.NewValidationError("signature", "invalid webhook signature")
	}

	// TODO(TEAM-PAYMENTS): Parse and process webhook payload
	s.logger.Info("Payment webhook processed")
	return nil
}

func (s *OrderService) sendOrderConfirmationNotification(ctx context.Context, order *models.Order) {
	// TODO(TEAM-NOTIFICATIONS): Use template-based notifications
	req := &models.SendNotificationRequest{
		Type:      models.NotificationTypeOrderConfirmation,
		Priority:  models.NotificationPriorityNormal,
		Recipient: order.UserID,
		Subject:   "Order Confirmation",
		Body:      fmt.Sprintf("Your order %s has been received.", order.ID),
		Metadata: map[string]string{
			"order_id": order.ID,
			"total":    fmt.Sprintf("%.2f %s", order.Total.ToFloat(), order.Total.Currency),
		},
	}

	if _, err := s.notificationClient.Send(ctx, req); err != nil {
		s.logger.Error("Failed to send order confirmation", logging.Fields{
			"order_id": order.ID,
			"error":    err.Error(),
		})
	}
}

func (s *OrderService) sendStatusChangeNotification(ctx context.Context, order *models.Order, previousStatus models.OrderStatus) {
	var notificationType models.NotificationType
	var subject, body string

	switch order.Status {
	case models.OrderStatusShipped:
		notificationType = models.NotificationTypeOrderShipped
		subject = "Order Shipped"
		body = fmt.Sprintf("Your order %s has been shipped.", order.ID)
	case models.OrderStatusDelivered:
		notificationType = models.NotificationTypeOrderDelivered
		subject = "Order Delivered"
		body = fmt.Sprintf("Your order %s has been delivered.", order.ID)
	default:
		return // No notification for other status changes
	}

	req := &models.SendNotificationRequest{
		Type:      notificationType,
		Priority:  models.NotificationPriorityNormal,
		Recipient: order.UserID,
		Subject:   subject,
		Body:      body,
	}

	if _, err := s.notificationClient.Send(ctx, req); err != nil {
		s.logger.Error("Failed to send status change notification", logging.Fields{
			"order_id": order.ID,
			"error":    err.Error(),
		})
	}
}

func (s *OrderService) sendCancellationNotification(ctx context.Context, order *models.Order, previousStatus models.OrderStatus) {
	req := &models.SendNotificationRequest{
		Type:      models.NotificationTypeOrderCancelled,
		Priority:  models.NotificationPriorityNormal,
		Recipient: order.UserID,
		Subject:   "Order Cancelled",
		Body:      fmt.Sprintf("Your order %s has been cancelled.", order.ID),
		Metadata: map[string]string{
			"order_id": order.ID,
			"reason":   order.Notes,
		},
	}

	if _, err := s.notificationClient.Send(ctx, req); err != nil {
		s.logger.Error("Failed to send cancellation notification", logging.Fields{
			"order_id": order.ID,
			"error":    err.Error(),
		})
	}
}

func isValidStatusTransition(from, to models.OrderStatus) bool {
	validTransitions := map[models.OrderStatus][]models.OrderStatus{
		models.OrderStatusPending:    {models.OrderStatusConfirmed, models.OrderStatusCancelled},
		models.OrderStatusConfirmed:  {models.OrderStatusProcessing, models.OrderStatusCancelled},
		models.OrderStatusProcessing: {models.OrderStatusShipped, models.OrderStatusCancelled},
		models.OrderStatusShipped:    {models.OrderStatusDelivered},
		models.OrderStatusDelivered:  {models.OrderStatusRefunded},
		models.OrderStatusCancelled:  {},
		models.OrderStatusRefunded:   {},
	}

	allowed, ok := validTransitions[from]
	if !ok {
		return false
	}

	for _, status := range allowed {
		if status == to {
			return true
		}
	}
	return false
}
