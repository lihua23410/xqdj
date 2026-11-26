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
