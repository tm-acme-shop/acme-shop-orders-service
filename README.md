# AcmeShop Orders Service

Order lifecycle management service for the AcmeShop e-commerce platform.

## Overview

The Orders Service handles:
- Order creation and management
- Order lifecycle (pending → confirmed → processing → shipped → delivered)
- Payment processing integration
- Order event publishing
- User order history

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Orders Service                              │
├─────────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                 │
│  │   Handlers  │  │   Service   │  │ Repository  │                 │
│  │  (v1 & v2)  │──│   Layer     │──│  (Postgres) │                 │
│  └─────────────┘  └─────────────┘  └─────────────┘                 │
│         │               │                │                          │
│         │         ┌─────┴─────┐    ┌─────┴─────┐                   │
│         │         │  Clients  │    │   Cache   │                   │
│         │         │ (Payment, │    │  (Redis)  │                   │
│         │         │  User,    │    └───────────┘                   │
│         │         │  Notify)  │                                     │
│         │         └───────────┘                                     │
│         │               │                                           │
│         └───────────────┴──────────────────────┐                   │
│                                                 │                   │
│  ┌───────────────────────────────────────────────────────────────┐ │
│  │                      Events (Kafka)                           │ │
│  │  - Order Created/Updated/Cancelled                            │ │
│  │  - Payment Events Consumer                                    │ │
│  └───────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────┘
```

## API Endpoints

### V2 API (Current)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v2/orders` | Create new order |
| GET | `/api/v2/orders` | List orders |
| GET | `/api/v2/orders/:id` | Get order by ID |
| PATCH | `/api/v2/orders/:id/status` | Update order status |
| POST | `/api/v2/orders/:id/cancel` | Cancel order |
| POST | `/api/v2/orders/:id/payment` | Process payment |
| GET | `/api/v2/orders/:id/payment` | Get order payment |
| POST | `/api/v2/orders/:id/refund` | Refund order |
| GET | `/api/v2/users/:user_id/orders` | Get user orders |
| GET | `/api/v2/payments/:id` | Get payment status |
| POST | `/api/v2/payments/:id/cancel` | Cancel payment |
| POST | `/api/v2/payments/:id/refund` | Process refund |

### V1 API (Deprecated)

> **TODO(TEAM-API)**: Remove after v1 API migration complete

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/orders` | Create order (legacy) |
| GET | `/api/v1/orders` | List orders (legacy) |
| GET | `/api/v1/orders/:id` | Get order (legacy) |
| POST | `/api/v1/orders/:id/status` | Update status (legacy) |
| POST | `/api/v1/orders/:id/pay` | Process payment (legacy) |
| GET | `/api/v1/users/:user_id/orders` | Get user orders (legacy) |
| GET | `/api/v1/payments/status` | Get payment status (legacy) |

### Health Endpoints

| Endpoint | Description |
|----------|-------------|
| `/health` | Service health check |
| `/ready` | Readiness probe |
| `/live` | Liveness probe |
| `/metrics` | Prometheus metrics |
| `/version` | Service version info |

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | 8082 | HTTP server port |
| `DB_HOST` | localhost | PostgreSQL host |
| `DB_PORT` | 5432 | PostgreSQL port |
| `DB_USER` | acme | Database user |
| `DB_PASSWORD` | acme | Database password |
| `DB_NAME` | acme_orders | Database name |
| `REDIS_HOST` | localhost | Redis host |
| `REDIS_PORT` | 6379 | Redis port |
| `KAFKA_BROKERS` | localhost:9092 | Kafka brokers |
| `PAYMENT_SERVICE_URL` | http://localhost:8083 | Payment service URL |
| `USER_SERVICE_URL` | http://localhost:8081 | User service URL |
| `NOTIFICATION_SERVICE_URL` | http://localhost:8084 | Notification service URL |

### Feature Flags

| Flag | Default | Description |
|------|---------|-------------|
| `ENABLE_V1_API` | true | Enable deprecated v1 API |
| `ENABLE_LEGACY_PAYMENTS` | true | Enable legacy payment path |
| `ENABLE_ORDER_EVENTS` | true | Enable Kafka event publishing |
| `ENABLE_ORDER_CACHING` | true | Enable Redis caching |

## Development

### Prerequisites

- Go 1.21+
- PostgreSQL 14+
- Redis 7+
- Apache Kafka (optional, for events)

### Running Locally

```bash
# Start dependencies
docker-compose up -d postgres redis

# Run the service
go run ./cmd/orders
```

### Running Tests

```bash
# Unit tests
go test ./...

# With coverage
go test -coverprofile=coverage.out ./...

# Integration tests (requires services)
go test -tags=integration ./...
```

### Building

```bash
# Build binary
go build -o bin/orders-service ./cmd/orders

# Build Docker image
docker build -t acme-shop-orders-service .
```

## Dependencies

- [acme-shop-shared-go](../acme-shop-shared-go) - Shared Go library
- [gin-gonic/gin](https://github.com/gin-gonic/gin) - HTTP framework
- [lib/pq](https://github.com/lib/pq) - PostgreSQL driver
- [redis/go-redis](https://github.com/redis/go-redis) - Redis client
- [segmentio/kafka-go](https://github.com/segmentio/kafka-go) - Kafka client

## Service Integrations

### Payment Service
- Process payments via `POST /api/v2/payments`
- Get payment status via `GET /api/v2/payments/:id`
- Process refunds via `POST /api/v2/payments/:id/refund`

### User Service
- Validate users via `GET /api/v2/users/:id`
- Get user details for order processing

### Notification Service
- Send order confirmation emails
- Send shipping notifications
- Send cancellation notifications

## Events

### Published Events (Kafka)

| Event Type | Description |
|------------|-------------|
| `order.created` | New order created |
| `order.status_changed` | Order status updated |
| `order.cancelled` | Order cancelled |

### Consumed Events (Kafka)

| Event Type | Description |
|------------|-------------|
| `payment.completed` | Payment succeeded → confirm order |
| `payment.failed` | Payment failed → cancel order |
| `payment.refunded` | Payment refunded → update order |

## TODO

- [ ] TODO(TEAM-API): Remove v1 API after migration
- [ ] TODO(TEAM-PAYMENTS): Remove legacy payment client
- [ ] TODO(TEAM-SEC): Add authentication middleware
- [ ] TODO(TEAM-PLATFORM): Add proper metrics and tracing
- [ ] TODO(TEAM-PLATFORM): Update GitHub Actions to v4/v5
