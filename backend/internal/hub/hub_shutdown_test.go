package hub

import (
	"context"
	"sync"
	"testing"
	"time"
)

// waitForClients polls h.Clients under RLock until the count reaches want,
// bounded by a 2-second timeout. Returns the observed count on timeout.
func waitForClients(t *testing.T, h *Hub, want int) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		h.mu.RLock()
		n := len(h.Clients)
		h.mu.RUnlock()
		if n == want {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for %d clients, got %d", want, n)
		default:
			// yield to the hub goroutine
		}
	}
}

// TestHubShutdown verifies that Hub.Run exits cleanly when its
// context is cancelled, closes all client Send channels, drains all clients,
// and that Shutdown() blocks until Run has fully returned.
// This is the DEFECT-6 exit gate.
func TestHubShutdown(t *testing.T) {
	h := New()
	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		h.Run(ctx)
	}()

	// Register three clients with buffered Send channels
	clients := make([]*Client, 3)
	for i := range clients {
		clients[i] = &Client{Hub: h, Send: make(chan interface{}, 16)}
		h.Register <- clients[i]
	}

	// Wait deterministically for all three registrations to be processed
	waitForClients(t, h, 3)

	// Broadcast one message so the hub exercises the active path.
	// After waitForClients returned, the hub's select loop is idle and will
	// pick up the broadcast on the next iteration.
	h.Broadcast <- "test-message"

	// Cancel context to initiate shutdown
	cancel()

	// Shutdown should unblock once Run exits
	shutdownDone := make(chan struct{})
	go func() {
		h.Shutdown()
		close(shutdownDone)
	}()

	select {
	case <-shutdownDone:
		// Success: Shutdown returned
	case <-time.After(2 * time.Second):
		t.Fatal("Hub.Shutdown() did not return within 2s after context cancellation")
	}

	// Wait for Run goroutine to complete
	runDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(runDone)
	}()

	select {
	case <-runDone:
		// Success: Run goroutine exited
	case <-time.After(2 * time.Second):
		t.Fatal("Hub.Run goroutine did not terminate within 2s")
	}

	// Verify all client Send channels are closed
	for i, c := range clients {
		select {
		case _, ok := <-c.Send:
			if ok {
				// Channel still has the buffered message, drain until closed
				for range c.Send {
				}
			}
			// Channel is closed (ok == false or drained to close)
		default:
			t.Errorf("client %d Send channel was not closed on shutdown", i)
		}
	}

	// Verify Clients map is drained
	h.mu.RLock()
	if len(h.Clients) != 0 {
		t.Errorf("expected 0 clients remaining after shutdown, got %d", len(h.Clients))
	}
	h.mu.RUnlock()
}
