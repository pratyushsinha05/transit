# D28 — Hub shutdown discards buffered client messages: execution spec

**Audience:** an executor with no prior context on this repository. Everything needed is
quoted inline. Do not consult other documents in this repo; where one disagrees with the
code quoted here, the code wins.

**Repository root:** the directory containing `backend/`, `frontend/`, `infra/`.
**Go module:** `transit-backend` (declared in `backend/go.mod`), Go 1.23.0, toolchain
observed at go1.26.0.
**Files you will modify:** exactly two.

| File | Action |
|---|---|
| `backend/internal/hub/hub.go` | UPDATE — add two constants, one helper, rewrite one `case` |
| `backend/internal/hub/hub_shutdown_test.go` | UPDATE — append two new test functions |

Nothing else. Not `client.go`, not `main.go`, not any other test.

---

## 0. Correction to the recorded defect statement — read before anything else

D28 is recorded in this repo's `IDEAS.md` as:

> **D28 — Hub shutdown discards buffered client messages.**
> When `Hub.Run` exits on `ctx.Done()`, it immediately closes all client `Send` channels.
> `Client.WritePump` handles channel closure by immediately sending a WebSocket close frame,
> potentially dropping messages that were buffered in `client.Send` but not yet written.

**The stated mechanism is wrong.** Closing a Go channel does not discard values already
buffered in it. Per the Go specification, receive operations on a closed channel return the
previously sent values first, and only return the zero value with `ok == false` after the
buffer is exhausted. Verified directly:

```go
ch := make(chan int, 8)
for i := 0; i < 8; i++ { ch <- i }
close(ch)
n := 0
for range ch { n++ }
// n == 8
```

So `Client.WritePump`, quoted in §1.2, does **not** jump to the close-frame branch while
messages remain buffered. It receives all of them with `ok == true`, writes each one, and
only then observes the close.

**The message loss is real, but it happens one level up.** `Hub.Shutdown()` returns as soon
as `h.done` is closed, which happens immediately after the `Send` channels are closed. It
does not wait for any `WritePump` to flush those messages onto its socket. In
`backend/cmd/server/main.go`, `wsHub.Shutdown()` is the last statement before `main` returns,
so the process exits with every `WritePump` goroutine still mid-flight. The queued messages
are lost to process exit, not to `close()`.

**Consequences for this task, all of which are already reflected below:**

1. The fix shape the defect prescribes — stop broadcasting, let each client drain, bounded
   by a timeout, then close — is still correct. Only the reason it works changes: it makes
   `Shutdown()` block until the write pumps have consumed the backlog, instead of returning
   while they are still working.
2. The drain condition is "the client's `WritePump` has **received** every buffered message",
   not "has **written** every buffered message". One message per client may still be inside
   `Conn.WriteJSON` when `Shutdown()` returns. That residual is the accepted ceiling; see
   §3.4.
3. A test that only asserts "the client receives all N messages before its channel closes"
   passes against the **unfixed** code, because `close()` preserves the buffer. The test that
   actually pins the defect is the one in §6.2, which asserts that `Shutdown()` *waits*. Both
   are specified; do not drop the second one.

After the code change lands, update the D28 entry in `IDEAS.md` to describe the real
mechanism. Do not leave the incorrect wording in place.

---

## 1. Current source, verbatim

### 1.1 `backend/internal/hub/hub.go` — the entire file as it stands

```go
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
```

### 1.2 `backend/internal/hub/client.go` — the parts that matter (read-only context)

```go
type Client struct {
	Hub *Hub

	// The websocket connection.
	Conn *websocket.Conn

	// Buffered channel of outbound messages.
	Send chan interface{}
}
```

```go
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteJSON(message); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
```

`client.go` also declares `writeWait = 10 * time.Second`, `pongWait = 60 * time.Second`,
`pingPeriod = (pongWait * 9) / 10`, and `maxMessageSize = 512`. **Do not modify `client.go`.**

### 1.3 Who creates clients — `backend/internal/handlers/websocket.go` (read-only context)

```go
	client := &hub.Client{Hub: h.hub, Conn: conn, Send: make(chan interface{}, 256)}
	client.Hub.Register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.WritePump()
	go client.ReadPump()
```

