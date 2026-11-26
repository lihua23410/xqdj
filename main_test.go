package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"
	"xqdj/internal/sim"

	"github.com/gorilla/websocket"
)

func TestStaticAndWSStart(t *testing.T) {
	match := sim.NewMatch()
	h := newHub()
	halt := make(chan struct{})
	defer close(halt)
	go match.Loop(h.broadcast, halt)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		s := h.add(conn)
		defer func() {
			h.remove(conn)
			_ = conn.Close()
		}()
		h.send(s, match.SnapshotJSON())
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var msg clientMsg
			if json.Unmarshal(data, &msg) != nil {
				continue
			}
			if msg.Type == "start" {
				match.Start()
			}
			h.broadcast(match.SnapshotJSON())
		}
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	_, first, err := c.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(first), `"phase":"select"`) {
		t.Fatalf("lobby snapshot: %s", first)
	}
	if err := c.WriteJSON(clientMsg{Type: "start"}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	sawRunning := false
	sawUnits := false
	last := ""
	for time.Now().Before(deadline) {
		_ = c.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		_, data, err := c.ReadMessage()
		if err != nil {
			continue
		}
		s := string(data)
		last = s
		if strings.Contains(s, `"phase":"running"`) {
			sawRunning = true
		}
		if strings.Contains(s, `"role":"fighter"`) {
			sawUnits = true
		}
		if sawRunning && sawUnits {
			return
		}
	}
	t.Fatalf("running=%v units=%v last=%s", sawRunning, sawUnits, last)
}

func TestRefreshDoesNotLeakSpectatorWriters(t *testing.T) {
	h := newHub()
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		s := h.add(conn)
		defer func() {
			h.remove(conn)
			_ = conn.Close()
		}()
		h.send(s, []byte(`{"phase":"select"}`))
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"

	dial := func() {
		t.Helper()
		c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		_ = c.SetReadDeadline(time.Now().Add(time.Second))
		if _, _, err := c.ReadMessage(); err != nil {
			t.Fatal(err)
		}
	}
	dial()
	waitSpectatorWriters(t, 0)

	for i := 0; i < 25; i++ {
		dial()
	}
	waitSpectatorWriters(t, 0)
}

func waitSpectatorWriters(t *testing.T, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		if n := spectatorWriters(); n == want {
			return
		} else if time.Now().After(deadline) {
			t.Fatalf("spectator writers = %d, want %d", n, want)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func spectatorWriters() int {
	buf := make([]byte, 1<<20)
	n := runtime.Stack(buf, true)
	return strings.Count(string(buf[:n]), "(*spectator).write")
}

func TestSnapshotDoesNotDisplaceQueuedLog(t *testing.T) {
	ch := make(chan outbound, 1)
	offer(ch, outbound{body: []byte("snap")})
	offer(ch, outbound{body: []byte("log"), keep: true})
	offer(ch, outbound{body: []byte("newer-snap")})
	got := <-ch
	if string(got.body) != "log" || !got.keep {
		t.Fatalf("snapshot displaced the log: %+v", got)
	}
}

func TestNewerLogReplacesQueuedLog(t *testing.T) {
	ch := make(chan outbound, 1)
	offer(ch, outbound{body: []byte("log-1"), keep: true})
	offer(ch, outbound{body: []byte("log-2"), keep: true})
	got := <-ch
	if string(got.body) != "log-2" {
		t.Fatalf("queued log = %s", got.body)
	}
}

func TestSnapshotsStillCoalesce(t *testing.T) {
	ch := make(chan outbound, 1)
	offer(ch, outbound{body: []byte("old")})
	offer(ch, outbound{body: []byte("new")})
	got := <-ch
	if string(got.body) != "new" {
		t.Fatalf("queued snapshot = %s", got.body)
	}
}

func TestLogReachesClientWhileSnapshotsBroadcast(t *testing.T) {
	match := sim.NewMatch()
	h := newHub()
	halt := make(chan struct{})
	defer close(halt)
	go match.Loop(h.broadcast, halt)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		s := h.add(conn)
		defer func() {
			h.remove(conn)
			_ = conn.Close()
		}()
		h.send(s, match.SnapshotJSON())
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var msg clientMsg
			if json.Unmarshal(data, &msg) != nil {
				continue
			}
			if msg.Type != "log" {
				h.broadcast(match.SnapshotJSON())
				continue
			}
			b, err := json.Marshal(struct {
				Type string `json:"type"`
			}{Type: "log"})
			if err == nil {
				h.sendKept(s, b)
			}
		}
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	for n := 0; n < 30; n++ {
		if err := c.WriteJSON(clientMsg{Type: "log"}); err != nil {
			t.Fatal(err)
		}
		deadline := time.Now().Add(time.Second)
		saw := false
		for time.Now().Before(deadline) {
			_ = c.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
			_, data, err := c.ReadMessage()
			if err != nil {
				continue
			}
			if strings.Contains(string(data), `"type":"log"`) {
				saw = true
				break
			}
		}
		if !saw {
			t.Fatalf("log request %d was overwritten by a snapshot", n)
		}
	}
}
