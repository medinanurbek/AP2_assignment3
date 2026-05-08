package usecase

import (
	"context"
	"errors"
	"payment-service/internal/domain"

	"github.com/google/uuid"
)

type PaymentUseCase interface {
	ProcessPayment(orderID string, amount int64, customerEmail string) (*domain.Payment, error)
	GetPaymentStatus(orderID string) (*domain.Payment, error)
	GetAllPayments() ([]*domain.Payment, error)
	ListPayments(min, max int64, status string) ([]*domain.Payment, error)
}

type paymentUseCase struct {
	repo      domain.PaymentRepository
	publisher interface {
		PublishPaymentCompleted(ctx context.Context, payment *domain.Payment) error
	}
}

func NewPaymentUseCase(repo domain.PaymentRepository, publisher interface {
	PublishPaymentCompleted(ctx context.Context, payment *domain.Payment) error
}) PaymentUseCase {
	return &paymentUseCase{repo: repo, publisher: publisher}
}

func (u *paymentUseCase) ProcessPayment(orderID string, amount int64, customerEmail string) (*domain.Payment, error) {
	if amount <= 0 {
		return nil, errors.New("invalid amount")
	}

	status := "Authorized"
	if amount > 100000 {
		status = "Declined"
	}

	payment := &domain.Payment{
		ID:            uuid.New().String(),
		OrderID:       orderID,
		CustomerEmail: customerEmail,
		TransactionID: uuid.New().String(),
		Amount:        amount,
		Status:        status,
	}

	err := u.repo.Save(payment)
	if err != nil {
		return nil, err
	}

	// Publish event to message broker
	if u.publisher != nil {
		err = u.publisher.PublishPaymentCompleted(context.Background(), payment)
		if err != nil {
			// Just log the error, don't fail the payment since it's already in the DB.
			// In a more robust system, we would use the Outbox pattern.
			// log.Printf("Failed to publish payment completed event: %v", err)
		}
	}

	return payment, nil
}

func (u *paymentUseCase) GetPaymentStatus(orderID string) (*domain.Payment, error) {
	return u.repo.GetByOrderID(orderID)
}

func (u *paymentUseCase) GetAllPayments() ([]*domain.Payment, error) {
	return u.repo.GetAll()
}

func (u *paymentUseCase) ListPayments(min, max int64, status string) ([]*domain.Payment, error) {
	// If status is provided, it takes priority (based on assignment requirement)
	if status != "" {
		if status != "Authorized" && status != "Declined" {
			return nil, errors.New("invalid status: must be 'Authorized' or 'Declined'")
		}
		return u.repo.ListByStatus(status)
	}

	// Otherwise, fallback to amount range filtering
	if max > 0 && min > max {
		return nil, errors.New("min_amount cannot be greater than max_amount")
	}
	return u.repo.FindByAmountRange(min, max)
}
