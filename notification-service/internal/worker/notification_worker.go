package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"notification-service/internal/provider"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

type Payment struct {
	ID            string `json:"ID"`
	OrderID       string `json:"OrderID" json:"order_id"`
	CustomerEmail string `json:"CustomerEmail" json:"customer_email"`
	TransactionID string `json:"TransactionID" json:"transaction_id"`
	Amount        int64  `json:"Amount" json:"amount"`
	Status        string `json:"Status" json:"status"`
}

type NotificationWorker struct {
	rdb      *redis.Client
	provider provider.EmailSender
	ch       *amqp.Channel
}

func NewNotificationWorker(rdb *redis.Client, provider provider.EmailSender, ch *amqp.Channel) *NotificationWorker {
	return &NotificationWorker{
		rdb:      rdb,
		provider: provider,
		ch:       ch,
	}
}

func (w *NotificationWorker) Start(msgs <-chan amqp.Delivery) {
	for d := range msgs {
		w.processMessage(d)
	}
}

func (w *NotificationWorker) processMessage(d amqp.Delivery) {
	var payment Payment
	if err := json.Unmarshal(d.Body, &payment); err != nil {
		log.Printf("[Error] Invalid JSON payload: %v", err)
		d.Nack(false, false)
		return
	}

	// 1. Idempotency Check using Redis
	ctx := context.Background()
	idempotencyKey := fmt.Sprintf("notification:processed:%s", payment.ID)
	
	// SetNX (Set if Not Exists) to ensure we only process once
	// We use a long TTL (e.g., 24h) for idempotency records
	success, err := w.rdb.SetNX(ctx, idempotencyKey, "processed", 24*time.Hour).Result()
	if err != nil {
		log.Printf("[Error] Redis error during idempotency check: %v", err)
		d.Nack(false, true) // Requeue to try again later
		return
	}

	if !success {
		log.Printf("[Notification] Ignoring duplicate message for Payment #%s", payment.ID)
		d.Ack(false)
		return
	}

	// 2. Process with Retries and Exponential Backoff
	backoff := []time.Duration{2 * time.Second, 4 * time.Second, 8 * time.Second}
	maxRetries := len(backoff)
	
	var sendErr error
	for i := 0; i <= maxRetries; i++ {
		sendErr = w.provider.SendEmail(
			payment.CustomerEmail,
			fmt.Sprintf("Order #%s Confirmation", payment.OrderID),
			fmt.Sprintf("Thank you for your purchase of $%v!", payment.Amount),
		)

		if sendErr == nil {
			break
		}

		if i < maxRetries {
			log.Printf("[Retry] Attempt %d failed for Payment #%s. Retrying in %v...", i+1, payment.ID, backoff[i])
			time.Sleep(backoff[i])
		}
	}

	if sendErr != nil {
		log.Printf("[Error] Failed to send notification after %d retries: %v", maxRetries, sendErr)
		// If it still fails, we could either DLQ it or delete the idempotency key to allow manual retry later
		// For now, we'll let it go to DLQ via RabbitMQ (if configured) or just fail.
		// Let's delete the key so it can be retried if the message is requeued.
		w.rdb.Del(ctx, idempotencyKey)
		d.Nack(false, false) // Send to DLQ
		return
	}

	log.Printf("[Success] Notification processed for Order #%s", payment.OrderID)
	d.Ack(false)
}
