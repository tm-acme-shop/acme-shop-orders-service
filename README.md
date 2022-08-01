# ACME Shop Orders Service

Orders management service for ACME Shop platform.

## Overview

This service handles order creation, retrieval, and management for the ACME Shop e-commerce platform. The service supports both V1 (legacy) and V2 API endpoints.

## API Endpoints

### V1 API (Legacy)

- `POST /api/v1/orders` - Create a new order
- `GET /api/v1/orders/:id` - Get order by ID
- `GET /api/v1/orders` - List orders for a user
- `POST /api/v1/orders/:id/status` - Update order status

### V2 API

- `POST /api/v2/orders` - Create a new order
- `GET /api/v2/orders/:id` - Get order by ID  
- `GET /api/v2/orders` - List orders
- `PATCH /api/v2/orders/:id/status` - Update order status

## Request Tracing

The V2 API supports the `X-Acme-Request-ID` header for distributed tracing.

## Configuration

Environment variables:
- `SERVER_PORT` - Server port (default: 8082)
- `DB_HOST` - Database host
- `DB_PORT` - Database port
- `DB_USER` - Database user
- `DB_PASSWORD` - Database password
- `DB_NAME` - Database name
- `REDIS_HOST` - Redis host
- `REDIS_PORT` - Redis port
- `PAYMENT_SERVICE_URL` - Payment service URL
- `ENABLE_LEGACY_PAYMENTS` - Use legacy payment client (default: true)

## Running

```bash
go run ./cmd/orders
```
