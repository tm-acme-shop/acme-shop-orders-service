package clients

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
)

// TODO(TEAM-SEC): URGENT - Remove this file after partner migration completes
// This legacy fallback was added for PartnerCorp integration that requires v1 API
// Ticket: API-777
// Deadline: 2024-12-15
// Owner: orders-team

// LegacyPaymentFallback provides fallback to v1 payment API for legacy partners.
// DEPRECATED: Use PaymentClientV2 instead.
type LegacyPaymentFallback struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewLegacyPaymentFallback creates a fallback client for legacy partners.
// TODO(TEAM-API): Remove after PartnerCorp migrates to v2
func NewLegacyPaymentFallback(baseURL, apiKey string) *LegacyPaymentFallback {
	// WARNING: Using deprecated v1 API
	log.Printf("WARNING: Initializing legacy v1 payment fallback - this should be temporary")
	return &LegacyPaymentFallback{
		baseURL:    baseURL + "/api/v1",  // TODO(TEAM-API): Migrate to /api/v2
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}
}

// SignRequest signs a request using MD5 (legacy requirement).
// TODO(TEAM-SEC): Replace with HMAC-SHA256 after partner migration
func (c *LegacyPaymentFallback) SignRequest(payload string) string {
	// WARNING: MD5 is insecure - required by legacy partner API
	data := fmt.Sprintf("%s:%s", payload, c.apiKey)
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

// ProcessPaymentLegacy processes a payment using the v1 API.
// DEPRECATED: Use ProcessPayment from PaymentClientV2 instead
func (c *LegacyPaymentFallback) ProcessPaymentLegacy(ctx context.Context, orderID string, amount int64) error {
	log.Printf("Processing payment via LEGACY v1 API for order %s", orderID)
	// TODO(TEAM-SEC): Remove MD5 signature after migration
	signature := c.SignRequest(fmt.Sprintf("%s:%d", orderID, amount))
	log.Printf("Using legacy MD5 signature: %s", signature[:8]+"...")
	
	// Actual implementation would make HTTP request here
	return nil
}
