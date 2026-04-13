package repository

import (
	"database/sql"
	"order-service/internal/domain"
)

type postgresOrderRepository struct {
	db *sql.DB
}

func NewPostgresOrderRepository(db *sql.DB) domain.OrderRepository {
	return &postgresOrderRepository{db: db}
}

func (r *postgresOrderRepository) Save(order *domain.Order) error {
	query := `INSERT INTO orders (id, customer_id, item_name, amount, status, created_at)
			  VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.Exec(query, order.ID, order.CustomerID, order.ItemName, order.Amount, order.Status, order.CreatedAt)
	return err
}

func (r *postgresOrderRepository) GetByID(id string) (*domain.Order, error) {
	query := `SELECT id, customer_id, item_name, amount, status, created_at FROM orders WHERE id = $1`
	row := r.db.QueryRow(query, id)

	var o domain.Order
	err := row.Scan(&o.ID, &o.CustomerID, &o.ItemName, &o.Amount, &o.Status, &o.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &o, nil
}

func (r *postgresOrderRepository) GetAll() ([]*domain.Order, error) {
	query := `SELECT id, customer_id, item_name, amount, status, created_at FROM orders`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []*domain.Order
	for rows.Next() {
		var o domain.Order
		if err := rows.Scan(&o.ID, &o.CustomerID, &o.ItemName, &o.Amount, &o.Status, &o.CreatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, &o)
	}
	return orders, nil
}

func (r *postgresOrderRepository) UpdateStatus(id string, status string) error {
	query := `UPDATE orders SET status = $1 WHERE id = $2`
	_, err := r.db.Exec(query, status, id)
	return err
}

func (r *postgresOrderRepository) GetByAmountRange(min, max int64) ([]*domain.Order, error) {
	query := `SELECT id, customer_id, item_name, amount, status, created_at FROM orders WHERE amount >= $1 AND amount <= $2`

	rows, err := r.db.Query(query, min, max)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []*domain.Order
	for rows.Next() {
		var o domain.Order
		if err := rows.Scan(&o.ID, &o.CustomerID, &o.ItemName, &o.Amount, &o.Status, &o.CreatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, &o)
	}
	return orders, nil
}

func (r *postgresOrderRepository) GetCustomerRevenue(customerID string) (*domain.CustomerRevenue, error) {
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM orders WHERE customer_id = $1)`
	err := r.db.QueryRow(checkQuery, customerID).Scan(&exists)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, sql.ErrNoRows
	}

	query := `SELECT COALESCE(SUM(amount), 0), COUNT(id) FROM orders WHERE customer_id = $1 AND status = 'Paid'`

	var totalAmount, ordersCount int64
	err = r.db.QueryRow(query, customerID).Scan(&totalAmount, &ordersCount)
	if err != nil {
		return nil, err
	}

	return &domain.CustomerRevenue{
		CustomerID:  customerID,
		TotalAmount: totalAmount,
		OrdersCount: ordersCount,
	}, nil
}
