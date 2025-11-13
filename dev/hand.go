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
	bigBlindName         string
	skipToShowdown       bool
	showDownHands        []showDownHand
	showDownMessage      string
	currentActionMessage string
	actionPlayerDeadline time.Time
	//for locking hand to prevent race conditions
	lock sync.Mutex
}

const (
	timeLimitPerAction = 30 //seconds

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

	// if the player that folded is acting
	if h.actionPlayerIndex == idx {
		// set to the seat just before the removed one, so +1 below hits the correct next actor
		h.actionPlayerIndex = (idx - 1 + n) % n
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

func shouldEndHand(H *Hand) bool {
	return len(H.Players) == 1 || H.pot == 0
}

// if hand ended before showdown(1 player left), award pot to winning player, send message
func endHandEarly(H *Hand) {
	winningPlayer := H.Players[0]
	if isSevenTwoOff(winningPlayer) {
		//change state to showdown because frontend needs that to display the message
		H.currentState = "showdown"
		H.showDownMessage = fmt.Sprintf("%s wins %s and gets 7 2 bounty", winningPlayer.Name, strconv.FormatFloat(H.pot, 'f', 2, 64))
		time.Sleep(4 * time.Second)
		H.collectSevenTwoBounty(H.Room, winningPlayer)
		winningPlayer.Stack += H.pot
		H.pot = 0
	}
	H.currentState = "showdown"
	H.showDownMessage = fmt.Sprintf("%s wins %s", winningPlayer.Name, strconv.FormatFloat(H.pot, 'f', 2, 64))
	time.Sleep(4 * time.Second)
	winningPlayer.Stack += H.pot
	H.pot = 0
}

// if there are 2 or more players and they is only 1 or 0 players with anything left in stack
// we skip to showdown as they cannot act and are all in
func skipToShowdown(H *Hand) bool {
	playersCanAct := 0
	totalPlayers := len(H.Players)
	for i := range H.Players {
		if checkPlayerCanAct(H.Players[i]) {
			playersCanAct++
		}
	}
	if totalPlayers >= 2 && playersCanAct <= 1 {
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
	//round float to nearest cent
	action.Amount = roundToNearestCent(action.Amount)

	switch action.Action {
	case "check":
		H.Players[H.actionPlayerIndex].canAct = false
		H.currentActionMessage = H.Players[H.actionPlayerIndex].Name + " checked"
		println(H.Players[H.actionPlayerIndex].Name, "checked")
		time.Sleep(200 * time.Millisecond)
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
			println(p.Name, "called all in for", contrib)
			time.Sleep(200 * time.Millisecond)
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
			H.pot += totalPut

			p.canAct = false
			H.currentActionMessage = p.Name + " went all in for " + strconv.FormatFloat(totalPut, 'f', -1, 64)
			println(p.Name, "went all in for", totalPut)

			// NOTE: we do NOT change:
			//   H.currentBet
			//   H.raiseAmount
			//   other players' canAct flags
			time.Sleep(200 * time.Millisecond)
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
		println(p.Name, " raised the current bet by", desiredRaise)
		time.Sleep(200 * time.Millisecond)
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
		println(H.Players[H.actionPlayerIndex].Name, " called")
		time.Sleep(200 * time.Millisecond)
		return true

	case "fold":
		H.lock.Lock()
		found := false
		for i, p := range H.Players {
			if p.ID == action.PlayerID {
				H.currentActionMessage = p.Name + " folded"
				println(p.Name, "folded")
				p.folded = true
				removePlayerAt(H, i)
				found = true
				break
			}
		}
		H.lock.Unlock()
		time.Sleep(200 * time.Millisecond)
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
		bigBlindName:      players[(smallBlindPosition+1)%len(players)].Name,
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
		if h.currentBet == (h.smallBlindSize*2) && h.bigBlindName == h.Players[h.actionPlayerIndex].Name && h.Players[h.actionPlayerIndex].currentBet == h.smallBlindSize*2 {
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
		h.actionPlayerDeadline = time.Now().Add(timeLimitPerAction * time.Second)
		timer := time.NewTimer(time.Duration(timeLimitPerAction)*time.Second + cur.timebank)
		for {
			var act Action

			select {
			case act = <-cur.pendingAction:
				// Ignore messages not from this player or not currently allowed
				//see if player used up timebank time
				if time.Now().After(h.actionPlayerDeadline) {
					usedTime := time.Since(h.actionPlayerDeadline)
					cur.timebank -= time.Duration(usedTime)
				}
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
				// Timeout -> default move for THIS
				//if they are sitting out fold to timeout
				cur.timebank = 0
				if cur.sittingOut {
					handleAction(h, Action{PlayerID: cur.ID, Action: "fold"})
				}
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
		h.Players[i].timebank = time.Second * 60
		h.Players[i].ShowCards = false
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
		} else if shouldEndHand(h) {
			endHandEarly(h)
			return
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
		}

		//check if we should skip to showdown/end
		if skipToShowdown(h) {
			h.skipToShowdown = true
			h.actionPlayerIndex = 0
		} else if shouldEndHand(h) {
			endHandEarly(h)
			return
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
		}

		if skipToShowdown(h) {
			h.skipToShowdown = true
			h.actionPlayerIndex = 0
		} else if shouldEndHand(h) {
			endHandEarly(h)
			return
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
		}
		//no need to check for a skip to showdown since it is next street,but still check end
		if shouldEndHand(h) {
			endHandEarly(h)
			return
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
		showdownhands[i] = showDownHand{PlayerName: h.Players[i].Name, Hand: h.Players[i].Hand}

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
					tmpIndex := FindPlayerIndex(h, winningHands[i].playerId)
					//can win the amount they committed divided by the number of players(but they are chopping)
					amountCanWin := currentPot.Amount / float64(len(winningHands))
					h.Players[tmpIndex].Stack += amountCanWin
					h.pot -= amountCanWin
					showdownmessage += h.Players[tmpIndex].Name + " wins " + strconv.FormatFloat(amountCanWin, 'f', -1, 64) + " with " + string(winningHands[i].hand.Type)
					println(showdownmessage)

				}
			} else {
				tmpIndex := FindPlayerIndex(h, winningHands[0].playerId)
				amountCanWin := currentPot.Amount
				h.Players[tmpIndex].Stack += amountCanWin
				h.pot -= amountCanWin
				showdownmessage += h.Players[tmpIndex].Name + " wins " + strconv.FormatFloat(amountCanWin, 'f', -1, 64) + " with " + string(winningHands[0].hand.Type)
				println(showdownmessage)
				if isSevenTwoOff(h.Players[tmpIndex]) {
					h.collectSevenTwoBounty(h.Room, h.Players[tmpIndex])
					showdownmessage += h.Players[tmpIndex].Name + " collected 7 2 bounty"
				}
			}
		} else {
			print("side pot is:", currentPot.Amount, "\n")
			//if the player id is not in the side pot they can't win(remove them from playerhands)
			tmpPlayerHands := make([]playerHand, 0, len(playerHands))
			for j := range playerHands {
				if contains(currentPot.eligiblePlayerIDs, playerHands[j].playerId) {
					tmpPlayerHands = append(tmpPlayerHands, playerHands[j])

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
	if len(showdownhands) <= 1 {
		if ((showdownhands[0].Hand[0].Rank == "7" && showdownhands[0].Hand[1].Rank == "2") && showdownhands[0].Hand[0].Suit != showdownhands[0].Hand[1].Suit) || ((showdownhands[0].Hand[1].Rank == "7" && showdownhands[0].Hand[0].Rank == "2") && showdownhands[0].Hand[0].Suit != showdownhands[0].Hand[1].Suit) {
		} else {
			showdownmessage = showdownhands[0].PlayerName + " wins"
			showdownhands = nil
		}
	}

	h.currentActionMessage = ""
	h.showDownHands = showdownhands
	h.showDownMessage = showdownmessage
	time.Sleep(5 * time.Second)
	h.showDownMessage = ""

	//make all players with 0 stack sitting out
	for i := range h.Players {
		if h.Players[i].Stack <= 0 {
			h.Players[i].sittingOut = true
		}
	}

}
