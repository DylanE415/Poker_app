package main

import (
	"testing"
	"time"
)

func Test_AllInPreflop_SplitPot_withThreePlayers(t *testing.T) {
	// Stacks:
	p1 := &Player{ID: "P1", Name: "Player 1", Stack: 1800, pendingAction: make(chan Action, 1)}  // BB (per engine log)
	p2 := &Player{ID: "P2", Name: "Player 2", Stack: 4400, pendingAction: make(chan Action, 1)}  // UTG (acts first preflop)
	p3 := &Player{ID: "P3", Name: "Player 3", Stack: 10000, pendingAction: make(chan Action, 1)} // SB (acts second preflop)

	players := []*Player{p1, p2, p3}

	// Choose SB index so engine ends up with SB=P3, BB=P1 (as your log shows).
	// Using sbIndex=2 with sbSize=0 keeps blinds from auto-posting and produces:
	// preflop order: P2 → P3 → P1
	const sbIndex = 2
	const sbSize = 0.0

	h := newHand(players, sbIndex, sbSize, nil)
	for _, p := range h.Players {
		p.canAct = true
	}

	// Deterministic deck:
	deck := []Card{
		// hole 1 (P1,P2,P3), hole 2 (P1,P2,P3)
		C("H", 13), // P1: K♥
		C("H", 12), // P2: Q♥
		C("S", 13), // P3: K♠
		C("S", 14), // P1: A♠
		C("C", 9),  // P2: 9♣
		C("S", 11), // P3: J♠
		// burn
		C("C", 3),
		// flop
		C("C", 13), // K♣
		C("D", 13), // K♦
		C("C", 12), // Q♣
		// burn
		C("D", 4),
		// turn
		C("D", 12), // Q♦
		// burn
		C("H", 5),
		// river
		C("S", 2), // 2♠
	}
	for len(deck) < 52 {
		deck = append(deck, C("C", 2))
	}
	h.deck = deck

	done := make(chan struct{})
	go func() { h.run(); close(done) }()

	// PRE-FLOP (actor order: P2 → P3 → P1, then continues around)
	// We follow the engine’s order but still realize the intended all-in line:
	// P2 "call" (checks, since currentBet=0),
	// P3 "call" (checks),
	// P1 RAISE to 1800 (shove),
	// P2 RERAISE to 4400 total (amount = 2600),
	// P3 CALL to 4400.
	time.Sleep(20 * time.Millisecond) // let dealing finish

	// P2 first (UTG after BB)
	p2.pendingAction <- Action{PlayerID: "P2", Action: "call"} // equivalent to check at 0

	time.Sleep(120 * time.Millisecond)
	// P3 second (SB)
	p3.pendingAction <- Action{PlayerID: "P3", Action: "call"} // check

	time.Sleep(120 * time.Millisecond)
	// P1 third (BB) — now shove to 1800 total
	p1.pendingAction <- Action{PlayerID: "P1", Action: "raise", Amount: 1800}

	time.Sleep(140 * time.Millisecond)
	// Next around: P2 acts again — reraise to 4400 total (delta = 2600)
	p2.pendingAction <- Action{PlayerID: "P2", Action: "raise", Amount: 2600}

	time.Sleep(140 * time.Millisecond)
	// P3 calls to match 4400
	p3.pendingAction <- Action{PlayerID: "P3", Action: "call"}

	// Optional: tiny nudge to close if your loop re-opens action
	time.Sleep(80 * time.Millisecond)

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatalf("hand.run did not finish in time")
	}

	// Expected results:
	// - Main pot: 1800*3 = 5400 -> split P1 & P3 => 2700 each
	// - Side pot: (4400-1800)*2 = 5200 -> P3 wins
	if p1.Stack != 2700 {
		t.Fatalf("P1 stack = %.0f, want 2700", p1.Stack)
	}
	if p2.Stack != 0 {
		t.Fatalf("P2 stack = %.0f, want 0", p2.Stack)
	}
	if p3.Stack != 13500 {
		t.Fatalf("P3 stack = %.0f, want 13500", p3.Stack)
	}
	if h.pot != 0 {
		t.Fatalf("remaining pot = %.0f, want 0", h.pot)
	}
}

func Test_AllInPreflop_splitPot_with2Players(t *testing.T) {
	p1 := &Player{ID: "P1", Name: "P1", Stack: 1000, pendingAction: make(chan Action, 1)}
	p2 := &Player{ID: "P2", Name: "P2", Stack: 1000, pendingAction: make(chan Action, 1)}
	players := []*Player{p1, p2}

	const sbIndex = 0
	const sbSize = 0.0

	h := newHand(players, sbIndex, sbSize, nil)
	for _, p := range h.Players {
		p.canAct = true
	}

	deck := []Card{
		C("H", 13), C("H", 8),
		C("S", 8), C("C", 3),
		C("C", 3),
		C("C", 8), C("D", 8), C("S", 10),
		C("D", 4),
		C("D", 10),
		C("H", 5),
		C("S", 2),
	}
	for len(deck) < 52 {
		deck = append(deck, C("C", 2))
	}
	h.deck = deck

	done := make(chan struct{})
	go func() { h.run(); close(done) }()

	time.Sleep(20 * time.Millisecond)
	p1.pendingAction <- Action{PlayerID: "P1", Action: "raise", Amount: 1000}

	time.Sleep(140 * time.Millisecond)
	p2.pendingAction <- Action{PlayerID: "P2", Action: "call"}

	// Optional safety clear (usually unnecessary in heads-up all-in)
	select {
	case p1.pendingAction <- Action{PlayerID: "P1", Action: "clear"}:
	default:
	}

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatalf("hand.run did not finish in time")
	}

	if p1.Stack != 1000 {
		t.Fatalf("P1 stack = %.0f, want 1000", p1.Stack)
	}
	if p2.Stack != 1000 {
		t.Fatalf("P2 stack = %.0f, want 1000", p2.Stack)
	}
	if h.pot != 0 {
		t.Fatalf("remaining pot = %.0f, want 0", h.pot)
	}
}
