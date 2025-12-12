package service

import (
	"context"

	"github.com/tm-acme-shop/acme-shop-orders-service/internal/config"
	"github.com/tm-acme-shop/acme-shop-shared-go/errors"
	"github.com/tm-acme-shop/acme-shop-shared-go/interfaces"
	"github.com/tm-acme-shop/acme-shop-shared-go/logging"
	"github.com/tm-acme-shop/acme-shop-shared-go/models"
)

// PaymentService handles payment-related business logic.
type PaymentService struct {
	paymentClient       interfaces.PaymentClient
	legacyPaymentClient interfaces.LegacyPaymentClient
	orderRepo           interfaces.OrderRepository
	config              *config.Config
	logger              *logging.LoggerV2
}

// NewPaymentService creates a new payment service.
func NewPaymentService(
	paymentClient interfaces.PaymentClient,
	legacyPaymentClient interfaces.LegacyPaymentClient,
	orderRepo interfaces.OrderRepository,
	cfg *config.Config,
) *PaymentService {
	return &PaymentService{
		paymentClient:       paymentClient,
		legacyPaymentClient: legacyPaymentClient,
		orderRepo:           orderRepo,
		config:              cfg,
		logger:              logging.NewLoggerV2("payment-service"),
	}
}

// GetPaymentStatus retrieves the status of a payment.
func (s *PaymentService) GetPaymentStatus(ctx context.Context, paymentID string) (*models.Payment, error) {
	s.logger.Debug("Getting payment status", logging.Fields{"payment_id": paymentID})

	payment, err := s.paymentClient.GetPaymentStatus(ctx, paymentID)
	if err != nil {
		s.logger.Error("Failed to get payment status", logging.Fields{
			"payment_id": paymentID,
			"error":      err.Error(),
		})
		return nil, err
	}

	if payment == nil {
		return nil, errors.ErrNotFound
	}

	return payment, nil
}

// GetPaymentStatusV1 retrieves payment status using the deprecated v1 format.
// Deprecated: Use GetPaymentStatus instead.
// TODO(TEAM-PAYMENTS): Remove after v1 API migration complete
func (s *PaymentService) GetPaymentStatusV1(ctx context.Context, orderID string) (string, error) {
	// TODO(TEAM-PAYMENTS): Migrate callers to GetPaymentStatus
	logging.Infof("Legacy: Getting payment status for order: %s", orderID)
	return s.legacyPaymentClient.GetStatus(ctx, orderID)
}

// GetPaymentByOrderID retrieves the payment associated with an order.
func (s *PaymentService) GetPaymentByOrderID(ctx context.Context, orderID string) (*models.Payment, error) {
	s.logger.Debug("Getting payment for order", logging.Fields{"order_id": orderID})

	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, errors.ErrNotFound
	}

	if order.PaymentID == "" {
		return nil, nil
	}

	return s.paymentClient.GetPaymentStatus(ctx, order.PaymentID)
}

// ProcessRefund processes a refund for a payment.
func (s *PaymentService) ProcessRefund(ctx context.Context, paymentID string, amount models.Money, reason string) (*models.RefundResponse, error) {
	s.logger.Info("Processing refund", logging.Fields{
		"payment_id": paymentID,
		"amount":     amount.Amount,
		"reason":     reason,
	})

	// Validate payment exists and can be refunded
	payment, err := s.paymentClient.GetPaymentStatus(ctx, paymentID)
	if err != nil {
		return nil, err
	}
	if payment == nil {
		return nil, errors.ErrNotFound
	}

	if !payment.CanRefund() {
		return nil, errors.NewValidationError("payment_id", "payment cannot be refunded")
	}

	// Validate refund amount
	if amount.Amount <= 0 {
		return nil, errors.NewValidationError("amount", "refund amount must be positive")
	}
	if amount.Amount > payment.Amount.Amount {
		return nil, errors.NewValidationError("amount", "refund amount exceeds payment amount")
	}

	// Process refund
	refundReq := &models.RefundRequest{
		PaymentID: paymentID,
		Amount:    amount,
		Reason:    reason,
	}

	refundResp, err := s.paymentClient.Refund(ctx, refundReq)
	if err != nil {
		s.logger.Error("Refund failed", logging.Fields{
			"payment_id": paymentID,
			"error":      err.Error(),
		})
		return nil, err
	}

	s.logger.Info("Refund processed", logging.Fields{
		"payment_id": paymentID,
		"refund_id":  refundResp.RefundID,
		"status":     refundResp.Status,
	})

	return refundResp, nil
}

// CancelPayment cancels a pending payment.
func (s *PaymentService) CancelPayment(ctx context.Context, paymentID string) error {
	s.logger.Info("Cancelling payment", logging.Fields{"payment_id": paymentID})

	// Validate payment exists and is pending
	payment, err := s.paymentClient.GetPaymentStatus(ctx, paymentID)
	if err != nil {
		return err
	}
	if payment == nil {
		return errors.ErrNotFound
	}

	if payment.Status != models.PaymentStatusPending {
		return errors.NewValidationError("payment_id", "only pending payments can be cancelled")
	}

	if err := s.paymentClient.CancelPayment(ctx, paymentID); err != nil {
		s.logger.Error("Failed to cancel payment", logging.Fields{
			"payment_id": paymentID,
			"error":      err.Error(),
		})
		return err
	}

	s.logger.Info("Payment cancelled", logging.Fields{"payment_id": paymentID})
	return nil
}

// ValidatePaymentMethod validates a payment method.
func (s *PaymentService) ValidatePaymentMethod(method models.PaymentMethod) error {
	switch method {
	case models.PaymentMethodCreditCard,
		models.PaymentMethodDebitCard,
		models.PaymentMethodPayPal,
		models.PaymentMethodBankTransfer:
		return nil
	case models.PaymentMethodCrypto:
		// TODO(TEAM-PAYMENTS): Enable crypto payments
		return errors.NewValidationError("method", "crypto payments not yet supported")
	default:
		return errors.NewValidationError("method", "invalid payment method")
	}
}

// ProcessWebhook processes a payment webhook.
func (s *PaymentService) ProcessWebhook(ctx context.Context, payload []byte, signature string) error {
	s.logger.Debug("Processing payment webhook")

	// Validate signature
	valid, err := s.paymentClient.ValidateWebhook(ctx, payload, signature)
	if err != nil {
		s.logger.Error("Webhook validation failed", logging.Fields{"error": err.Error()})
		return err
	}
	if !valid {
		return errors.NewValidationError("signature", "invalid webhook signature")
	}

	// TODO(TEAM-PAYMENTS): Parse webhook payload and update order status
	s.logger.Info("Payment webhook processed")
	return nil
}

// LegacyProcessPayment processes a payment using the deprecated format.
// Deprecated: Use ProcessPayment in OrderService instead.
// TODO(TEAM-PAYMENTS): Remove after migration complete
func (s *PaymentService) LegacyProcessPayment(ctx context.Context, orderID string, amount float64, currency string) (string, error) {
	// TODO(TEAM-PAYMENTS): Migrate callers to new payment processing
	logging.Infof("Legacy: Processing payment for order: %s", orderID)

	req := &models.LegacyPaymentRequest{
		OrderID:  orderID,
		Amount:   amount,
		Currency: currency,
	}

	return s.legacyPaymentClient.ProcessLegacyPayment(ctx, req)
}
