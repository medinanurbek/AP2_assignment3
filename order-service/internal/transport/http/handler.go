package http

import (
	"net/http"
	"strconv"
	"sync"

	"order-service/internal/domain"
	"order-service/internal/usecase"

	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	useCase usecase.OrderUseCase
}

func NewOrderHandler(uc usecase.OrderUseCase) *OrderHandler {
	return &OrderHandler{useCase: uc}
}

var (
	idempotencyCache = make(map[string]*domain.Order)
	cacheMutex       sync.Mutex
)

func (h *OrderHandler) CreateOrder(c *gin.Context) {
	idemKey := c.GetHeader("Idempotency-Key")
	if idemKey != "" {
		cacheMutex.Lock()
		if order, exists := idempotencyCache[idemKey]; exists {
			cacheMutex.Unlock()
			c.JSON(http.StatusOK, gin.H{"message": "returned from cache", "order": order})
			return
		}
		cacheMutex.Unlock()
	}

	var req struct {
		CustomerID    string `json:"customer_id"`
		CustomerEmail string `json:"customer_email"`
		ItemName      string `json:"item_name"`
		Amount        int64  `json:"amount"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order, err := h.useCase.CreateOrder(req.CustomerID, req.CustomerEmail, req.ItemName, req.Amount)

	if err != nil && order != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Payment service unavailable. Order failed.",
			"order": order,
		})
		return
	} else if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if idemKey != "" {
		cacheMutex.Lock()
		idempotencyCache[idemKey] = order
		cacheMutex.Unlock()
	}

	c.JSON(http.StatusCreated, order)
}

func (h *OrderHandler) GetOrder(c *gin.Context) {
	id := c.Param("id")

	order, err := h.useCase.GetOrder(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if order == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}

	c.JSON(http.StatusOK, order)
}

func (h *OrderHandler) GetAllOrders(c *gin.Context) {
	minStr := c.Query("min_amount")
	maxStr := c.Query("max_amount")

	if minStr != "" && maxStr != "" {
		min, err1 := strconv.ParseInt(minStr, 10, 64)
		max, err2 := strconv.ParseInt(maxStr, 10, 64)

		if err1 != nil || err2 != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "min_amount and max_amount must be numbers"})
			return
		}

		orders, err := h.useCase.GetOrdersByAmount(min, max)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, orders)
		return
	}

	orders, err := h.useCase.GetAllOrders()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, orders)
}

func (h *OrderHandler) CancelOrder(c *gin.Context) {
	id := c.Param("id")

	err := h.useCase.CancelOrder(id)
	if err != nil {
		if err == usecase.ErrOrderNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err == usecase.ErrCannotCancel {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "order cancelled successfully"})
}

func (h *OrderHandler) GetCustomerRevenue(c *gin.Context) {
	customerID := c.Query("customer_id")
	if customerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "customer_id is required"})
		return
	}

	revenue, err := h.useCase.GetCustomerRevenue(customerID)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "customer not found in database"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, revenue)
}

func RegisterRoutes(r *gin.Engine, handler *OrderHandler) {
	r.POST("/orders", handler.CreateOrder)
	r.GET("/orders", handler.GetAllOrders)
	r.GET("/orders/revenue", handler.GetCustomerRevenue)
	r.GET("/orders/:id", handler.GetOrder)
	r.PATCH("/orders/:id/cancel", handler.CancelOrder)
}
