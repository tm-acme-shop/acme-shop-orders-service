package main

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tm-acme-shop/acme-shop-orders-service/internal/clients"
	"github.com/tm-acme-shop/acme-shop-orders-service/internal/config"
	"github.com/tm-acme-shop/acme-shop-orders-service/internal/events"
	"github.com/tm-acme-shop/acme-shop-orders-service/internal/handlers"
	"github.com/tm-acme-shop/acme-shop-orders-service/internal/repository"
	"github.com/tm-acme-shop/acme-shop-orders-service/internal/server"
	"github.com/tm-acme-shop/acme-shop-orders-service/internal/service"
	"github.com/tm-acme-shop/acme-shop-shared-go/logging"

	_ "github.com/lib/pq"
)

func main() {
	cfg := config.Load()

	logger := logging.NewLoggerV2("orders-service")

	// TODO(TEAM-PLATFORM): Migrate all legacy logging to structured logging
	logging.Infof("Starting orders-service on port %d", cfg.Server.Port)

	db, err := initDatabase(cfg)
	if err != nil {
		logger.Fatal("Failed to connect to database", logging.Fields{"error": err.Error()})
	}
	defer db.Close()

	orderRepo := repository.NewPostgresOrderRepository(db, logger)
	orderCache := repository.NewRedisOrderCache(cfg.Redis)

	// TODO(TEAM-API): Remove legacy repository after migration complete
	legacyRepo := repository.NewPostgresOrderRepositoryV1(db)

	paymentClient := clients.NewHTTPPaymentClient(cfg.PaymentService, logger)
	// TODO(TEAM-PAYMENTS): Remove legacy payment client after migration
	legacyPaymentClient := clients.NewLegacyHTTPPaymentClient(cfg.PaymentService)

	userClient := clients.NewHTTPUserClient(cfg.UserService, logger)
	notificationClient := clients.NewHTTPNotificationClient(cfg.NotificationService, logger)

	eventPublisher := events.NewKafkaPublisher(cfg.Kafka, logger)
	defer eventPublisher.Close()

	orderService := service.NewOrderService(
		orderRepo,
		orderCache,
		legacyRepo,
		paymentClient,
		legacyPaymentClient,
		userClient,
		notificationClient,
		eventPublisher,
		cfg,
	)

	paymentService := service.NewPaymentService(
		paymentClient,
		legacyPaymentClient,
		orderRepo,
		cfg,
	)

	h := handlers.NewHandlers(orderService, paymentService, cfg)

	srv := server.New(h, cfg)

	go func() {
		logger.Info("Server starting", logging.Fields{
			"port":                  cfg.Server.Port,
			"enable_legacy_payments": cfg.Features.EnableLegacyPayments,
			"enable_v1_api":         cfg.Features.EnableV1API,
		})
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed to start", logging.Fields{"error": err.Error()})
		}
	}()

	// Start event consumer
	eventConsumer := events.NewKafkaConsumer(cfg.Kafka, orderService, logger)
	go func() {
		if err := eventConsumer.Start(context.Background()); err != nil {
			logger.Error("Event consumer failed", logging.Fields{"error": err.Error()})
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	eventConsumer.Stop()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", logging.Fields{"error": err.Error()})
	}

	logger.Info("Server exited")
}

func initDatabase(cfg *config.Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.Database.ConnectionString())
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.Database.MaxLifetime)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	// TODO(TEAM-PLATFORM): Run migrations automatically in development
	logging.Info("Database connected", logging.Fields{
		"host": cfg.Database.Host,
		"name": cfg.Database.Name,
	})

	return db, nil
}
