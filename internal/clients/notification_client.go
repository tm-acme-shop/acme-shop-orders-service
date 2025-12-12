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

// Ensure HTTPNotificationClient implements interfaces.NotificationSender
var _ interfaces.NotificationSender = (*HTTPNotificationClient)(nil)

// HTTPNotificationClient implements interfaces.NotificationSender using HTTP.
type HTTPNotificationClient struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
	logger     *logging.LoggerV2
}

// NewHTTPNotificationClient creates a new HTTP-based notification client.
func NewHTTPNotificationClient(cfg config.ServiceConfig, logger *logging.LoggerV2) *HTTPNotificationClient {
	return &HTTPNotificationClient{
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		apiKey: cfg.APIKey,
		logger: logger,
	}
}

// SendNotification sends a notification to a user.
func (c *HTTPNotificationClient) SendNotification(ctx context.Context, notification *models.Notification) error {
	c.logger.Debug("Sending notification", logging.Fields{
		"user_id": notification.UserID,
		"type":    notification.Type,
		"channel": notification.Channel,
	})

	body, err := json.Marshal(notification)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/v2/notifications", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	c.setHeaders(ctx, req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to send notification", logging.Fields{
			"user_id": notification.UserID,
			"error":   err.Error(),
		})
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("notification service returned status %d", resp.StatusCode)
	}

	c.logger.Info("Notification sent", logging.Fields{
		"user_id": notification.UserID,
		"type":    notification.Type,
	})

	return nil
}

// SendEmail sends an email notification.
func (c *HTTPNotificationClient) SendEmail(ctx context.Context, req *models.SendEmailRequest) error {
	c.logger.Debug("Sending email", logging.Fields{
		"to":       req.To,
		"template": req.Template,
	})

	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/v2/notifications/email", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	c.setHeaders(ctx, httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("email service returned status %d", resp.StatusCode)
	}

	c.logger.Info("Email sent", logging.Fields{"to": req.To})
	return nil
}

// SendEmailLegacy sends an email using the deprecated API.
// Deprecated: Use SendEmail instead.
// TODO(TEAM-API): Remove after v1 API migration complete
func (c *HTTPNotificationClient) SendEmailLegacy(ctx context.Context, to, subject, body string) error {
	// TODO(TEAM-API): Migrate callers to SendEmail
	logging.Infof("Legacy: Sending email to: %s", to)

	payload := map[string]string{
		"to":      to,
		"subject": subject,
		"body":    body,
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// Use deprecated v1 API endpoint
	url := fmt.Sprintf("%s/api/v1/notifications/email", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}

	c.setLegacyHeaders(ctx, req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		logging.Infof("Legacy: Failed to send email: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("legacy email service returned status %d", resp.StatusCode)
	}

	return nil
}

// SendSMS sends an SMS notification.
func (c *HTTPNotificationClient) SendSMS(ctx context.Context, req *models.SendSMSRequest) error {
	c.logger.Debug("Sending SMS", logging.Fields{
		"to":       req.To,
		"template": req.Template,
	})

	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/v2/notifications/sms", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	c.setHeaders(ctx, httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("SMS service returned status %d", resp.StatusCode)
	}

	c.logger.Info("SMS sent", logging.Fields{"to": req.To})
	return nil
}

// SendPush sends a push notification.
func (c *HTTPNotificationClient) SendPush(ctx context.Context, req *models.SendPushRequest) error {
	c.logger.Debug("Sending push notification", logging.Fields{
		"user_id": req.UserID,
		"title":   req.Title,
	})

	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/v2/notifications/push", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	c.setHeaders(ctx, httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("push service returned status %d", resp.StatusCode)
	}

	c.logger.Info("Push notification sent", logging.Fields{"user_id": req.UserID})
	return nil
}

func (c *HTTPNotificationClient) setHeaders(ctx context.Context, req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	if requestID := ctx.Value(middleware.RequestIDKey); requestID != nil {
		req.Header.Set(middleware.HeaderRequestID, requestID.(string))
	}
}

func (c *HTTPNotificationClient) setLegacyHeaders(ctx context.Context, req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	
	if c.apiKey != "" {
		// TODO(TEAM-SEC): Migrate to Bearer token
		req.Header.Set("X-API-Key", c.apiKey)
	}

	if requestID := ctx.Value(middleware.RequestIDKey); requestID != nil {
		req.Header.Set(middleware.HeaderRequestID, requestID.(string))
	}
}

// MockNotificationClient is a mock implementation for testing.
type MockNotificationClient struct {
	notifications []*models.Notification
	logger        *logging.LoggerV2
}

// NewMockNotificationClient creates a mock notification client.
func NewMockNotificationClient() *MockNotificationClient {
	return &MockNotificationClient{
		notifications: make([]*models.Notification, 0),
		logger:        logging.NewLoggerV2("mock-notification-client"),
	}
}

func (m *MockNotificationClient) SendNotification(ctx context.Context, notification *models.Notification) error {
	m.notifications = append(m.notifications, notification)
	return nil
}

func (m *MockNotificationClient) SendEmail(ctx context.Context, req *models.SendEmailRequest) error {
	return nil
}

func (m *MockNotificationClient) SendEmailLegacy(ctx context.Context, to, subject, body string) error {
	return nil
}

func (m *MockNotificationClient) SendSMS(ctx context.Context, req *models.SendSMSRequest) error {
	return nil
}

func (m *MockNotificationClient) SendPush(ctx context.Context, req *models.SendPushRequest) error {
	return nil
}
