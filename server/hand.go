package main

import (
	"math/rand"
	"time"
)

type Action struct {
	PlayerID string  `json:"id"`
	Action   string  `json:"action"` // "raise", "call", "fold", "check"
	Amount   float64 `json:"amount"`
}

type Hand struct {
	Players           []Player
	actionPlayerIndex int
	deck              []Card
	currentState      string // "pre-flop", "flop", "turn", "river"
	board             []Card
	pot               float64
	avaliableActions  []string // "raise", "call", "fold", "check"( will change based on player actions)
}

func shuffleDeck(deck []Card) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := len(deck) - 1; i > 0; i-- {
		j := r.Intn(i + 1)
		deck[i], deck[j] = deck[j], deck[i]
	}
}

func checkPlayerCanAct(H *Hand, p Player) bool {
	if p.Stack > 0 && p.canAct {
		return true
	}
	return false
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

// take action from channel and do it
func handleAction(H *Hand, action Action) {
	// if action cannot be done, return
	if !contains(H.avaliableActions, action.Action) {
		return
	}

	//RAISE
	if action.Action == "raise" {
		H.Players[H.actionPlayerIndex].Stack -= action.Amount
		H.avaliableActions = []string{"call", "fold", "raise"}
		H.pot += action.Amount
		H.Players[H.actionPlayerIndex].canAct = false
		// since there is a raise every player still in hand can now act again
		for _, p := range H.Players {
			p.canAct = true
		}

		//CALL
	} else if action.Action == "call" {
		H.Players[H.actionPlayerIndex].Stack -= action.Amount
		H.pot += action.Amount
		H.Players[H.actionPlayerIndex].canAct = false

		//FOLD
	} else if action.Action == "fold" {
		for i, p := range H.Players {
			//remove player from hand
			if p.ID == action.PlayerID {
				// remove element i(:i is everything before i, :i+1 is everything after i)
				H.Players = append(H.Players[:i], H.Players[i+1:]...)
				break
			}
		}
	}

}

func newHand(players []Player, smallBlindPosition int) Hand {

	suits := []string{"Spades", "Hearts", "Diamonds", "Clubs"}
	ranks := []string{"Ace", "2", "3", "4", "5", "6", "7", "8", "9", "10", "Jack", "Queen", "King"}
	deck := make([]Card, 0)

	for _, suit := range suits {
		for _, rank := range ranks {
			deck = append(deck, Card{Suit: suit, Rank: rank})
		}
	}

	shuffleDeck(deck)

	return Hand{
		Players:           players,
		actionPlayerIndex: smallBlindPosition,
		deck:              deck,
		currentState:      "pre-flop",
		board:             make([]Card, 0),
		pot:               0,
		avaliableActions:  []string{"raise", "fold", "check"},
	}
}

func (h *Hand) run() {

	//pre flop
	//action loop

	if h.currentState == "pre-flop" {
		for {
			//preflop first player is the person after blinds
			actingPlayerIndex := nextEligible(h, h.actionPlayerIndex)
			if actingPlayerIndex == -1 {
				break
			}
			// it's this player's turn
			h.actionPlayerIndex = actingPlayerIndex
			cur := &h.Players[actingPlayerIndex]

			var currentPlayerAction Action
			for {
				select {
				case action := <-cur.pendingAction:
					// Handle the action
					// first check if action is valid
					currentPlayerAction = action
					if contains(h.avaliableActions, action.Action) {
						handleAction(h, action)
						break
					} else {
						// invaid action await new response in channel
					}
				default:
					// wait for action to come in channel
					time.Sleep(1 * time.Second)
					if currentPlayerAction != (Action{}) {
						// Do something with the action
						// For example, you could call a method on the current player
						currentPlayerAction.Player.CanAct = true
					}
				}
			}

			// TODO: wait/apply real action (fold/call/raise/all-in) here.

			// For now, just mark them as having acted this round:
			cur.canAct = false

			// Advance pointer so the next search starts from the next seat
			h.actionPlayerIndex = (h.actionPlayerIndex + 1) % len(h.Players)
		}

	}

	if h.currentState == "flop" {
		h.deck = h.deck[1:]                      // burn
		h.board = append(h.board, h.deck[:3]...) // flop
		h.deck = h.deck[3:]

		// ---- FLOP BETTING LOOP ----

		for {

			actingPlayerIndex := nextEligible(h, h.actionPlayerIndex)
			if actingPlayerIndex == -1 {
				// no one left to act on the flop
				break
			}
			// it's this player's turn
			h.actionPlayerIndex = actingPlayerIndex
			cur := &h.Players[actingPlayerIndex]

			// TODO: wait/apply real action (fold/call/raise/all-in) here.

			// For now, just mark them as having acted this round:
			cur.canAct = false

			// Advance pointer so the next search starts from the next seat
			h.actionPlayerIndex = (h.actionPlayerIndex + 1) % len(h.Players)
		}

	}
}
