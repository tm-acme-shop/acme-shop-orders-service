module github.com/tm-acme-shop/acme-shop-orders-service

go 1.21

require (
	github.com/tm-acme-shop/acme-shop-shared-go v0.0.0
	github.com/gin-gonic/gin v1.9.1
	github.com/lib/pq v1.10.9
	github.com/prometheus/client_golang v1.17.0
	github.com/redis/go-redis/v9 v9.3.0
	github.com/segmentio/kafka-go v0.4.46
)

replace github.com/tm-acme-shop/acme-shop-shared-go => ../acme-shop-shared-go
