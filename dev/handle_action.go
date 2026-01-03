package main

import (
	"strconv"
	"time"
)

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

	case "small_blind":
		p := H.Players[H.actionPlayerIndex]
		if p.Stack <= H.smallBlindSize {
			amt := p.Stack
			p.currentBet = amt
			H.pot += amt
			p.Stack = 0
			H.currentActionMessage = p.Name + " went all in for " + strconv.FormatFloat(amt, 'f', -1, 64)
			p.potCommitment += amt
			H.raiseAmount = H.smallBlindSize
			H.currentBet = amt
			return true
		}
		amt := H.smallBlindSize
		p.currentBet = amt
		H.pot += amt
		p.Stack -= amt
		p.potCommitment += amt
		H.raiseAmount = amt
		H.currentBet = amt
		return true

	case "big_blind":
		p := H.Players[H.actionPlayerIndex]
		if p.Stack <= 2*H.smallBlindSize {
			//if less than the small blind
			if p.Stack < H.currentBet {
				p.currentBet = p.Stack
				H.pot += p.Stack
				p.Stack = 0
				H.currentActionMessage = p.Name + " went all in for " + strconv.FormatFloat(p.Stack, 'f', -1, 64)
				p.potCommitment += p.Stack
				return true
				//else amt was bigger than sb but smaller than bb
			} else {
				p.currentBet = p.Stack
				H.pot += p.Stack
				p.Stack = 0
				H.currentActionMessage = p.Name + " went all in for " + strconv.FormatFloat(p.Stack, 'f', -1, 64)
				p.potCommitment += p.Stack
				H.currentBet = p.Stack
				H.raiseAmount = H.smallBlindSize
				return true
			}
		}
		amt := H.smallBlindSize * 2
		p.currentBet = amt
		H.pot += amt
		p.Stack -= amt
		p.potCommitment += amt
		H.raiseAmount = 0.25
		H.currentBet = amt
		H.currentActionMessage = p.Name + " and " + H.smallBlindName + " posted the blinds"
		return true

	case "raise":
		p := H.Players[H.actionPlayerIndex]

		// how much do I have to put in just to call right now?
		toCall := H.currentBet - p.currentBet
		if toCall < 0 {
			toCall = 0
		}

		// If I can't even cover the call, this is an all-in CALL, not a raise.
		// Line does NOT move in this case.
		if p.Stack <= toCall {
			contrib := p.Stack
			p.Stack = 0
			p.currentBet += contrib
			p.potCommitment += contrib
			H.pot += contrib

			p.canAct = false
			H.currentActionMessage = p.Name + " called all in for " + strconv.FormatFloat(contrib, 'f', -1, 64)
			println(p.Name, "called all in for", contrib)
			time.Sleep(actionDelayMS * time.Millisecond)
			return true
		}

		// Player is asking to "raise BY" this amount
		desiredRaise := action.Amount
		if desiredRaise < 0 {
			desiredRaise = 0
		}

		// How many chips I have LEFT AFTER calling
		canRaiseWith := p.Stack - toCall

		// If I can't afford the raise I'm asking for -> short all-in.
		// We now treat this as "raise all-in" and MOVE the line (reopen action).
		if desiredRaise > canRaiseWith {
			desiredRaise = canRaiseWith
		}

		// If, after capping, there's no raise left, just treat as a call.
		if desiredRaise == 0 {
			totalToPut := toCall
			p.Stack -= totalToPut
			p.currentBet += totalToPut
			p.potCommitment += totalToPut
			H.pot += totalToPut

			p.canAct = false
			H.currentActionMessage = p.Name + " called"
			println(p.Name, "called")
			time.Sleep(actionDelayMS * time.Millisecond)
			return true
		}

		// Normal (or short all-in) raise
		totalToPut := toCall + desiredRaise
		finalPlayerBet := p.currentBet + totalToPut

		p.Stack -= totalToPut
		p.currentBet = finalPlayerBet
		p.potCommitment += totalToPut
		H.pot += totalToPut

		// Move the table line to this bet (always, including short all-in)
		prevLine := H.currentBet
		H.currentBet = finalPlayerBet
		H.raiseAmount = finalPlayerBet - prevLine

		// Everyone can act again, except the raiser
		H.avaliableActions = []string{"call", "fold", "raise", "clear"}
		for i := range H.Players {
			H.Players[i].canAct = true
			drainCallsAndRaises(H.Players[i].pendingAction)
		}
		p.canAct = false

		// Message: distinguish between “all in” vs normal raise
		if desiredRaise == canRaiseWith {
			// short (or exact) all-in raise that moved the line
			H.currentActionMessage = p.Name + " went all in for " + strconv.FormatFloat(totalToPut, 'f', 2, 64)
			p.Stack = 0
			println(p.Name, "went all in for", totalToPut)
		} else {
			H.currentActionMessage = p.Name + " raised the current bet by " + strconv.FormatFloat(desiredRaise, 'f', 2, 64)
			println(p.Name, "raised the current bet by", desiredRaise)
		}

		time.Sleep(actionDelayMS * time.Millisecond)
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
