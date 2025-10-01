package main

import (
	"fmt"
	"time"
)

type Command struct {
	Kind   string // "join, leave, sit_out"
	Player Player
}

type Room struct {
	id                 int
	joinAndLeaveChan   chan Command
	players            []Player
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
		players:            make([]Player, 0),
		minStack:           minStack,
		maxStack:           maxStack,
		smallBlindPosition: 0,
		handDone:           make(chan struct{}, 1),
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
	eligible := make([]Player, 0, len(r.players))
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

// function operates on a pointer receiver to actually change the room in memory, r Room would make a copy
func (r *Room) run() {
	ticker := time.NewTicker(400 * time.Millisecond) // light heartbeat
	defer ticker.Stop()

	for {
		select {
		case cmd := <-r.joinAndLeaveChan:
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
