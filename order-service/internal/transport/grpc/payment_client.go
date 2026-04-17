package grpc

import (
	"context"
	"log"

	"order-service/internal/domain"

	paymentpb "github.com/medinanurbek/generated-repo/go/payment"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// PaymentGRPCClient implements domain.PaymentGateway using gRPC.
type PaymentGRPCClient struct {
	client paymentpb.PaymentServiceClient
	conn   *grpc.ClientConn
}

// NewPaymentGRPCClient dials the payment gRPC server at addr and returns a
// PaymentGateway that can be injected into the order use-case.
func NewPaymentGRPCClient(addr string) (domain.PaymentGateway, func(), error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}

	log.Printf("[PaymentGRPCClient] connected to payment service at %s", addr)

	cleanup := func() {
		if err := conn.Close(); err != nil {
			log.Printf("[PaymentGRPCClient] error closing connection: %v", err)
		}
	}

	return &PaymentGRPCClient{
		client: paymentpb.NewPaymentServiceClient(conn),
		conn:   conn,
	}, cleanup, nil
}

// ProcessPayment calls the Payment service via gRPC.
func (c *PaymentGRPCClient) ProcessPayment(orderID string, amount int64) (*domain.PaymentInfo, error) {
	resp, err := c.client.ProcessPayment(context.Background(), &paymentpb.PaymentRequest{
		OrderId: orderID,
		Amount:  float64(amount),
	})
	if err != nil {
		return nil, err
	}

	return &domain.PaymentInfo{
		Status:        resp.GetStatus(),
		TransactionID: resp.GetTransactionId(),
	}, nil
}
