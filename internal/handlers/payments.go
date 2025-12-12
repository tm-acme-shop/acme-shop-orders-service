package handlers

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tm-acme-shop/acme-shop-orders-service/internal/service"
	"github.com/tm-acme-shop/acme-shop-shared-go/logging"
	"github.com/tm-acme-shop/acme-shop-shared-go/models"
)

// ProcessOrderPayment handles POST /api/v2/orders/:id/payment
func (h *Handlers) ProcessOrderPayment(c *gin.Context) {
	orderID := c.Param("id")

	var req models.ProcessPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind payment request", logging.Fields{"error": err.Error()})
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	req.OrderID = orderID

	// Get user ID from context
	if userID, exists := c.Get("user_id"); exists {
		req.UserID = userID.(string)
	}

	if err := service.ValidatePaymentRequest(&req); err != nil {
		handleError(c, err)
		return
	}

	resp, err := h.orderService.ProcessOrderPayment(c.Request.Context(), orderID, &req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ProcessOrderPaymentV1 handles POST /api/v1/orders/:id/pay
// Deprecated: Use ProcessOrderPayment (v2) instead.
// TODO(TEAM-PAYMENTS): Remove after v1 API migration complete
func (h *Handlers) ProcessOrderPaymentV1(c *gin.Context) {
	logging.Infof("Legacy: Processing order payment via v1 API")

	orderID := c.Param("id")

	var req models.LegacyPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	req.OrderID = orderID

	// TODO(TEAM-SEC): Remove card number validation - legacy pattern
	if err := service.ValidateLegacyPaymentRequest(&req); err != nil {
		handleError(c, err)
		return
	}

	// Convert to v2 and process
	v2Req := &models.ProcessPaymentRequest{
		OrderID:   orderID,
		Amount:    models.NewMoney(req.Amount, req.Currency),
		Method:    models.PaymentMethodCreditCard,
		CardToken: "legacy_token", // Legacy format doesn't use tokens
	}

	resp, err := h.orderService.ProcessOrderPayment(c.Request.Context(), orderID, v2Req)
	if err != nil {
		handleError(c, err)
		return
	}

	// Return legacy format
	c.JSON(http.StatusOK, gin.H{
		"transaction_id": resp.PaymentID,
		"status":         resp.Status,
	})
}

// GetPaymentStatus handles GET /api/v2/payments/:id
func (h *Handlers) GetPaymentStatus(c *gin.Context) {
	paymentID := c.Param("id")

	payment, err := h.paymentService.GetPaymentStatus(c.Request.Context(), paymentID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, payment)
}

// GetPaymentStatusV1 handles GET /api/v1/payments/status
// Deprecated: Use GetPaymentStatus (v2) instead.
// TODO(TEAM-PAYMENTS): Remove after v1 API migration complete
func (h *Handlers) GetPaymentStatusV1(c *gin.Context) {
	logging.Infof("Legacy: Getting payment status via v1 API")

	// Legacy API uses order_id query param instead of payment_id path param
	orderID := c.Query("order_id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order_id is required"})
		return
	}

	status, err := h.paymentService.GetPaymentStatusV1(c.Request.Context(), orderID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"order_id": orderID,
		"status":   status,
	})
}

// GetOrderPayment handles GET /api/v2/orders/:id/payment
func (h *Handlers) GetOrderPayment(c *gin.Context) {
	orderID := c.Param("id")

	payment, err := h.paymentService.GetPaymentByOrderID(c.Request.Context(), orderID)
	if err != nil {
		handleError(c, err)
		return
	}

	if payment == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no payment found for order"})
		return
	}

	c.JSON(http.StatusOK, payment)
}

// CancelPayment handles POST /api/v2/payments/:id/cancel
func (h *Handlers) CancelPayment(c *gin.Context) {
	paymentID := c.Param("id")

	if err := h.paymentService.CancelPayment(c.Request.Context(), paymentID); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"payment_id": paymentID,
		"status":     "cancelled",
	})
}

// ProcessRefund handles POST /api/v2/payments/:id/refund
func (h *Handlers) ProcessRefund(c *gin.Context) {
	paymentID := c.Param("id")

	var req struct {
		Amount models.Money `json:"amount"`
		Reason string       `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	refundResp, err := h.paymentService.ProcessRefund(
		c.Request.Context(),
		paymentID,
		req.Amount,
		req.Reason,
	)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, refundResp)
}

// ProcessRefundV1 handles POST /api/v1/payments/:id/refund
// Deprecated: Use ProcessRefund (v2) instead.
// TODO(TEAM-PAYMENTS): Remove after v1 API migration complete
func (h *Handlers) ProcessRefundV1(c *gin.Context) {
	logging.Infof("Legacy: Processing refund via v1 API")

	paymentID := c.Param("id")

	var req struct {
		Amount   float64 `json:"amount"`
		Currency string  `json:"currency"`
		Reason   string  `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	amount := models.NewMoney(req.Amount, req.Currency)

	refundResp, err := h.paymentService.ProcessRefund(
		c.Request.Context(),
		paymentID,
		amount,
		req.Reason,
	)
	if err != nil {
		handleError(c, err)
		return
	}

	// Return legacy format
	c.JSON(http.StatusOK, gin.H{
		"refund_id":      refundResp.RefundID,
		"transaction_id": refundResp.PaymentID,
		"status":         refundResp.Status,
		"amount":         req.Amount,
		"currency":       req.Currency,
	})
}

// PaymentWebhook handles POST /api/v2/webhooks/payment
func (h *Handlers) PaymentWebhook(c *gin.Context) {
	signature := c.GetHeader("X-Payment-Signature")

	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.Error("Failed to read webhook payload", logging.Fields{"error": err.Error()})
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	if err := h.paymentService.ProcessWebhook(c.Request.Context(), payload, signature); err != nil {
		h.logger.Error("Webhook processing failed", logging.Fields{"error": err.Error()})
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "received"})
}

// PaymentWebhookV1 handles POST /api/v1/webhooks/payment
// Deprecated: Use PaymentWebhook (v2) instead.
// TODO(TEAM-PAYMENTS): Remove after v1 API migration complete
func (h *Handlers) PaymentWebhookV1(c *gin.Context) {
	logging.Infof("Legacy: Processing payment webhook via v1 API")

	// Legacy uses different signature header
	// TODO(TEAM-API): Remove legacy header after migration
	signature := c.GetHeader("X-Legacy-Signature")

	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	if err := h.paymentService.ProcessWebhook(c.Request.Context(), payload, signature); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"received": true})
}
