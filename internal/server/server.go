package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tm-acme-shop/acme-shop-orders-service/internal/config"
	"github.com/tm-acme-shop/acme-shop-orders-service/internal/handlers"
	"github.com/tm-acme-shop/acme-shop-shared-go/logging"
	"github.com/tm-acme-shop/acme-shop-shared-go/middleware"
)

// Server represents the HTTP server.
type Server struct {
	router     *gin.Engine
	httpServer *http.Server
	handlers   *handlers.Handlers
	config     *config.Config
	logger     *logging.LoggerV2
}

// New creates a new server instance.
func New(h *handlers.Handlers, cfg *config.Config) *Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	s := &Server{
		router:   router,
		handlers: h,
		config:   cfg,
		logger:   logging.NewLoggerV2("server"),
	}

	s.setupMiddleware()
	s.setupRoutes()

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	return s
}

func (s *Server) setupMiddleware() {
	s.router.Use(gin.Recovery())
	s.router.Use(middleware.RequestIDMiddleware())
	s.router.Use(middleware.LoggingMiddleware())
	s.router.Use(middleware.CORSMiddleware())

	// TODO(TEAM-PLATFORM): Add rate limiting middleware
	// TODO(TEAM-SEC): Add authentication middleware for protected routes
}

func (s *Server) setupRoutes() {
	// Health endpoints
	s.router.GET("/health", s.handlers.Health)
	s.router.GET("/ready", s.handlers.Ready)
	s.router.GET("/live", s.handlers.Live)
	s.router.GET("/version", s.handlers.Version)
	s.router.GET("/metrics/prometheus", gin.WrapH(promhttp.Handler()))
	s.router.GET("/metrics", s.handlers.Metrics)

	// Debug endpoints (disable in production)
	// TODO(TEAM-SEC): Add authentication for debug endpoints
	s.router.GET("/debug", s.handlers.Debug)

	// V1 API routes (deprecated)
	// TODO(TEAM-API): Remove after v1 API migration complete
	if s.config.Features.EnableV1API {
		v1 := s.router.Group("/api/v1")
		s.setupV1Routes(v1)
	}

	// V2 API routes
	v2 := s.router.Group("/api/v2")
	s.setupV2Routes(v2)

	// Webhook routes
	webhooks := s.router.Group("/api/webhooks")
	s.setupWebhookRoutes(webhooks)
}

func (s *Server) setupV1Routes(rg *gin.RouterGroup) {
	// TODO(TEAM-API): Remove all v1 routes after migration complete
	logging.Infof("Setting up deprecated v1 API routes")

	// Order routes (legacy)
	orders := rg.Group("/orders")
	{
		orders.POST("", s.handlers.CreateOrderV1)
		orders.GET("", s.handlers.ListOrdersV1)
		orders.GET("/:id", s.handlers.GetOrderV1)
		orders.POST("/:id/status", s.handlers.UpdateOrderStatusV1)
		orders.POST("/:id/pay", s.handlers.ProcessOrderPaymentV1)
	}

	// User order routes (legacy)
	users := rg.Group("/users")
	{
		users.GET("/:user_id/orders", s.handlers.GetUserOrdersV1)
	}

	// Payment routes (legacy)
	payments := rg.Group("/payments")
	{
		payments.GET("/status", s.handlers.GetPaymentStatusV1)
		payments.POST("/:id/refund", s.handlers.ProcessRefundV1)
	}

	// Legacy webhook endpoint
	rg.POST("/webhooks/payment", s.handlers.PaymentWebhookV1)
}

func (s *Server) setupV2Routes(rg *gin.RouterGroup) {
	s.logger.Info("Setting up v2 API routes")

	// Order routes
	orders := rg.Group("/orders")
	{
		orders.POST("", s.handlers.CreateOrder)
		orders.GET("", s.handlers.ListOrders)
		orders.GET("/:id", s.handlers.GetOrder)
		orders.PATCH("/:id/status", s.handlers.UpdateOrderStatus)
		orders.POST("/:id/cancel", s.handlers.CancelOrder)
		orders.POST("/:id/payment", s.handlers.ProcessOrderPayment)
		orders.GET("/:id/payment", s.handlers.GetOrderPayment)
		orders.POST("/:id/refund", s.handlers.RefundOrder)
	}

	// User order routes
	users := rg.Group("/users")
	{
		users.GET("/:user_id/orders", s.handlers.GetUserOrders)
	}

	// Payment routes
	payments := rg.Group("/payments")
	{
		payments.GET("/:id", s.handlers.GetPaymentStatus)
		payments.POST("/:id/cancel", s.handlers.CancelPayment)
		payments.POST("/:id/refund", s.handlers.ProcessRefund)
	}
}

func (s *Server) setupWebhookRoutes(rg *gin.RouterGroup) {
	// V2 webhooks
	rg.POST("/v2/payment", s.handlers.PaymentWebhook)

	// V1 webhooks (deprecated)
	// TODO(TEAM-API): Remove after v1 webhook migration
	if s.config.Features.EnableV1API {
		rg.POST("/payment", s.handlers.PaymentWebhookV1)
	}
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	s.logger.Info("Starting HTTP server", logging.Fields{
		"port": s.config.Server.Port,
	})
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down HTTP server")
	return s.httpServer.Shutdown(ctx)
}

// Router returns the gin router for testing.
func (s *Server) Router() *gin.Engine {
	return s.router
}
