package main

import (
	"context"
	"database/sql"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"order-service/internal/repository"
	grpctransport "order-service/internal/transport/grpc"
	httptransport "order-service/internal/transport/http"
	"order-service/internal/transport/http/middleware"
	"order-service/internal/usecase"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	orderpb "github.com/medinanurbek/generated-repo/go/order"
	"github.com/redis/go-redis/v9"
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

	// 3.5 Initialize Redis for Caching and Rate Limiting
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6379"
	}
	rdb := redis.NewClient(&redis.Options{
		Addr: redisURL,
	})

	// 4. Initialize Core Logic
	repo := repository.NewPostgresOrderRepository(db)
	cache := repository.NewRedisOrderCache(rdb)
	useCase := usecase.NewOrderUseCase(repo, paymentClient, cache)

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

	// 6. Setup REST HTTP Server
	httpHandler := httptransport.NewOrderHandler(useCase)
	r := gin.Default()
	// Apply rate limit: 10 requests per minute
	r.Use(middleware.RateLimiter(rdb, 10, time.Minute))
	httptransport.RegisterRoutes(r, httpHandler)

	httpPort := os.Getenv("PORT")
	if httpPort == "" {
		httpPort = "8082"
	}

	srv := &http.Server{
		Addr:    ":" + httpPort,
		Handler: r,
	}

	log.Printf("Starting Order HTTP Service on port %s...", httpPort)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down Order Service servers...")

	grpcServer.GracefulStop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Order HTTP Server Shutdown Error:", err)
	}

	log.Println("Order Service exiting")
}


