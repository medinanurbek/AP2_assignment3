package repository

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/lib/pq"
)

// OrderStatusEvent is the payload decoded from the pg_notify JSON.
type OrderStatusEvent struct {
	OrderID   string `json:"order_id"`
	Status    string `json:"status"`
	UpdatedAt string `json:"updated_at"`
}

// PubSub is a simple in-memory fan-out broker.
// When Postgres fires a NOTIFY, Publish sends the event to every subscriber.
type PubSub struct {
	mu          sync.RWMutex
	subscribers map[string][]chan OrderStatusEvent // keyed by order_id ("" = all)
}

func NewPubSub() *PubSub {
	return &PubSub{
		subscribers: make(map[string][]chan OrderStatusEvent),
	}
}

// Subscribe registers a channel that will receive events for the given orderID.
// Use "" to receive all events.
func (ps *PubSub) Subscribe(orderID string) chan OrderStatusEvent {
	ch := make(chan OrderStatusEvent, 8)
	ps.mu.Lock()
	ps.subscribers[orderID] = append(ps.subscribers[orderID], ch)
	ps.mu.Unlock()
	return ch
}

// Unsubscribe removes the channel from the subscriber list.
func (ps *PubSub) Unsubscribe(orderID string, ch chan OrderStatusEvent) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	subs := ps.subscribers[orderID]
	filtered := subs[:0]
	for _, s := range subs {
		if s != ch {
			filtered = append(filtered, s)
		}
	}
	ps.subscribers[orderID] = filtered
	close(ch)
}

// Publish fans-out an event to subscribers whose key matches orderID or "".
func (ps *PubSub) Publish(event OrderStatusEvent) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	send := func(ch chan OrderStatusEvent) {
		select {
		case ch <- event:
		default:
			// drop if consumer is slow — prevents blocking the listener goroutine
		}
	}

	for _, ch := range ps.subscribers[event.OrderID] {
		send(ch)
	}
	for _, ch := range ps.subscribers[""] {
		send(ch)
	}
}

// StartListener opens a pq.Listener on the given DSN, listens on the
// "order_status_updates" channel and publishes decoded events to ps.
// This function blocks; run it in a goroutine.
func StartListener(dsn string, ps *PubSub) {
	onProblem := func(ev pq.ListenerEventType, err error) {
		if err != nil {
			log.Printf("[Postgres Listener] error: %v", err)
		}
	}

	listener := pq.NewListener(dsn, 5*time.Second, time.Minute, onProblem)
	if err := listener.Listen("order_status_updates"); err != nil {
		log.Fatalf("[Postgres Listener] failed to listen: %v", err)
	}

	log.Println("[Postgres Listener] listening on channel 'order_status_updates'")

	for {
		select {
		case notification := <-listener.Notify:
			if notification == nil {
				// nil means the listener reconnected; nothing to decode
				continue
			}
			var event OrderStatusEvent
			if err := json.Unmarshal([]byte(notification.Extra), &event); err != nil {
				log.Printf("[Postgres Listener] failed to parse notification: %v", err)
				continue
			}
			log.Printf("[Postgres Listener] received: order=%s status=%s", event.OrderID, event.Status)
			ps.Publish(event)

		case <-time.After(90 * time.Second):
			// keepalive ping so the connection doesn't time out
			go func() {
				if err := listener.Ping(); err != nil {
					log.Printf("[Postgres Listener] ping error: %v", err)
				}
			}()
		}
	}
}
