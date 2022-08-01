package main

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
	"github.com/tm-acme-shop/acme-shop-orders-service/internal/clients"
	"github.com/tm-acme-shop/acme-shop-orders-service/internal/config"
	"github.com/tm-acme-shop/acme-shop-orders-service/internal/handlers"
	"github.com/tm-acme-shop/acme-shop-orders-service/internal/repository"
	"github.com/tm-acme-shop/acme-shop-orders-service/internal/server"
	"github.com/tm-acme-shop/acme-shop-orders-service/internal/service"
)

func main() {
	log.Printf("Starting Orders Service")

	cfg := config.Load()

	db, err := sql.Open("postgres", cfg.Database.ConnectionString())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Printf("Connected to database")

	// V1 clients and services
	orderRepo := repository.NewOrderRepository(db)
	paymentClientV1 := clients.NewPaymentClientV1(cfg.PaymentService.BaseURL)
	orderService := service.NewOrderService(orderRepo, paymentClientV1)
	orderHandlers := handlers.NewOrderHandlers(orderService)

	// V2 clients and services
	orderRepoV2 := repository.NewOrderRepositoryV2(db)
	paymentClientV2 := clients.NewPaymentClientV2(cfg.PaymentService.BaseURL)
	paymentClient := clients.NewPaymentClient(paymentClientV1, paymentClientV2, cfg.Features.EnableLegacyPayments)
	orderServiceV2 := service.NewOrderServiceV2(orderRepoV2, paymentClient, cfg)
	orderHandlersV2 := handlers.NewOrderHandlersV2(orderServiceV2)

	srv := server.NewServer(cfg, orderHandlers, orderHandlersV2)

	log.Printf("Feature flags: EnableLegacyPayments=%v", cfg.Features.EnableLegacyPayments)

	if err := srv.Run(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
