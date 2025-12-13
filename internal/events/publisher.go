package events

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/tm-acme-shop/acme-shop-orders-service/internal/config"
	"github.com/tm-acme-shop/acme-shop-shared-go/interfaces"
	"github.com/tm-acme-shop/acme-shop-shared-go/logging"
	"github.com/tm-acme-shop/acme-shop-shared-go/middleware"
	"github.com/tm-acme-shop/acme-shop-shared-go/models"
)

// Ensure KafkaPublisher implements interfaces.OrderEventPublisher
var _ interfaces.OrderEventPublisher = (*KafkaPublisher)(nil)

// EventType represents the type of order event.
type EventType string

const (
	EventTypeOrderCreated       EventType = "order.created"
	EventTypeOrderStatusChanged EventType = "order.status_changed"
	EventTypeOrderCancelled     EventType = "order.cancelled"
	EventTypeOrderRefunded      EventType = "order.refunded"
)

// OrderEvent represents an order-related event.
type OrderEvent struct {
	ID             string            `json:"id"`
	Type           EventType         `json:"type"`
	OrderID        string            `json:"order_id"`
	UserID         string            `json:"user_id"`
	Data           json.RawMessage   `json:"data"`
	Metadata       map[string]string `json:"metadata"`
	Timestamp      time.Time         `json:"timestamp"`
	CorrelationID  string            `json:"correlation_id,omitempty"`
}

// KafkaPublisher publishes order events to Kafka.
type KafkaPublisher struct {
	writer *kafka.Writer
	topic  string
	logger *logging.LoggerV2
}

// NewKafkaPublisher creates a new Kafka-based event publisher.
func NewKafkaPublisher(cfg config.KafkaConfig, logger *logging.LoggerV2) *KafkaPublisher {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        cfg.OrdersTopic,
		Balancer:     &kafka.LeastBytes{},
		WriteTimeout: 10 * time.Second,
		RequiredAcks: kafka.RequireOne,
	}

	return &KafkaPublisher{
		writer: writer,
		topic:  cfg.OrdersTopic,
		logger: logger,
	}
}

// PublishOrderCreated publishes an order created event.
func (p *KafkaPublisher) PublishOrderCreated(ctx context.Context, order *models.Order) error {
	p.logger.Debug("Publishing order created event", logging.Fields{
		"order_id": order.ID,
	})

	data, err := json.Marshal(order)
	if err != nil {
		return err
	}

	event := p.createEvent(ctx, EventTypeOrderCreated, order.ID, order.UserID, data)
	return p.publish(ctx, event)
}

