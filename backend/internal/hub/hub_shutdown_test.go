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

// TestShutdownDrainsBufferedMessages verifies that messages already sitting in a
// client's Send buffer when shutdown begins are delivered, in order, before the
// channel is closed, and that Shutdown() does not return until the buffer is empty.
//
// Note: this test also passes against the pre-D28-fix hub, because closing a Go
// channel does not discard buffered values. It is a guard against the new drain
// path dropping, reordering, or deadlocking — not the D28 regression pin. That is
// TestShutdownDropsAfterDrainTimeout.
func TestShutdownDrainsBufferedMessages(t *testing.T) {
	h := New()
	ctx, cancel := context.WithCancel(context.Background())

	go h.Run(ctx)

	const bufferedCount = 8
	c := &Client{Hub: h, Send: make(chan interface{}, 16)}
	h.Register <- c
	waitForClients(t, h, 1)

	// Fill the buffer with nobody reading, so all bufferedCount messages are
	// still queued at the moment shutdown starts.
	for i := 0; i < bufferedCount; i++ {
		h.Broadcast <- i
	}

	deadline := time.After(2 * time.Second)
	for len(c.Send) < bufferedCount {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for %d buffered messages, got %d", bufferedCount, len(c.Send))
		default:
			// yield to the hub goroutine
		}
	}

	// The reader is created up front and parked on startReading so that it is
	// already scheduled when shutdown begins, rather than being spawned inside
	// the clientDrainTimeout window.
	startReading := make(chan struct{})
	readerDone := make(chan struct{})
	var received []interface{}
	go func() {
		defer close(readerDone)
		<-startReading
		for msg := range c.Send {
			received = append(received, msg)
		}
	}()

	cancel()
	close(startReading)

	shutdownDone := make(chan struct{})
	go func() {
		h.Shutdown()
		close(shutdownDone)
	}()

	select {
	case <-shutdownDone:
	case <-time.After(2 * time.Second):
		t.Fatal("Hub.Shutdown() did not return within 2s after context cancellation")
	}

	// Shutdown returned, so the drain either completed or timed out. With an
	// active reader it must have completed.
	if n := len(c.Send); n != 0 {
		t.Errorf("client buffer length after Shutdown() = %d, want 0", n)
	}

	select {
	case <-readerDone:
	case <-time.After(2 * time.Second):
		t.Fatal("client Send channel was not closed after shutdown")
	}

	if len(received) != bufferedCount {
		t.Fatalf("client received %d messages, want %d", len(received), bufferedCount)
	}
	for i, msg := range received {
		if msg != i {
			t.Errorf("received[%d] = %v, want %d", i, msg, i)
		}
	}
}

// TestShutdownDropsAfterDrainTimeout verifies the drain bound for a client that
// never reads: Shutdown() waits for the client rather than returning immediately,
// gives up after clientDrainTimeout, drops what is left, closes the channel, and
// returns. This is the D28 regression pin — against the unfixed hub the elapsed
// time is ~0 and the lower-bound assertion fails.
func TestShutdownDropsAfterDrainTimeout(t *testing.T) {
	h := New()
	ctx, cancel := context.WithCancel(context.Background())

	go h.Run(ctx)

	const bufferedCount = 4
	// This client is never drained by anyone.
	c := &Client{Hub: h, Send: make(chan interface{}, 16)}
	h.Register <- c
	waitForClients(t, h, 1)

	for i := 0; i < bufferedCount; i++ {
		h.Broadcast <- i
	}

	deadline := time.After(2 * time.Second)
	for len(c.Send) < bufferedCount {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for %d buffered messages, got %d", bufferedCount, len(c.Send))
		default:
			// yield to the hub goroutine
		}
	}

	DroppedMessages.Store(0)

	start := time.Now()
	cancel()

	shutdownDone := make(chan struct{})
	go func() {
		h.Shutdown()
		close(shutdownDone)
	}()

	select {
	case <-shutdownDone:
	case <-time.After(2 * time.Second):
		t.Fatal("Hub.Shutdown() did not return within 2s: the drain bound is not being honored")
	}
	elapsed := time.Since(start)

	// Lower bound: Shutdown must have waited for the client instead of closing
	// its channel immediately. This is what D28 broke.
	if elapsed < clientDrainTimeout {
		t.Errorf("Shutdown() returned after %v, want at least %v", elapsed, clientDrainTimeout)
	}

	// Upper bound: one wedged client must not extend shutdown indefinitely.
	if elapsed >= 2*time.Second {
		t.Errorf("Shutdown() took %v, want well under 2s", elapsed)
	}

	// The undrained messages are counted as dropped, not silently discarded.
	if got := DroppedMessages.Load(); got != bufferedCount {
		t.Errorf("DroppedMessages = %d, want exactly %d", got, bufferedCount)
	}

	// The channel is closed regardless of the timeout, and the buffered
	// messages are still readable from it — what was lost is the delivery
	// deadline, not the values.
	drained := 0
	for range c.Send {
		drained++
	}
	if drained != bufferedCount {
		t.Errorf("drained %d messages from the closed channel, want %d", drained, bufferedCount)
	}
}
