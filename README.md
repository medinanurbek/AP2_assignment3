# AP2 Assignment 1 - Clean Architecture based Microservices (Order & Payment)

**Name:** Medina Nurbek 
**Course:** Advanced Programming 2  

## Architecture Overview

The system is composed of two independent microservices built using Go and the Gin framework. Each microservice follows the principles of Clean Architecture to ensure separation of concerns, independency from frameworks, and testability.

### Bounded Contexts
1. **Order Service** (`order-service/`): Manages the ordering process, maintaining the lifecycle of an order from "Pending" to "Paid", "Failed", or "Cancelled".
2. **Payment Service** (`payment-service/`): Evaluates transactions and limits. It rejects amounts over 100,000 units independently without needing context regarding the broader e-commerce platform.

### Strict Microservices Rules Observed
- **No Shared Code:** Each service has its own copy of the `domain` models inside its module. Code is completely decoupled. No shared `/pkg` module exists.
- **Database per Service:** The `migrations/` folder in each service defines independent schemas, expecting separate databases per microservice architecture rules.
- **Dependency Inversion / Composition Root:** Services declare `Repository` and outgoing `Gateway` interfaces within their domain layer. Implementations (HTTP, Postgres) are passed down at the start wire up in `.cmd/*/main.go`.

### Failure Handling & Resilience
- **Network Resiliency:** Order Service performs a synchronous call to Payment Service using a custom HTTP client configured with a strict `2-second timeout`.
- **Fault Tolerance:** If Payment Service is down or times out, the `POST /orders` endpoint immediately aborts waiting, and falls back to a gracefully handled HTTP 503 (Service Unavailable) message. The `Order` is marked as `Failed` in the database, allowing for audit-tracing.

### Idempotency (Bonus)
The `Order Service` features a cross-goroutine caching mechanism that filters duplicate `POST /orders` requests based on an optional `Idempotency-Key` header. Sending duplicate requests with the same key returns the historically processed order transparently, without duplicating order database inserts or repeating upstream HTTP calls to `Payment Service`.

---

## Instructions

### Setup Database
1. Execute `init.sql` located inside `order-service/migrations/` on your Order Database instance.
2. Execute `init.sql` located inside `payment-service/migrations/` on your Payment Database instance.

### Build & Run
Run Payment Service:
```bash
cd payment-service
go run cmd/payment-service/main.go
# Starts on port 8081
```

Run Order Service:
```bash
cd order-service
go run cmd/order-service/main.go
# Starts on port 8080
```

---

## API Examples

### 1. Create an Order (Success Case)
```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: test-key-01" \
  -d '{"customer_id": "CUST123", "item_name": "Laptop", "amount": 15000}'
```

### 2. Create an Order with Duplicate Key (Idempotency Demo)
*Run the command above twice.* The second time, no database row is added, representing duplicate protection.

### 3. Create an Order (Declined Case - Amount > 100000)
```bash
curl -X POST http://localhost:8082/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id": "CUST123", "item_name": "Expensive Car", "amount": 250000}'
```

### 4. Retrieve an Order
```bash
# Substitute the returned database UUID
curl -X GET http://localhost:8082/orders/{order_id}
```

### 5. Simulate 2-second Timeout & Failure Cas
1. **Stop** the Payment Service terminal.
2. Post an order to Order Service.
3. Order Service waits 2 seconds max, then returns a **503 Service Unavailable** with the order status correctly marked as `Failed`.
