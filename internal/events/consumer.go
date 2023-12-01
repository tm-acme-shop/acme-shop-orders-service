package events

import (
	"context"
	"encoding/json"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/tm-acme-shop/acme-shop-orders-service/internal/config"
	"github.com/tm-acme-shop/acme-shop-orders-service/internal/service"
	"github.com/tm-acme-shop/acme-shop-shared-go/logging"
	"github.com/tm-acme-shop/acme-shop-shared-go/models"
)

// PaymentEventType represents the type of payment event.
type PaymentEventType string

const (
	PaymentEventCompleted PaymentEventType = "payment.completed"
	PaymentEventFailed    PaymentEventType = "payment.failed"
	PaymentEventRefunded  PaymentEventType = "payment.refunded"
)

// PaymentEvent represents a payment-related event.
type PaymentEvent struct {
	ID        string           `json:"id"`
	Type      PaymentEventType `json:"type"`
	PaymentID string           `json:"payment_id"`
	OrderID   string           `json:"order_id"`
	Status    string           `json:"status"`
	Data      json.RawMessage  `json:"data"`
	Timestamp time.Time        `json:"timestamp"`
}

// KafkaConsumer consumes events from Kafka.
type KafkaConsumer struct {
	reader       *kafka.Reader
	orderService *service.OrderService
	logger       *logging.LoggerV2
	stopCh       chan struct{}
}

// NewKafkaConsumer creates a new Kafka-based event consumer.
func NewKafkaConsumer(cfg config.KafkaConfig, orderService *service.OrderService, logger *logging.LoggerV2) *KafkaConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  cfg.Brokers,
		Topic:    cfg.PaymentsTopic,
		GroupID:  cfg.ConsumerGroup,
		MinBytes: 1,
		MaxBytes: 10e6,
		MaxWait:  time.Second,
	})

	return &KafkaConsumer{
		reader:       reader,
		orderService: orderService,
		logger:       logger,
		stopCh:       make(chan struct{}),
	}
}

// Start begins consuming events.
func (c *KafkaConsumer) Start(ctx context.Context) error {
	c.logger.Info("Starting Kafka consumer")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.stopCh:
			c.logger.Info("Kafka consumer stopped")
			return nil
		default:
			msg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				c.logger.Error("Failed to read message", logging.Fields{"error": err.Error()})
				continue
			}

			c.handleMessage(ctx, msg)
		}
	}
}

// Stop stops the consumer.
func (c *KafkaConsumer) Stop() {
	close(c.stopCh)
	c.reader.Close()
}

func (c *KafkaConsumer) handleMessage(ctx context.Context, msg kafka.Message) {
	c.logger.Debug("Received message", logging.Fields{
		"topic":     msg.Topic,
		"partition": msg.Partition,
		"offset":    msg.Offset,
	})

	var event PaymentEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		c.logger.Error("Failed to unmarshal event", logging.Fields{"error": err.Error()})
		return
	}

	switch event.Type {
	case PaymentEventCompleted:
		c.handlePaymentCompleted(ctx, &event)
	case PaymentEventFailed:
		c.handlePaymentFailed(ctx, &event)
	case PaymentEventRefunded:
		c.handlePaymentRefunded(ctx, &event)
	default:
		c.logger.Debug("Ignoring unknown event type", logging.Fields{"type": event.Type})
	}
}

func (c *KafkaConsumer) handlePaymentCompleted(ctx context.Context, event *PaymentEvent) {
	c.logger.Info("Handling payment completed event", logging.Fields{
		"payment_id": event.PaymentID,
		"order_id":   event.OrderID,
	})

	// Update order status to confirmed
	req := &models.UpdateOrderStatusRequest{
		Status: models.OrderStatusConfirmed,
		Notes:  "Payment completed via event",
	}

	_, err := c.orderService.UpdateOrderStatus(ctx, event.OrderID, req)
	if err != nil {
		c.logger.Error("Failed to update order status", logging.Fields{
			"order_id": event.OrderID,
			"error":    err.Error(),
		})
	}
}

func (c *KafkaConsumer) handlePaymentFailed(ctx context.Context, event *PaymentEvent) {
	c.logger.Info("Handling payment failed event", logging.Fields{
		"payment_id": event.PaymentID,
		"order_id":   event.OrderID,
	})

	// Cancel the order due to payment failure
	_, err := c.orderService.CancelOrder(ctx, event.OrderID, "Payment failed")
	if err != nil {
		c.logger.Error("Failed to cancel order", logging.Fields{
			"order_id": event.OrderID,
			"error":    err.Error(),
		})
	}
}

func (c *KafkaConsumer) handlePaymentRefunded(ctx context.Context, event *PaymentEvent) {
	c.logger.Info("Handling payment refunded event", logging.Fields{
		"payment_id": event.PaymentID,
		"order_id":   event.OrderID,
	})

	// Update order status to refunded
	req := &models.UpdateOrderStatusRequest{
		Status: models.OrderStatusRefunded,
		Notes:  "Payment refunded via event",
	}

	_, err := c.orderService.UpdateOrderStatus(ctx, event.OrderID, req)
	if err != nil {
		c.logger.Error("Failed to update order status", logging.Fields{
			"order_id": event.OrderID,
			"error":    err.Error(),
		})
	}
}

// LegacyEventConsumer is the deprecated event consumer.
// Deprecated: Use KafkaConsumer instead.
// TODO(TEAM-PLATFORM): Remove after migration to Kafka complete
type LegacyEventConsumer struct {
	orderService *service.OrderService
	logger       *logging.LoggerV2
}

// NewLegacyEventConsumer creates a deprecated event consumer.
// Deprecated: Use NewKafkaConsumer instead.
func NewLegacyEventConsumer(orderService *service.OrderService) *LegacyEventConsumer {
	// TODO(TEAM-PLATFORM): Migrate to Kafka
	logging.Infof("Warning: Using legacy event consumer")
	return &LegacyEventConsumer{
		orderService: orderService,
		logger:       logging.NewLoggerV2("legacy-consumer"),
	}
}

// Start is a deprecated method.
// Deprecated: Use KafkaConsumer.Start instead.
func (c *LegacyEventConsumer) Start(ctx context.Context) error {
	logging.Infof("Legacy: Starting event consumer (no-op)")
	<-ctx.Done()
	return ctx.Err()
}

// HandlePaymentEventLegacy handles payment events in legacy format.
// Deprecated: Use KafkaConsumer.handleMessage instead.
// TODO(TEAM-PLATFORM): Remove after migration
func (c *LegacyEventConsumer) HandlePaymentEventLegacy(orderID string, status string) error {
	logging.Infof("Legacy: Handling payment event for order: %s, status: %s", orderID, status)

	var orderStatus models.OrderStatus
	switch status {
	case "completed":
		orderStatus = models.OrderStatusConfirmed
	case "failed":
		orderStatus = models.OrderStatusCancelled
	case "refunded":
		orderStatus = models.OrderStatusRefunded
	default:
		return nil
	}

	req := &models.UpdateOrderStatusRequest{
		Status: orderStatus,
		Notes:  "Updated via legacy event",
	}

	_, err := c.orderService.UpdateOrderStatus(context.Background(), orderID, req)
	return err
}
