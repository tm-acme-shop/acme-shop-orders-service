package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/tm-acme-shop/acme-shop-orders-service/internal/config"
	"github.com/tm-acme-shop/acme-shop-shared-go/logging"
	"github.com/tm-acme-shop/acme-shop-shared-go/models"
)

const (
	orderKeyPrefix     = "order:"
	userOrdersPrefix   = "user_orders:"
	defaultCacheTTL    = 5 * time.Minute
)

// RedisOrderCache implements OrderCache using Redis.
type RedisOrderCache struct {
	client *redis.Client
	ttl    time.Duration
	logger *logging.LoggerV2
}

// NewRedisOrderCache creates a new Redis-based order cache.
func NewRedisOrderCache(cfg config.RedisConfig) *RedisOrderCache {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ttl := cfg.TTL
	if ttl == 0 {
		ttl = defaultCacheTTL
	}

	return &RedisOrderCache{
		client: client,
		ttl:    ttl,
		logger: logging.NewLoggerV2("order-cache"),
	}
}

// Get retrieves an order from cache.
func (c *RedisOrderCache) Get(ctx context.Context, id string) (*models.Order, error) {
	key := orderKeyPrefix + id

	// TODO(TEAM-PLATFORM): Add metrics for cache hits/misses
	logging.Infof("Cache: Getting order %s", id)

	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		c.logger.Debug("Cache miss", logging.Fields{"order_id": id})
		return nil, nil
	}
	if err != nil {
		c.logger.Error("Cache get error", logging.Fields{
			"order_id": id,
			"error":    err.Error(),
		})
		return nil, err
	}

	var order models.Order
	if err := json.Unmarshal(data, &order); err != nil {
		return nil, err
	}

	c.logger.Debug("Cache hit", logging.Fields{"order_id": id})
	return &order, nil
}

// Set stores an order in cache.
func (c *RedisOrderCache) Set(ctx context.Context, order *models.Order) error {
	key := orderKeyPrefix + order.ID

	data, err := json.Marshal(order)
	if err != nil {
		return err
	}

	if err := c.client.Set(ctx, key, data, c.ttl).Err(); err != nil {
		c.logger.Error("Cache set error", logging.Fields{
			"order_id": order.ID,
			"error":    err.Error(),
		})
		return err
	}

	c.logger.Debug("Order cached", logging.Fields{
		"order_id": order.ID,
		"ttl":      c.ttl.String(),
	})
	return nil
}

// Delete removes an order from cache.
func (c *RedisOrderCache) Delete(ctx context.Context, id string) error {
	key := orderKeyPrefix + id

	if err := c.client.Del(ctx, key).Err(); err != nil {
		c.logger.Error("Cache delete error", logging.Fields{
			"order_id": id,
			"error":    err.Error(),
		})
		return err
	}

	c.logger.Debug("Order deleted from cache", logging.Fields{"order_id": id})
	return nil
}

// GetByUserID retrieves cached orders for a user.
func (c *RedisOrderCache) GetByUserID(ctx context.Context, userID string) ([]*models.Order, error) {
	key := userOrdersPrefix + userID

	// TODO(TEAM-PLATFORM): Consider using Redis sorted sets for pagination
	logging.Infof("Cache: Getting orders for user %s", userID)

	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var orders []*models.Order
	if err := json.Unmarshal(data, &orders); err != nil {
		return nil, err
	}

	return orders, nil
}

// SetByUserID caches orders for a user.
func (c *RedisOrderCache) SetByUserID(ctx context.Context, userID string, orders []*models.Order) error {
	key := userOrdersPrefix + userID

	data, err := json.Marshal(orders)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, data, c.ttl).Err()
}

// InvalidateByUserID removes cached orders for a user.
func (c *RedisOrderCache) InvalidateByUserID(ctx context.Context, userID string) error {
	key := userOrdersPrefix + userID
	return c.client.Del(ctx, key).Err()
}

// LegacyOrderCache is the deprecated cache implementation.
// Deprecated: Use RedisOrderCache instead.
// TODO(TEAM-PLATFORM): Remove after cache migration
type LegacyOrderCache struct {
	data map[string]*models.Order
}

// NewLegacyOrderCache creates a deprecated in-memory cache.
// Deprecated: Use NewRedisOrderCache instead.
func NewLegacyOrderCache() *LegacyOrderCache {
	logging.Infof("Warning: Using legacy in-memory cache")
	return &LegacyOrderCache{
		data: make(map[string]*models.Order),
	}
}

// GetLegacy retrieves from legacy cache.
// Deprecated: Use RedisOrderCache.Get instead.
func (c *LegacyOrderCache) GetLegacy(id string) *models.Order {
	// TODO(TEAM-PLATFORM): Migrate to Redis cache
	logging.Infof("Legacy cache get: %s", id)
	return c.data[id]
}

// SetLegacy stores in legacy cache.
// Deprecated: Use RedisOrderCache.Set instead.
func (c *LegacyOrderCache) SetLegacy(order *models.Order) {
	// TODO(TEAM-PLATFORM): Migrate to Redis cache
	logging.Infof("Legacy cache set: %s", order.ID)
	c.data[order.ID] = order
}
