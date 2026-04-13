package main

import (
	"database/sql"
	"log"
	"os"

	"order-service/internal/repository"
	httptransport "order-service/internal/transport/http"
	"order-service/internal/usecase"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {
	dsn := os.Getenv("ORDER_DB_DSN")
	if dsn == "" {
		dsn = "user=postgres password=1234 dbname=order_db sslmode=disable"
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Printf("Failed to ping DB: %v. Continuing without DB for demonstration...", err)
	}
	paymentServiceURL := os.Getenv("PAYMENT_SERVICE_URL")
	if paymentServiceURL == "" {
		paymentServiceURL = "http://localhost:8081"
	}

	repo := repository.NewPostgresOrderRepository(db)
	paymentClient := httptransport.NewPaymentClient(paymentServiceURL)
	useCase := usecase.NewOrderUseCase(repo, paymentClient)
	handler := httptransport.NewOrderHandler(useCase)

	r := gin.Default()
	httptransport.RegisterRoutes(r, handler)

	// Start Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}
	log.Printf("Starting Order Service on port %s...", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
