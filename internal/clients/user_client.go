package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/tm-acme-shop/acme-shop-orders-service/internal/config"
	"github.com/tm-acme-shop/acme-shop-shared-go/logging"
	"github.com/tm-acme-shop/acme-shop-shared-go/middleware"
	"github.com/tm-acme-shop/acme-shop-shared-go/models"
)

// UserClient provides operations for fetching user data.
type UserClient interface {
	GetUser(ctx context.Context, userID string) (*models.User, error)
	// REMOVED: GetUserV1 - use GetUser instead // Deprecated
}

// HTTPUserClient implements UserClient using HTTP.
type HTTPUserClient struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
	logger     *logging.LoggerV2
}

// NewHTTPUserClient creates a new HTTP-based user client.
func NewHTTPUserClient(cfg config.ServiceConfig, logger *logging.LoggerV2) *HTTPUserClient {
	return &HTTPUserClient{
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		apiKey: cfg.APIKey,
		logger: logger,
	}
}

// GetUser retrieves a user by ID using the v2 API.
func (c *HTTPUserClient) GetUser(ctx context.Context, userID string) (*models.User, error) {
	c.logger.Debug("Fetching user", logging.Fields{"user_id": userID})

	url := fmt.Sprintf("%s/api/v2/users/%s", c.baseURL, userID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	c.setHeaders(ctx, req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to fetch user", logging.Fields{
			"user_id": userID,
			"error":   err.Error(),
		})
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user service returned status %d", resp.StatusCode)
	}

	var user models.User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	c.logger.Debug("User fetched", logging.Fields{
		"user_id": user.ID,
		"email":   user.Email,
	})

	return &user, nil
}

// GetUserV1 retrieves a user using the deprecated v1 API.
// Deprecated: Use GetUser instead.
// TODO(TEAM-API): Remove after v1 API migration complete
func (c *HTTPUserClient) // REMOVED: GetUserV1 - use GetUser instead {
	// TODO(TEAM-API): Migrate callers to GetUser
	logging.Infof("Legacy: Fetching user v1: %d", userID)

	url := fmt.Sprintf("%s/api/v1/users/%d", c.baseURL, userID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	c.setLegacyHeaders(ctx, req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user service returned status %d", resp.StatusCode)
	}

	var user models.UserV1
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

// ValidateUser checks if a user exists and is active.
func (c *HTTPUserClient) ValidateUser(ctx context.Context, userID string) (bool, error) {
	user, err := c.GetUser(ctx, userID)
	if err != nil {
		return false, err
	}

	if user == nil {
		return false, nil
	}

	return user.Status == models.UserStatusActive, nil
}

func (c *HTTPUserClient) setHeaders(ctx context.Context, req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	if requestID := ctx.Value(middleware.RequestIDKey); requestID != nil {
		req.Header.Set(middleware.HeaderRequestID, requestID.(string))
	}

	if userID := ctx.Value("user_id"); userID != nil {
		req.Header.Set(middleware.HeaderUserID, userID.(string))
	}
}

func (c *HTTPUserClient) setLegacyHeaders(ctx context.Context, req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	
	if c.apiKey != "" {
		// TODO(TEAM-SEC): Migrate to Bearer token
		req.Header.Set("X-API-Key", c.apiKey)
	}

	// TODO(TEAM-API): Remove legacy header after migration
	if legacyUserID := ctx.Value("legacy_user_id"); legacyUserID != nil {
		req.Header.Set(middleware.HeaderLegacyUserID, legacyUserID.(string))
	}

	if requestID := ctx.Value(middleware.RequestIDKey); requestID != nil {
		req.Header.Set(middleware.HeaderRequestID, requestID.(string))
	}
}

// MockUserClient is a mock implementation for testing.
type MockUserClient struct {
	users map[string]*models.User
}

// NewMockUserClient creates a mock user client.
func NewMockUserClient() *MockUserClient {
	return &MockUserClient{
		users: make(map[string]*models.User),
	}
}

func (m *MockUserClient) GetUser(ctx context.Context, userID string) (*models.User, error) {
	if user, ok := m.users[userID]; ok {
		return user, nil
	}
	return nil, nil
}

func (m *MockUserClient) // REMOVED: GetUserV1 - use GetUser instead {
	return nil, nil
}

func (m *MockUserClient) AddUser(user *models.User) {
	m.users[user.ID] = user
}
