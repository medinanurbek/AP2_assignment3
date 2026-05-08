package repository

import (
	"database/sql"
	"payment-service/internal/domain"
)

type postgresPaymentRepository struct {
	db *sql.DB
}

func NewPostgresPaymentRepository(db *sql.DB) domain.PaymentRepository {
	return &postgresPaymentRepository{db: db}
}

func (r *postgresPaymentRepository) Save(payment *domain.Payment) error {
	query := `INSERT INTO payments (id, order_id, customer_email, transaction_id, amount, status)
			  VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.Exec(query, payment.ID, payment.OrderID, payment.CustomerEmail, payment.TransactionID, payment.Amount, payment.Status)
	return err
}

func (r *postgresPaymentRepository) GetByOrderID(orderID string) (*domain.Payment, error) {
	query := `SELECT id, order_id, customer_email, transaction_id, amount, status FROM payments WHERE order_id = $1`
	row := r.db.QueryRow(query, orderID)

	var p domain.Payment
	err := row.Scan(&p.ID, &p.OrderID, &p.CustomerEmail, &p.TransactionID, &p.Amount, &p.Status)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // or domain specific error
		}
		return nil, err
	}
	return &p, nil
}

func (r *postgresPaymentRepository) GetAll() ([]*domain.Payment, error) {
	query := `SELECT id, order_id, customer_email, transaction_id, amount, status FROM payments`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var payments []*domain.Payment
	for rows.Next() {
		var p domain.Payment
		if err := rows.Scan(&p.ID, &p.OrderID, &p.CustomerEmail, &p.TransactionID, &p.Amount, &p.Status); err != nil {
			return nil, err
		}
		payments = append(payments, &p)
	}
	return payments, nil
}

func (r *postgresPaymentRepository) ListByStatus(status string) ([]*domain.Payment, error) {
	query := `SELECT id, order_id, customer_email, transaction_id, amount, status FROM payments WHERE status = $1`
	
	rows, err := r.db.Query(query, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var payments []*domain.Payment
	for rows.Next() {
		var p domain.Payment
		if err := rows.Scan(&p.ID, &p.OrderID, &p.CustomerEmail, &p.TransactionID, &p.Amount, &p.Status); err != nil {
			return nil, err
		}
		payments = append(payments, &p)
	}
	return payments, nil
}

func (r *postgresPaymentRepository) GetByAmountRange(min, max int64) ([]*domain.Payment, error) {
	return r.FindByAmountRange(min, max)
}

func (r *postgresPaymentRepository) FindByAmountRange(min, max int64) ([]*domain.Payment, error) {
	// If max is 0, we consider it as unlimited.
	var query string
	var args []interface{}

	if max > 0 {
		query = `SELECT id, order_id, customer_email, transaction_id, amount, status FROM payments WHERE amount >= $1 AND amount <= $2`
		args = []interface{}{min, max}
	} else {
		query = `SELECT id, order_id, customer_email, transaction_id, amount, status FROM payments WHERE amount >= $1`
		args = []interface{}{min}
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var payments []*domain.Payment
	for rows.Next() {
		var p domain.Payment
		if err := rows.Scan(&p.ID, &p.OrderID, &p.CustomerEmail, &p.TransactionID, &p.Amount, &p.Status); err != nil {
			return nil, err
		}
		payments = append(payments, &p)
	}
	return payments, nil
}
