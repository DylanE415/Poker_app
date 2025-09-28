package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

/* === HTTP server handlers === */

// the server containing all rooms
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

/*
for users to join a room, if valid sends a command to the command channel of that room

	request would have the form

	curl -X POST "http://localhost:8080/join?room=1" \
	  -H "Content-Type: application/json" \
	  -d '{"id":"1234","name":"Alice","stack":100}'
*/
func (s *Server) joinHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "use POST", http.StatusMethodNotAllowed)
		return
	}

	// parse request body
	var p Player
	if err := json.NewDecoder(req.Body).Decode(&p); err != nil || p.ID == "" || p.Name == "" || p.Stack <= 0 {
		http.Error(w, "bad json (need id, name, stack)", http.StatusBadRequest)
		return
	}
	// has to be a valid room id
	roomID, err := room_request_to_int(req.URL.Query().Get("room"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	rm := s.getRoom(fmt.Sprint(roomID))

	//check if id is an int
	if _, err := strconv.Atoi(p.ID); err != nil {
		http.Error(w, "id must be an int", http.StatusBadRequest)
		return
	}

	// check if player already exists in that room
	if rm.has(p.ID) {
		http.Error(w, "player already in room", http.StatusBadRequest)
		return
	}

	//stack must be positive and at least minStack and not greater than maxStack
	print("stack: ", p.Stack, "minStack: ", p.Stack, rm.minStack)
	if p.Stack < rm.minStack || p.Stack > rm.maxStack {
		http.Error(w, fmt.Sprintf("stack must be within %f and %f", rm.minStack, rm.maxStack), http.StatusBadRequest)
		return
	}

	// add player
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

// simple get request to return players in room for display purposes

type PlayersResponse struct {
	Count   int      `json:"count"`
	Players []Player `json:"players"`
	Room    int      `json:"room"`
}

// a request would have the form http://localhost:8080/players?room=1
func (s *Server) playersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "use GET", http.StatusMethodNotAllowed)
		return
	}

	// parse and validate room id
	roomID, err := room_request_to_int(r.URL.Query().Get("room"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	rm := s.getRoom(r.URL.Query().Get("room"))

	resp := PlayersResponse{
		Count:   len(rm.players),
		Players: rm.players,
		Room:    roomID,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
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
