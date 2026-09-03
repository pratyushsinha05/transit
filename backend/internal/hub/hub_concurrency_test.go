package hub

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// TestBroadcastToMultipleClients verifies that a message sent to Broadcast
// is delivered to all currently registered clients.
func TestBroadcastToMultipleClients(t *testing.T) {
	h := New()
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		h.Shutdown()
	}()

	go h.Run(ctx)

	const clientCount = 5
	clients := make([]*Client, clientCount)
	for i := range clients {
		clients[i] = &Client{Hub: h, Send: make(chan interface{}, 16)}
		h.Register <- clients[i]
	}

	waitForClients(t, h, clientCount)

	testMsg := "broadcast-test-message"
	h.Broadcast <- testMsg

	for i, c := range clients {
		select {
		case msg, ok := <-c.Send:
			if !ok {
				t.Fatalf("client %d Send channel closed unexpectedly", i)
			}
			if msg != testMsg {
				t.Fatalf("client %d received %v, want %v", i, msg, testMsg)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for client %d to receive message", i)
		}
	}
}

// TestClientUnregister verifies that unregistering a client removes it
// from the hub's client map and closes its Send channel.
func TestClientUnregister(t *testing.T) {
	h := New()
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		h.Shutdown()
	}()

	go h.Run(ctx)

	c1 := &Client{Hub: h, Send: make(chan interface{}, 16)}
	c2 := &Client{Hub: h, Send: make(chan interface{}, 16)}

	h.Register <- c1
	h.Register <- c2
	waitForClients(t, h, 2)

	h.Unregister <- c1
	waitForClients(t, h, 1)

	// c1.Send should be closed
	select {
	case _, ok := <-c1.Send:
		if ok {
			t.Error("expected c1.Send to be closed, but received a value")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for c1.Send close")
	}

	// c2 should still receive broadcast
	h.Broadcast <- "for-c2"
	select {
	case msg := <-c2.Send:
		if msg != "for-c2" {
			t.Errorf("c2 received %v, want for-c2", msg)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for c2 to receive message")
	}
}

// TestUnregisterIdempotent verifies that unregistering a client twice does not panic.
func TestUnregisterIdempotent(t *testing.T) {
	h := New()
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		h.Shutdown()
	}()

	go h.Run(ctx)

	c := &Client{Hub: h, Send: make(chan interface{}, 16)}
	h.Register <- c
	waitForClients(t, h, 1)

	h.Unregister <- c
	waitForClients(t, h, 0)

	// Second unregister should be a no-op without panic
	h.Unregister <- c

	// Verify hub continues operating normally
	c2 := &Client{Hub: h, Send: make(chan interface{}, 16)}
	h.Register <- c2
	waitForClients(t, h, 1)
}

// TestDisconnectMidBroadcast verifies that clients disconnecting while
// broadcasts are actively occurring does not cause race conditions or deadlocks.
func TestDisconnectMidBroadcast(t *testing.T) {
	h := New()
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		h.Shutdown()
	}()

	go h.Run(ctx)

	const totalClients = 10
	clients := make([]*Client, totalClients)
	for i := range clients {
		clients[i] = &Client{Hub: h, Send: make(chan interface{}, 256)}
		h.Register <- clients[i]
	}
	waitForClients(t, h, totalClients)

	// Drain goroutines for all clients to prevent blocking
	var drainWg sync.WaitGroup
	stopDrain := make(chan struct{})
	for _, c := range clients {
		drainWg.Add(1)
		go func(cl *Client) {
			defer drainWg.Done()
			for {
				select {
				case <-stopDrain:
					return
				case _, ok := <-cl.Send:
					if !ok {
						return
					}
				}
			}
		}(c)
	}

	var opWg sync.WaitGroup
	opWg.Add(2)

	// Goroutine 1: Continuous broadcast
	go func() {
		defer opWg.Done()
		for i := 0; i < 100; i++ {
			h.Broadcast <- i
		}
	}()

	// Goroutine 2: Unregister half of the clients
	go func() {
		defer opWg.Done()
		for i := 0; i < totalClients/2; i++ {
			h.Unregister <- clients[i]
		}
	}()

	opWg.Wait()
	close(stopDrain)
	drainWg.Wait()
}

