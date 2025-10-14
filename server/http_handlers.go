package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

/* === HTTP server handlers === */
///////////////////////////////////////////////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////////////////////////////////////////

// login handler
/*
a POST request with the follwing body:

{
	"username": "username",
	"password": "password"
}

*/
type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "use POST", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}

	uname := strings.ToLower(strings.TrimSpace(req.Username))
	u, ok := s.usersByUsername[uname]
	if !ok || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)) != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	// create session + set cookie
	if err := s.createSessionFor(w, u.ID); err != nil {
		http.Error(w, "could not create session", http.StatusInternalServerError)
		return
	}

	// success (no body needed)
	w.WriteHeader(http.StatusNoContent)
}

/*
for users to join a room, if valid sends a command to the command channel of that room

	request would have the form, and cookie with id and session token

	curl -X POST "http://localhost:8080/join?room=1" \
	  -H "Content-Type: application/json" \
	  -d '{"stack":100}'
*/
type joinBody struct {
	Stack float64 `json:"stack"`
}

func (s *Server) joinHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "use POST", http.StatusMethodNotAllowed)
		return
	}
	// auth
	uid, ok := s.userIDFromRequest(r)
	if !ok {
		http.Error(w, "auth required", http.StatusUnauthorized)
		return
	}

	// room (query only; don’t mix body + query)
	roomID := r.URL.Query().Get("room")
	if roomID == "" {
		http.Error(w, "missing room", http.StatusBadRequest)
		return
	}
	room := s.getRoom(roomID)
	if room == nil {
		http.Error(w, "no such room", http.StatusBadRequest)
		return
	}

	// decode body safely
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB
	var b joinBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil || b.Stack < 0 {
		http.Error(w, "bad json (need stack>=0)", http.StatusBadRequest)
		return
	}

	// user lookup
	u, ok := s.usersByID[uid]
	if !ok {
		http.Error(w, "no such user", http.StatusBadRequest)
		return
	}

	p := newPlayer(u.ID, u.Username, b.Stack)

	room.joinAndLeaveChan <- Command{Kind: "join", Player: p}
	w.WriteHeader(http.StatusNoContent)

}

// for users to leave a room, if valid sends a command to the command channel of that room, same format as join
// body has room id: http://localhost:8080/leave?room=1

func (s *Server) leaveHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "use POST", http.StatusMethodNotAllowed)
		return
	}
	uid, ok := s.userIDFromRequest(req)
	if !ok {
		http.Error(w, "auth required", http.StatusUnauthorized)
		return
	}
	//parse which room from body
	var rmNumber int
	if err := json.NewDecoder(req.Body).Decode(&rmNumber); err != nil || rmNumber < 0 {
		http.Error(w, "bad json (need room)", http.StatusBadRequest)
		return
	}

	rm := s.getRoom(req.URL.Query().Get("room"))

	//get player pointer
	p := getPlayerFromID(uid, rm.players)

	rm.joinAndLeaveChan <- Command{Kind: "leave", Player: p}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("left\n"))
}

// simple get request to return players in room for display purposes
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////
type PlayersResponse struct {
	Count   int       `json:"count"`
	Players []*Player `json:"players"`
	Room    int       `json:"room"`
}

// a request would have the form http://localhost:8080/players?room=1
func (s *Server) playersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "use GET", http.StatusMethodNotAllowed)
		return
	}

	//make sure valid cookie
	_, ok := s.userIDFromRequest(r)
	if !ok {
		http.Error(w, "auth required", http.StatusUnauthorized)
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

	//make sure valid cookie
	_, ok := s.userIDFromRequest(r)
	if !ok {
		http.Error(w, "auth required", http.StatusUnauthorized)
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
		Room              int       `json:"room"`
		ActionPlayerIndex int       `json:"actionPlayerIndex"`
		Players           []*Player `json:"players"`
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
	//make sure valid cookie
	uid, ok := s.userIDFromRequest(r)
	if !ok {
		http.Error(w, "auth required", http.StatusUnauthorized)
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
	// make sure cookie matches
	if a.PlayerID != uid {
		http.Error(w, "auth required", http.StatusUnauthorized)
		return
	}

	// find player in hand
	idx := FindPlayerIndexInHand(h, a.PlayerID)
	if idx < 0 {
		http.Error(w, "unknown player", http.StatusBadRequest)
		return
	}

	//check if valid action
	if !contains(h.avaliableActions, a.Action) {
		http.Error(w, "invalid action", http.StatusBadRequest)
		return
	}

	// enqueue latest action into channel
	p := h.Players[idx]

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

	//make sure valid cookie
	_, ok := s.userIDFromRequest(r)
	if !ok {
		http.Error(w, "auth required", http.StatusUnauthorized)
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
	if getPlayerFromID(r.URL.Query().Get("playerId"), rm.players) == nil {
		http.Error(w, "unknown player", http.StatusBadRequest)
		return
	}
	// check if player is in a hand, if they are they can not sit in or out
	h := rm.currentHand
	p := rm.players[FindPlayerIndexInRoom(rm, r.URL.Query().Get("playerId"))]

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
