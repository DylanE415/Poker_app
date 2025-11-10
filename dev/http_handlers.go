package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	// when func returns, close body http req .body is a stream
	defer r.Body.Close()
	ip := s.getIpAddress(r)
	ok, wait := s.canAttemptLogin(ip)
	if !ok {
		w.Header().Set("Retry-After", strconv.Itoa(int(wait.Seconds())+1))
		http.Error(w, "too many login attempts, try again in "+wait.String(), http.StatusTooManyRequests)
		return
	}

	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.recordLoginFail(ip)
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}

	uname := strings.ToLower(strings.TrimSpace(req.Username))
	u, ok := s.usersByUsername[uname]
	if !ok || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)) != nil {
		s.recordLoginFail(ip)
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	s.recordLoginSuccess(ip)
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
	reply := make(chan error, 1)
	room.commandChan <- Command{Kind: "join", PlayerID: u.ID, PlayerName: u.Username, stack: b.Stack, reply: reply}
	//now we awaiteither the reply or a timeout
	select {
	case err := <-reply:
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent) // done
	case <-time.After(2 * time.Second):
		http.Error(w, "timeout waiting for join ack", http.StatusGatewayTimeout)
		return
	case <-r.Context().Done():
		http.Error(w, "request canceled", http.StatusGatewayTimeout)
		return
	}

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
	roomID := req.URL.Query().Get("room")
	if roomID == "" {
		http.Error(w, "missing room", http.StatusBadRequest)
		return
	}
	room := s.getRoom(roomID)

	if room == nil {
		http.Error(w, "no such room", http.StatusBadRequest)
		return
	}

	// user lookup
	u, ok := s.usersByID[uid]
	if !ok {
		http.Error(w, "no such user", http.StatusBadRequest)
		return
	}

	//send leave command
	reply := make(chan error, 1)
	room.commandChan <- Command{Kind: "leave", PlayerID: u.ID, PlayerName: u.Username, reply: reply}

	//now we awaiteither the reply or a timeout
	select {
	case err := <-reply:
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent) // done
	case <-time.After(2 * time.Second):
		http.Error(w, "timeout waiting for leave ack", http.StatusGatewayTimeout)
		return
	case <-req.Context().Done():
		http.Error(w, "request canceled", http.StatusGatewayTimeout)
		return
	}

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
func (s *Server) getStateHandler(w http.ResponseWriter, r *http.Request) {
	//TODO var hand = s.getRoom(r.URL.Query().Get("room")).currentHand
	if r.Method != http.MethodGet {
		http.Error(w, "use GET", http.StatusMethodNotAllowed)
		return
	}

	//make sure valid cookie
	uid, ok := s.userIDFromRequest(r)
	if !ok {
		http.Error(w, "auth required", http.StatusUnauthorized)
		return
	}
	_, err := room_request_to_int(r.URL.Query().Get("room"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// user lookup
	u, ok := s.usersByID[uid]
	if !ok {
		http.Error(w, "no such user", http.StatusBadRequest)
		return
	}

	room := s.getRoom(r.URL.Query().Get("room"))
	reply := make(chan error, 1)
	stateReply := make(chan roomState, 1)
	room.commandChan <- Command{Kind: "getState", PlayerID: u.ID, PlayerName: u.Username, reply: reply, stateReply: stateReply}
	select {
	case err := <-reply:
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		} else {
			//await state reply
			select {
			case state := <-stateReply:
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(state)
			case <-time.After(2 * time.Second):
				http.Error(w, "timeout waiting for getState ack", http.StatusGatewayTimeout)
				return
			case <-r.Context().Done():
				http.Error(w, "request canceled", http.StatusGatewayTimeout)
				return

			}
		}
	case <-time.After(2 * time.Second):
		http.Error(w, "timeout waiting for getState ack", http.StatusGatewayTimeout)
		return
	case <-r.Context().Done():
		http.Error(w, "request canceled", http.StatusGatewayTimeout)
		return
	}

}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////////////////////////////////////////
// exaample request
/*POST http://localhost:8080/action?room=1
//actions: call, raise, fold, check
Content-Type: application/json
{
  "action": "fold"
  "amount": 0
}
*/

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
	// get room
	room := s.getRoom(fmt.Sprint(roomID))
	if room == nil {
		http.Error(w, "no such room", http.StatusBadRequest)
		return
	}

	// decode action from body
	var a Action
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	a.PlayerID = uid
	// make sure cookie matches
	if a.PlayerID != uid {
		http.Error(w, "auth required", http.StatusUnauthorized)
		return
	}

	if a.Amount < 0 {
		http.Error(w, "amount must be greater than 0", http.StatusBadRequest)
		return
	}

	reply := make(chan error, 1)
	command := Command{
		Kind:       "action",
		ActionType: a.Action,
		PlayerID:   a.PlayerID,
		reply:      reply,
		actionAmt:  a.Amount,
	}
	//sned command to room
	room.commandChan <- command

	//wait for timeout,reply, or the client to cancel
	select {
	case err := <-reply:
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent) // done
	case <-time.After(2 * time.Second):
		http.Error(w, "timeout waiting for action ack", http.StatusGatewayTimeout)
		return
	case <-r.Context().Done():
		http.Error(w, "request canceled", http.StatusGatewayTimeout)
		return
	}
}

