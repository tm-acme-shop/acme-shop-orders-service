# ACME Shop Orders Service

Orders management service for ACME Shop platform.

## Overview

This service handles order creation, retrieval, and management for the ACME Shop e-commerce platform.

## API Endpoints

### V1 API

- `POST /api/v1/orders` - Create a new order
- `GET /api/v1/orders/:id` - Get order by ID
- `GET /api/v1/orders` - List orders for a user
- `POST /api/v1/orders/:id/status` - Update order status

## Configuration

Environment variables:
- `SERVER_PORT` - Server port (default: 8082)
- `DB_HOST` - Database host
- `DB_PORT` - Database port
- `DB_USER` - Database user
- `DB_PASSWORD` - Database password
- `DB_NAME` - Database name
- `PAYMENT_SERVICE_URL` - Payment service URL

## Running

```bash
go run ./cmd/orders
```
