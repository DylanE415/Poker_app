package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type Command struct {
	Kind       string // "join, leave, sitOut, sitIn, action, "getState"
	ActionType string // "fold, call, raise, check, "
	EmoteType  string // "angle, ..."
	PlayerID   string
	PlayerName string
	stack      float64
	reply      chan error //for sending a reply to client
	stateReply chan roomState
	actionAmt  float64
}

type Room struct {
	id                 int
	RoomName           string
	commandChan        chan Command
	players            []*Player
	minStack           float64
	maxStack           float64
	smallBlindPosition int
	smallBlindSize     float64
	currentHand        *Hand
	previousHand       *Hand
	handDone           chan struct{}
	sitOutSince        map[string]time.Time
	timeSinceLastHand  time.Time
}

type playerState struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Stack           float64 `json:"stack"`
	SittingOut      bool    `json:"sittingOut"`
	Hand            []Card  `json:"hand"`
	CurrentBet      float64 `json:"currentBet"`
	Folded          bool    `json:"folded"`
	TimeBankUnixMs  int64   `json:"timeBankUnixMs,omitempty"`
	EmoteText       string  `json:"emoteText,omitempty"`
	EmoteAudio      string  `json:"emoteAudio,omitempty"`
	EmoteEndsUnixMs int64   `json:"emoteEndsUnixMs,omitempty"`
	NextEmoteAt     int64   `json:"nextEmoteAt,omitempty"`
}

type handState struct {
	Board                   []Card         `json:"board"`
	Pot                     float64        `json:"pot"`
	AvaliableActions        []string       `json:"avaliableActions"`
	RaiseAmount             float64        `json:"raiseAmount"`
	CurrentBet              float64        `json:"currentBet"`
	ActionPlayerName        string         `json:"actionPlayerName"`
	Street                  string         `json:"street"`
	ShowDownMessage         string         `json:"showDownMessage"`
	ShowDownHands           []showDownHand `json:"showDownHands"`
	CurrentActionMessage    string         `json:"currentActionMessage"`
	SmallBlindName          string         `json:"smallBlindName"`
	ActionDeadlineUnixMs    int64          `json:"actionDeadlineUnixMs"`
	ServerCurrentTimeUnixMs int64          `json:"serverCurrentTimeUnixMs"`
}

type roomState struct {
	Players []playerState `json:"players"`
	Hand    handState     `json:"hand"`
}

