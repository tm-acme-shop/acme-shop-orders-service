package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/tm-acme-shop/acme-shop-orders-service/internal/repository"
	"github.com/tm-acme-shop/acme-shop-orders-service/internal/service"
	"github.com/tm-acme-shop/acme-shop-shared-go/errors"
	"github.com/tm-acme-shop/acme-shop-shared-go/logging"
	"github.com/tm-acme-shop/acme-shop-shared-go/middleware"
	"github.com/tm-acme-shop/acme-shop-shared-go/models"
)

// API-155: v2 order endpoints with structured responses (2023-05)
// CreateOrder handles POST /api/v2/orders
func (h *Handlers) CreateOrder(c *gin.Context) {
	var req models.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind request", logging.Fields{"error": err.Error()})
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Get user ID from context if not provided
	if req.UserID == "" {
		if userID, exists := c.Get("user_id"); exists {
			req.UserID = userID.(string)
		}
	}

	order, err := h.orderService.CreateOrder(c.Request.Context(), &req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, order)
}

// API-105: Initial orders v1 API (2022-04)
// CreateOrderV1 handles POST /api/v1/orders
// Deprecated: Use CreateOrder (v2) instead.
// TODO(TEAM-API): Remove after v1 API migration complete
func (h *Handlers) CreateOrderV1(c *gin.Context) {
	logging.Infof("Legacy: Creating order via v1 API")

	var req repository.LegacyCreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// TODO(TEAM-API): Migrate to v2 API
	// For now, convert to v2 format internally
	v2Req := &models.CreateOrderRequest{
		UserID: strconv.FormatInt(req.UserID, 10),
		Items:  make([]models.OrderItem, len(req.Items)),
	}

	for i, item := range req.Items {
		v2Req.Items[i] = models.OrderItem{
			ProductID: strconv.FormatInt(item.ProductID, 10),
			Quantity:  item.Quantity,
			UnitPrice: models.NewMoney(item.Price, req.Currency),
			Total:     models.NewMoney(item.Price*float64(item.Quantity), req.Currency),
		}
	}

	order, err := h.orderService.CreateOrder(c.Request.Context(), v2Req)
	if err != nil {
		handleError(c, err)
		return
	}

	// Convert to legacy response format
	legacyResp := repository.ConvertToLegacyOrder(
		order.ID,
		order.UserID,
		string(order.Status),
		order.Total.ToFloat(),
		order.Total.Currency,
	)

	c.JSON(http.StatusCreated, legacyResp)
}

// GetOrder handles GET /api/v2/orders/:id
func (h *Handlers) GetOrder(c *gin.Context) {
	orderID := c.Param("id")

	order, err := h.orderService.GetOrder(c.Request.Context(), orderID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, order)
}

// GetOrderV1 handles GET /api/v1/orders/:id
// Deprecated: Use GetOrder (v2) instead.
// TODO(TEAM-API): Remove after v1 API migration complete
func (h *Handlers) GetOrderV1(c *gin.Context) {
	logging.Infof("Legacy: Getting order via v1 API")

	orderIDStr := c.Param("id")
	orderID, err := strconv.ParseInt(orderIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order ID"})
		return
	}

	order, err := h.orderService.GetOrderV1(c.Request.Context(), orderID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, order)
}

// UpdateOrderStatus handles PATCH /api/v2/orders/:id/status
func (h *Handlers) UpdateOrderStatus(c *gin.Context) {
	orderID := c.Param("id")

	var req models.UpdateOrderStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := service.ValidateUpdateOrderStatusRequest(&req); err != nil {
		handleError(c, err)
		return
	}

	order, err := h.orderService.UpdateOrderStatus(c.Request.Context(), orderID, &req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, order)
}

