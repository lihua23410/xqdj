package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"

	"xqdj/internal/sim"
)

func registerScheduledBattles(mux *http.ServeMux, match *sim.Match) {
	writeJSON := func(w http.ResponseWriter, battle sim.ScheduledBattle) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(battle)
	}
	mux.HandleFunc("/api/scheduled-battles/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		id := r.PathValue("id")
		if _, err := hex.DecodeString(id); len(id) != 32 || err != nil {
			http.Error(w, "对局编号无效", http.StatusBadRequest)
			return
		}
		switch r.Method {
		case http.MethodGet:
			battle, ok := match.ScheduledStatus(id)
			if !ok {
				http.Error(w, "找不到本次对局", http.StatusNotFound)
				return
			}
			writeJSON(w, battle)
		case http.MethodPost:
			var request struct {
				Sides [][]string `json:"sides"`
			}
			if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192)).Decode(&request); err != nil {
				http.Error(w, "无法读取对阵名单", http.StatusBadRequest)
				return
			}
			battle, err := match.StartScheduled(id, request.Sides)
			switch {
			case errors.Is(err, sim.ErrBattleInProgress):
				http.Error(w, err.Error(), http.StatusConflict)
			case errors.Is(err, sim.ErrScheduledRoster):
				http.Error(w, err.Error(), http.StatusBadRequest)
			case err != nil:
				http.Error(w, err.Error(), http.StatusInternalServerError)
			default:
				writeJSON(w, battle)
			}
		default:
			w.Header().Set("Allow", "GET, POST")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}
