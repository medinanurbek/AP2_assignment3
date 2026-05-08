package domain

type Payment struct {
	ID            string
	OrderID       string
	CustomerEmail string
	TransactionID string
	Amount        int64
	Status        string // "Authorized", "Declined"
}

type PaymentRepository interface {
	Save(payment *Payment) error
	GetByOrderID(orderID string) (*Payment, error)
	GetAll() ([]*Payment, error)
	ListByStatus(status string) ([]*Payment, error)
	GetByAmountRange(min, max int64) ([]*Payment, error)
	FindByAmountRange(min, max int64) ([]*Payment, error)
}
