package hub

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// DroppedMessages tracks how many messages were dropped due to full client buffers.
var DroppedMessages atomic.Int64

const (
	// clientDrainTimeout bounds how long shutdown waits for a single client's
	// buffered messages to be consumed by its WritePump. A client that has not
	// drained within this window has its remaining buffered messages dropped —
	// an accepted trade-off so that one wedged client cannot hold up shutdown.
	clientDrainTimeout = 250 * time.Millisecond

	// clientDrainPollInterval is how often the drain loop re-checks len(c.Send).
	// A ticker rather than a busy spin: a worst-case drain wakes 250 times per
	// client instead of pegging a core for a quarter second per client.
	clientDrainPollInterval = 1 * time.Millisecond
)

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
			// Stop reading Broadcast: from here on nothing new is forwarded.
			// Snapshot and clear the client set under the lock, then release it
			// before waiting — a drain can take up to clientDrainTimeout and
			// must not block Register/Unregister or any reader of h.Clients.
			h.mu.Lock()
			clients := make([]*Client, 0, len(h.Clients))
			for client := range h.Clients {
				clients = append(clients, client)
				delete(h.Clients, client)
			}
			h.mu.Unlock()

			// Drain every client in parallel. Sequential draining would make
			// shutdown cost N * clientDrainTimeout.
			var wg sync.WaitGroup
			for _, client := range clients {
				wg.Add(1)
				go func(c *Client) {
					defer wg.Done()
					drainAndClose(c)
				}(client)
			}
			wg.Wait()

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

// drainAndClose waits for a client's already-buffered messages to be consumed by
// its WritePump, then closes the channel. The hub is the only writer to Send and
// is therefore the only closer.
//
// Closing the channel does not itself discard buffered messages — a receiver still
// drains them before observing the close. What D28 actually costs is that
// Shutdown() used to return while those messages were still queued, so the process
// exited with the write pumps mid-flight. Waiting here is what makes Shutdown()'s
// return mean "the backlog has been handed to the write pumps".
//
// ponytail: this waits for WritePump to *receive* the backlog, not to finish
// writing it — the last message may still be inside Conn.WriteJSON when this
// returns. Closing that gap requires a per-client done signal that WritePump
// closes in its defer; not built, because it would make every bare test client
// (one with no WritePump) block for the full timeout.
func drainAndClose(c *Client) {
	defer close(c.Send)

	if len(c.Send) == 0 {
		return
	}

	deadline := time.After(clientDrainTimeout)
	ticker := time.NewTicker(clientDrainPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			remaining := len(c.Send)
			DroppedMessages.Add(int64(remaining))
			log.Printf("hub: shutdown drain timed out for client %p, dropped %d buffered messages", c, remaining)
			return
		case <-ticker.C:
			if len(c.Send) == 0 {
				return
			}
		}
	}
}
