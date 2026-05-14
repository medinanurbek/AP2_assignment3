package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"notification-service/internal/provider"
	"notification-service/internal/worker"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

func main() {
	// 1. Initialize RabbitMQ
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://user:password@localhost:5672/"
	}

	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer ch.Close()

	// 2. Setup Queues (Main and DLQ)
	setupQueues(ch)

	// 3. Initialize Redis
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6379"
	}
	rdb := redis.NewClient(&redis.Options{
		Addr: redisURL,
	})

	// 4. Initialize Provider based on Environment
	var emailProvider provider.EmailSender
	mode := os.Getenv("PROVIDER_MODE")
	if mode == "REAL" {
		log.Println("[Init] Using REAL Email Provider")
		emailProvider = provider.NewRealEmailSender()
	} else {
		log.Println("[Init] Using SIMULATED (Mock) Email Provider")
		emailProvider = provider.NewMockEmailSender()
	}

	// 5. Start Background Worker
	qName := "payment.completed"
	msgs, err := ch.Consume(
		qName,
		"",
		false, // auto-ack = false
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to register a consumer: %v", err)
	}

	notificationWorker := worker.NewNotificationWorker(rdb, emailProvider, ch)
	go notificationWorker.Start(msgs)

	log.Printf(" [*] Notification Service (Worker) is running. Mode: %s", mode)

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down Notification Service...")
}

func setupQueues(ch *amqp.Channel) {
	// Setup DLX
	_ = ch.ExchangeDeclare("dlx", "direct", true, false, false, false, nil)
	
	// Setup DLQ
	_, _ = ch.QueueDeclare("payment.dlq", true, false, false, false, nil)
	_ = ch.QueueBind("payment.dlq", "payment.dlq", "dlx", false, nil)

	// Setup Main Queue with DLX
	args := amqp.Table{
		"x-dead-letter-exchange":    "dlx",
		"x-dead-letter-routing-key": "payment.dlq",
	}
	_, _ = ch.QueueDeclare("payment.completed", true, false, false, false, args)
}
