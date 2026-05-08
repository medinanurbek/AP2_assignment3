FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /app/notification-service cmd/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/notification-service .
CMD ["./notification-service"]
