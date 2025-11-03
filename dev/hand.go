package main

import (
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Action struct {
	PlayerID string  `json:"playerId"`
	Action   string  `json:"action"` // "raise", "call", "fold", "check", "clear"
	Amount   float64 `json:"amount"`
}

type showDownHand struct {
	PlayerName string
	Hand       []Card
}
type Hand struct {
	//all pointers to players
	Room                 *Room
	Players              []*Player
	actionPlayerIndex    int
	deck                 []Card
	currentState         string // "pre-flop", "flop", "turn", "river", "showdown"
	board                []Card
	pot                  float64
	avaliableActions     []string // "raise", "call", "fold", "check" (changes based on state)
	raiseAmount          float64
	currentBet           float64
	smallBlindIndex      int
	smallBlindSize       float64
	smallBlindName       string
	skipToShowdown       bool
	showDownHands        []showDownHand
	showDownMessage      string
	currentActionMessage string
	actionPlayerDeadline time.Time
	//for locking hand to prevent race conditions
	lock sync.Mutex
}

const (
	timeLimit = 45 //seconds

)

// 7 2 off bounty
func isSevenTwoOff(player *Player) bool {
	if (player.Hand[0].Rank == "2" && player.Hand[1].Rank == "7" && (player.Hand[0].Suit != player.Hand[1].Suit)) || (player.Hand[0].Rank == "7" && player.Hand[1].Rank == "2" && (player.Hand[0].Suit != player.Hand[1].Suit)) {
		return true

	}
	return false
}
func (h *Hand) collectSevenTwoBounty(r *Room, p *Player) {
	for i := range r.players {
		if r.players[i].ID == p.ID {
			continue
		} else if r.players[i].Stack < h.smallBlindSize*2 {
			p.Stack += r.players[i].Stack
			r.players[i].Stack = 0
		} else {
			p.Stack += h.smallBlindSize * 2
			r.players[i].Stack -= h.smallBlindSize * 2
		}
	}

}

func shuffleDeck(deck []Card) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := len(deck) - 1; i > 0; i-- {
		j := r.Intn(i + 1)
		deck[i], deck[j] = deck[j], deck[i]
	}
}

