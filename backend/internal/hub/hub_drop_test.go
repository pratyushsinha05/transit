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
	defer cancel()

	go h.Run(ctx)

	// Register a client with a buffer of 1.
	slowClient := &Client{Hub: h, Send: make(chan interface{}, 1)}
	h.Register <- slowClient
	time.Sleep(10 * time.Millisecond)

	// Reset the counter.
	DroppedMessages.Store(0)

	// Fill the buffer.
	h.Broadcast <- "msg1"
	time.Sleep(10 * time.Millisecond)

	// This message should be dropped because buffer is full and client is not reading.
	h.Broadcast <- "msg2"
	time.Sleep(10 * time.Millisecond)

	got := DroppedMessages.Load()
	if got < 1 {
		t.Errorf("DroppedMessages = %d, want >= 1", got)
	}
}