Production clients therefore have a 256-slot `Send` buffer and exactly one consumer
(`WritePump`). Test clients in `internal/hub` are bare structs with a buffered channel and
**no** `WritePump` — the drain mechanism must not assume a consumer exists.

### 1.4 Shutdown ordering — `backend/cmd/server/main.go` (read-only context)

Hub construction:

```go
	// 6. Initialize Hub
	wsHub := hub.New()
	hubCtx, hubCancel := context.WithCancel(context.Background())
	go wsHub.Run(hubCtx)
	log.Println("WebSocket hub started")
```

Shutdown:

```go
	// 11. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		log.Printf("Echo shutdown error: %v", err)
	}

	hubCancel()
	wsHub.Shutdown()

	log.Println("Server stopped")
```

**This ordering is load-bearing and must not change.** The HTTP server is fully shut down
*before* the hub context is cancelled, so no request handler is still producing broadcasts
by the time the hub begins draining. §3.1 depends on this.

---

## 2. The five specification points

### 2.1 Where "stop accepting new broadcasts" happens

**Decision: `Run` stops reading `h.Broadcast` the instant it selects `ctx.Done()`, and never
reads it again.** The drain phase forwards nothing; late broadcasts are neither delivered nor
discarded-with-a-log, they simply have no reader.

Rationale: this is already the behavior today — `Run` returns from the `ctx.Done()` branch
without further reads — so keeping it is the smaller diff and introduces no new semantics.
The alternative, continuing to read `h.Broadcast` during the drain window and dropping what
arrives, only narrows the window in which a late producer can block; it cannot close it,
because the producer can still block after `Run` finally returns. It would buy a partial
guarantee at the cost of a new silent-discard path, which is exactly the kind of half-truth
this codebase already has too much of. The reason a hard cutoff is safe here is structural,
not lucky: `main.go` (§1.4) completes `e.Shutdown(ctx)` before calling `hubCancel()`, so
every HTTP handler that could call `Broadcast` has already returned, and the 256-slot buffer
absorbs anything queued but unprocessed. A caller that blocks on `h.Broadcast` after
shutdown has begun is a caller that outlived the server, which is a different defect from
D28 and is not in scope.

### 2.2 The drain mechanism

**Decision: poll `len(client.Send) == 0`.** Rejected alternative: a per-client `Done` channel
that `WritePump` closes to signal drain completion.

Why polling wins:

- **`WritePump` cannot signal "the buffer is empty" without observing `len` itself.** Its
  only completion signal is the channel *close*, and the close is precisely what we are
  trying to defer. Any signal it could send would be derived from the same `len` check, just
  moved into the client.
- **Test clients have no `WritePump`.** As shown in §1.3, `Send` is filled by the hub but
  consumed by a goroutine that only exists in production and in the two websocket tests. A
  drain that waits on a `Done` channel would block on a nil or never-closed channel for every
  bare test client, forcing every shutdown in the test suite to eat the full timeout and
  making the hub untestable without a live socket.
- **`len` on a channel needs no lock and is not a data race.** It is a runtime read of the
  queue count, safe for concurrent use. The existing tests already rely on this while the
  hub is concurrently sending — see `hub_concurrency_test.go` (`for len(slowClient.Send) < 1`)
  and `hub_drop_test.go` (`for len(slowClient.Send) == 0`), both of which pass under
  `go test -race` today.

**What the lock does and does not cover.** `h.mu` protects the `h.Clients` map only. The
`ctx.Done()` branch takes `h.mu.Lock()` once, copies the client set into a slice, empties the
map, and **releases the lock before any waiting begins**. Holding a write lock across a
250 ms wait would block every `waitForClients` helper in the test suite and any concurrent
`Register`/`Unregister` for the duration, for no benefit — once a client is out of the map it
is unreachable to every other code path, and `Run` is the only goroutine that ever closes a
`Send` channel, so there is no double-close hazard after the map is cleared.

**Exact wait, per client:**

1. Fast path: if `len(c.Send) == 0`, close immediately and return.
2. Otherwise start a `time.After(clientDrainTimeout)` deadline and a
   `time.NewTicker(clientDrainPollInterval)`.
3. On each tick, re-check `len(c.Send)`; when it reaches 0, close and return.
4. On the deadline, record the remaining count in `DroppedMessages`, log it, close, return.

