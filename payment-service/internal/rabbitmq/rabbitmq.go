package rabbitmq

import (
	"context"
	"encoding/json"
	"log"
	"payment-service/internal/domain"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Publisher interface {
	PublishPaymentCompleted(ctx context.Context, payment *domain.Payment) error
	Close() error
}

type rabbitMQPublisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewRabbitMQPublisher(url string) (Publisher, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	// Declare DLX exchange first
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
		ch.Close()
		conn.Close()
		return nil, err
	}

	// Declare the queue as durable
	// Declare queue with same arguments as consumer to avoid PRECONDITION_FAILED
	_, err = ch.QueueDeclare(
		"payment.completed", // name
		true,                // durable
		false,               // delete when unused
		false,               // exclusive
		false,               // no-wait
		amqp.Table{
			"x-dead-letter-exchange":    "dlx",
			"x-dead-letter-routing-key": "payment.dlq",
		}, // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}

	return &rabbitMQPublisher{
		conn:    conn,
		channel: ch,
	}, nil
}

func (p *rabbitMQPublisher) PublishPaymentCompleted(ctx context.Context, payment *domain.Payment) error {
	body, err := json.Marshal(payment)
	if err != nil {
		return err
	}

	// Create a context with timeout for publishing
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = p.channel.PublishWithContext(ctx,
		"",                  // exchange
		"payment.completed", // routing key
		false,               // mandatory
		false,               // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent, // message survives broker restart
			MessageId:    payment.ID,      // using payment ID for idempotency check later
		})
	
	if err != nil {
		log.Printf("[RabbitMQ] Failed to publish message: %v", err)
		return err
	}

	log.Printf("[RabbitMQ] Published payment.completed for OrderID: %s", payment.OrderID)
	return nil
}

func (p *rabbitMQPublisher) Close() error {
	if err := p.channel.Close(); err != nil {
		return err
	}
	return p.conn.Close()
}
