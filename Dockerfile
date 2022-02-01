FROM golang:1.19-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /orders-service ./cmd/orders

FROM alpine:3.17

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /orders-service .

EXPOSE 8082

CMD ["./orders-service"]