// TestSlowConsumerNeverReads verifies that when a client's Send channel buffer is full,
// the hub drops messages to that client without blocking other clients, and increments
// DroppedMessages by exactly the number of dropped messages.
func TestSlowConsumerNeverReads(t *testing.T) {
	h := New()
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		h.Shutdown()
	}()

	go h.Run(ctx)

	// slowClient has a buffer capacity of 1 and will NOT be drained.
	slowClient := &Client{Hub: h, Send: make(chan interface{}, 1)}
	// fastClient has ample buffer capacity and WILL be drained.
	fastClient := &Client{Hub: h, Send: make(chan interface{}, 64)}

	h.Register <- slowClient
	h.Register <- fastClient
	waitForClients(t, h, 2)

	// Drain fastClient in background to collect its messages
	var fastReceivedMu sync.Mutex
	var fastReceived []interface{}
	fastDone := make(chan struct{})
	go func() {
		defer close(fastDone)
		for msg := range fastClient.Send {
			fastReceivedMu.Lock()
			fastReceived = append(fastReceived, msg)
			fastReceivedMu.Unlock()
			if msg == "msg-done" {
				return
			}
		}
	}()

	DroppedMessages.Store(0)

	// Step 1: Send 1 message to fill slowClient's buffer of capacity 1.
	h.Broadcast <- "fill-buffer"

	// Wait deterministically for slowClient to receive the filling message
	deadline := time.After(2 * time.Second)
	for len(slowClient.Send) < 1 {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for slowClient to receive initial message")
		default:
		}
	}

	// At this point, slowClient.Send is full (length 1 of capacity 1).
	// DroppedMessages must be 0 so far.
	if got := DroppedMessages.Load(); got != 0 {
		t.Fatalf("DroppedMessages before drops = %d, want 0", got)
	}

	// Step 2: Broadcast exactly 5 more messages.
	// Since slowClient is full and not reading, all 5 messages MUST be dropped for slowClient.
	const dropCount = 5
	for i := 1; i <= dropCount; i++ {
		h.Broadcast <- i
	}

	// Step 3: Wait deterministically for DroppedMessages to reach exactly 5.
	deadline = time.After(2 * time.Second)
	for DroppedMessages.Load() < dropCount {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for DroppedMessages to reach %d, got %d", dropCount, DroppedMessages.Load())
		default:
		}
	}

	// Assert exact drop count
	gotDrops := DroppedMessages.Load()
	if gotDrops != dropCount {
		t.Errorf("DroppedMessages = %d, want exactly %d", gotDrops, dropCount)
	}

	// Send terminal message to signal fast consumer
	h.Broadcast <- "msg-done"

	select {
	case <-fastDone:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for fastClient drain to complete")
	}

	// Fast client must have received all 7 messages (1 fill + 5 intermediate + 1 done)
	fastReceivedMu.Lock()
	totalFast := len(fastReceived)
	fastReceivedMu.Unlock()
	expectedFast := 1 + dropCount + 1
	if totalFast != expectedFast {
		t.Errorf("fastClient received %d messages, want %d", totalFast, expectedFast)
	}

	// slowClient must still have only 1 message in its buffer (the original fill-buffer message)
	if len(slowClient.Send) != 1 {
		t.Errorf("slowClient buffer length = %d, want 1", len(slowClient.Send))
	}
}

// TestClientWritePump verifies that WritePump writes messages from Send to websocket,
// and exits cleanly with a CloseMessage when the Send channel is closed.
func TestClientWritePump(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	serverMsgChan := make(chan string, 10)
	serverClosed := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade error: %v", err)
			return
		}
		defer func() {
			conn.Close()
			close(serverClosed)
		}()

		for {
			var msg string
			err := conn.ReadJSON(&msg)
			if err != nil {
				return
			}
			serverMsgChan <- msg
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}

	h := New()
	client := &Client{
		Hub:  h,
		Conn: clientConn,
		Send: make(chan interface{}, 16),
	}

	writePumpDone := make(chan struct{})
	go func() {
		client.WritePump()
		close(writePumpDone)
	}()

	// Send a message through client.Send
	client.Send <- "hello-websocket"

	select {
	case msg := <-serverMsgChan:
		if msg != "hello-websocket" {
			t.Errorf("server received %q, want %q", msg, "hello-websocket")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for message to be written over websocket")
	}

	// Close Send to signal WritePump to terminate
	close(client.Send)

	select {
	case <-writePumpDone:
		// WritePump exited cleanly
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for WritePump to exit after channel close")
	}

	select {
	case <-serverClosed:
		// Server saw the close frame
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for server connection close")
	}
}

// TestClientReadPump verifies that ReadPump unregisters the client
// and closes connection upon websocket closure.
func TestClientReadPump(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

	var srvConnMu sync.Mutex
	var srvConn *websocket.Conn

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		srvConnMu.Lock()
		srvConn = conn
		srvConnMu.Unlock()
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}

	h := New()
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		h.Shutdown()
	}()
	go h.Run(ctx)

	client := &Client{
		Hub:  h,
		Conn: clientConn,
		Send: make(chan interface{}, 16),
	}
	h.Register <- client
	waitForClients(t, h, 1)

	readPumpDone := make(chan struct{})
	go func() {
		client.ReadPump()
		close(readPumpDone)
	}()

	// Close the connection from server side to trigger ReadPump exit
	deadline := time.After(2 * time.Second)
	for {
		srvConnMu.Lock()
		c := srvConn
		srvConnMu.Unlock()
		if c != nil {
			c.Close()
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for server connection to be established")
		default:
		}
	}

	select {
	case <-readPumpDone:
		// ReadPump returned
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for ReadPump to exit")
	}

	// ReadPump defer unregisters client
	waitForClients(t, h, 0)
}
