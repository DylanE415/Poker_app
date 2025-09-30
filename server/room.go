package main

import (
	"fmt"
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
	currentHand        Hand
	previousHand       Hand
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

// function operates on a pointer receiver to actually change the room in memory, r Room would make a copy
func (r *Room) run() {
	for {
		// take a command from the join and leave channel
		cmd := <-r.joinAndLeaveChan
		switch cmd.Kind {
		case "join":
			if !r.has(cmd.Player.ID) {
				r.players = append(r.players, cmd.Player)
			} else {
				fmt.Printf("Player %s already in room %d\n", cmd.Player.ID, r.id)
			}
		case "leave":
			//making new array without the player
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
				fmt.Printf("- %s (%s) stack: %.2f\n", pl.Name, pl.ID, pl.Stack)
			}
		}
		fmt.Println()

	}
}
