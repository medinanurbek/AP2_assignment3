package main

import (
	"database/sql"
	"log"
	"net"
	"os"

	"payment-service/internal/repository"
	grpctransport "payment-service/internal/transport/grpc"
	httptransport "payment-service/internal/transport/http"
	"payment-service/internal/usecase"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	paymentpb "github.com/medinanurbek/generated-repo/go/payment"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
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

	// REST HTTP setup
	httpHandler := httptransport.NewPaymentHandler(useCase)
	r := gin.Default()
	httptransport.RegisterRoutes(r, httpHandler)

	httpPort := os.Getenv("PORT")
	if httpPort == "" {
		httpPort = "8081"
	}

	// gRPC Setup
	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "50051"
	}

	listener, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("Failed to listen for gRPC: %v", err)
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(grpctransport.LoggingInterceptor),
	)
	grpcHandler := grpctransport.NewPaymentHandler(useCase)
	paymentpb.RegisterPaymentServiceServer(grpcServer, grpcHandler)
	reflection.Register(grpcServer)

	log.Printf("Starting Payment Service gRPC on port %s...", grpcPort)
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("Failed to start gRPC server: %v", err)
		}
	}()

	log.Printf("Starting Payment Service HTTP on port %s...", httpPort)
	if err := r.Run(":" + httpPort); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}
