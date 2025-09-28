package main

import (
	"fmt"
)

type Command struct {
	Kind   string // "join" or "leave"
	Player Player
}

type Room struct {
	id       int
	cmdChan  chan Command
	players  []Player
	minStack float64
	maxStack float64
}

// has a command buffer of 16 commands
func newRoom(id int, minStack float64, maxStack float64) *Room {
	return &Room{
		id:       id,
		cmdChan:  make(chan Command, 16),
		players:  make([]Player, 0),
		minStack: minStack,
		maxStack: maxStack,
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
				fmt.Printf("- %s (%s) stack: %.2f\n", pl.Name, pl.ID, pl.Stack)
			}
		}
		fmt.Println()
	}
}
