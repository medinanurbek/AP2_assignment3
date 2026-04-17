package main

import (
	"database/sql"
	"log"
	"net"
	"os"

	"order-service/internal/repository"
	grpctransport "order-service/internal/transport/grpc"
	httptransport "order-service/internal/transport/http"
	"order-service/internal/usecase"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	orderpb "github.com/medinanurbek/generated-repo/go/order"
	"google.golang.org/grpc"
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

	// 1. Initialize PubSub broker for internal event distribution
	ps := repository.NewPubSub()

	// 2. Start PostgreSQL Listener for real-time DB changes (NOTIFY)
	go repository.StartListener(dsn, ps)

	// 3. Setup Payment Service gRPC Client
	paymentGRPCAddr := os.Getenv("PAYMENT_GRPC_URL")
	if paymentGRPCAddr == "" {
		paymentGRPCAddr = "localhost:50051"
	}
	paymentClient, cleanup, err := grpctransport.NewPaymentGRPCClient(paymentGRPCAddr)
	if err != nil {
		log.Fatalf("Failed to initialize Payment gRPC Client: %v", err)
	}
	defer cleanup()

	// 4. Initialize Core Logic
	repo := repository.NewPostgresOrderRepository(db)
	useCase := usecase.NewOrderUseCase(repo, paymentClient)

	// 5. Setup gRPC Server (for Streaming)
	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "50052"
	}
	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("Failed to listen for gRPC: %v", err)
	}

	grpcServer := grpc.NewServer()
	orderHandler := grpctransport.NewOrderHandler(ps)
	orderpb.RegisterOrderServiceServer(grpcServer, orderHandler)

	log.Printf("Starting Order gRPC Server on port %s (Streaming)...", grpcPort)
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to start gRPC server: %v", err)
		}
	}()

	// 6. Setup REST HTTP Server (Existing)
	httpHandler := httptransport.NewOrderHandler(useCase)
	r := gin.Default()
	httptransport.RegisterRoutes(r, httpHandler)

	httpPort := os.Getenv("PORT")
	if httpPort == "" {
		httpPort = "8082"
	}
	log.Printf("Starting Order HTTP Service on port %s...", httpPort)
	if err := r.Run(":" + httpPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

