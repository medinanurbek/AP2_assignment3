package main

import (
	"database/sql"
	"log"
	"net"
	"net/http"
	"os"

	"payment-service/internal/repository"
	grpctransport "payment-service/internal/transport/grpc"
	httptransport "payment-service/internal/transport/http"
	"payment-service/internal/usecase"
	"payment-service/internal/rabbitmq"
	"os/signal"
	"syscall"
	"context"
	"time"

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

	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://user:password@localhost:5672/"
	}

	publisher, err := rabbitmq.NewRabbitMQPublisher(rabbitURL)
	if err != nil {
		log.Printf("Failed to connect to RabbitMQ, running without publisher: %v", err)
		publisher = nil // will be handled in useCase
	} else {
		defer publisher.Close()
	}

	repo := repository.NewPostgresPaymentRepository(db)
	useCase := usecase.NewPaymentUseCase(repo, publisher)

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

	srv := &http.Server{
		Addr:    ":" + httpPort,
		Handler: r,
	}

	log.Printf("Starting Payment Service HTTP on port %s...", httpPort)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down servers...")

	grpcServer.GracefulStop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("HTTP Server Shutdown Error:", err)
	}

	log.Println("Server exiting")
}