func checkPlayerCanAct(p *Player) bool {
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

// remove player at idx and keep SB + action index sane
func removePlayerAt(h *Hand, idx int) {
	n := len(h.Players)
	if idx < 0 || idx >= n {
		return
	}

	// delete
	h.Players = append(h.Players[:idx], h.Players[idx+1:]...)
	n-- // new length

	if n == 0 {
		h.actionPlayerIndex = -1
		h.smallBlindIndex = -1
		h.smallBlindName = ""
		return
	}

	// fix small blind
	if h.smallBlindIndex == idx {
		// SB folded → seat that slid in becomes SB
		h.smallBlindIndex = idx
		if h.smallBlindIndex >= n {
			h.smallBlindIndex = 0
		}
	} else if h.smallBlindIndex > idx {
		h.smallBlindIndex--
	}

	// refresh SB name
	if h.smallBlindIndex >= 0 && h.smallBlindIndex < n {
		h.smallBlindName = h.Players[h.smallBlindIndex].Name
	}

	// fix action player
	if h.actionPlayerIndex == idx {
		// actor folded
		// put action on the LAST seat so the loop's "+1" hops to seat 0 (SB)
		h.actionPlayerIndex = n - 1
	} else if h.actionPlayerIndex > idx {
		h.actionPlayerIndex--
	}
}

func nextEligible(H *Hand, start int) int {

	n := len(H.Players)
	for step := 0; step < n; step++ {
		i := (start + step) % n
		if checkPlayerCanAct(H.Players[i]) {
			return i
		}
	}
	return -1
}

type SidePot struct {
	eligiblePlayerIDs []string
	Amount            float64
}

func buildSidePots(H *Hand) []SidePot {
	totalPot := H.pot
	var sidePots []SidePot
	potCommitmentAccountedFor := 0.0
	//first sort players by pot commitment
	players := make([]*Player, len(H.Players))
	copy(players, H.Players)

	// ascending order(lowest to highest)
	sort.Slice(players, func(i, j int) bool {
		return players[i].potCommitment < players[j].potCommitment
	})

	//each side pot is the lowest pot commitment player and everyone else(then next side pot remove the lowest player and so on)
	for i := range players {
		newPot := SidePot{eligiblePlayerIDs: []string{players[i].ID}, Amount: players[i].potCommitment}
		for j := i + 1; j < len(players); j++ {
			newPot.eligiblePlayerIDs = append(newPot.eligiblePlayerIDs, players[j].ID)
		}
		newPot.Amount = (players[i].potCommitment - potCommitmentAccountedFor) * float64(len(newPot.eligiblePlayerIDs))
		potCommitmentAccountedFor += players[i].potCommitment - potCommitmentAccountedFor
		//if side pot amount is greater than total pot, reduce to total pot
		if newPot.Amount > totalPot {
			newPot.Amount = totalPot
		}
		//if its the final pot and total pot is not 0, increase to total pot
		if len(newPot.eligiblePlayerIDs) == 1 && totalPot != 0 {
			newPot.Amount = totalPot
		}
		totalPot -= newPot.Amount
		sidePots = append(sidePots, newPot)

	}
	return sidePots

}

// if only 1 or 0 players can act, skip to showdown
func skipToShowdown(H *Hand) bool {
	playersLeft := 0
	for i := range H.Players {
		if checkPlayerCanAct(H.Players[i]) {
			playersLeft++
		}
	}
	if playersLeft <= 1 {
		return true
	}
	return false
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
		H.currentActionMessage = H.Players[H.actionPlayerIndex].Name + " checked"
		return true
	case "raise":
		p := H.Players[H.actionPlayerIndex]

		// how much do I have to put in just to call right now?
		toCall := H.currentBet - p.currentBet
		if toCall < 0 {
			toCall = 0
		}

		//  if can't even call, this is just an all-in call, line does NOT move
		if p.Stack <= toCall {
			contrib := p.Stack
			p.Stack = 0
			p.currentBet += contrib
			p.potCommitment += contrib
			H.pot += contrib

			p.canAct = false
			H.currentActionMessage = p.Name + " called all in for " + strconv.FormatFloat(contrib, 'f', -1, 64)
			return true
		}

		// player is asking to "raise BY" this amount
		desiredRaise := action.Amount

		// how many chips I have LEFT AFTER calling
		canRaiseWith := p.Stack - toCall

		// if I can't afford the raise I'm asking for -> short all-in
		//    in real NLHE this does NOT change currentBet or reopen
		if desiredRaise > canRaiseWith {
			// just put everything in, but do NOT move table line
			totalPut := p.Stack
			p.potCommitment += totalPut
			p.Stack = 0
			p.currentBet += totalPut
			p.potCommitment += totalPut
			H.pot += totalPut

			p.canAct = false
			H.currentActionMessage = p.Name + " went all in for " + strconv.FormatFloat(totalPut, 'f', -1, 64)
			// NOTE: we do NOT change:
			//   H.currentBet
			//   H.raiseAmount
			//   other players' canAct flags
			return true
		}

		// normal full raise
		totalToPut := toCall + desiredRaise
		finalPlayerBet := p.currentBet + totalToPut

		p.Stack -= totalToPut
		p.currentBet = finalPlayerBet
		p.potCommitment += totalToPut
		H.pot += totalToPut

		// move the table line to this bet
		H.currentBet = finalPlayerBet
		H.raiseAmount = desiredRaise

		// everyone can act again, except the raiser
		H.avaliableActions = []string{"call", "fold", "raise", "clear"}
		for i := range H.Players {
			H.Players[i].canAct = true
			drainCallsAndRaises(H.Players[i].pendingAction)
		}
		p.canAct = false

		H.currentActionMessage = p.Name + " raised the current bet by " + strconv.FormatFloat(desiredRaise, 'f', -1, 64)
		return true

	case "call":
		// amount needed to call is what the most recent raiser bet(current bet in hand) - what the player has in front of them
		amountToCall := H.currentBet - H.Players[H.actionPlayerIndex].currentBet
		if amountToCall > H.Players[H.actionPlayerIndex].Stack {
			amountToCall = H.Players[H.actionPlayerIndex].Stack
		}
		//update player stack/pot and what the current player has in front of them
		H.Players[H.actionPlayerIndex].Stack -= amountToCall
		H.pot += amountToCall
		H.Players[H.actionPlayerIndex].potCommitment += amountToCall
		H.Players[H.actionPlayerIndex].currentBet = amountToCall + H.Players[H.actionPlayerIndex].currentBet
		H.Players[H.actionPlayerIndex].canAct = false

		H.currentActionMessage = H.Players[H.actionPlayerIndex].Name + " called"
		return true

	case "fold":
		H.lock.Lock()
		found := false
		for i, p := range H.Players {
			if p.ID == action.PlayerID {
				H.currentActionMessage = p.Name + " folded"
				p.folded = true
				removePlayerAt(H, i)
				found = true
				break
			}
		}
		H.lock.Unlock()
		return found

		//for players to clear whatever action is in their action queue
	case "clear":
		drainPendingActions(H.Players[H.actionPlayerIndex].pendingAction)

	}

	return false
}

