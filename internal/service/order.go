package service

import (
	"context"
	"log"

	"github.com/tm-acme-shop/acme-shop-orders-service/internal/clients"
	"github.com/tm-acme-shop/acme-shop-orders-service/internal/models"
	"github.com/tm-acme-shop/acme-shop-orders-service/internal/repository"
)

type OrderService struct {
	orderRepo     *repository.OrderRepository
	paymentClient *clients.PaymentClientV1
}

func NewOrderService(orderRepo *repository.OrderRepository, paymentClient *clients.PaymentClientV1) *OrderService {
	return &OrderService{
		orderRepo:     orderRepo,
		paymentClient: paymentClient,
	}
}

func (s *OrderService) CreateOrderV1(ctx context.Context, userID int64, items []models.OrderItem, currency string) (*models.Order, error) {
	log.Printf("Creating order for user %d", userID)

	var total float64
	for _, item := range items {
		total += item.Price * float64(item.Quantity)
	}

	order, err := s.orderRepo.CreateOrder(ctx, userID, total, currency)
	if err != nil {
		log.Printf("Failed to create order in database: %v", err)
		return nil, err
	}

	log.Printf("Order %d created with total %.2f %s", order.ID, total, currency)
	return order, nil
}

func (s *OrderService) GetOrderV1(ctx context.Context, orderID int64) (*models.Order, error) {
	log.Printf("Getting order %d", orderID)
	return s.orderRepo.GetOrderByID(ctx, orderID)
}

func (s *OrderService) GetUserOrdersV1(ctx context.Context, userID int64) ([]*models.Order, error) {
	log.Printf("Getting orders for user %d", userID)
	return s.orderRepo.GetOrdersByUserID(ctx, userID)
}

func (s *OrderService) UpdateOrderStatusV1(ctx context.Context, orderID int64, status string) (*models.Order, error) {
	log.Printf("Updating order %d status to %s", orderID, status)
	return s.orderRepo.UpdateOrderStatus(ctx, orderID, status)
}

func (s *OrderService) ProcessPaymentV1(ctx context.Context, orderID int64) error {
	order, err := s.orderRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		return err
	}

	log.Printf("Processing payment for order %d", orderID)

	_, err = s.paymentClient.Charge(
		string(rune(order.ID)),
		order.Total,
		order.Currency,
	)
	if err != nil {
		log.Printf("Payment failed for order %d: %v", orderID, err)
		return err
	}

	_, err = s.orderRepo.UpdateOrderStatus(ctx, orderID, "paid")
	return err
}
