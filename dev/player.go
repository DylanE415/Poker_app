package main

type Player struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Stack         float64 `json:"stack"`
	canAct        bool
	timebank      float64
	sittingOut    bool
	Hand          []Card `json:"hand"`
	pendingAction chan Action
	currentBet    float64
	potCommitment float64
	folded        bool
	buyIn         float64
}

func newPlayer(id string, name string, stack float64) *Player {
	return &Player{
		ID:            id,
		Name:          name,
		Stack:         stack,
		sittingOut:    true,
		canAct:        true,
		timebank:      60,
		pendingAction: make(chan Action, 1),
		currentBet:    0,
		potCommitment: 0,
		folded:        false,
		buyIn:         stack,
	}
}

func getPlayerFromID(id string, players []*Player) *Player {
	for _, p := range players {
		if p.ID == id {
			return p
		}
	}
	return nil
}
