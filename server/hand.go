package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

type Action struct {
	PlayerID string  `json:"playerId"`
	Action   string  `json:"action"` // "raise", "call", "fold", "check"
	Amount   float64 `json:"amount"`
}

type Hand struct {
	Players           []Player
	actionPlayerIndex int
	deck              []Card
	currentState      string // "pre-flop", "flop", "turn", "river", "showdown", "over"
	board             []Card
	pot               float64
	avaliableActions  []string // "raise", "call", "fold", "check" (changes based on state)
	raiseAmount       float64
}

func shuffleDeck(deck []Card) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := len(deck) - 1; i > 0; i-- {
		j := r.Intn(i + 1)
		deck[i], deck[j] = deck[j], deck[i]
	}
}

func checkPlayerCanAct(H *Hand, p Player) bool {
	return p.Stack > 0 && p.canAct
}

func FindPlayerIndexInHand(H *Hand, id string) int {
	for i, p := range H.Players {
		if p.ID == id {
			return i
		}
	}
	return -1
}

func nextEligible(H *Hand, start int) int {
	n := len(H.Players)
	for step := 0; step < n; step++ {
		i := (start + step) % n
		if checkPlayerCanAct(H, H.Players[i]) {
			return i
		}
	}
	return -1
}

// take action from channel and do it (mutates H via pointer)
func handleAction(H *Hand, action Action) {
	// if action cannot be done, return
	if !contains(H.avaliableActions, action.Action) {
		return
	}

	switch action.Action {
	case "raise":
		//must raise by at least the current raise amount
		if action.Amount < H.raiseAmount {
			print("cannot raise by less than current raise amount\n")
			return
		}
		H.Players[H.actionPlayerIndex].Stack -= action.Amount
		H.pot += action.Amount
		H.avaliableActions = []string{"call", "fold", "raise"}
		H.Players[H.actionPlayerIndex].canAct = false
		H.raiseAmount = action.Amount

		// everyone still in hand can act again
		for i := range H.Players {
			H.Players[i].canAct = true
		}

	case "call":
		H.Players[H.actionPlayerIndex].Stack -= action.Amount
		H.pot += action.Amount
		H.Players[H.actionPlayerIndex].canAct = false

	case "fold":
		for i, p := range H.Players {
			if p.ID == action.PlayerID {
				H.Players = append(H.Players[:i], H.Players[i+1:]...)
				break
			}
		}
	}
}

