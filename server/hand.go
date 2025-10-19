package main

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"
)

type Action struct {
	PlayerID string  `json:"playerId"`
	Action   string  `json:"action"` // "raise", "call", "fold", "check"
	Amount   float64 `json:"amount"`
}

type Hand struct {
	//need to be a reference to rooms players
	Players           []*Player
	actionPlayerIndex int
	deck              []Card
	currentState      string // "pre-flop", "flop", "turn", "river", "showdown", "over"
	board             []Card
	pot               float64
	avaliableActions  []string // "raise", "call", "fold", "check" (changes based on state)
	raiseAmount       float64
	currentBet        float64
	smallBlindIndex   int
	//for locking hand to prevent race conditions
	lock sync.Mutex
}

func shuffleDeck(deck []Card) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := len(deck) - 1; i > 0; i-- {
		j := r.Intn(i + 1)
		deck[i], deck[j] = deck[j], deck[i]
	}
}

func checkPlayerCanAct(H *Hand, p *Player) bool {
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

// take action from channel and do it (mutates H via pointer), returns if action was valid
func handleAction(H *Hand, action Action) bool {
	// if action cannot be done, return
	if !contains(H.avaliableActions, action.Action) {
		print("invalid action\n")
		return false
	}

	switch action.Action {
	case "check":
		H.Players[H.actionPlayerIndex].canAct = false
		return true
	case "raise":
		//must raise by double the previous raise

		if action.Amount > H.Players[H.actionPlayerIndex].Stack {
			action.Amount = H.Players[H.actionPlayerIndex].Stack
		} else if action.Amount < (H.raiseAmount) {
			print("cannot raise by less than current raise amount\n")
			return false
		}
		//current bet
		newBet := action.Amount + H.currentBet
		H.Players[H.actionPlayerIndex].Stack -= newBet
		H.pot += newBet
		H.avaliableActions = []string{"call", "fold", "raise"}

		H.raiseAmount = action.Amount
		//new current bet everyone has to match
		H.currentBet = newBet
		//chips in front of player
		H.Players[H.actionPlayerIndex].currentBet = newBet

		// everyone still in hand can act again execpt raiser
		for i := range H.Players {
			H.Players[i].canAct = true
		}
		H.Players[H.actionPlayerIndex].canAct = false

		return true

	case "call":
		if action.Amount > H.Players[H.actionPlayerIndex].Stack {
			action.Amount = H.Players[H.actionPlayerIndex].Stack

		}
		// amount needed to call is what the most recent raiser bet(current bet in hand) - what the player has in front of them
		amountToCall := H.currentBet - H.Players[H.actionPlayerIndex].currentBet
		//update player stack/pot and what the current player has in front of them
		H.Players[H.actionPlayerIndex].Stack -= amountToCall
		H.pot += amountToCall
		H.Players[H.actionPlayerIndex].currentBet = amountToCall + H.Players[H.actionPlayerIndex].currentBet
		H.Players[H.actionPlayerIndex].canAct = false
		return true

	case "fold":
		for i, p := range H.Players {
			if p.ID == action.PlayerID {
				H.Players = append(H.Players[:i], H.Players[i+1:]...)
				return true
			}
		}
	}

	return false
}

func newHand(players []*Player, smallBlindPosition int) *Hand {
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
		smallBlindIndex:   smallBlindPosition,
		deck:              deck,
		currentState:      "pre-flop",
		pot:               0,
		avaliableActions:  []string{"raise", "fold", "check"},
	}
}

func streetLoop(h *Hand) {
	h.actionPlayerIndex = h.smallBlindIndex
	//if pre flop small blind raises
	if h.currentState == "pre-flop" {
		print("small blind is: ", h.Players[h.actionPlayerIndex].ID, "\n")
		handleAction(h, Action{PlayerID: h.Players[h.actionPlayerIndex].ID, Action: "raise", Amount: 1})
		//blind can still act after raising
		h.actionPlayerIndex = (h.actionPlayerIndex + 1) % len(h.Players)
	}
	//at start everyone can act
	for i := range h.Players {
		h.Players[i].canAct = true
	}
	for {

		//check if everyone called the small blind, then set actions to either check/raise/fold for small blind
		if h.currentBet == 1 && h.actionPlayerIndex == h.smallBlindIndex && h.Players[h.actionPlayerIndex].currentBet == 1 {
			h.avaliableActions = []string{"check", "raise", "fold"}
		}
		// if only one player left, street over
		if len(h.Players) == 1 {
			break
		}

		acting := nextEligible(h, h.actionPlayerIndex)
		if acting == -1 {
			break // no one else can act -> street over
		}

		h.actionPlayerIndex = acting
		cur := h.Players[acting]
		println("player:", cur.ID, "is acting")
		fmt.Printf("can do: %s\n", strings.Join(h.avaliableActions, ", "))

		// Wait on THIS player until valid action or timeout
		timer := time.NewTimer(30 * time.Second)
		for {
			var act Action

			select {
			case act = <-cur.pendingAction:
				// Ignore messages not from this player or not currently allowed
				if act.PlayerID != cur.ID || !contains(h.avaliableActions, act.Action) {
					continue
				}
				//  if valid -> stop waiting
				if handleAction(h, act) {
					if !timer.Stop() {
						<-timer.C
					}
					goto nextActor
				}
				// invalid -> keep waiting; timer continues

			case <-timer.C:
				// Timeout -> default move for THIS player
				if contains(h.avaliableActions, "check") {
					handleAction(h, Action{PlayerID: cur.ID, Action: "check"})
				} else {
					handleAction(h, Action{PlayerID: cur.ID, Action: "fold"})
				}
				goto nextActor
			}
		}

	nextActor:
		fmt.Printf("pot: %.2f\n", h.pot)
		// Advance seat (slice may have shrunk on fold; modulo keeps us in range)
		if len(h.Players) == 0 {
			break
		}
		h.actionPlayerIndex %= len(h.Players)
		h.actionPlayerIndex = (h.actionPlayerIndex + 1) % len(h.Players)
	}

	h.avaliableActions = []string{"raise", "check", "fold"}
	h.raiseAmount = 0
}

func (h *Hand) run() {

	//clear player cards
	for i := range h.Players {
		h.Players[i].Hand = []Card{}
	}
	h.board = []Card{}
	h.pot = 0
	//deal players 2 cards, 1 card at a time
	for i := 0; i < 2; i++ {
		for j := range h.Players {
			h.Players[j].Hand = append(h.Players[j].Hand, h.deck[0])
			h.deck = h.deck[1:]
		}
	}

	println("players have cards:")
	for _, p := range h.Players {
		print(p.Name, ": ")
		for _, c := range p.Hand {
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
	print(len(winningHands), " winners\n")

	if len(winningHands) > 1 {
		println("chopping")
		for i := range winningHands {
			println("player ", winningHands[i].playerId, " wins with ", winningHands[i].hand.Type)
			tmpIndex := FindPlayerIndex(h, winningHands[i].playerId)
			h.Players[tmpIndex].Stack += h.pot / float64(len(winningHands))
		}

	} else {
		println("player ", winningHands[0].playerId, " wins with ", winningHands[0].hand.Type)
		tmpIndex := FindPlayerIndex(h, winningHands[0].playerId)
		h.Players[tmpIndex].Stack += h.pot
	}

	// take players with 0 stack out of hand
	tmp := h.Players[:0] // reuse capacity
	for _, p := range h.Players {
		if p.Stack > 0 { // consider <= 0 if using floats
			tmp = append(tmp, p)
		}
		if p.Stack <= 0 {
			p.sittingOut = true
		}
	}
	h.Players = tmp
}
