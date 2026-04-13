package main

import (
	"database/sql"
	"log"
	"os"

	"payment-service/internal/repository"
	httptransport "payment-service/internal/transport/http"
	"payment-service/internal/usecase"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {
	dsn := os.Getenv("PAYMENT_DB_DSN")
	if dsn == "" {
		dsn = "user=postgres password=1234 dbname=payment_db sslmode=disable"
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Printf("Failed to ping DB: %v. Continuing without DB for demonstration...", err)
	}

	repo := repository.NewPostgresPaymentRepository(db)
	useCase := usecase.NewPaymentUseCase(repo)
	handler := httptransport.NewPaymentHandler(useCase)

	r := gin.Default()
	httptransport.RegisterRoutes(r, handler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	log.Printf("Starting Payment Service on port %s...", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
