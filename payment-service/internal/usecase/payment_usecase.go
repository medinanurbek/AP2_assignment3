package usecase

import (
	"errors"
	"payment-service/internal/domain"

	"github.com/google/uuid"
)

type PaymentUseCase interface {
	ProcessPayment(orderID string, amount int64) (*domain.Payment, error)
	GetPaymentStatus(orderID string) (*domain.Payment, error)
	GetAllPayments() ([]*domain.Payment, error)
}

type paymentUseCase struct {
	repo domain.PaymentRepository
}

func NewPaymentUseCase(repo domain.PaymentRepository) PaymentUseCase {
	return &paymentUseCase{repo: repo}
}

func (u *paymentUseCase) ProcessPayment(orderID string, amount int64) (*domain.Payment, error) {
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
		TransactionID: uuid.New().String(),
		Amount:        amount,
		Status:        status,
	}

	err := u.repo.Save(payment)
	if err != nil {
		return nil, err
	}

	return payment, nil
}

func (u *paymentUseCase) GetPaymentStatus(orderID string) (*domain.Payment, error) {
	return u.repo.GetByOrderID(orderID)
}

func (u *paymentUseCase) GetAllPayments() ([]*domain.Payment, error) {
	return u.repo.GetAll()
}
