package grpc

import (
	"context"

	paymentpb "github.com/medinanurbek/generated-repo/go/payment"
	"payment-service/internal/usecase"
)

type PaymentHandler struct {
	paymentpb.UnimplementedPaymentServiceServer
	useCase usecase.PaymentUseCase
}

func NewPaymentHandler(uc usecase.PaymentUseCase) *PaymentHandler {
	return &PaymentHandler{useCase: uc}
}

func (h *PaymentHandler) ProcessPayment(ctx context.Context, req *paymentpb.PaymentRequest) (*paymentpb.PaymentResponse, error) {
	// Cast float64 to int64 because the proto amount is double while the logic expects int64
	payment, err := h.useCase.ProcessPayment(req.GetOrderId(), int64(req.GetAmount()))
	if err != nil {
		return nil, err
	}

	return &paymentpb.PaymentResponse{
		Status:        payment.Status,
		TransactionId: payment.TransactionID,
	}, nil
}