func newHand(players []*Player, smallBlindPosition int, smallBlindSize float64, r *Room) *Hand {
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
		smallBlindName:    players[smallBlindPosition].Name,
		deck:              deck,
		currentState:      "pre-flop",
		pot:               0,
		avaliableActions:  []string{"raise", "fold", "check", "clear"},
		smallBlindSize:    smallBlindSize,
		currentBet:        0,
		raiseAmount:       0,
		skipToShowdown:    false,
		Room:              r,
	}
}

func streetLoop(h *Hand) {
	h.actionPlayerIndex = h.smallBlindIndex
	h.smallBlindName = h.Players[h.smallBlindIndex].Name
	//if pre flop small blind raises
	if h.currentState == "pre-flop" {
		print("small blind is: ", h.Players[h.actionPlayerIndex].ID, "\n")
		handleAction(h, Action{PlayerID: h.Players[h.actionPlayerIndex].ID, Action: "raise", Amount: h.smallBlindSize})
		//big blind raise
		h.actionPlayerIndex = (h.actionPlayerIndex + 1) % len(h.Players)
		print("big blind is: ", h.Players[h.actionPlayerIndex].ID, "\n")
		handleAction(h, Action{PlayerID: h.Players[h.actionPlayerIndex].ID, Action: "raise", Amount: h.smallBlindSize})
		//blind can still act after raising
		h.actionPlayerIndex = (h.actionPlayerIndex + 1) % len(h.Players)
		for i := range h.Players {
			h.Players[i].canAct = true
		}
	}
	for {

		//check if everyone called the blind, then set actions to either check/raise/fold for big blind
		if h.currentBet == (h.smallBlindSize*2) && h.actionPlayerIndex == ((h.smallBlindIndex+1)%len(h.Players)) && h.Players[h.actionPlayerIndex].currentBet == h.smallBlindSize*2 {
			h.avaliableActions = []string{"check", "raise", "fold", "clear"}
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
		h.actionPlayerDeadline = time.Now().Add(timeLimit * time.Second)
		timer := time.NewTimer(timeLimit * time.Second)
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
					// if they folded because of timeout make them sit out
					cur.sittingOut = true
					handleAction(h, Action{PlayerID: cur.ID, Action: "fold"})
				}
				goto nextActor

			}
		}

	nextActor:
		// Advance seat (slice may have shrunk on fold; modulo keeps us in range)
		if len(h.Players) == 0 {
			break
		}
		h.actionPlayerIndex %= len(h.Players)
		h.actionPlayerIndex = (h.actionPlayerIndex + 1) % len(h.Players)
	}
	//resert all current bets and actions
	h.avaliableActions = []string{"raise", "check", "fold", "clear"}
	h.raiseAmount = 0
	h.currentBet = 0
	for i := range h.Players {
		h.Players[i].canAct = true
		h.Players[i].currentBet = 0
		drainPendingActions(h.Players[i].pendingAction)
	}

}

