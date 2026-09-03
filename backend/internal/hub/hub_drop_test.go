package hub

import (
	"context"
	"testing"
	"time"
)

// TestDropCounterIncrements verifies that broadcasting to a client with a
// full Send channel increments DroppedMessages and does not block.
func TestDropCounterIncrements(t *testing.T) {
	h := New()
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		h.Shutdown()
	}()

	go h.Run(ctx)

	// Register a client with a buffer of 1.
	slowClient := &Client{Hub: h, Send: make(chan interface{}, 1)}
	h.Register <- slowClient
	waitForClients(t, h, 1)

	// Reset the counter.
	DroppedMessages.Store(0)

	// Fill the buffer.
	h.Broadcast <- "msg1"

	// Wait deterministically until slowClient buffer receives msg1
	deadline := time.After(2 * time.Second)
	for len(slowClient.Send) == 0 {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for slowClient to receive msg1")
		default:
			// yield
		}
	}

	// This message should be dropped because buffer is full and client is not reading.
	h.Broadcast <- "msg2"

	// Wait deterministically for DroppedMessages to increment
	deadline = time.After(2 * time.Second)
	for DroppedMessages.Load() < 1 {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for DroppedMessages to increment")
		default:
			// yield
		}
	}

	got := DroppedMessages.Load()
	if got < 1 {
		t.Errorf("DroppedMessages = %d, want >= 1", got)
	}
}
