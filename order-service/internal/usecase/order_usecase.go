package usecase

import (
	"errors"
	"time"

	"order-service/internal/domain"

	"github.com/google/uuid"
)

var (
	ErrInvalidAmount = errors.New("amount must be > 0")
	ErrOrderNotFound = errors.New("order not found")
	ErrCannotCancel  = errors.New("cannot cancel an order that is not Pending")
)

type OrderUseCase interface {
	CreateOrder(customerID, customerEmail, itemName string, amount int64) (*domain.Order, error)
	GetOrder(id string) (*domain.Order, error)
	GetAllOrders() ([]*domain.Order, error)
	CancelOrder(id string) error
	GetOrdersByAmount(minAmount, maxAmount int64) ([]*domain.Order, error)
	GetCustomerRevenue(customerID string) (*domain.CustomerRevenue, error)
}

type orderUseCase struct {
	repo    domain.OrderRepository
	payment domain.PaymentGateway
	cache   domain.OrderCache
}

func NewOrderUseCase(repo domain.OrderRepository, payment domain.PaymentGateway, cache domain.OrderCache) OrderUseCase {
	return &orderUseCase{repo: repo, payment: payment, cache: cache}
}

func (u *orderUseCase) CreateOrder(customerID, customerEmail, itemName string, amount int64) (*domain.Order, error) {
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}

	order := &domain.Order{
		ID:            uuid.New().String(),
		CustomerID:    customerID,
		CustomerEmail: customerEmail,
		ItemName:      itemName,
		Amount:        amount,
		Status:        "Pending",
		CreatedAt:     time.Now(),
	}

	err := u.repo.Save(order)
	if err != nil {
		return nil, err
	}

	payInfo, err := u.payment.ProcessPayment(order.ID, order.Amount, order.CustomerEmail)
	if err != nil {
		u.repo.UpdateStatus(order.ID, "Failed")
		order.Status = "Failed"
		return order, err
	}

	status := "Failed"
	if payInfo.Status == "Authorized" {
		status = "Paid"
	}

	u.repo.UpdateStatus(order.ID, status)
	order.Status = status

	// Invalidate cache after status change
	_ = u.cache.Delete(order.ID)

	return order, nil
}

func (u *orderUseCase) GetOrder(id string) (*domain.Order, error) {
	// Try cache first
	order, err := u.cache.Get(id)
	if err == nil && order != nil {
		return order, nil
	}

	// Cache miss, get from DB
	order, err = u.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if order != nil {
		// Store in cache for 5 minutes
		_ = u.cache.Set(order, 5*time.Minute)
	}

	return order, nil
}

func (u *orderUseCase) GetAllOrders() ([]*domain.Order, error) {
	return u.repo.GetAll()
}

func (u *orderUseCase) CancelOrder(id string) error {
	order, err := u.repo.GetByID(id)
	if err != nil {
		return err
	}
	if order == nil {
		return ErrOrderNotFound
	}
	if order.Status != "Pending" {
		return ErrCannotCancel
	}

	err = u.repo.UpdateStatus(id, "Cancelled")
	if err == nil {
		_ = u.cache.Delete(id)
	}
	return err
}

func (u *orderUseCase) GetOrdersByAmount(minAmount, maxAmount int64) ([]*domain.Order, error) {
	if minAmount < 0 || maxAmount < 0 {
		return nil, errors.New("negative amounts are not allowed")
	}

	orders, err := u.repo.GetByAmountRange(minAmount, maxAmount)
	if err != nil {
		return nil, err
	}

	if len(orders) == 0 {
		return nil, errors.New("no orders found in this range")
	}
	return orders, nil
}

func (u *orderUseCase) GetCustomerRevenue(customerID string) (*domain.CustomerRevenue, error) {
	if customerID == "" {
		return nil, errors.New("customerID is empty")
	}
	return u.repo.GetCustomerRevenue(customerID)
}
