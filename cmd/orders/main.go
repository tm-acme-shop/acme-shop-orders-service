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

	orderRepo := repository.NewOrderRepository(db)
	paymentClient := clients.NewPaymentClientV1(cfg.PaymentService.BaseURL)
	orderService := service.NewOrderService(orderRepo, paymentClient)
	orderHandlers := handlers.NewOrderHandlers(orderService)

	srv := server.NewServer(cfg, orderHandlers)

	if err := srv.Run(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
