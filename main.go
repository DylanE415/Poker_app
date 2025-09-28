package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
)

/* === Data types === */

type Player struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Stack int    `json:"stack"`
}

type Command struct {
	Kind   string // "join" or "leave"
	Player Player
}

type Room struct {
	id      int
	cmdChan chan Command
	players []Player
}

// has a command buffer of 16 commands
func newRoom(id int) *Room {
	return &Room{
		id:      id,
		cmdChan: make(chan Command, 16),
		players: make([]Player, 0),
	}
}

func (r *Room) has(id string) bool {
	for _, p := range r.players {
		if p.ID == id {
			return true
		}
	}
	return false
}

func (r *Room) run() {
	for {
		// take a command from the channel
		cmd := <-r.cmdChan
		switch cmd.Kind {
		case "join":
			if !r.has(cmd.Player.ID) {
				r.players = append(r.players, cmd.Player)
			} else {
				fmt.Printf("Player %s already in room %d\n", cmd.Player.ID, r.id)
			}
		case "leave":
			id := cmd.Player.ID
			dst := r.players[:0]
			for _, p := range r.players {
				if p.ID != id {
					dst = append(dst, p)
				}
			}
			r.players = dst
		}
		fmt.Println("Players in room", r.id, ":")
		if len(r.players) == 0 {
			fmt.Println("(none)")
		} else {
			for _, pl := range r.players {
				fmt.Printf("- %s (%s) stack: %d\n", pl.Name, pl.ID, pl.Stack)
			}
		}
		fmt.Println()
	}
}

/* === HTTP server handlers === */

type Server struct {
	room1 *Room
	room2 *Room
}

func (s *Server) getRoom(q string) *Room {
	if q == "2" {
		return s.room2
	}
	return s.room1
}

func (s *Server) joinHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "use POST", http.StatusMethodNotAllowed)
		return
	}
	var p Player
	if err := json.NewDecoder(req.Body).Decode(&p); err != nil || p.ID == "" {
		http.Error(w, "bad json (need id, name optional)", http.StatusBadRequest)
		return
	}
	rm := s.getRoom(req.URL.Query().Get("room"))
	rm.cmdChan <- Command{Kind: "join", Player: p}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("joined\n"))
}

func (s *Server) leaveHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "use POST", http.StatusMethodNotAllowed)
		return
	}
	var p Player
	if err := json.NewDecoder(req.Body).Decode(&p); err != nil || p.ID == "" {
		http.Error(w, "bad json (need id)", http.StatusBadRequest)
		return
	}
	rm := s.getRoom(req.URL.Query().Get("room"))
	rm.cmdChan <- Command{Kind: "leave", Player: p}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("left\n"))
}

func (s *Server) playersHandler(w http.ResponseWriter, req *http.Request) {
	rm := s.getRoom(req.URL.Query().Get("room"))
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		Count   int      `json:"count"`
		Players []Player `json:"players"`
		Room    int      `json:"room"`
	}{
		Count:   len(rm.players),
		Players: rm.players,
		Room:    atoiDefault(req.URL.Query().Get("room"), 1),
	})
}

/* wrapper for CORS */

func withCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
}

/* === main === */

func main() {
	s := &Server{
		room1: newRoom(1),
		room2: newRoom(2),
	}
	go s.room1.run()
	go s.room2.run()

	mux := http.NewServeMux()
	mux.HandleFunc("/join", s.joinHandler)
	mux.HandleFunc("/leave", s.leaveHandler)
	mux.HandleFunc("/players", s.playersHandler)

	log.Println("Server on :8080 | POST /join  POST /leave  GET /players  (use ?room=2 for room 2)")
	log.Fatal(http.ListenAndServe(":8080", withCORS(mux)))
}

/* === utils === */

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}
