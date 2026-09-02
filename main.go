package main

import (
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"sync"
	"xqdj/internal/sim"
	"xqdj/internal/web"

	_ "xqdj/character"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type hub struct {
	mu      sync.Mutex
	clients map[*websocket.Conn]struct{}
}

func newHub() *hub {
	return &hub{clients: make(map[*websocket.Conn]struct{})}
}

func (h *hub) add(c *websocket.Conn) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
}

func (h *hub) remove(c *websocket.Conn) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
}

func (h *hub) send(c *websocket.Conn, msg []byte) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return c.WriteMessage(websocket.TextMessage, msg)
}

func (h *hub) broadcast(msg []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		if err := c.WriteMessage(websocket.TextMessage, msg); err != nil {
			_ = c.Close()
			delete(h.clients, c)
		}
	}
}

type clientMsg struct {
	Type string `json:"type"`
	Slot int    `json:"slot"`
	Kind string `json:"kind"`
}

func main() {
	match := sim.NewMatch()
	h := newHub()
	halt := make(chan struct{})
	go match.Loop(h.broadcast, halt)

	static, err := fs.Sub(web.FS, ".")
	if err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(static)))
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		h.add(conn)
		defer func() {
			h.remove(conn)
			_ = conn.Close()
		}()
		_ = h.send(conn, match.SnapshotJSON())
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var msg clientMsg
			if json.Unmarshal(data, &msg) != nil {
				continue
			}
			switch msg.Type {
			case "select":
				match.SetSlot(msg.Slot, msg.Kind)
			case "start":
				match.Start()
			case "pause":
				match.Pause()
			case "end":
				match.End()
			}
			h.broadcast(match.SnapshotJSON())
		}
	})

	addr := ":8080"
	log.Printf("open http://127.0.0.1%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
