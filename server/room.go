package main

import (
	"fmt"
	"net/http"
	"time"
)

type Command struct {
	Kind       string // "join, leave, sit_out, sit_in"
	PlayerID   string
	PlayerName string
	stack      float64
	reply      chan error //for sending a reply to client
}

type Room struct {
	id                 int
	joinAndLeaveChan   chan Command
	players            []*Player
	minStack           float64
	maxStack           float64
	smallBlindPosition int
	currentHand        *Hand
	previousHand       *Hand
	handDone           chan struct{}
}

// has a command buffer of 16 commands
func newRoom(id int, minStack float64, maxStack float64) *Room {
	return &Room{
		id:                 id,
		joinAndLeaveChan:   make(chan Command, 16),
		players:            make([]*Player, 0),
		minStack:           minStack,
		maxStack:           maxStack,
		smallBlindPosition: 0,
		handDone:           make(chan struct{}, 1),
	}
}

// returns true if playerid is in room
func (r *Room) hasPlayer(id string) bool {
	for _, p := range r.players {
		if p.ID == id {
			return true
		}
	}
	return false
}

func FindPlayerIndexInRoom(r *Room, id string) int {
	for i, p := range r.players {
		if p.ID == id {
			return i
		}
	}
	return -1
}

// assumes: type Room struct { currentHand *Hand; previousHand *Hand; players []Player; smallBlindPosition int }

func (r *Room) startNextHandIfReady() {
	// if an old hand exists and is over, archive it
	if r.currentHand != nil && r.currentHand.currentState == "over" {
		r.previousHand = r.currentHand
		r.currentHand = nil
	}
	// if a hand is still running, don't start a new one
	if r.currentHand != nil {
		return
	}
	// collect players who are NOT sitting out
	eligible := make([]*Player, 0, len(r.players))
	for i := range r.players {
		if !r.players[i].sittingOut {
			// reset per-hand flags
			r.players[i].canAct = true
			// optional: drain any stale pendingAction
			select {
			case <-r.players[i].pendingAction:
			default:
			}
			eligible = append(eligible, r.players[i])
		}
	}
	// need at least 2 players to start a hand
	if len(eligible) < 2 {
		return
	}
	// ensure smallBlindPosition is in range of eligible players
	if r.smallBlindPosition < 0 {
		r.smallBlindPosition = 0
	}
	r.smallBlindPosition %= len(eligible)

	// create the new hand (newHand returns *Hand)
	r.currentHand = newHand(eligible, r.smallBlindPosition)
	// advance blinds for the NEXT hand
	r.smallBlindPosition = (r.smallBlindPosition + 1) % len(eligible)

	// run the hand as a go routine
	go func(h *Hand) {
		h.run()
		h.currentState = "over"
		// notify the room that this hand finished (dont need a value just anything)
		select {
		case r.handDone <- struct{}{}:
		default:
		}
	}(r.currentHand)
}

// for sending a reply to the client
func safeReply(ch chan error, err error) {
	select {
	case ch <- err: // this means that there is a channel to receive
	default:
		// receiver not ready; drop so the room goroutine keeps running
	}
}

func SitInOrOut(r *Room, id string, sitIn bool) {
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
}

// function operates on a pointer receiver to actually change the room in memory, r Room would make a copy
func (r *Room) run() {
	ticker := time.NewTicker(400 * time.Millisecond) // every 400ms the room checks for new joins/leaves also checks if a hand is over
	defer ticker.Stop()                              // stop the ticker when the function returns

	for {
		select {
		case cmd := <-r.joinAndLeaveChan:
			switch cmd.Kind {
			case "join":
				// if the player is not in room
				if !r.hasPlayer(cmd.PlayerID) {
					//make new player and add to room
					newPlayer := newPlayer(cmd.PlayerID, cmd.PlayerName, cmd.stack)
					r.players = append(r.players, newPlayer)
					//send good reply to client
					safeReply(cmd.reply, nil)
				} else {
					//send bad reply to client
					safeReply(cmd.reply, fmt.Errorf("player already in room"))
				}
			case "leave":
				//see if player is in room
				id := cmd.PlayerID
				//if player is in room
				if r.hasPlayer(id) {
					//leave room and send good reply to client
					for i, p := range r.players {
						if p.ID == id {
							// remove player (appends all elements before i and all elements after i)
							r.players = append(r.players[:i], r.players[i+1:]...)
							//send good reply to client
							safeReply(cmd.reply, nil)
							break
						}
					}
				} else {
					//send bad reply to client
					safeReply(cmd.reply, fmt.Errorf("player not in room"))
				}
			default:
				panic("unknown command kind")
			}
			// optional: print roster
			fmt.Println("Players in room", r.id, ":")
			if len(r.players) == 0 {
				fmt.Println("(none)")
			} else {
				for _, pl := range r.players {
					fmt.Printf("- %s (%s) stack: %.2f\n", pl.Name, pl.ID, pl.Stack)
				}
			}
			fmt.Println()
			// After any roster change, we might now be eligible to start a hand:
			r.startNextHandIfReady()

		case <-r.handDone:
			// hand finished; try to start the next one right away
			if r.currentHand != nil {
				r.previousHand = r.currentHand
				r.currentHand = nil
			}
			r.startNextHandIfReady()

		case <-ticker.C:
			// periodic check keeps things moving even without joins/leaves
			r.startNextHandIfReady()
		}
	}
}