The close happens on both exits via a single `defer close(c.Send)` — the hub is the writer and
therefore the closer, which is the project's channel-ownership rule.

### 2.3 The bound

Two named constants, declared in `backend/internal/hub/hub.go` in a new `const` block placed
directly beneath the existing `var DroppedMessages atomic.Int64` declaration. Neither value
may appear inline.

```go
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
```

`hub.go` does not currently import `time`. Add it.

**Total worst-case shutdown duration: `clientDrainTimeout`, i.e. 250 ms, for any N.**

**Draining is parallel — one goroutine per client, joined by a `sync.WaitGroup`.** This is not
optional. Sequential draining would make the worst case `N × 250 ms`: 2.5 s at 10 clients,
250 s at the 1,000-concurrent-client figure this system targets. A shutdown path whose
duration scales with connection count is itself a defect, and a worse one than D28. With
parallel drains the wall-clock cost is one timeout regardless of N; the only thing that scales
is N transient goroutines, each of which does nothing but read an integer once a millisecond.

Where 250 ms sits in the overall budget: `main.go` gives Echo its own 10 s context, then calls
`hubCancel()` and `wsHub.Shutdown()` outside that budget. 250 ms is invisible next to it, and
is roughly two orders of magnitude above the cost of a `WriteJSON` to a loopback or LAN
socket, so a healthy client drains a full 256-slot buffer with room to spare.

### 2.4 What does not change

- **`Shutdown()`'s signature is unchanged**: `func (h *Hub) Shutdown()`, no arguments, no
  return value, still blocking on `<-h.done`. Its doc comment (*"blocks until Run has drained
  all clients and exited"*) becomes more literally true than it was.
- **`main.go` needs no edit.** `hubCancel()` followed by `wsHub.Shutdown()` already expresses
  exactly the right thing; the difference is that `Shutdown()` now returns later and with the
  work actually done.
- **`client.go` needs no edit.** `WritePump`'s close-frame branch is correct as written.
- **`Hub` struct fields, `New()`, and the `Register` / `Unregister` / `Broadcast` cases are
  untouched.** Only the `ctx.Done()` case is rewritten.

**`TestHubShutdown` compatibility — verified against the test as committed.** The test
(`backend/internal/hub/hub_shutdown_test.go`) registers three clients with `make(chan
interface{}, 16)` and **no reader**, waits for all three registrations, broadcasts one
`"test-message"`, cancels the context, and then asserts four things. Each is checked against
the new logic:

| What `TestHubShutdown` asserts | Under the new drain logic |
|---|---|
| `Shutdown()` returns within 2 s of cancellation | Passes. No client is ever read, so all three drains hit `clientDrainTimeout` — **in parallel**, so ~250 ms total, well inside 2 s. |
| The `Run` goroutine terminates within 2 s | Passes. `Run` returns immediately after `wg.Wait()` and `close(h.done)`. |
| Every client's `Send` channel is closed | Passes. `drainAndClose` closes on both exit paths via `defer`. |
| `len(h.Clients) == 0` | Passes. The map is emptied under `h.mu.Lock()` in the same statement that snapshots it, before any waiting. |

Note the test's own tolerance for a scheduling race: because `select` picks randomly among
ready cases, the broadcast may or may not be processed before `ctx.Done()` wins, so a client
buffer holds either 0 or 1 message. The test handles both (`case _, ok := <-c.Send` → if
`ok`, `for range c.Send` drains to close). The new logic handles both too: 0 takes the fast
path, 1 takes the 250 ms timeout. **`TestHubShutdown` must pass unmodified. Do not edit it.**

**Expected runtime change across the existing suite.** Every test whose teardown is
`defer func() { cancel(); h.Shutdown() }()` while a client still holds buffered messages now
pays 250 ms at teardown: `TestHubShutdown`, `TestSlowConsumerNeverReads`,
`TestDropCounterIncrements`, and possibly `TestDisconnectMidBroadcast`. Tests whose clients
are fully drained (`TestBroadcastToMultipleClients`, `TestClientUnregister`,
`TestUnregisterIdempotent`, `TestClientReadPump`) pay nothing. Package wall time goes from
~1.6 s to roughly ~2.7 s. That is expected, not a hang.

### 2.5 The new tests

Two functions, appended to `backend/internal/hub/hub_shutdown_test.go`. Full code in §6.
They follow the file's existing style: flat top-level functions (no `t.Run` subtests appear
anywhere in this package), deterministic polling loops with a `time.After` deadline and a
`default:` yield, exact counts asserted, and **no `time.Sleep` anywhere** — that is a hard
project rule.

