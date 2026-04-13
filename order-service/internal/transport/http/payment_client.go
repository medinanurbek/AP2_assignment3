package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"order-service/internal/domain"
)

type PaymentClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewPaymentClient(baseURL string) domain.PaymentGateway {
	return &PaymentClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 2 * time.Second,
		},
	}
}

func (c *PaymentClient) ProcessPayment(orderID string, amount int64) (*domain.PaymentInfo, error) {
	url := fmt.Sprintf("%s/payments", c.baseURL)

	payload := map[string]interface{}{
		"order_id": orderID,
		"amount":   amount,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("payment service unavailable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("payment service returned non-200 status: %d", resp.StatusCode)
	}

	var res struct {
		Status        string `json:"status"`
		TransactionID string `json:"transaction_id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	return &domain.PaymentInfo{
		Status:        res.Status,
		TransactionID: res.TransactionID,
	}, nil
}