func newHand(players []Player, smallBlindPosition int) *Hand {
	suits := []string{"S", "H", "D", "C"}
	ranks := []string{"14", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12", "13"}
	deck := make([]Card, 0, 52)

	for _, suit := range suits {
		for _, rank := range ranks {
			deck = append(deck, Card{Suit: suit, Rank: rank})
		}
	}
	shuffleDeck(deck)

	return &Hand{
		Players:           players,
		actionPlayerIndex: smallBlindPosition,
		deck:              deck,
		currentState:      "pre-flop",
		pot:               0,
		avaliableActions:  []string{"raise", "fold", "check"},
	}
}

func streetLoop(h *Hand) {
	for {
		// if everyone but 1 player folded, go to showdown
		if len(h.Players) == 1 {
			break
		}

		actingPlayerIndex := nextEligible(h, h.actionPlayerIndex)
		if actingPlayerIndex == -1 {
			break
		}
		println("player:", h.Players[actingPlayerIndex].ID, "is acting")
		fmt.Printf("can do: %s\n", strings.Join(h.avaliableActions, ", "))

		h.actionPlayerIndex = actingPlayerIndex
		cur := &h.Players[actingPlayerIndex]

		// wait until player's action or timeout (no polling)
		var act Action
		got := false
		timer := time.NewTimer(30 * time.Second)
		select {
		case act = <-cur.pendingAction:
			got = true
		case <-timer.C:
			if contains(h.avaliableActions, "check") {
				act = Action{PlayerID: cur.ID, Action: "check"}
				got = true
			} else {
				act = Action{PlayerID: cur.ID, Action: "fold"}
				got = true
			}
		}
		timer.Stop()

		// if got no action check/fold
		print("player ", cur.ID, " got action: ", act.Action, "\n")
		if got && contains(h.avaliableActions, act.Action) && act.PlayerID == cur.ID {
			handleAction(h, act)
		} else if contains(h.avaliableActions, "check") {
			handleAction(h, Action{PlayerID: cur.ID, Action: "check"})
		} else {
			handleAction(h, Action{PlayerID: cur.ID, Action: "fold"})
		}
		print("pot: ", h.pot, "\n")

		cur.canAct = false
		h.actionPlayerIndex = (h.actionPlayerIndex + 1) % len(h.Players)
	}

	//reset everyone can act
	for i := range h.Players {
		h.Players[i].canAct = true
	}
	//reset previous raise to 0
	h.raiseAmount = 0
}

func (h *Hand) run() {

	//clear player cards
	for i := range h.Players {
		h.Players[i].hand = []Card{}
	}
	h.board = []Card{}
	h.pot = 0
	//deal players 2 cards, 1 card at a time
	for i := 0; i < 2; i++ {
		for j := range h.Players {
			h.Players[j].hand = append(h.Players[j].hand, h.deck[0])
			h.deck = h.deck[1:]
		}
	}

	println("players have cards:")
	for _, p := range h.Players {
		print(p.Name, ": ")
		for _, c := range p.hand {
			print(c.Rank, c.Suit)
			print(", ")
		}
		println()
	}

	// ===== PRE-FLOP =====
	print("pre-flop\n")
	if h.currentState == "pre-flop" {
		streetLoop(h)
		h.currentState = "flop"
	}

	print("Pre-flop done, moving to flop\n")
	// ===== FLOP =====
	if h.currentState == "flop" {
		h.deck = h.deck[1:]                      // burn
		h.board = append(h.board, h.deck[:3]...) // flop
		h.deck = h.deck[3:]

		print("Flop: ")
		for _, c := range h.board {
			print(c.Rank, c.Suit)
			print(", ")
		}

		streetLoop(h)
		h.currentState = "turn"
	}

	print("Flop done, moving to turn\n")
	// ===== TURN =====
	if h.currentState == "turn" {
		h.deck = h.deck[1:]                  // burn
		h.board = append(h.board, h.deck[0]) // turn
		h.deck = h.deck[1:]

		print("Turn: ")
		for _, c := range h.board {
			print(c.Rank, c.Suit)
			print(", ")
		}

		streetLoop(h)
		h.currentState = "river"
	}

	print("Turn done, moving to river\n")
	// ===== RIVER =====
	if h.currentState == "river" {
		h.deck = h.deck[1:]                  // burn
		h.board = append(h.board, h.deck[0]) // river
		h.deck = h.deck[1:]

		print("River: ")
		for _, c := range h.board {
			print(c.Rank, c.Suit)
			print(", ")
		}

		streetLoop(h)

	}

	print("River done, moving to showdown\n")
	//playerhand is struct with player id and best hand
	playerHands := make([]playerHand, len(h.Players))
	for i := range h.Players {
		playerHands[i] = playerHand{playerId: h.Players[i].ID, hand: getPlayerBestHand(h, h.Players[i])}
	}
	// showdown

	winningHands := getShowdownBestHand(playerHands)

	if len(winningHands) > 1 {
		println("chopping")
		for i := range winningHands {
			println("player ", winningHands[i].playerId, " wins with ", winningHands[i].hand.Type)
		}

	} else {
		println("player ", winningHands[0].playerId, " wins with ", winningHands[0].hand.Type)
	}

	WinningID := winningHands[0].playerId

	for i := range h.Players {
		if h.Players[i].ID == WinningID {
			h.Players[i].Stack += h.pot
		}
	}
}
