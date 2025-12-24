package hub

import (
	"sync"
)

type Hub struct {
	Clients    map[*Client]bool
	Broadcast  chan interface{}
	Register   chan *Client
	Unregister chan *Client
	mu         sync.RWMutex
}

func New() *Hub {
	return &Hub{
		Broadcast:  make(chan interface{}, 256),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Clients:    make(map[*Client]bool),
	}
}

func (h *Hub) Run() {
	for {
		select {
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
					// If the client's buffer is full, we assume it's dead or stuck.
					// In a more robust system we might disconnect them.
					// For now we just skip dropping the message for this client.
					// To strictly follow prompt "send non-blocking (skip if buffer full)"
					// we just skip here.
				}
			}
			h.mu.RUnlock()
		}
	}
}
