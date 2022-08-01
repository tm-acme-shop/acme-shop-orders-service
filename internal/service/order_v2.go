package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/tm-acme-shop/acme-shop-orders-service/internal/clients"
	"github.com/tm-acme-shop/acme-shop-orders-service/internal/config"
	"github.com/tm-acme-shop/acme-shop-orders-service/internal/models"
	"github.com/tm-acme-shop/acme-shop-orders-service/internal/repository"
)

type OrderServiceV2 struct {
	orderRepo     *repository.OrderRepositoryV2
	paymentClient *clients.PaymentClient
	config        *config.Config
}

func NewOrderServiceV2(orderRepo *repository.OrderRepositoryV2, paymentClient *clients.PaymentClient, cfg *config.Config) *OrderServiceV2 {
	return &OrderServiceV2{
		orderRepo:     orderRepo,
		paymentClient: paymentClient,
		config:        cfg,
	}
}

func (s *OrderServiceV2) CreateOrder(ctx context.Context, userID string, items []models.OrderItemV2, currency string, requestID string) (*models.OrderV2, error) {
	log.Printf("Creating order for user %s", userID)

	var total float64
	for _, item := range items {
		total += item.UnitPrice * float64(item.Quantity)
	}

	order, err := s.orderRepo.CreateOrder(ctx, userID, total, currency)
	if err != nil {
		log.Printf("Failed to create order in database: %v", err)
		return nil, err
	}

	log.Printf("Order %s created with total %.2f %s", order.ID, total, currency)
	return order, nil
}

func (s *OrderServiceV2) GetOrder(ctx context.Context, orderID string, requestID string) (*models.OrderV2, error) {
	log.Printf("Getting order %s", orderID)
	return s.orderRepo.GetOrderByID(ctx, orderID)
}

func (s *OrderServiceV2) GetUserOrders(ctx context.Context, userID string) ([]*models.OrderV2, error) {
	log.Printf("Getting orders for user %s", userID)
	return s.orderRepo.GetOrdersByUserID(ctx, userID)
}

func (s *OrderServiceV2) UpdateOrderStatus(ctx context.Context, orderID string, status string, requestID string) (*models.OrderV2, error) {
	log.Printf("Updating order %s status to %s", orderID, status)
	return s.orderRepo.UpdateOrderStatus(ctx, orderID, status)
}

func (s *OrderServiceV2) ProcessPayment(ctx context.Context, orderID string, requestID string) error {
	order, err := s.orderRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		return err
	}

	log.Printf("Processing payment for order %s", orderID)

	paymentID, err := s.paymentClient.Charge(ctx, order.ID, order.Total, order.Currency, requestID)
	if err != nil {
		log.Printf("Payment failed for order %s: %v", orderID, err)
		return err
	}

	log.Printf("Payment %s completed for order %s", paymentID, orderID)
	_, err = s.orderRepo.UpdateOrderStatus(ctx, orderID, "paid")
	return err
}

func generateOrderID() string {
	return fmt.Sprintf("ord_%d", time.Now().UnixNano())
}
