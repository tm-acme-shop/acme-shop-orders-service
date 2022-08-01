package server

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/tm-acme-shop/acme-shop-orders-service/internal/config"
	"github.com/tm-acme-shop/acme-shop-orders-service/internal/handlers"
)

type Server struct {
	config          *config.Config
	router          *gin.Engine
	orderHandlers   *handlers.OrderHandlers
	orderHandlersV2 *handlers.OrderHandlersV2
}

func NewServer(cfg *config.Config, orderHandlers *handlers.OrderHandlers, orderHandlersV2 *handlers.OrderHandlersV2) *Server {
	router := gin.Default()

	s := &Server{
		config:          cfg,
		router:          router,
		orderHandlers:   orderHandlers,
		orderHandlersV2: orderHandlersV2,
	}

	s.setupRoutes()

	return s
}

func (s *Server) setupRoutes() {
	s.router.GET("/health", handlers.HealthCheck)
	s.router.GET("/ready", handlers.ReadinessCheck)

	v1 := s.router.Group("/api/v1")
	{
		v1.POST("/orders", s.orderHandlers.CreateOrderV1)
		v1.GET("/orders/:id", s.orderHandlers.GetOrderV1)
		v1.GET("/orders", s.orderHandlers.ListOrdersV1)
		v1.POST("/orders/:id/status", s.orderHandlers.UpdateOrderStatusV1)
	}

	v2 := s.router.Group("/api/v2")
	{
		v2.POST("/orders", s.orderHandlersV2.CreateOrder)
		v2.GET("/orders/:id", s.orderHandlersV2.GetOrder)
		v2.GET("/orders", s.orderHandlersV2.ListOrders)
		v2.PATCH("/orders/:id/status", s.orderHandlersV2.UpdateOrderStatus)
	}
}

func (s *Server) Run() error {
	addr := fmt.Sprintf(":%d", s.config.Server.Port)
	log.Printf("Starting server on %s", addr)
	return s.router.Run(addr)
}
