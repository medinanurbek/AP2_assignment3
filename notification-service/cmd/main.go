package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Payment struct matches the payload from payment-service
type Payment struct {
	ID            string `json:"ID"`
	OrderID       string `json:"OrderID" json:"order_id"`
	CustomerEmail string `json:"CustomerEmail" json:"customer_email"`
	TransactionID string `json:"TransactionID" json:"transaction_id"`
	Amount        int64  `json:"Amount" json:"amount"`
	Status        string `json:"Status" json:"status"`
}

var (
	// Idempotency cache (in-memory for assignment purposes)
	processedMessages = make(map[string]bool) // кэш в памяти
	cacheMutex        sync.Mutex

	// Retry tracker to implement DLQ logic (Bonus)
	retryTracker = make(map[string]int)
	retryMutex   sync.Mutex
)

func main() {
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

	// --- Bonus: Setup Dead Letter Queue (DLQ) ---
	err = ch.ExchangeDeclare(
		"dlx",    // name
		"direct", // type
		true,     // durable
		false,    // auto-deleted
		false,    // internal
		false,    // no-wait
		nil,      // arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare DLX: %v", err)
	}

	dlq, err := ch.QueueDeclare(
		"payment.dlq", // name
		true,          // durable
		false,         // delete when unused
		false,         // exclusive
		false,         // no-wait
		nil,           // arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare DLQ: %v", err)
	}

	err = ch.QueueBind(
		dlq.Name,      // queue name
		"payment.dlq", // routing key
		"dlx",         // exchange
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to bind DLQ: %v", err)
	}

	// --- Setup Main Queue ---
	args := amqp.Table{
		"x-dead-letter-exchange":    "dlx",
		"x-dead-letter-routing-key": "payment.dlq",
	}
	q, err := ch.QueueDeclare(
		"payment.completed", // name
		true,                // durable (Persistence)
		false,               // delete when unused
		false,               // exclusive
		false,               // no-wait
		args,                // arguments including DLX configuration
	)
	if err != nil {
		log.Fatalf("Failed to declare a queue: %v", err)
	}

	// Consume messages with auto-ack = false (Manual ACKs)
	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto-ack (FALSE AS REQUIRED)
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		log.Fatalf("Failed to register a consumer: %v", err)
	}

	_, cancel := context.WithCancel(context.Background())

	go func() {
		for d := range msgs {
			var payment Payment
			if err := json.Unmarshal(d.Body, &payment); err != nil {
				log.Printf("[Error] Invalid JSON payload. Rejecting message to DLQ.")
				d.Nack(false, false) // Reject and don't requeue -> goes to DLX
				continue
			}

			// --- Idempotency Check ---
			cacheMutex.Lock()
			if processedMessages[payment.ID] {
				log.Printf("[Notification] Ignoring duplicate message (Idempotency), Order #%s", payment.OrderID)
				cacheMutex.Unlock()
				d.Ack(false) // Acknowledge as we already processed it
				continue
			}
			cacheMutex.Unlock()

			// --- Process Logic ---
			// Simulate sending email
			log.Printf("[Notification] Sent email to %s for Order #%s. Amount: $%v",
				payment.CustomerEmail, payment.OrderID, payment.Amount)

			// --- Store in Idempotency cache ---
			cacheMutex.Lock()
			processedMessages[payment.ID] = true
			cacheMutex.Unlock()

			// --- Manual ACK ---
			d.Ack(false) // Acknowledge successful processing
		}
	}()

	log.Printf(" [*] Notification Service is waiting for messages. To exit press CTRL+C")

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down Notification Service...")
	cancel()
}
