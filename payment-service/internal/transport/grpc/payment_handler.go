package grpc

import (
	"context"

	"payment-service/internal/usecase"

	paymentpb "github.com/medinanurbek/generated-repo/go/payment"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	payment, err := h.useCase.ProcessPayment(req.GetOrderId(), int64(req.GetAmount()), req.GetCustomerEmail())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to process payment: %v", err)
	}

	return &paymentpb.PaymentResponse{
		Status:        payment.Status,
		TransactionId: payment.TransactionID,
		Amount:        float64(payment.Amount),
	}, nil
}

func (h *PaymentHandler) ListPayments(ctx context.Context, req *paymentpb.ListPaymentsRequest) (*paymentpb.ListPaymentsResponse, error) {
	payments, err := h.useCase.ListPayments(req.GetMinAmount(), req.GetMaxAmount(), req.GetStatus())
	if err != nil {
		// If the error is from our validation, use InvalidArgument
		errMsg := err.Error()
		if errMsg == "invalid status: must be 'Authorized' or 'Declined'" || errMsg == "min_amount cannot be greater than max_amount" {
			return nil, status.Errorf(codes.InvalidArgument, "validation error: %v", err)
		}
		return nil, status.Errorf(codes.Internal, "failed to list payments: %v", err)
	}

	var pbPayments []*paymentpb.PaymentResponse
	for _, p := range payments {
		pbPayments = append(pbPayments, &paymentpb.PaymentResponse{
			Status:        p.Status,
			TransactionId: p.TransactionID,
			Amount:        float64(p.Amount),
		})
	}

	return &paymentpb.ListPaymentsResponse{
		Payments: pbPayments,
	}, nil
}
