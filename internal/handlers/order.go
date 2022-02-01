package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/tm-acme-shop/acme-shop-orders-service/internal/models"
	"github.com/tm-acme-shop/acme-shop-orders-service/internal/service"
)

type OrderHandlers struct {
	orderService *service.OrderService
}

func NewOrderHandlers(orderService *service.OrderService) *OrderHandlers {
	return &OrderHandlers{
		orderService: orderService,
	}
}

type CreateOrderV1Request struct {
	UserID   int64              `json:"user_id"`
	Items    []models.OrderItem `json:"items"`
	Currency string             `json:"currency"`
}

func (h *OrderHandlers) CreateOrderV1(c *gin.Context) {
	log.Printf("Creating order via v1 API")

	var req CreateOrderV1Request
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Failed to bind request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	order, err := h.orderService.CreateOrderV1(c.Request.Context(), req.UserID, req.Items, req.Currency)
	if err != nil {
		log.Printf("Failed to create order: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create order"})
		return
	}

	log.Printf("Order created: %d", order.ID)
	c.JSON(http.StatusCreated, order)
}

func (h *OrderHandlers) GetOrderV1(c *gin.Context) {
	orderIDStr := c.Param("id")
	orderID, err := strconv.ParseInt(orderIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order ID"})
		return
	}

	log.Printf("Fetching order %d via v1 API", orderID)

	order, err := h.orderService.GetOrderV1(c.Request.Context(), orderID)
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

func (h *OrderHandlers) ListOrdersV1(c *gin.Context) {
	userIDStr := c.Query("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	log.Printf("Listing orders for user %d via v1 API", userID)

	orders, err := h.orderService.GetUserOrdersV1(c.Request.Context(), userID)
	if err != nil {
		log.Printf("Failed to list orders: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list orders"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"orders": orders,
		"count":  len(orders),
	})
}

func (h *OrderHandlers) UpdateOrderStatusV1(c *gin.Context) {
	orderIDStr := c.Param("id")
	orderID, err := strconv.ParseInt(orderIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order ID"})
		return
	}

	var req struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	log.Printf("Updating order %d status to %s via v1 API", orderID, req.Status)

	order, err := h.orderService.UpdateOrderStatusV1(c.Request.Context(), orderID, req.Status)
	if err != nil {
		log.Printf("Failed to update order status: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update order status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":     order.ID,
		"status": order.Status,
	})
}