// PublishOrderStatusChanged publishes an order status change event.
func (p *KafkaPublisher) PublishOrderStatusChanged(ctx context.Context, order *models.Order, previousStatus models.OrderStatus) error {
	p.logger.Debug("Publishing order status changed event", logging.Fields{
		"order_id":        order.ID,
		"previous_status": previousStatus,
		"new_status":      order.Status,
	})

	payload := struct {
		Order          *models.Order      `json:"order"`
		PreviousStatus models.OrderStatus `json:"previous_status"`
		NewStatus      models.OrderStatus `json:"new_status"`
	}{
		Order:          order,
		PreviousStatus: previousStatus,
		NewStatus:      order.Status,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	event := p.createEvent(ctx, EventTypeOrderStatusChanged, order.ID, order.UserID, data)
	return p.publish(ctx, event)
}

// PublishOrderCancelled publishes an order cancellation event.
func (p *KafkaPublisher) PublishOrderCancelled(ctx context.Context, order *models.Order, reason string) error {
	p.logger.Debug("Publishing order cancelled event", logging.Fields{
		"order_id": order.ID,
		"reason":   reason,
	})

	payload := struct {
		Order  *models.Order `json:"order"`
		Reason string        `json:"reason"`
	}{
		Order:  order,
		Reason: reason,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	event := p.createEvent(ctx, EventTypeOrderCancelled, order.ID, order.UserID, data)
	return p.publish(ctx, event)
}

func (p *KafkaPublisher) createEvent(ctx context.Context, eventType EventType, orderID, userID string, data []byte) *OrderEvent {
	event := &OrderEvent{
		ID:        generateEventID(),
		Type:      eventType,
		OrderID:   orderID,
		UserID:    userID,
		Data:      data,
		Metadata:  make(map[string]string),
		Timestamp: time.Now(),
	}

	// Add correlation ID from context
	if requestID := ctx.Value(middleware.RequestIDKey); requestID != nil {
		event.CorrelationID = requestID.(string)
	}

	return event
}

func (p *KafkaPublisher) publish(ctx context.Context, event *OrderEvent) error {
	eventData, err := json.Marshal(event)
	if err != nil {
		return err
	}

	msg := kafka.Message{
		Key:   []byte(event.OrderID),
		Value: eventData,
		Headers: []kafka.Header{
			{Key: "event_type", Value: []byte(event.Type)},
			{Key: "event_id", Value: []byte(event.ID)},
		},
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		p.logger.Error("Failed to publish event", logging.Fields{
			"event_id":   event.ID,
			"event_type": event.Type,
			"order_id":   event.OrderID,
			"error":      err.Error(),
		})
		return err
	}

	p.logger.Info("Event published", logging.Fields{
		"event_id":   event.ID,
		"event_type": event.Type,
		"order_id":   event.OrderID,
	})

	return nil
}

// Close closes the Kafka writer.
func (p *KafkaPublisher) Close() error {
	p.logger.Info("Closing Kafka publisher")
	return p.writer.Close()
}

func generateEventID() string {
	// TODO(TEAM-PLATFORM): Use proper UUID generation
	return "evt_" + time.Now().Format("20060102150405.000000")
}

// LegacyEventPublisher is the deprecated event publisher.
// Deprecated: Use KafkaPublisher instead.
// TODO(TEAM-PLATFORM): Remove after migration to Kafka complete
type LegacyEventPublisher struct {
	logger *logging.LoggerV2
}

// NewLegacyEventPublisher creates a deprecated event publisher.
// Deprecated: Use NewKafkaPublisher instead.
func NewLegacyEventPublisher() *LegacyEventPublisher {
	log.Printf("Warning: Using legacy event publisher - migrate to Kafka")
	return &LegacyEventPublisher{
		logger: logging.NewLoggerV2("legacy-publisher"),
	}
}

// PublishOrderCreated is a deprecated method.
// Deprecated: Use KafkaPublisher.PublishOrderCreated instead.
func (p *LegacyEventPublisher) PublishOrderCreated(ctx context.Context, order *models.Order) error {
	// TODO(TEAM-PLATFORM): Migrate to Kafka
	log.Printf("Legacy: Publishing order created event for order: %s", order.ID)
	return nil
}

// PublishOrderStatusChanged is a deprecated method.
// Deprecated: Use KafkaPublisher.PublishOrderStatusChanged instead.
func (p *LegacyEventPublisher) PublishOrderStatusChanged(ctx context.Context, order *models.Order, previousStatus models.OrderStatus) error {
	log.Printf("Legacy: Publishing order status changed event for order: %s", order.ID)
	return nil
}

// PublishOrderCancelled is a deprecated method.
// Deprecated: Use KafkaPublisher.PublishOrderCancelled instead.
func (p *LegacyEventPublisher) PublishOrderCancelled(ctx context.Context, order *models.Order, reason string) error {
	log.Printf("Legacy: Publishing order cancelled event for order: %s", order.ID)
	return nil
}

// MockEventPublisher is a mock implementation for testing.
type MockEventPublisher struct {
	Events []*OrderEvent
}

func NewMockEventPublisher() *MockEventPublisher {
	return &MockEventPublisher{
		Events: make([]*OrderEvent, 0),
	}
}

func (m *MockEventPublisher) PublishOrderCreated(ctx context.Context, order *models.Order) error {
	m.Events = append(m.Events, &OrderEvent{
		Type:    EventTypeOrderCreated,
		OrderID: order.ID,
	})
	return nil
}

func (m *MockEventPublisher) PublishOrderStatusChanged(ctx context.Context, order *models.Order, previousStatus models.OrderStatus) error {
	m.Events = append(m.Events, &OrderEvent{
		Type:    EventTypeOrderStatusChanged,
		OrderID: order.ID,
	})
	return nil
}

func (m *MockEventPublisher) PublishOrderCancelled(ctx context.Context, order *models.Order, reason string) error {
	m.Events = append(m.Events, &OrderEvent{
		Type:    EventTypeOrderCancelled,
		OrderID: order.ID,
	})
	return nil
}
