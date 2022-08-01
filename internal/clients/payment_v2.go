package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type PaymentClientV2 struct {
	baseURL    string
	httpClient *http.Client
}

func NewPaymentClientV2(baseURL string) *PaymentClientV2 {
	return &PaymentClientV2{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type ChargeRequestV2 struct {
	OrderID   string  `json:"order_id"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	RequestID string  `json:"request_id,omitempty"`
}

type ChargeResponseV2 struct {
	PaymentID string `json:"payment_id"`
	Status    string `json:"status"`
}

func (c *PaymentClientV2) Charge(ctx context.Context, orderID string, amount float64, currency string, requestID string) (*ChargeResponseV2, error) {
	log.Printf("Calling v2 payment API for order %s, amount %.2f %s, requestID: %s", orderID, amount, currency, requestID)

	req := ChargeRequestV2{
		OrderID:   orderID,
		Amount:    amount,
		Currency:  currency,
		RequestID: requestID,
	}

	return c.chargeWithRetry(ctx, req, requestID, 3)
}

func (c *PaymentClientV2) chargeWithRetry(ctx context.Context, req ChargeRequestV2, requestID string, maxRetries int) (*ChargeResponseV2, error) {
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("Retrying payment request, attempt %d", attempt+1)
			time.Sleep(time.Duration(attempt) * 100 * time.Millisecond)
		}

		body, err := json.Marshal(req)
		if err != nil {
			return nil, err
		}

		url := fmt.Sprintf("%s/api/v2/payments", c.baseURL)
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set("Content-Type", "application/json")
		
		if requestID != "" {
			httpReq.Header.Set("X-Acme-Request-ID", requestID)
		}

		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("payment service returned status %d", resp.StatusCode)
			continue
		}

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			return nil, fmt.Errorf("payment service returned status %d", resp.StatusCode)
		}

		var result ChargeResponseV2
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, err
		}

		log.Printf("Payment processed via v2 API, payment_id: %s", result.PaymentID)
		return &result, nil
	}

	return nil, fmt.Errorf("payment failed after %d retries: %v", maxRetries, lastErr)
}

func (c *PaymentClientV2) GetPaymentStatus(ctx context.Context, paymentID string, requestID string) (string, error) {
	log.Printf("Getting payment status via v2 API, payment_id: %s", paymentID)

	url := fmt.Sprintf("%s/api/v2/payments/%s", c.baseURL, paymentID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	if requestID != "" {
		httpReq.Header.Set("X-Acme-Request-ID", requestID)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", nil
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status check failed with status %d", resp.StatusCode)
	}

	var result struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Status, nil
}

func (c *PaymentClientV2) Refund(ctx context.Context, paymentID string, amount float64, requestID string) error {
	log.Printf("Processing refund via v2 API, payment_id: %s, amount: %.2f", paymentID, amount)

	req := map[string]interface{}{
		"payment_id": paymentID,
		"amount":     amount,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/v2/payments/%s/refund", c.baseURL, paymentID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	if requestID != "" {
		httpReq.Header.Set("X-Acme-Request-ID", requestID)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("refund failed with status %d", resp.StatusCode)
	}

	log.Printf("Refund processed via v2 API")
	return nil
}

type PaymentClient struct {
	v1           *PaymentClientV1
	v2           *PaymentClientV2
	enableLegacy bool
}

func NewPaymentClient(v1 *PaymentClientV1, v2 *PaymentClientV2, enableLegacy bool) *PaymentClient {
	return &PaymentClient{
		v1:           v1,
		v2:           v2,
		enableLegacy: enableLegacy,
	}
}

func (c *PaymentClient) Charge(ctx context.Context, orderID string, amount float64, currency string, requestID string) (string, error) {
	if c.enableLegacy {
		log.Printf("Using legacy v1 payment client")
		resp, err := c.v1.Charge(orderID, amount, currency)
		if err != nil {
			return "", err
		}
		return resp.TransactionID, nil
	}
	
	resp, err := c.v2.Charge(ctx, orderID, amount, currency, requestID)
	if err != nil {
		return "", err
	}
	return resp.PaymentID, nil
}