func (h *Hand) run() {

	//clear player cards and amount they can win this hand
	playerCount := 0
	for i := range h.Players {
		h.Players[i].Hand = []Card{}
		h.Players[i].potCommitment = 0
		h.Players[i].folded = false
		h.Players[i].currentBet = 0
		playerCount++

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
		if skipToShowdown(h) {
			h.skipToShowdown = true
			h.actionPlayerIndex = 0
		}
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

		if !h.skipToShowdown {
			streetLoop(h)
		} else {
			h.actionPlayerIndex = 0
		}
		//check if we should skip to showdown
		if skipToShowdown(h) {
			h.skipToShowdown = true
			h.actionPlayerIndex = 0
		}
		//wait 1 sec to display board
		time.Sleep(1 * time.Second)
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
		if !h.skipToShowdown {
			streetLoop(h)
		} else {
			h.actionPlayerIndex = 0
		}

		if skipToShowdown(h) {
			h.skipToShowdown = true
			h.actionPlayerIndex = 0
		}
		time.Sleep(1 * time.Second)
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
		if !h.skipToShowdown {
			streetLoop(h)
		} else {
			h.actionPlayerIndex = 0
		}
		time.Sleep(1 * time.Second)

	}

	print("River done, moving to showdown\n")
	h.currentState = "showdown"

	// showdown

	//make all side pots
	SidePots := buildSidePots(h)
	//get all players best hands
	showdownhands := make([]showDownHand, len(h.Players))
	playerHands := make([]playerHand, len(h.Players))
	for i := range h.Players {
		playerHands[i] = playerHand{playerId: h.Players[i].ID, hand: getPlayerBestHand(h, h.Players[i])}
		showdownhands = append(showdownhands, showDownHand{PlayerName: h.Players[i].Name, Hand: h.Players[i].Hand})

	}

	//for each side pot
	showdownmessage := ""
	for i := range SidePots {
		currentPot := SidePots[i]
		// get playerands with only the ids that can win the side pot(for i=0 its all players)
		if i == 0 {
			print("first side pot is:", currentPot.Amount, "\n")
			winningHands := getShowdownBestHand(playerHands)
			if len(winningHands) > 1 {
				println("chopping")
				showdownmessage += "chopping with " + strconv.Itoa(len(winningHands)) + " players \n"
				for i := range winningHands {
					println(winningHands[i].playerId, " wins with ", winningHands[i].hand.Type)
					tmpIndex := FindPlayerIndex(h, winningHands[i].playerId)
					//can win the amount they committed divided by the number of players(but they are chopping)
					amountCanWin := currentPot.Amount / float64(len(winningHands))
					h.Players[tmpIndex].Stack += amountCanWin
					h.pot -= amountCanWin
					showdownmessage += h.Players[tmpIndex].Name + " wins " + strconv.FormatFloat(amountCanWin, 'f', -1, 64) + " with " + string(winningHands[i].hand.Type)

				}
			} else {
				println(winningHands[0].playerId, " wins with ", winningHands[0].hand.Type)
				tmpIndex := FindPlayerIndex(h, winningHands[0].playerId)
				amountCanWin := currentPot.Amount
				h.Players[tmpIndex].Stack += amountCanWin
				h.pot -= amountCanWin
				showdownmessage += h.Players[tmpIndex].Name + " wins " + strconv.FormatFloat(amountCanWin, 'f', -1, 64) + " with " + string(winningHands[i].hand.Type)
				if isSevenTwoOff(h.Players[tmpIndex]) {
					h.collectSevenTwoBounty(h.Room, h.Players[tmpIndex])
					showdownmessage += h.Players[tmpIndex].Name + " collected 7 2 bounty"
				}
			}
		} else {
			print("side pot is:", currentPot.Amount, "\n")
			//if the player id is not in the side pot they can't win(remove them from playerhands)
			tmpPlayerHands := make([]playerHand, len(playerHands))
			for j := range playerHands {
				if contains(currentPot.eligiblePlayerIDs, playerHands[j].playerId) {
					tmpPlayerHands[j] = playerHands[j]

				}
			}
			winningHands := getShowdownBestHand(tmpPlayerHands)
			if len(winningHands) > 1 {
				println("chopping")
				for i := range winningHands {
					println("player ", winningHands[i].playerId, " wins with ", winningHands[i].hand.Type)
					tmpIndex := FindPlayerIndex(h, winningHands[i].playerId)
					//can win the amount they committed divided by the number of players(but they are chopping)
					amountCanWin := currentPot.Amount / float64(len(winningHands))
					h.Players[tmpIndex].Stack += amountCanWin
					h.pot -= amountCanWin
				}
			} else {
				println("player ", winningHands[0].playerId, " wins with ", winningHands[0].hand.Type)
				tmpIndex := FindPlayerIndex(h, winningHands[0].playerId)
				amountCanWin := currentPot.Amount
				h.Players[tmpIndex].Stack += amountCanWin
				h.pot -= amountCanWin
				if isSevenTwoOff(h.Players[tmpIndex]) {
					h.collectSevenTwoBounty(h.Room, h.Players[tmpIndex])
				}
			}
		}

	}

	//if only 1 player dont show hand unless it is 7 2 off
	if len(showdownhands) <= 2 {
		if (showdownhands[1].Hand[0].Rank == "7" && showdownhands[1].Hand[1].Rank == "2") || (showdownhands[1].Hand[1].Rank == "7" && showdownhands[1].Hand[0].Rank == "2") {
		} else {
			showdownmessage = showdownhands[1].PlayerName + " wins"
			showdownhands = nil
		}
	}

	h.currentActionMessage = ""
	h.showDownHands = showdownhands
	h.showDownMessage = showdownmessage
	time.Sleep(5 * time.Second)
	h.showDownMessage = ""
	h.showDownHands = nil
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

	h.skipToShowdown = false
}