// UpdateOrderStatusV1 handles POST /api/v1/orders/:id/status
// Deprecated: Use UpdateOrderStatus (v2) instead.
// TODO(TEAM-API): Remove after v1 API migration complete
func (h *Handlers) UpdateOrderStatusV1(c *gin.Context) {
	logging.Infof("Legacy: Updating order status via v1 API")

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

	// Use v2 internally but convert IDs
	v2Req := &models.UpdateOrderStatusRequest{
		Status: models.OrderStatus(req.Status),
	}

	order, err := h.orderService.UpdateOrderStatus(
		c.Request.Context(),
		strconv.FormatInt(orderID, 10),
		v2Req,
	)
	if err != nil {
		handleError(c, err)
		return
	}

	// Return legacy format
	c.JSON(http.StatusOK, gin.H{
		"id":     orderID,
		"status": order.Status,
	})
}

// CancelOrder handles POST /api/v2/orders/:id/cancel
func (h *Handlers) CancelOrder(c *gin.Context) {
	orderID := c.Param("id")

	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := service.ValidateCancellationReason(req.Reason); err != nil {
		handleError(c, err)
		return
	}

	order, err := h.orderService.CancelOrder(c.Request.Context(), orderID, req.Reason)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, order)
}

// ListOrders handles GET /api/v2/orders
func (h *Handlers) ListOrders(c *gin.Context) {
	filter := &models.OrderListFilter{}

	// Parse query parameters
	if userID := c.Query("user_id"); userID != "" {
		filter.UserID = userID
	}

	if status := c.Query("status"); status != "" {
		s := models.OrderStatus(status)
		filter.Status = &s
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = limit
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			filter.Offset = offset
		}
	}

	if err := service.ValidateOrderListFilter(filter); err != nil {
		handleError(c, err)
		return
	}

	orders, total, err := h.orderService.ListOrders(c.Request.Context(), filter)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"orders": orders,
		"total":  total,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

// ListOrdersV1 handles GET /api/v1/orders
// Deprecated: Use ListOrders (v2) instead.
// TODO(TEAM-API): Remove after v1 API migration complete
func (h *Handlers) ListOrdersV1(c *gin.Context) {
	logging.Infof("Legacy: Listing orders via v1 API")

	userIDStr := c.Query("user_id")
	if userIDStr == "" {
		// Try to get from legacy header
		// TODO(TEAM-API): Remove legacy header support
		userIDStr = c.GetHeader(middleware.HeaderLegacyUserID)
	}

	if userIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	orders, err := h.orderService.GetUserOrdersV1(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"orders": orders,
		"count":  len(orders),
	})
}

// GetUserOrders handles GET /api/v2/users/:user_id/orders
func (h *Handlers) GetUserOrders(c *gin.Context) {
	userID := c.Param("user_id")

	limit := 20
	offset := 0

	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			offset = o
		}
	}

	orders, total, err := h.orderService.GetUserOrders(c.Request.Context(), userID, limit, offset)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"orders": orders,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GetUserOrdersV1 handles GET /api/v1/users/:user_id/orders
// Deprecated: Use GetUserOrders (v2) instead.
// TODO(TEAM-API): Remove after v1 API migration complete
func (h *Handlers) GetUserOrdersV1(c *gin.Context) {
	logging.Infof("Legacy: Getting user orders via v1 API")

	userIDStr := c.Param("user_id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	orders, err := h.orderService.GetUserOrdersV1(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, orders)
}

// RefundOrder handles POST /api/v2/orders/:id/refund
func (h *Handlers) RefundOrder(c *gin.Context) {
	orderID := c.Param("id")

	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if req.Reason == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "reason is required"})
		return
	}

	refundResp, err := h.orderService.RefundOrder(c.Request.Context(), orderID, req.Reason)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, refundResp)
}

func handleError(c *gin.Context, err error) {
	if err == errors.ErrNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	if validationErr, ok := err.(*errors.ValidationError); ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   validationErr.Message,
			"details": validationErr.Details,
		})
		return
	}

	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}
