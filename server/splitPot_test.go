package main

import (
	"testing"
)

func Test_AllInPreflop_SplitPot_withThreePlayers(t *testing.T) {
	// Initial stacks before hand:
	p1 := &Player{ID: "P1", Name: "Player 1", Stack: 1800, pendingAction: make(chan Action, 2)}
	p2 := &Player{ID: "P2", Name: "Player 2", Stack: 4400, pendingAction: make(chan Action, 2)}
	p3 := &Player{ID: "P3", Name: "Player 3", Stack: 10000, pendingAction: make(chan Action, 2)}

	players := []*Player{p1, p2, p3}

	// We want actor order preflop to start at P1.
	// streetLoop sets actionPlayerIndex = smallBlindIndex, then does an initial SB "raise" of smallBlindSize
	// and advances to (smallBlindIndex+1) next. So pick SB = P3 (index 2) so next is P1.
	const sbIndex = 2
	const sbSize = 0.0 // avoid auto-bet interfering

	h := newHand(players, sbIndex, sbSize)

	// Everyone can act initially so nextEligible picks them.
	for _, p := range h.Players {
		p.canAct = true
	}

	// Build a deterministic deck so your evaluator gets:
	// Board: K♣ K♦ Q♣ Q♦ 2♠
	// P1: K♥ A♠   -> Full House KKKQQ
	// P2: Q♥ 9♣   -> Full House QQQKK
	// P3: K♠ J♠   -> Full House KKKQQ
	//
	// Dealing order in run():
	//  - 2 rounds of hole cards: P1, P2, P3, then P1, P2, P3 (taking deck[0] each time)
	//  - Flop: burn(1), 3 cards
	//  - Turn: burn(1), 1 card
	//  - River: burn(1), 1 card
	deck := []Card{
		// hole 1 (P1,P2,P3), hole 2 (P1,P2,P3)
		C("H", 13), // P1: K♥
		C("H", 12), // P2: Q♥
		C("S", 13), // P3: K♠
		C("S", 14), // P1: A♠
		C("C", 9),  // P2: 9♣
		C("S", 11), // P3: J♠

		// burn for flop
		C("C", 3),

		// flop
		C("C", 13), // K♣
		C("D", 13), // K♦
		C("C", 12), // Q♣

		// burn for turn
		C("D", 4),

		// turn
		C("D", 12), // Q♦

		// burn for river
		C("H", 5),

		// river
		C("S", 2), // 2♠
	}

	// Pad deck with junk to reach 52 if your code ever expects it
	for len(deck) < 52 {
		deck = append(deck, C("C", 2))
	}
	h.deck = deck

	// Preflop betting we want:
	// 1) P1 raises to 1800 total (all-in)        -> amount=1800 since currentBet=0
	// 2) P2 re-raises so currentBet becomes 4400 -> amount=(4400-1800)=2600
	// 3) P3 calls to 4400                        -> Action "call" (your code computes amount)
	//
	// streetLoop will read from the acting player's pendingAction channel.
	// We enqueue exactly one action for each in turn.
	p1.pendingAction <- Action{PlayerID: "P1", Action: "raise", Amount: 1800}
	p2.pendingAction <- Action{PlayerID: "P2", Action: "raise", Amount: 2600}
	p3.pendingAction <- Action{PlayerID: "P3", Action: "call"}

	// Now run the full hand with your real logic, including side pots you built.
	h.run()

	// With correct side-pot logic, expected results:
	// - Main pot: 1800*3 = 5400 -> split P1 & P3 => 2700 each
	// - Side pot: (4400-1800)*2 = 5200 -> P3 wins
	// Final stacks:
	// P1: 1800 - 1800 + 2700       = 2700
	// P2: 4400 - 4400 + 0          = 0
	// P3: 10000 - 4400 + 2700+5200 = 13500
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
