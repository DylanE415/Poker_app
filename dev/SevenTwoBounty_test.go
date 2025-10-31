// hand_bounty_test.go
package main

import (
	"testing"
)

func TestCollectSevenTwoBounty(t *testing.T) {
	// Minimal Player/Card types are assumed to exist in your codebase:
	// type Card struct { Suit, Rank string }
	// type Player struct { ID, Name string; Stack float64; Hand []Card }

	r := newRoom(1, 0, 150)

	// Helper to make a player with a stack
	mk := func(id string, stack float64) *Player {
		return &Player{ID: id, Name: id, Stack: stack}
	}

	sb := 5.0
	// winner has 7-2 offsuit
	winner := mk("P1", 100)
	winner.Hand = []Card{{Rank: "2", Suit: "H"}, {Rank: "7", Suit: "S"}}

	// others: one deep, one exactly SB, one short-stacked
	p2 := mk("P2", 100) // pays 5
	p3 := mk("P3", 5)   // pays 5
	p4 := mk("P4", 3)   // pays 3 (short)

	h := &Hand{
		Players:        []*Player{winner, p2, p3, p4},
		smallBlindSize: sb,
	}
	r.players = []*Player{winner, p2, p3, p4}

	// Sanity: winner is indeed 7-2 offsuit
	if !isSevenTwoOff(winner) {
		t.Fatalf("expected winner to have 7-2 offsuit")
	}

	// Capture pre-values
	startWinner := winner.Stack
	startP2 := p2.Stack
	startP3 := p3.Stack
	startP4 := p4.Stack

	h.collectSevenTwoBounty(r, winner)

	// Expected transfers (excluding the winner themselves):
	expFromP2 := min(sb, startP2) // 5
	expFromP3 := min(sb, startP3) // 5
	expFromP4 := min(sb, startP4) // 3
	expectedGain := expFromP2 + expFromP3 + expFromP4

	// Winner: should only gain from others; must NOT pay themselves.
	if got, want := winner.Stack, startWinner+expectedGain; got != want {
		t.Errorf("winner stack = %.2f, want %.2f", got, want)
	}

	// P2: -5
	if got, want := p2.Stack, startP2-expFromP2; got != want {
		t.Errorf("P2 stack = %.2f, want %.2f", got, want)
	}
	// P3: -5
	if got, want := p3.Stack, startP3-expFromP3; got != want {
		t.Errorf("P3 stack = %.2f, want %.2f", got, want)
	}
	// P4: -3 (short)
	if got, want := p4.Stack, startP4-expFromP4; got != want {
		t.Errorf("P4 stack = %.2f, want %.2f", got, want)
	}
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
