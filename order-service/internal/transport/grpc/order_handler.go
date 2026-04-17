package grpc

import (
	"log"
	"time"

	orderpb "github.com/medinanurbek/generated-repo/go/order"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"order-service/internal/repository"
)

// OrderHandler implements the OrderServiceServer interface.
type OrderHandler struct {
	orderpb.UnimplementedOrderServiceServer
	pubSub *repository.PubSub
}

// NewOrderHandler creates a new streaming gRPC handler backed by the pubSub broker.
func NewOrderHandler(ps *repository.PubSub) *OrderHandler {
	return &OrderHandler{pubSub: ps}
}

// SubscribeToOrderUpdates is a server-side streaming RPC.
// It subscribes to Postgres LISTEN/NOTIFY events for the given order_id
// and streams each status change to the client until the connection is closed.
func (h *OrderHandler) SubscribeToOrderUpdates(
	req *orderpb.OrderRequest,
	stream grpc.ServerStreamingServer[orderpb.OrderStatusUpdate],
) error {
	orderID := req.GetOrderId()
	log.Printf("[OrderHandler] client subscribed to updates for order %s", orderID)

	// Subscribe to in-memory pub/sub keyed by orderID.
	// Use "" to receive all order updates regardless of ID.
	ch := h.pubSub.Subscribe(orderID)
	defer h.pubSub.Unsubscribe(orderID, ch)

	ctx := stream.Context()

	for {
		select {
		case <-ctx.Done():
			// Client disconnected or cancelled.
			log.Printf("[OrderHandler] client disconnected from order %s stream", orderID)
			return nil

		case event, ok := <-ch:
			if !ok {
				// Channel was closed by Unsubscribe.
				return nil
			}

			// Parse string time from DB to time.Time
			t, err := time.Parse(time.RFC3339, event.UpdatedAt)
			if err != nil {
				t = time.Now()
			}

			err = stream.Send(&orderpb.OrderStatusUpdate{
				OrderId:   event.OrderID,
				Status:    event.Status,
				UpdatedAt: timestamppb.New(t),
			})
			if err != nil {
				log.Printf("[OrderHandler] send error for order %s: %v", orderID, err)
				return err
			}

			log.Printf("[OrderHandler] sent update: order=%s status=%s", event.OrderID, event.Status)
		}
	}
}
