package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/tm-acme-shop/acme-shop-orders-service/internal/config"
	"github.com/tm-acme-shop/acme-shop-shared-go/interfaces"
	"github.com/tm-acme-shop/acme-shop-shared-go/logging"
	"github.com/tm-acme-shop/acme-shop-shared-go/middleware"
	"github.com/tm-acme-shop/acme-shop-shared-go/models"
)

// Ensure LegacyHTTPPaymentClient implements interfaces.LegacyPaymentClient
var _ interfaces.LegacyPaymentClient = (*LegacyHTTPPaymentClient)(nil)

// LegacyHTTPPaymentClient is the deprecated payment client.
// Deprecated: Use HTTPPaymentClient instead.
// TODO(TEAM-PAYMENTS): Remove after migration complete
type LegacyHTTPPaymentClient struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
}

// NewLegacyHTTPPaymentClient creates a deprecated legacy payment client.
// Deprecated: Use NewHTTPPaymentClient instead.
func NewLegacyHTTPPaymentClient(cfg config.ServiceConfig) *LegacyHTTPPaymentClient {
	// TODO(TEAM-PAYMENTS): Remove this constructor after migration
	logging.Infof("Warning: Creating legacy payment client")
	return &LegacyHTTPPaymentClient{
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		apiKey: cfg.APIKey,
	}
}

// ProcessLegacyPayment processes a payment using the old format.
// Deprecated: Use HTTPPaymentClient.ProcessPayment instead.
func (c *LegacyHTTPPaymentClient) ProcessLegacyPayment(ctx context.Context, req *models.LegacyPaymentRequest) (string, error) {
	// TODO(TEAM-PAYMENTS): Migrate callers to new payment client
	logging.Infof("Legacy: Processing payment for order: %s", req.OrderID)

	body, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	// Use deprecated v1 API endpoint
	// TODO(TEAM-API): Remove v1 endpoint after migration
	url := fmt.Sprintf("%s/api/v1/payments/process", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	c.setLegacyHeaders(ctx, httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		logging.Infof("Legacy: Payment request failed: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("legacy payment service returned status %d", resp.StatusCode)
	}

	var result struct {
		TransactionID string `json:"transaction_id"`
		Status        string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	logging.Infof("Legacy: Payment processed, transaction: %s", result.TransactionID)
	return result.TransactionID, nil
}

// GetStatus retrieves payment status by order ID (legacy behavior).
// Deprecated: Use HTTPPaymentClient.GetPaymentStatus with payment ID.
func (c *LegacyHTTPPaymentClient) GetStatus(ctx context.Context, orderID string) (string, error) {
	// TODO(TEAM-PAYMENTS): Migrate callers to new payment client
	logging.Infof("Legacy: Getting payment status for order: %s", orderID)

	// Use deprecated v1 API endpoint
	url := fmt.Sprintf("%s/api/v1/payments/status?order_id=%s", c.baseURL, orderID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	c.setLegacyHeaders(ctx, httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", nil
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("legacy payment service returned status %d", resp.StatusCode)
	}

	var result struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Status, nil
}

func (c *LegacyHTTPPaymentClient) setLegacyHeaders(ctx context.Context, req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	
	if c.apiKey != "" {
		// Legacy uses X-API-Key header
		// TODO(TEAM-SEC): Migrate to Bearer token
		req.Header.Set("X-API-Key", c.apiKey)
	}

	// Use legacy user ID header
	// TODO(TEAM-API): Remove legacy header after migration
	if legacyUserID := ctx.Value("legacy_user_id"); legacyUserID != nil {
		req.Header.Set(middleware.HeaderLegacyUserID, legacyUserID.(string))
	}

	// Propagate request ID for tracing
	if requestID := ctx.Value(middleware.RequestIDKey); requestID != nil {
		req.Header.Set(middleware.HeaderRequestID, requestID.(string))
	}
}

// LegacyPaymentResult is the deprecated payment result format.
// Deprecated: Use models.ProcessPaymentResponse instead.
// TODO(TEAM-PAYMENTS): Remove after migration
type LegacyPaymentResult struct {
	TransactionID string  `json:"transaction_id"`
	Status        string  `json:"status"`
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
	Timestamp     string  `json:"timestamp"`
}

// ConvertLegacyResultToPayment converts legacy result to new format.
// Deprecated: Remove after migration.
// TODO(TEAM-PAYMENTS): Remove this helper after migration complete
func ConvertLegacyResultToPayment(result *LegacyPaymentResult, orderID string) *models.Payment {
	logging.Infof("Converting legacy payment result to new format")
	return &models.Payment{
		ID:         result.TransactionID,
		OrderID:    orderID,
		Amount:     models.NewMoney(result.Amount, result.Currency),
		ProviderID: result.TransactionID,
	}
}

// LegacyMockPaymentClient is a deprecated mock for testing.
// Deprecated: Use MockPaymentClient instead.
type LegacyMockPaymentClient struct{}

// NewLegacyMockPaymentClient creates a deprecated mock client.
// Deprecated: Use NewMockPaymentClient instead.
func NewLegacyMockPaymentClient() *LegacyMockPaymentClient {
	// TODO(TEAM-PAYMENTS): Remove after migration
	logging.Infof("Warning: Creating legacy mock payment client")
	return &LegacyMockPaymentClient{}
}

func (m *LegacyMockPaymentClient) ProcessLegacyPayment(ctx context.Context, req *models.LegacyPaymentRequest) (string, error) {
	return "mock_txn_123", nil
}

func (m *LegacyMockPaymentClient) GetStatus(ctx context.Context, orderID string) (string, error) {
	return "completed", nil
}
