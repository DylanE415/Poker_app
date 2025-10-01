package main

type Player struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Stack         float64 `json:"stack"`
	canAct        bool
	timebank      float64
	sittingOut    bool
	hand          []Card
	pendingAction chan Action
}

func newPlayer(id string, name string, stack float64) Player {
	return Player{
		ID:            id,
		Name:          name,
		Stack:         stack,
		sittingOut:    true,
		canAct:        true,
		timebank:      60,
		pendingAction: make(chan Action, 1),
	}
}
