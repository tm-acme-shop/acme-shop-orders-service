package handlers

import (
	"github.com/tm-acme-shop/acme-shop-orders-service/internal/config"
	"github.com/tm-acme-shop/acme-shop-orders-service/internal/service"
	"github.com/tm-acme-shop/acme-shop-shared-go/logging"
)

// Handlers holds all HTTP handlers for the orders service.
type Handlers struct {
	orderService   *service.OrderService
	paymentService *service.PaymentService
	config         *config.Config
	logger         *logging.LoggerV2
}

// NewHandlers creates a new handlers instance.
func NewHandlers(
	orderService *service.OrderService,
	paymentService *service.PaymentService,
	cfg *config.Config,
) *Handlers {
	return &Handlers{
		orderService:   orderService,
		paymentService: paymentService,
		config:         cfg,
		logger:         logging.NewLoggerV2("handlers"),
	}
}
