package service

import (
	"strings"

	"github.com/tm-acme-shop/acme-shop-shared-go/errors"
	"github.com/tm-acme-shop/acme-shop-shared-go/logging"
	"github.com/tm-acme-shop/acme-shop-shared-go/models"
)

// ValidateCreateOrderRequest validates an order creation request.
func ValidateCreateOrderRequest(req *models.CreateOrderRequest) error {
	if req.UserID == "" {
		return errors.NewValidationError("user_id", "user ID is required")
	}

	if len(req.Items) == 0 {
		return errors.NewValidationError("items", "at least one item is required")
	}

	// Validate each item
	for i, item := range req.Items {
		if err := validateOrderItem(&item, i); err != nil {
			return err
		}
	}

	// Validate shipping address
	if err := validateAddress(&req.ShippingAddress, "shipping_address"); err != nil {
		return err
	}

	// Validate billing address
	if err := validateAddress(&req.BillingAddress, "billing_address"); err != nil {
		return err
	}

	return nil
}

func validateOrderItem(item *models.OrderItem, index int) error {
	if item.ProductID == "" {
		return errors.NewValidationError("items", "product ID is required for item")
	}

	if item.Quantity <= 0 {
		return errors.NewValidationError("items", "quantity must be positive")
	}

	if item.UnitPrice.Amount < 0 {
		return errors.NewValidationError("items", "unit price cannot be negative")
	}

	if item.UnitPrice.Currency == "" {
		return errors.NewValidationError("items", "currency is required for item")
	}

	return nil
}

func validateAddress(addr *models.Address, field string) error {
	if addr.Line1 == "" {
		return errors.NewValidationError(field, "address line 1 is required")
	}

	if addr.City == "" {
		return errors.NewValidationError(field, "city is required")
	}

	if addr.PostalCode == "" {
		return errors.NewValidationError(field, "postal code is required")
	}

	if addr.Country == "" {
		return errors.NewValidationError(field, "country is required")
	}

	// Validate country code
	if len(addr.Country) != 2 {
		return errors.NewValidationError(field, "country must be a 2-letter ISO code")
	}

	return nil
}

// ValidateUpdateOrderStatusRequest validates a status update request.
func ValidateUpdateOrderStatusRequest(req *models.UpdateOrderStatusRequest) error {
	if req.Status == "" {
		return errors.NewValidationError("status", "status is required")
	}

	// Validate status value
	switch req.Status {
	case models.OrderStatusPending,
		models.OrderStatusConfirmed,
		models.OrderStatusProcessing,
		models.OrderStatusShipped,
		models.OrderStatusDelivered,
		models.OrderStatusCancelled,
		models.OrderStatusRefunded:
		// Valid status
	default:
		return errors.NewValidationError("status", "invalid order status")
	}

	return nil
}

// ValidateOrderListFilter validates a list filter.
func ValidateOrderListFilter(filter *models.OrderListFilter) error {
	if filter.Limit < 0 {
		return errors.NewValidationError("limit", "limit cannot be negative")
	}

	if filter.Offset < 0 {
		return errors.NewValidationError("offset", "offset cannot be negative")
	}

	if filter.Limit > 100 {
		// TODO(TEAM-API): Make max limit configurable
		filter.Limit = 100
	}

	if filter.StartDate != nil && filter.EndDate != nil {
		if filter.StartDate.After(*filter.EndDate) {
			return errors.NewValidationError("start_date", "start date cannot be after end date")
		}
	}

	return nil
}

// ValidatePaymentRequest validates a payment request.
func ValidatePaymentRequest(req *models.ProcessPaymentRequest) error {
	if req.OrderID == "" {
		return errors.NewValidationError("order_id", "order ID is required")
	}

	if req.UserID == "" {
		return errors.NewValidationError("user_id", "user ID is required")
	}

	if req.Amount.Amount <= 0 {
		return errors.NewValidationError("amount", "amount must be positive")
	}

	if req.Amount.Currency == "" {
		return errors.NewValidationError("currency", "currency is required")
	}

	// Validate payment method
	switch req.Method {
	case models.PaymentMethodCreditCard, models.PaymentMethodDebitCard:
		if req.CardToken == "" {
			return errors.NewValidationError("card_token", "card token is required for card payments")
		}
	case models.PaymentMethodPayPal:
		if req.ReturnURL == "" {
			return errors.NewValidationError("return_url", "return URL is required for PayPal payments")
		}
	case models.PaymentMethodBankTransfer:
		// No additional validation needed
	case models.PaymentMethodCrypto:
		// TODO(TEAM-PAYMENTS): Add crypto validation
		return errors.NewValidationError("method", "crypto payments not yet supported")
	default:
		return errors.NewValidationError("method", "invalid payment method")
	}

	return nil
}

// ValidateLegacyPaymentRequest validates a legacy payment request.
// Deprecated: Use ValidatePaymentRequest instead.
// TODO(TEAM-PAYMENTS): Remove after migration complete
func ValidateLegacyPaymentRequest(req *models.LegacyPaymentRequest) error {
	logging.Infof("Validating legacy payment request for order: %s", req.OrderID)

	if req.OrderID == "" {
		return errors.NewValidationError("order_id", "order ID is required")
	}

	if req.Amount <= 0 {
		return errors.NewValidationError("amount", "amount must be positive")
	}

	if req.Currency == "" {
		return errors.NewValidationError("currency", "currency is required")
	}

	// TODO(TEAM-SEC): Never validate raw card numbers - this is a legacy pattern
	// that should be removed
	if req.CardNumber != "" {
		if len(req.CardNumber) < 13 || len(req.CardNumber) > 19 {
			return errors.NewValidationError("card_number", "invalid card number length")
		}
	}

	return nil
}

// SanitizeOrderNotes sanitizes order notes to prevent XSS.
func SanitizeOrderNotes(notes string) string {
	// TODO(TEAM-SEC): Use proper HTML sanitization library
	notes = strings.ReplaceAll(notes, "<", "&lt;")
	notes = strings.ReplaceAll(notes, ">", "&gt;")
	notes = strings.ReplaceAll(notes, "\"", "&quot;")
	notes = strings.TrimSpace(notes)

	// Limit length
	if len(notes) > 1000 {
		notes = notes[:1000]
	}

	return notes
}

// ValidateRefundRequest validates a refund request.
func ValidateRefundRequest(req *models.RefundRequest) error {
	if req.PaymentID == "" {
		return errors.NewValidationError("payment_id", "payment ID is required")
	}

	if req.Amount.Amount <= 0 {
		return errors.NewValidationError("amount", "refund amount must be positive")
	}

	if req.Reason == "" {
		return errors.NewValidationError("reason", "refund reason is required")
	}

	if len(req.Reason) > 500 {
		return errors.NewValidationError("reason", "refund reason too long (max 500 characters)")
	}

	return nil
}

// ValidateCancellationReason validates an order cancellation reason.
func ValidateCancellationReason(reason string) error {
	if reason == "" {
		return errors.NewValidationError("reason", "cancellation reason is required")
	}

	if len(reason) > 500 {
		return errors.NewValidationError("reason", "cancellation reason too long (max 500 characters)")
	}

	return nil
}