// has a command buffer of 16 commands
func newRoom(id int, minStack float64, maxStack float64, smallBlindSize float64, roomName string) *Room {
	return &Room{
		id:                 id,
		commandChan:        make(chan Command, 16),
		players:            make([]*Player, 0),
		minStack:           minStack,
		maxStack:           maxStack,
		smallBlindPosition: 0,
		handDone:           make(chan struct{}, 1),
		sitOutSince:        make(map[string]time.Time),
		smallBlindSize:     smallBlindSize,
		RoomName:           roomName,
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

func updateLedger(playerName string, playerID string, currentStack float64, buyIn float64) {
	// pick a path that exists
	path := "./static/ledger.json"
	if _, err := os.Stat(path); err != nil {
		if _, err2 := os.Stat("../static/ledger.json"); err2 == nil {
			path = "../static/ledger.json"
		}
	}

	type entry struct {
		ID        string  `json:"id"`
		Name      string  `json:"name,omitempty"`
		NetProfit float64 `json:"net_profit"`
	}
	var doc struct {
		Ledger []entry `json:"ledger"`
	}

	if b, err := os.ReadFile(path); err == nil && len(b) > 0 {
		_ = json.Unmarshal(b, &doc)
	}
	if doc.Ledger == nil {
		doc.Ledger = []entry{}
	}

	// locate by ID
	idx := -1
	for i := range doc.Ledger {
		if doc.Ledger[i].ID == playerID {
			idx = i
			break
		}
	}

	if idx >= 0 {
		// round to nearest cent
		doc.Ledger[idx].NetProfit = currentStack - buyIn
	} else {
		// new entry: initialize buy_in = currentStack so net starts at 0
		doc.Ledger = append(doc.Ledger, entry{
			ID:        playerID,
			Name:      playerName,
			NetProfit: 0,
		})
	}

	if out, err := json.MarshalIndent(doc, "", "  "); err == nil {
		_ = os.WriteFile(path, out, 0o644)
	}
}

func (r *Room) leavePlayerByID(id string) {
	for i, p := range r.players {
		if p.ID == id {
			// auto-fold if they’re in a hand
			enqueueLatestAction(p.pendingAction, Action{PlayerID: p.ID, Action: "fold", Amount: 0})
			//update ledger
			updateLedger(p.Name, p.ID, p.Stack, p.buyIn)
			// players = all payers before i + all players after i
			r.players = append(r.players[:i], r.players[i+1:]...)
			delete(r.sitOutSince, id)
			return
		}
	}
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
	r.currentHand = newHand(eligible, r.smallBlindPosition, r.smallBlindSize, r)
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
	r.timeSinceLastHand = time.Now()
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
	//time.NewTicker(d) gives you a ticker whose .C is a channel that delivers a time value every d. In a
	ticker := time.NewTicker(200 * time.Millisecond) // every 400ms the room checks for new joins/leaves also checks if a hand is over
	autoKickTicker := time.NewTicker(30 * time.Second)
	defer ticker.Stop() // stop the ticker when the function returns

	for {

		select {
		case cmd := <-r.commandChan:
			switch cmd.Kind {
			case "join":

				if cmd.stack < r.minStack || cmd.stack > r.maxStack {
					safeReply(cmd.reply, fmt.Errorf("stack must be between %d and %d", int(r.minStack), int(r.maxStack)))
					break
				}
				// if room is full
				if len(r.players) >= 9 {
					//send bad reply to client
					safeReply(cmd.reply, fmt.Errorf("join failed: room is full"))
					break
				}
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
					safeReply(cmd.reply, fmt.Errorf("join failed: player already in room"))
				}
			case "leave":
				//see if player is in room
				id := cmd.PlayerID
				//if player is in room
				if r.hasPlayer(id) {
					r.leavePlayerByID(id)
					//send good reply to client
					print(cmd.PlayerName, " left room\n")
					safeReply(cmd.reply, nil)
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
				if player.Stack <= 0 {
					safeReply(cmd.reply, fmt.Errorf("player stack too low"))
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
				//reset sitting out timeout
				delete(r.sitOutSince, player.ID)

			case "sitOut":
				player := getPlayerFromID(cmd.PlayerID, r.players)
				//see if player is in room
				if player == nil {
					//send bad reply to client
					safeReply(cmd.reply, fmt.Errorf("player not in room"))
					break
				}
				//if there is a current hand fold player
				if r.currentHand != nil {
					enqueueLatestAction(player.pendingAction, Action{PlayerID: player.ID, Action: "fold", Amount: 0})
				}
				//send good reply to client
				print(player.Name, " is sitting out\n")
				safeReply(cmd.reply, nil)
				player.sittingOut = true
				r.sitOutSince[player.ID] = time.Now()
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

				h.lock.Lock()
				validAction := contains(h.avaliableActions, cmd.ActionType)
				minRaise := h.raiseAmount
				h.lock.Unlock()

				if !validAction {
					safeReply(cmd.reply, fmt.Errorf("invalid action"))
					break
				}
				if cmd.ActionType == "raise" && cmd.actionAmt > player.Stack {
					cmd.actionAmt = player.Stack
				}

				//if player had already had 1 chip in front of them and they raised a raise of 2 by 1 then they would only need to bet 2 more chips

				if (cmd.actionAmt < minRaise) && cmd.ActionType == "raise" {
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

			case "emote":
				player := getPlayerFromID(cmd.PlayerID, r.players)
				if player == nil {
					safeReply(cmd.reply, fmt.Errorf("player not in room"))
					break
				}
				safeReply(cmd.reply, nil)
				r.handleEmote(player, cmd.EmoteType)
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
				now := time.Now()
				// only info needed to display to client about other players(name, stack, sittingOut, timebank, currentBet) and hand if it is clients id
				for i, pl := range r.players {
					ps := playerState{
						Name:           pl.Name,
						ID:             pl.ID,
						Stack:          pl.Stack,
						SittingOut:     pl.sittingOut,
						TimeBankUnixMs: pl.timebank.Milliseconds(),
						CurrentBet:     pl.currentBet,
						Folded:         pl.folded,
					}
					if pl.ID == cmd.PlayerID {
						ps.Hand = pl.Hand

					}

					// attach emote if still active
					if !pl.emoteUntil.IsZero() && now.Before(pl.emoteUntil) {
						ps.EmoteText = pl.emoteText
						ps.EmoteAudio = pl.emoteAudio
						ps.EmoteEndsUnixMs = pl.emoteUntil.UnixMilli()
					} else {
						// clear expired runtime so next poll has nothing
						pl.emoteText, pl.emoteAudio = "", ""
						pl.emoteUntil = time.Time{}
					}
					if !pl.nextEmoteAt.IsZero() && now.Before(pl.nextEmoteAt) {
						ps.NextEmoteAt = pl.nextEmoteAt.UnixMilli()
					} else {
						// zero means "no cooldown"; with `omitempty` this disappears from JSON
						ps.NextEmoteAt = 0
					}

					state.Players[i] = ps
				}
				//fill in hand state ( board, available actions, pot, currentBet, actionplayername, raiseAmount)
				//must now lock the hand thread to ensure that the hand is not being updated while we are sending it to the player
				if h != nil {
					h.lock.Lock()
					actionPlayerName := ""
					if h.actionPlayerIndex >= 0 && h.actionPlayerIndex < len(h.Players) {
						actionPlayerName = h.Players[h.actionPlayerIndex].Name
					}
					state.Hand = handState{
						Board:                   h.board,
						Pot:                     h.pot,
						AvaliableActions:        h.avaliableActions,
						RaiseAmount:             h.raiseAmount,
						CurrentBet:              h.currentBet,
						ActionPlayerName:        actionPlayerName,
						Street:                  h.currentState,
						ShowDownMessage:         h.showDownMessage,
						ShowDownHands:           h.showDownHands,
						CurrentActionMessage:    h.currentActionMessage,
						SmallBlindName:          h.smallBlindName,
						ActionDeadlineUnixMs:    h.actionPlayerDeadline.UnixMilli(),
						ServerCurrentTimeUnixMs: time.Now().UnixMilli(),
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
		case <-autoKickTicker.C:
			now := time.Now()

			for _, p := range r.players {
				if !p.sittingOut {
					delete(r.sitOutSince, p.ID) //remove from sitoutcounter if they sat back in
					continue
				}
				started, ok := r.sitOutSince[p.ID]
				if !ok {
					r.sitOutSince[p.ID] = now
					continue
				}
				//auto kick if they sat out for 5 minutes
				if now.Sub(started) > 5*time.Minute {
					r.leavePlayerByID(p.ID)
					print("auto kick\n")

				}

			}
			// if no hand in the last30 minutes, kick everyone
			if r.currentHand == nil && !r.timeSinceLastHand.IsZero() &&
				now.Sub(r.timeSinceLastHand) > 30*time.Minute {

				for len(r.players) > 0 {
					r.leavePlayerByID(r.players[0].ID)
				}
				fmt.Println("auto kick: no hand for 30 minutes")
			}

		} // select bracket
	} // room loop bracket
} //room run bracket