- **§6.1 `TestShutdownDrainsBufferedMessages`** — the delivery guard. A client with 8
  messages queued and a reader that is created up front and released the moment shutdown
  starts; asserts all 8 arrive in order, the channel then closes, `Shutdown()` returns, and
  `len(c.Send) == 0` once it does. Per §0.3 this test also passes against the unfixed code;
  it exists to prove the new drain path does not itself drop, reorder, or deadlock.
- **§6.2 `TestShutdownDropsAfterDrainTimeout`** — the actual D28 regression pin, and the
  "client that never reads" case. No reader is ever attached. Asserts `Shutdown()` takes **at
  least `clientDrainTimeout`** (proving it waited rather than returning instantly, which is
  what the unfixed code does) and **less than 2 s** (proving the bound is honored and shutdown
  cannot hang), that the channel is closed, and that `DroppedMessages` increased by exactly
  the number of undrained messages. Measuring elapsed time is a clock *read*, not a sleep,
  and is permitted.

---

## 3. The change, precisely

### 3.1 Imports

`backend/internal/hub/hub.go` currently imports `context`, `log`, `sync`, `sync/atomic`.
**Add `time`.** Final import block:

```go
import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"
)
```

### 3.2 Constants

Insert the `const` block from §2.3 immediately after
`var DroppedMessages atomic.Int64` and before `type Hub struct`.

### 3.3 Replace the `ctx.Done()` case

Replace this, and only this, inside `Run`:

```go
		case <-ctx.Done():
			h.mu.Lock()
			for client := range h.Clients {
				close(client.Send)
				delete(h.Clients, client)
			}
			h.mu.Unlock()
			close(h.done)
			return
```

with:

```go
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
```

### 3.4 Add the helper

Append to `backend/internal/hub/hub.go`, after `Shutdown()`:

```go
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
```

Note on the `DroppedMessages.Add` call: it reuses the existing counter for exactly what the
counter is named for, and it makes the timeout drop visible instead of silent. It cannot
disturb the two existing drop-counting tests — both call `DroppedMessages.Store(0)` after
registering their client and assert their counts *before* their deferred `Shutdown()` runs,
and no test in this package calls `t.Parallel()`.

Note on the `log.Printf`: this fires once per timed-out client during shutdown, not in the
ingestion loop, so it does not violate the project's rule against logging on hot paths.

---

## 4. Pre-flight

Run from the repository root. This must be green **before** you edit anything; if it is not,
stop and report, because the baseline is not what this document describes.

```bash
cd backend
gofmt -l internal/hub/
go vet ./internal/hub/...
go test -race -count=1 -cover ./internal/hub/...
go build ./...
```

Expected:

- `gofmt -l` prints **nothing**.
- `go vet` prints nothing and exits 0.
- `go test` prints `ok  	transit-backend/internal/hub	<time>	coverage: 87.0% of statements`.
- `go build ./...` prints nothing and exits 0.

---

## 5. Verification

```bash
cd backend
gofmt -l internal/hub/
go vet ./internal/hub/...
go test -race -count=1 -cover ./internal/hub/...
go build ./...
```

Expected:

- `gofmt -l` prints nothing.
- `go vet` clean.
- `ok  	transit-backend/internal/hub	<time>	coverage: >= 87.0% of statements` — coverage
  must not fall below the 87.0% baseline. Wall time rises to roughly 2.5–3 s for the reason
  given in §2.4; that is expected.
- `go build ./...` clean.

Then the targeted run:

```bash
cd backend
go test -race -count=1 -v -run 'TestHubShutdown|TestShutdownDrainsBufferedMessages|TestShutdownDropsAfterDrainTimeout' ./internal/hub/...
```

Expected — all three `PASS`, `TestHubShutdown` **unmodified**:

