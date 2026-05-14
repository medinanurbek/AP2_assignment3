package domain

import "time"

type Order struct {
	ID         string
	CustomerID    string
	CustomerEmail string
	ItemName      string
	Amount        int64
	Status        string // "Pending", "Paid", "Failed", "Cancelled"
	CreatedAt     time.Time
}

type OrderRepository interface {
	Save(order *Order) error
	GetByID(id string) (*Order, error)
	GetAll() ([]*Order, error)
	UpdateStatus(id string, status string) error
	GetByAmountRange(minAmount, maxAmount int64) ([]*Order, error)
	GetCustomerRevenue(customerID string) (*CustomerRevenue, error)
}

type OrderCache interface {
	Get(id string) (*Order, error)
	Set(order *Order, ttl time.Duration) error
	Delete(id string) error
}

type CustomerRevenue struct {
	CustomerID  string `json:"customer_id"`
	TotalAmount int64  `json:"total_amount"`
	OrdersCount int64  `json:"orders_count"`
}
type PaymentInfo struct {
	Status        string
	TransactionID string
}

type PaymentGateway interface {
	ProcessPayment(orderID string, amount int64, customerEmail string) (*PaymentInfo, error)
}
