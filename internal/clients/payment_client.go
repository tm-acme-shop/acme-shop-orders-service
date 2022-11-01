package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/tm-acme-shop/acme-shop-orders-service/internal/config"
	"github.com/tm-acme-shop/acme-shop-shared-go/interfaces"
	"github.com/tm-acme-shop/acme-shop-shared-go/logging"
	"github.com/tm-acme-shop/acme-shop-shared-go/middleware"
	"github.com/tm-acme-shop/acme-shop-shared-go/models"
)

// Ensure HTTPPaymentClient implements interfaces.PaymentClient
var _ interfaces.PaymentClient = (*HTTPPaymentClient)(nil)

// HTTPPaymentClient implements interfaces.PaymentClient using HTTP.
type HTTPPaymentClient struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
	logger     *logging.LoggerV2
}

// NewHTTPPaymentClient creates a new HTTP-based payment client.
func NewHTTPPaymentClient(cfg config.ServiceConfig, logger *logging.LoggerV2) *HTTPPaymentClient {
	return &HTTPPaymentClient{
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		apiKey: cfg.APIKey,
		logger: logger,
	}
}

// ProcessPayment initiates a payment for an order.
func (c *HTTPPaymentClient) ProcessPayment(ctx context.Context, req *models.ProcessPaymentRequest) (*models.ProcessPaymentResponse, error) {
	c.logger.Debug("Processing payment", logging.Fields{
		"order_id": req.OrderID,
		"amount":   req.Amount.Amount,
		"method":   req.Method,
	})

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	// Use v2 API endpoint
	url := fmt.Sprintf("%s/api/v2/payments", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	c.setHeaders(ctx, httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		c.logger.Error("Payment request failed", logging.Fields{
			"order_id": req.OrderID,
			"error":    err.Error(),
		})
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		c.logger.Error("Payment request returned error", logging.Fields{
			"order_id":    req.OrderID,
			"status_code": resp.StatusCode,
		})
		return nil, fmt.Errorf("payment service returned status %d", resp.StatusCode)
	}

	var result models.ProcessPaymentResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	c.logger.Info("Payment processed", logging.Fields{
		"order_id":   req.OrderID,
		"payment_id": result.PaymentID,
		"status":     result.Status,
	})

	return &result, nil
}

// GetPaymentStatus retrieves the current status of a payment.
func (c *HTTPPaymentClient) GetPaymentStatus(ctx context.Context, paymentID string) (*models.Payment, error) {
	c.logger.Debug("Getting payment status", logging.Fields{"payment_id": paymentID})

	url := fmt.Sprintf("%s/api/v2/payments/%s", c.baseURL, paymentID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	c.setHeaders(ctx, httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("payment service returned status %d", resp.StatusCode)
	}

	var payment models.Payment
	if err := json.NewDecoder(resp.Body).Decode(&payment); err != nil {
		return nil, err
	}

	return &payment, nil
}

// Refund processes a refund for a completed payment.
func (c *HTTPPaymentClient) Refund(ctx context.Context, req *models.RefundRequest) (*models.RefundResponse, error) {
	c.logger.Debug("Processing refund", logging.Fields{
		"payment_id": req.PaymentID,
		"amount":     req.Amount.Amount,
		"reason":     req.Reason,
	})

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/v2/payments/%s/refund", c.baseURL, req.PaymentID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	c.setHeaders(ctx, httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		c.logger.Error("Refund request failed", logging.Fields{
			"payment_id": req.PaymentID,
			"error":      err.Error(),
		})
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("refund service returned status %d", resp.StatusCode)
	}

	var result models.RefundResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	c.logger.Info("Refund processed", logging.Fields{
		"payment_id": req.PaymentID,
		"refund_id":  result.RefundID,
		"status":     result.Status,
	})

	return &result, nil
}

// CancelPayment cancels a pending payment.
func (c *HTTPPaymentClient) CancelPayment(ctx context.Context, paymentID string) error {
	c.logger.Debug("Cancelling payment", logging.Fields{"payment_id": paymentID})

	url := fmt.Sprintf("%s/api/v2/payments/%s/cancel", c.baseURL, paymentID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return err
	}

	c.setHeaders(ctx, httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("cancel payment returned status %d", resp.StatusCode)
	}

	c.logger.Info("Payment cancelled", logging.Fields{"payment_id": paymentID})
	return nil
}

// ValidateWebhook validates an incoming webhook from the payment provider.
func (c *HTTPPaymentClient) ValidateWebhook(ctx context.Context, payload []byte, signature string) (bool, error) {
	c.logger.Debug("Validating webhook", logging.Fields{
		"payload_size": len(payload),
		"has_signature": signature != "",
	})

	// TODO(TEAM-PAYMENTS): Implement proper webhook signature validation
	if signature == "" {
		return false, nil
	}

	return true, nil
}

// PLAT-025: Request ID propagation to payment service (2022-11)
func (c *HTTPPaymentClient) setHeaders(ctx context.Context, req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	// Propagate request ID for tracing
	if requestID := ctx.Value(middleware.RequestIDKey); requestID != nil {
		req.Header.Set(middleware.HeaderRequestID, requestID.(string))
	}

	// TODO(TEAM-API): Remove legacy header after migration
	if legacyUserID := ctx.Value("legacy_user_id"); legacyUserID != nil {
		req.Header.Set(middleware.HeaderLegacyUserID, legacyUserID.(string))
	}

	// Set new user ID header
	if userID := ctx.Value("user_id"); userID != nil {
		req.Header.Set(middleware.HeaderUserID, userID.(string))
	}
}

// MockPaymentClient is a mock implementation for testing.
type MockPaymentClient struct {
	payments map[string]*models.Payment
	logger   *logging.LoggerV2
}

// NewMockPaymentClient creates a mock payment client.
func NewMockPaymentClient() *MockPaymentClient {
	return &MockPaymentClient{
		payments: make(map[string]*models.Payment),
		logger:   logging.NewLoggerV2("mock-payment-client"),
	}
}

func (m *MockPaymentClient) ProcessPayment(ctx context.Context, req *models.ProcessPaymentRequest) (*models.ProcessPaymentResponse, error) {
	paymentID := fmt.Sprintf("pay_%d", time.Now().UnixNano())
	
	m.payments[paymentID] = &models.Payment{
		ID:        paymentID,
		OrderID:   req.OrderID,
		UserID:    req.UserID,
		Amount:    req.Amount,
		Method:    req.Method,
		Status:    models.PaymentStatusCompleted,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return &models.ProcessPaymentResponse{
		PaymentID: paymentID,
		Status:    models.PaymentStatusCompleted,
	}, nil
}

func (m *MockPaymentClient) GetPaymentStatus(ctx context.Context, paymentID string) (*models.Payment, error) {
	if payment, ok := m.payments[paymentID]; ok {
		return payment, nil
	}
	return nil, nil
}

func (m *MockPaymentClient) Refund(ctx context.Context, req *models.RefundRequest) (*models.RefundResponse, error) {
	return &models.RefundResponse{
		RefundID:  fmt.Sprintf("ref_%d", time.Now().UnixNano()),
		PaymentID: req.PaymentID,
		Amount:    req.Amount,
		Status:    models.PaymentStatusRefunded,
	}, nil
}

func (m *MockPaymentClient) CancelPayment(ctx context.Context, paymentID string) error {
	if payment, ok := m.payments[paymentID]; ok {
		payment.Status = models.PaymentStatusCancelled
	}
	return nil
}

func (m *MockPaymentClient) ValidateWebhook(ctx context.Context, payload []byte, signature string) (bool, error) {
	return signature != "", nil
}
