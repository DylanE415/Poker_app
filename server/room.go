package main

import (
	"fmt"
	"time"
)

type Command struct {
	Kind       string // "join, leave, sitOut, sitIn, action, "getState"
	ActionType string // "fold, call, raise, check"
	PlayerID   string
	PlayerName string
	stack      float64
	reply      chan error //for sending a reply to client
	stateReply chan roomState
	actionAmt  float64
}

type Room struct {
	id                 int
	commandChan        chan Command
	players            []*Player
	minStack           float64
	maxStack           float64
	smallBlindPosition int
	currentHand        *Hand
	previousHand       *Hand
	handDone           chan struct{}
}

type playerState struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Stack      float64 `json:"stack"`
	Timebank   float64 `json:"timebank"`
	SittingOut bool    `json:"sittingOut"`
	Hand       []Card  `json:"hand"`
	CurrentBet float64 `json:"currentBet"`
}

type handState struct {
	Board            []Card         `json:"board"`
	Pot              float64        `json:"pot"`
	AvaliableActions []string       `json:"avaliableActions"`
	RaiseAmount      float64        `json:"raiseAmount"`
	CurrentBet       float64        `json:"currentBet"`
	ActionPlayerName string         `json:"actionPlayerName"`
	Street           string         `json:"street"`
	ShowDownMessage  string         `json:"showDownMessage"`
	ShowDownHands    []showDownHand `json:"showDownHands"`
}

type roomState struct {
	Players []playerState `json:"players"`
	Hand    handState     `json:"hand"`
}