```
=== RUN   TestHubShutdown
--- PASS: TestHubShutdown (0.2Xs)
=== RUN   TestShutdownDrainsBufferedMessages
--- PASS: TestShutdownDrainsBufferedMessages (0.0Xs)
=== RUN   TestShutdownDropsAfterDrainTimeout
--- PASS: TestShutdownDropsAfterDrainTimeout (0.2Xs)
PASS
ok  	transit-backend/internal/hub
```

**Regression proof — confirm the new test actually pins the defect.** Temporarily revert only
the `ctx.Done()` case in `hub.go` to the version quoted in §3.3, leaving both new tests in
place, and run:

```bash
cd backend
go test -count=1 -run TestShutdownDropsAfterDrainTimeout ./internal/hub/...
```

Expected: **FAIL**, with a message of the form
`Shutdown() returned after 0s, want at least 250ms`. Restore the fix and re-run §5 before
committing. If this step does not fail, the test is not pinning anything and must be
rewritten.

---

## 6. New test code

Append both functions to `backend/internal/hub/hub_shutdown_test.go`. The existing imports of
that file are `context`, `sync`, `testing`, `time` — all four remain in use and no new import
is needed. The helper `waitForClients(t, h, want)` is already defined at the top of that file;
reuse it.

### 6.1 `TestShutdownDrainsBufferedMessages`

```go
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
```

Race-safety of `received`: it is written only by the reader goroutine and read by the test
goroutine only after `<-readerDone`, which happens-after `close(readerDone)`, which
happens-after the final write. Clean under `-race`.

### 6.2 `TestShutdownDropsAfterDrainTimeout`

```go
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
```

The `for range c.Send` at the end terminates only if the channel was closed; if the fix
forgot to close on the timeout path, this test hangs and the Go test binary panics on its own
10-minute deadline. That is an acceptable failure signal for a case that should be impossible.

---

## 7. Must not be touched

| Thing | Why |
|---|---|
| `backend/cmd/server/main.go` shutdown ordering — `e.Shutdown(ctx)` → `hubCancel()` → `wsHub.Shutdown()` | Established in an earlier phase. §2.1's hard broadcast cutoff is only safe because the HTTP server is already down when the hub cancels. |
| `Hub.Shutdown()`'s signature | Callers must not change. Only its timing changes. |
| `backend/internal/hub/client.go` | `WritePump`'s close-frame handling is correct. The fix is entirely hub-side. |
| `backend/internal/hub/message.go` — the `Message` struct | It is the canonical WebSocket envelope: flat (no `data` wrapper), no `heading` field, and deliberately **no `omitempty` on any numeric field** so a stopped device serializes `"speed": 0` rather than omitting the key. Two separate closed defects depend on this exact shape. |
| `backend/internal/hub/message_test.go` | It asserts the serialized JSON shape, including that `heading` is *absent*. Do not "tidy" that assertion away. |
| `TestHubShutdown` in `hub_shutdown_test.go` | Must pass **unmodified** — see §2.4. If you find yourself editing it, the fix is wrong. |
| Every other existing test in `internal/hub` | Only their teardown wall time changes. |
| The `Register` / `Unregister` / `Broadcast` cases in `Run` | Out of scope. |
| `h.Broadcast`'s 256-slot capacity and `New()`'s construction | Out of scope. |

**One known, pre-existing, out-of-scope interaction.** Once `Run` returns, nothing reads
`h.Unregister`, which is unbuffered. A `ReadPump` whose connection dies during or after
shutdown blocks forever on `c.Hub.Unregister <- c`. This is true of the code today and is
unchanged by this fix; the process is exiting. Do not attempt to solve it here — note it and
move on.

---

## 8. Definition of done

1. `backend/internal/hub/hub.go` changed exactly as specified in §3 — imports, constants,
   the `ctx.Done()` case, and `drainAndClose`.
2. Both test functions from §6 appended to `backend/internal/hub/hub_shutdown_test.go`.
3. §5 verification green, coverage at or above 87.0%.
4. The §5 regression proof demonstrated: `TestShutdownDropsAfterDrainTimeout` fails against
   the reverted `ctx.Done()` case.
5. The D28 entry in `IDEAS.md` rewritten to state the real mechanism per §0 — the loss is
   `Shutdown()` returning before the write pumps have consumed the backlog, not `close()`
   discarding buffered values.
6. One commit, conventional-commit format, e.g.
   `fix(hub): drain buffered client messages before closing on shutdown`.
