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

///////////////////////////////////////////////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////////////////////////////////////////
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
	var tmp Player
	if err := json.NewDecoder(req.Body).Decode(&tmp); err != nil || tmp.ID == "" || tmp.Name == "" || tmp.Stack <= 0 {
		http.Error(w, "bad json (need id, name, stack)", http.StatusBadRequest)
		return
	}
	p := newPlayer(tmp.ID, tmp.Name, tmp.Stack)
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
		http.Error(w, "player id already in room", http.StatusBadRequest)
		return
	}
	// check if name is already in that room
	for _, pl := range rm.players {
		if pl.Name == p.Name {
			http.Error(w, "name already in room", http.StatusBadRequest)
			return
		}
	}
	//check if room has less than 9 players
	if len(rm.players) >= 9 {
		http.Error(w, "room is full", http.StatusBadRequest)
		return
	}

	//stack must be positive and at least minStack and not greater than maxStack
	print("stack: ", p.Stack, "minStack: ", p.Stack, rm.minStack)
	if p.Stack < rm.minStack || p.Stack > rm.maxStack {
		http.Error(w, fmt.Sprintf("stack must be within %f and %f", rm.minStack, rm.maxStack), http.StatusBadRequest)
		return
	}

	// add player
	p.canAct = true
	rm.joinAndLeaveChan <- Command{Kind: "join", Player: p}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("joined\n"))
}

// for users to leave a room, if valid sends a command to the command channel of that room, same format as join

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
	rm.joinAndLeaveChan <- Command{Kind: "leave", Player: p}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("left\n"))
}

// simple get request to return players in room for display purposes
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////
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

///////////////////////////////////////////////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////////////////////////////////////////

// for return state of room to client
// GET /state?room=1  -> { room, actionPlayerIndex, players }
func (s *Server) stateHandler(w http.ResponseWriter, r *http.Request) {
	//TODO var hand = s.getRoom(r.URL.Query().Get("room")).currentHand
	if r.Method != http.MethodGet {
		http.Error(w, "use GET", http.StatusMethodNotAllowed)
		return
	}
	roomID, err := room_request_to_int(r.URL.Query().Get("room"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	rm := s.getRoom(fmt.Sprint(roomID))

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		Room              int      `json:"room"`
		ActionPlayerIndex int      `json:"actionPlayerIndex"`
		Players           []Player `json:"players"`
	}{
		Room: rm.id, ActionPlayerIndex: rm.smallBlindPosition + 1, Players: rm.players,
	})
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////////////////////////////////////////
// exaample request
/*POST http://localhost:8080/action?room=1
Content-Type: application/json
{
  "playerId": "123",
  "action": "fold"
}
*/

// for rmaking the latest action the current action in the channel
func enqueueLatest(ch chan Action, a Action) {
	for {
		select {
		case ch <- a:
			// sent successfully; done
			return
		case <-ch:
			// channel was full; drop the value; and repeat statement to do first case
		}
	}
}
func (s *Server) setActionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "use POST", http.StatusMethodNotAllowed)
		return
	}
	// check valid room
	roomID, err := room_request_to_int(r.URL.Query().Get("room"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	rm := s.getRoom(fmt.Sprint(roomID))
	if rm == nil || rm.currentHand == nil {
		http.Error(w, "no active hand", http.StatusConflict)
		return
	}
	h := rm.currentHand

	// decode body
	var a Action
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil || a.PlayerID == "" || a.Action == "" {
		http.Error(w, "bad json (need playerId, action)", http.StatusBadRequest)
		return
	}
	// find player
	idx := FindPlayerIndexInHand(h, a.PlayerID)
	if idx < 0 {
		http.Error(w, "unknown player", http.StatusBadRequest)
		return
	}

	// enqueue latest action into channel
	p := &h.Players[idx]
	enqueueLatest(p.pendingAction, a)

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("action queued\n"))
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////
// need id of player
// format is :8080/sitInOrOut?room=1&playerId=2&sitIn=true
func (s *Server) sitInOrOutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "use POST", http.StatusMethodNotAllowed)
		return
	}

	// check valid room
	roomID, err := room_request_to_int(r.URL.Query().Get("room"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// check if player is in room
	rm := s.getRoom(fmt.Sprint(roomID))
	if !rm.has(r.URL.Query().Get("playerId")) {
		http.Error(w, "player not in room", http.StatusConflict)
		return
	}
	// check if player is in a hand, if they are they can not sit in or out
	h := rm.currentHand
	p := &rm.players[FindPlayerIndexInRoom(rm, r.URL.Query().Get("playerId"))]

	if h == nil {
		if r.URL.Query().Get("sitIn") == "true" && p.sittingOut {
			p.sittingOut = false
		} else if r.URL.Query().Get("sitIn") == "false" && !p.sittingOut {
			p.sittingOut = true
		} else {
			http.Error(w, "already in that state", http.StatusBadRequest)
			return
		}
	} else if FindPlayerIndexInHand(h, r.URL.Query().Get("playerId")) >= 0 {
		http.Error(w, "player already in hand", http.StatusConflict)
		return
	} else {

		if r.URL.Query().Get("sitIn") == "true" && p.sittingOut {
			p.sittingOut = false
		} else if r.URL.Query().Get("sitIn") == "false" && !p.sittingOut {
			p.sittingOut = true
		} else {
			http.Error(w, "already in that state", http.StatusBadRequest)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK!\n"))

}