// emote handler
/*
POST http://localhost:8080/emote?room=1
Content-Type: application/json

{
  "emoteType": "angle"
}
*/
func (s *Server) setEmoteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "use POST", http.StatusMethodNotAllowed)
		return
	}

	uid, ok := s.userIDFromRequest(r)
	if !ok {
		http.Error(w, "auth required", http.StatusUnauthorized)
		return
	}

	roomID, err := room_request_to_int(r.URL.Query().Get("room"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	room := s.getRoom(fmt.Sprint(roomID))
	if room == nil {
		http.Error(w, "no such room", http.StatusBadRequest)
		return
	}

	// 👇 simple JSON decode into your existing Action struct
	var a Action
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if a.EmoteType == "" {
		http.Error(w, "missing emoteType", http.StatusBadRequest)
		return
	}

	reply := make(chan error, 1)
	room.commandChan <- Command{
		Kind:      "emote",
		PlayerID:  uid,
		reply:     reply,
		EmoteType: a.EmoteType,
	}

	select {
	case err := <-reply:
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case <-time.After(2 * time.Second):
		http.Error(w, "timeout waiting for emote ack", http.StatusGatewayTimeout)
	case <-r.Context().Done():
		http.Error(w, "request canceled", http.StatusGatewayTimeout)
	}
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////
// /////////////////////////////////////////////////////////////////////////////////////////////////////////////
// need id of player
// format is :8080/sitInOrOut?room=1&sitIn=true
func (s *Server) sitInOrOutHandler(w http.ResponseWriter, r *http.Request) {
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

	// user lookup
	_, ok = s.usersByID[uid]
	if !ok {
		http.Error(w, "no such user", http.StatusBadRequest)
		return
	}

	//parse the sitIn or sitOut
	sitIn, err := strconv.ParseBool(r.URL.Query().Get("sitIn"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	//send sitIn or sitOut command to room
	room := s.getRoom(fmt.Sprint(roomID))
	reply := make(chan error, 1)
	if sitIn {
		Command := Command{Kind: "sitIn", PlayerID: uid, reply: reply}
		room.commandChan <- Command
	} else {
		Command := Command{Kind: "sitOut", PlayerID: uid, reply: reply}
		room.commandChan <- Command
	}

	//wait for timeout,reply, or the client to cancel
	select {
	case err := <-reply:
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent) // done
	case <-time.After(2 * time.Second):
		http.Error(w, "timeout waiting for action ack", http.StatusGatewayTimeout)
		return
	case <-r.Context().Done():
		http.Error(w, "request canceled", http.StatusGatewayTimeout)
		return
	}

}

// format is :8080/showHand?room=1&showHand=true
func (s *Server) showHandHandler(w http.ResponseWriter, r *http.Request) {
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

	// user lookup
	_, ok = s.usersByID[uid]
	if !ok {
		http.Error(w, "no such user", http.StatusBadRequest)
		return
	}

	//parse either show or hide
	showHand, err := strconv.ParseBool(r.URL.Query().Get("showHand"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	//send  command to room
	room := s.getRoom(fmt.Sprint(roomID))
	reply := make(chan error, 1)
	if showHand {
		Command := Command{Kind: "showHand", PlayerID: uid, reply: reply}
		room.commandChan <- Command
	} else {
		Command := Command{Kind: "hideHand", PlayerID: uid, reply: reply}
		room.commandChan <- Command
	}

	//wait for timeout,reply, or the client to cancel
	select {
	case err := <-reply:
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent) // done
	case <-time.After(2 * time.Second):
		http.Error(w, "timeout waiting for action ack", http.StatusGatewayTimeout)
		return
	case <-r.Context().Done():
		http.Error(w, "request canceled", http.StatusGatewayTimeout)
		return
	}

}