// has a command buffer of 16 commands
func newRoom(id int, minStack float64, maxStack float64) *Room {
	return &Room{
		id:                 id,
		commandChan:        make(chan Command, 16),
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
	r.currentHand = newHand(eligible, r.smallBlindPosition, 1)
	// advance blinds for the NEXT hand
	r.smallBlindPosition = (r.smallBlindPosition + 1) % len(eligible)

	//clear everyones action channel
	for _, p := range r.players {
		select {
		case <-p.pendingAction:
		default:
		}
	}

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

// for rmaking the latest action the current action in the channel
func enqueueLatestAction(ch chan Action, a Action) {
	for {
		select {
		case ch <- a:
			// sent successfully; done
			return
		case <-ch:
			// channel was full; drop the value; and repeat the loop
		}
	}
}

// for sending a reply to the client
func safeReply(ch chan error, err error) {
	select {
	case ch <- err: // this means that there is a channel to receive
	default:
		// receiver not ready; drop so the room goroutine keeps running
	}
}

func sendStateReply(ch chan roomState, rs roomState) {
	select {
	case ch <- rs: // this means that there is a channel to receive
	default:
		// receiver not ready; drop so the room goroutine keeps running
	}
}

// function operates on a pointer receiver to actually change the room in memory, r Room would make a copy
func (r *Room) run() {
	ticker := time.NewTicker(400 * time.Millisecond) // every 400ms the room checks for new joins/leaves also checks if a hand is over
	defer ticker.Stop()                              // stop the ticker when the function returns

	for {
		select {
		case cmd := <-r.commandChan:
			switch cmd.Kind {
			case "join":
				// if the player is not in room
				if !r.hasPlayer(cmd.PlayerID) {
					//make new player and add to room
					newPlayer := newPlayer(cmd.PlayerID, cmd.PlayerName, cmd.stack)
					r.players = append(r.players, newPlayer)
					//send good reply to client
					print(cmd.PlayerName, " joined room\n")
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
							// send fold action to hand
							enqueueLatestAction(p.pendingAction, Action{id, "fold", 0})
							// remove player (appends all elements before i and all elements after i)
							r.players = append(r.players[:i], r.players[i+1:]...)
							//send good reply to client
							print(p.Name, " left room\n")
							safeReply(cmd.reply, nil)
							break
						}
					}
				} else {
					//send bad reply to client
					safeReply(cmd.reply, fmt.Errorf("player not in room"))
				}
			case "sitIn":
				player := getPlayerFromID(cmd.PlayerID, r.players)
				//see if player is in room
				if player == nil {
					//send bad reply to client
					safeReply(cmd.reply, fmt.Errorf("player not in room"))
					break
				}
				//send good reply to client
				print("players in room:\n")
				for _, p := range r.players {
					print(p.Name, "\n")
				}
				print(player.Name, " is sitting in\n")
				safeReply(cmd.reply, nil)
				player.sittingOut = false
			case "sitOut":
				player := getPlayerFromID(cmd.PlayerID, r.players)
				//see if player is in room
				if player == nil {
					//send bad reply to client
					safeReply(cmd.reply, fmt.Errorf("player not in room"))
					break
				}
				//send good reply to client
				print(player.Name, " is sitting out\n")
				safeReply(cmd.reply, nil)
				player.sittingOut = true
			case "action":
				player := getPlayerFromID(cmd.PlayerID, r.players)
				if player == nil {
					//send bad reply to client
					safeReply(cmd.reply, fmt.Errorf("player not in room"))
					break
				}
				if player.sittingOut {
					safeReply(cmd.reply, fmt.Errorf("player is sitting out"))
					break
				}
				//check if valid action
				h := r.currentHand
				if h == nil {
					safeReply(cmd.reply, fmt.Errorf("no hand in progress"))
					break
				}
				if !contains(h.avaliableActions, cmd.ActionType) {
					safeReply(cmd.reply, fmt.Errorf("invalid action"))
					break
				}
				if cmd.ActionType == "raise" && cmd.actionAmt > player.Stack {
					cmd.actionAmt = player.Stack
				}

				//if player had already had 1 chip in front of them and they raised a raise of 2 by 1 then they would only need to bet 2 more chips
				playerBet := cmd.actionAmt + player.currentBet
				if (playerBet < h.currentBet) && cmd.ActionType == "raise" {
					safeReply(cmd.reply, fmt.Errorf("not enough chips to raise"))
					break
				}

				if cmd.ActionType == "call" && cmd.actionAmt > player.Stack {
					cmd.actionAmt = player.Stack
				}
				//send good reply to client
				safeReply(cmd.reply, nil)
				// enqueue the action
				enqueueLatestAction(player.pendingAction, Action{PlayerID: player.ID, Action: cmd.ActionType, Amount: cmd.actionAmt})

			case "getState":
				//send state to client
				p := getPlayerFromID(cmd.PlayerID, r.players)
				if p == nil {
					//send bad reply to client
					safeReply(cmd.reply, fmt.Errorf("player not in room"))
					break
				}
				h := r.currentHand

				//fill in various fields of the state
				state := roomState{}
				state.Players = make([]playerState, len(r.players))
				// only info needed to display to client about other players(name, stack, sittingOut, timebank, currentBet) and hand if it is clients id
				for i, p := range r.players {
					state.Players[i] = playerState{
						Name:       p.Name,
						Stack:      p.Stack,
						SittingOut: p.sittingOut,
						Timebank:   p.timebank,
						CurrentBet: p.currentBet,
					}
					if p.ID == cmd.PlayerID {
						state.Players[i].Hand = p.Hand
					}
				}
				//fill in hand state ( board, available actions, pot, currentBet, actionplayername, raiseAmount)
				//must now lock the hand thread to ensure that the hand is not being updated while we are sending it to the player
				if h != nil {
					h.lock.Lock()
					state.Hand = handState{
						Board:            h.board,
						Pot:              h.pot,
						AvaliableActions: h.avaliableActions,
						RaiseAmount:      h.raiseAmount,
						CurrentBet:       h.currentBet,
						ActionPlayerName: h.Players[h.actionPlayerIndex].Name,
						Street:           h.currentState,
						ShowDownMessage:  h.showDownMessage,
						ShowDownHands:    h.showDownHands,
					}
					h.lock.Unlock()
				}
				//send reply and statereply
				safeReply(cmd.reply, nil)
				sendStateReply(cmd.stateReply, state)
			default:
				panic("unknown command kind")
			}
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
