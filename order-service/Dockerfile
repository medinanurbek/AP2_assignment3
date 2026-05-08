FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY pkg/ ./pkg/
COPY order-service/go.mod order-service/go.sum ./order-service/
WORKDIR /app/order-service
RUN go mod download
WORKDIR /app
COPY order-service/ ./order-service/
WORKDIR /app/order-service
RUN go build -o /app/order-service-bin cmd/order-service/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/order-service-bin ./order-service
COPY --from=builder /app/order-service/migrations ./migrations
CMD ["./order-service"]
