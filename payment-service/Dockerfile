FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY pkg/ ./pkg/
COPY payment-service/go.mod payment-service/go.sum ./payment-service/
WORKDIR /app/payment-service
RUN go mod download
WORKDIR /app
COPY payment-service/ ./payment-service/
WORKDIR /app/payment-service
RUN go build -o /app/payment-service-bin cmd/payment-service/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/payment-service-bin ./payment-service
COPY --from=builder /app/payment-service/migrations ./migrations
CMD ["./payment-service"]
