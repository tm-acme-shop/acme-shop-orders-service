package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tm-acme-shop/acme-shop-orders-service/internal/models"
	"github.com/tm-acme-shop/acme-shop-orders-service/internal/service"
)

type OrderHandlersV2 struct {
	orderService *service.OrderServiceV2
}

func NewOrderHandlersV2(orderService *service.OrderServiceV2) *OrderHandlersV2 {
	return &OrderHandlersV2{
		orderService: orderService,
	}
}

type CreateOrderV2Request struct {
	UserID   string             `json:"user_id"`
	Items    []models.OrderItemV2 `json:"items"`
	Currency string             `json:"currency"`
}

func (h *OrderHandlersV2) CreateOrder(c *gin.Context) {
	log.Printf("Creating order via v2 API")

	var req CreateOrderV2Request
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Failed to bind request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	requestID := c.GetHeader("X-Acme-Request-ID")
	if requestID != "" {
		log.Printf("Processing request with ID: %s", requestID)
	}

	order, err := h.orderService.CreateOrder(c.Request.Context(), req.UserID, req.Items, req.Currency, requestID)
	if err != nil {
		log.Printf("Failed to create order: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create order"})
		return
	}

	log.Printf("Order created: %s", order.ID)
	c.JSON(http.StatusCreated, order)
}

func (h *OrderHandlersV2) GetOrder(c *gin.Context) {
	orderID := c.Param("id")
	log.Printf("Getting order %s via v2 API", orderID)

	requestID := c.GetHeader("X-Acme-Request-ID")

	order, err := h.orderService.GetOrder(c.Request.Context(), orderID, requestID)
	if err != nil {
		log.Printf("Failed to get order: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get order"})
		return
	}

	if order == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}

	c.JSON(http.StatusOK, order)
}

func (h *OrderHandlersV2) ListOrders(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	log.Printf("Listing orders for user %s via v2 API", userID)

	orders, err := h.orderService.GetUserOrders(c.Request.Context(), userID)
	if err != nil {
		log.Printf("Failed to list orders: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list orders"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"orders": orders,
		"total":  len(orders),
	})
}

type UpdateOrderStatusV2Request struct {
	Status string `json:"status"`
	Notes  string `json:"notes,omitempty"`
}

func (h *OrderHandlersV2) UpdateOrderStatus(c *gin.Context) {
	orderID := c.Param("id")

	var req UpdateOrderStatusV2Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	log.Printf("Updating order %s status to %s via v2 API", orderID, req.Status)

	requestID := c.GetHeader("X-Acme-Request-ID")

	order, err := h.orderService.UpdateOrderStatus(c.Request.Context(), orderID, req.Status, requestID)
	if err != nil {
		log.Printf("Failed to update order status: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update order status"})
		return
	}

	c.JSON(http.StatusOK, order)
}
