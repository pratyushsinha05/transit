package hub

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
)

// DroppedMessages tracks how many messages were dropped due to full client buffers.
var DroppedMessages atomic.Int64

type Hub struct {
	Clients    map[*Client]bool
	Broadcast  chan interface{}
	Register   chan *Client
	Unregister chan *Client
	mu         sync.RWMutex
	done       chan struct{}
}

func New() *Hub {
	return &Hub{
		Broadcast:  make(chan interface{}, 256),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Clients:    make(map[*Client]bool),
		done:       make(chan struct{}),
	}
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			h.mu.Lock()
			for client := range h.Clients {
				close(client.Send)
				delete(h.Clients, client)
			}
			h.mu.Unlock()
			close(h.done)
			return
		case client := <-h.Register:
			h.mu.Lock()
			h.Clients[client] = true
			h.mu.Unlock()
		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				close(client.Send)
			}
			h.mu.Unlock()
		case message := <-h.Broadcast:
			h.mu.RLock()
			for client := range h.Clients {
				select {
				case client.Send <- message:
				default:
					DroppedMessages.Add(1)
					log.Printf("hub: dropped message for client %p (buffer full)", client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Shutdown blocks until Run has drained all clients and exited.
func (h *Hub) Shutdown() {
	<-h.done
}
