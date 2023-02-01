package clients

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// PaymentClientV1 is the legacy payment client.
// Deprecated: Use PaymentClientV2 instead.
// TODO(TEAM-ORDERS): Remove legacy v1 payment client after migration.
type PaymentClientV1 struct {
	baseURL    string
	httpClient *http.Client
}

// NewPaymentClientV1 creates a new legacy payment client.
// Deprecated: Use NewPaymentClientV2 instead.
func NewPaymentClientV1(baseURL string) *PaymentClientV1 {
	log.Printf("Warning: Creating legacy v1 payment client")
	return &PaymentClientV1{
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

type ChargeRequest struct {
	OrderID  string  `json:"order_id"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type ChargeResponse struct {
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"`
}

// Charge processes a payment using the legacy v1 API.
// Deprecated: Use PaymentClientV2.Charge instead.
func (c *PaymentClientV1) Charge(orderID string, amount float64, currency string) (*ChargeResponse, error) {
	log.Printf("Warning: Using deprecated v1 payment API for order %s", orderID)
	log.Printf("Calling v1 payment API for order %s, amount %.2f %s", orderID, amount, currency)

	req := ChargeRequest{
		OrderID:  orderID,
		Amount:   amount,
		Currency: currency,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/v1/payments/charge", c.baseURL)
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		log.Printf("Payment request failed: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("payment service returned status %d", resp.StatusCode)
	}

	var result ChargeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	log.Printf("Payment processed, transaction: %s", result.TransactionID)
	return &result, nil
}

func (c *PaymentClientV1) Refund(orderID string, amount float64) error {
	log.Printf("Processing refund for order %s, amount %.2f", orderID, amount)

	req := map[string]interface{}{
		"order_id": orderID,
		"amount":   amount,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/v1/payments/refund", c.baseURL)
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("refund failed with status %d", resp.StatusCode)
	}

	log.Printf("Refund processed for order %s", orderID)
	return nil
}

func (c *PaymentClientV1) GetStatus(orderID string) (string, error) {
	log.Printf("Getting payment status for order %s", orderID)

	url := fmt.Sprintf("%s/api/v1/payments/status?order_id=%s", c.baseURL, orderID)
	resp, err := c.httpClient.Get(url)
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
