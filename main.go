package main

import (
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
	"xqdj/internal/export"
	"xqdj/internal/sim"
	"xqdj/internal/unit"
	"xqdj/internal/web"

	_ "xqdj/character"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// outbound 是写给一个观众的一条消息。keep 为真时是战报：后面的快照不能把它挤掉。
type outbound struct {
	body []byte
	keep bool
}

// spectator 单独写连接。广播只保留最新一帧，慢的页面不会堵住比赛和其他观众。
type spectator struct {
	conn *websocket.Conn
	send chan outbound
}

type hub struct {
	mu      sync.Mutex
	clients map[*websocket.Conn]*spectator
}

func newHub() *hub {
	return &hub{clients: make(map[*websocket.Conn]*spectator)}
}

func (h *hub) add(c *websocket.Conn) *spectator {
	s := &spectator{conn: c, send: make(chan outbound, 1)}
	h.mu.Lock()
	h.clients[c] = s
	h.mu.Unlock()
	go s.write(h)
	return s
}

func (s *spectator) write(h *hub) {
	for msg := range s.send {
		_ = s.conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
		if err := s.conn.WriteMessage(websocket.TextMessage, msg.body); err != nil {
			_ = s.conn.Close()
			h.remove(s.conn)
			return
		}
	}
}

func (h *hub) remove(c *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	s, ok := h.clients[c]
	if !ok {
		return
	}
	delete(h.clients, c)
	close(s.send)
}

func offer(ch chan outbound, msg outbound) {
	select {
	case ch <- msg:
		return
	default:
	}
	select {
	case old := <-ch:
		if old.keep && !msg.keep {
			ch <- old
			return
		}
	default:
	}
	select {
	case ch <- msg:
	default:
	}
}

func (h *hub) enqueue(s *spectator, msg outbound) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[s.conn]; !ok {
		return
	}
	offer(s.send, msg)
}

func (h *hub) send(s *spectator, msg []byte) {
	h.enqueue(s, outbound{body: msg})
}

func (h *hub) sendKept(s *spectator, msg []byte) {
	h.enqueue(s, outbound{body: msg, keep: true})
}

func (h *hub) broadcast(msg []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	frame := outbound{body: msg}
	for _, s := range h.clients {
		offer(s.send, frame)
	}
}

type clientMsg struct {
	Type  string `json:"type"`
	ID    string `json:"id"`
	Slot  int    `json:"slot"`
	Index *int   `json:"index"`
	Kind  string `json:"kind"`
	Field string `json:"field"`
	Team  *bool  `json:"team"`
	Count int    `json:"count"`
}

func main() {
	match := sim.NewMatch()
	h := newHub()
	halt := make(chan struct{})
	go match.Loop(h.broadcast, halt)

	addr := os.Getenv("XQDJ_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	base := "http://" + addr
	if strings.HasPrefix(addr, ":") {
		base = "http://127.0.0.1" + addr
	}

	static, err := fs.Sub(web.FS, ".")
	if err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/team", func(w http.ResponseWriter, r *http.Request) {
		b, err := web.FS.ReadFile("team.html")
		if err != nil {
			http.Error(w, "team page missing", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(b)
	})
	mux.Handle("/", http.FileServer(http.FS(static)))
	mux.HandleFunc("GET /api/pairing-roster", web.PairingRoster(os.DirFS("character")))
	registerScheduledBattles(mux, match)
	mux.Handle("/ball/", http.StripPrefix("/ball/", http.FileServer(http.FS(unit.BallFS()))))
	exporter := export.NewManager(base, match.Recording)
	exporter.Handle(mux)
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
			switch msg.Type {
			case "select":
				if msg.Index != nil {
					if msg.Kind == "" {
						match.RemoveSeat(msg.Slot, *msg.Index)
					} else {
						match.SetSeat(msg.Slot, *msg.Index, msg.Kind)
					}
				} else {
					match.SetSlot(msg.Slot, msg.Kind)
				}
			case "side":
				match.SetSideLen(msg.Slot, msg.Count)
			case "mode":
				if msg.Team != nil && *msg.Team {
					match.UseTeam()
				} else {
					match.UseDuel()
				}
			case "field":
				match.SetField(msg.Field)
			case "start":
				match.Start()
			case "presentation-end":
				match.FinishOpening(msg.ID)
			case "pause":
				match.Pause()
			case "end":
				match.End()
			case "log":
				recap, trace := match.LogText()
				b, err := json.Marshal(struct {
					Type  string `json:"type"`
					Recap string `json:"recap"`
					Trace string `json:"trace"`
				}{Type: "log", Recap: recap, Trace: trace})
				if err == nil {
					h.sendKept(s, b)
				}
				continue
			}
			h.broadcast(match.SnapshotJSON())
		}
	})

	log.Printf("open http://127.0.0.1%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
