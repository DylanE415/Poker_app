package main

import (
	"strconv"
	"testing"
)

// helper to build a card with numeric rank (Ace = 14)
func C(suit string, rank int) Card {
	return Card{Suit: suit, Rank: strconv.Itoa(rank)}
}

func evalTwo(board []Card, h1 []Card, h2 []Card) (BestHand, BestHand) {
	h := Hand{}
	p1 := Player{ID: "P1", Name: "P1"}
	p2 := Player{ID: "P2", Name: "P2"}
	h.board = append([]Card{}, board...)
	p1.hand = append([]Card{}, h1...)
	p2.hand = append([]Card{}, h2...)
	b1 := getPlayerBestHand(&h, &p1) // your function taking *Hand, *Player
	b2 := getPlayerBestHand(&h, &p2)
	return b1, b2
}

// expect: 0 = tie, 1 = p1 wins, 2 = p2 wins
type cmp int

const (
	tie cmp = iota
	p1
	p2
)

func checkCompare(t *testing.T, name string, board, h1, h2 []Card, expect cmp, wantType HandType, note string) {
	t.Run(name, func(t *testing.T) {
		b1, b2 := evalTwo(board, h1, h2)

		switch expect {
		case p1:
			if !isHandBetter(b1, b2) {
				t.Fatalf("expected P1 to win. note: %s\nP1=%+v\nP2=%+v", note, b1, b2)
			}
			if isHandBetter(b2, b1) {
				t.Fatalf("asymmetry: P2 also better than P1")
			}
		case p2:
			if !isHandBetter(b2, b1) {
				t.Fatalf("expected P2 to win. note: %s\nP1=%+v\nP2=%+v", note, b1, b2)
			}
			if isHandBetter(b1, b2) {
				t.Fatalf("asymmetry: P1 also better than P2")
			}
		case tie:
			if !handsEqual(b1, b2) {
				t.Fatalf("expected tie. note: %s\nP1=%+v\nP2=%+v", note, b1, b2)
			}
		}
	})
}

func TestCompare_TwoPlayers_KickersAndEdges(t *testing.T) {
	// 1) High card vs high card, kicker decides (A K Q vs A K J)
	checkCompare(t,
		"HighCard kicker breaks tie",
		[]Card{C("C", 14), C("D", 7), C("S", 5), C("H", 4), C("C", 2)},
		[]Card{C("D", 13), C("C", 12)}, // K,Q
		[]Card{C("S", 13), C("D", 11)}, // K,J
		p1, HighCard, "A-high: compare K then Q vs J")

	// 2) Pair on board, kicker battle (Q Q … ; A K vs A J)
	checkCompare(t,
		"Pair on board; kickers decide",
		[]Card{C("C", 12), C("D", 12), C("S", 7), C("H", 5), C("D", 2)}, // Q Q 7 5 2
		[]Card{C("C", 14), C("C", 13)},                                  // A K
		[]Card{C("D", 14), C("D", 11)},                                  // A J
		p1, Pair, "Both have pair of Q; K kicker beats J")

	// 3) Pair (not on board), kicker stack
	checkCompare(t,
		"Pair with kicker hierarchy",
		[]Card{C("C", 9), C("D", 7), C("S", 4), C("H", 3), C("D", 2)},
		[]Card{C("S", 13), C("C", 9)}, // Pair 9s with A,K kickers
		[]Card{C("H", 13), C("D", 8)}, // Pair 8s only; P1 should win
		p1, Pair, "9s vs 8s")

	// 4) Two pair same top pair, different second pair
	checkCompare(t,
		"TwoPair same top-pair; second pair decides",
		[]Card{C("C", 13), C("D", 13), C("S", 5), C("H", 4), C("D", 2)}, // K K 5 4 2
		[]Card{C("S", 5), C("C", 14)},                                   // K K 5 (A kicker)
		[]Card{C("H", 4), C("D", 14)},                                   // K K 4 (A kicker)
		p1, TwoPair, "KK+5 vs KK+4")

	// 5) Two pair same pairs, different kicker
	checkCompare(t,
		"TwoPair same pairs; kicker decides",
		[]Card{C("C", 13), C("D", 13), C("S", 5), C("H", 5), C("D", 2)}, // K K 5 5 2
		[]Card{C("S", 14), C("C", 12)},                                  // kicker A
		[]Card{C("H", 14), C("D", 11)},                                  // kicker A
		tie, TwoPair, "KK55A-Q vs KK55A-J")

	// 6) Trips with kicker comparison (lexicographic)
	checkCompare(t,
		"Trips with kicker comparison",
		[]Card{C("D", 12), C("C", 12), C("H", 12), C("D", 7), C("C", 2)}, // Q Q Q
		[]Card{C("H", 14), C("C", 13)},                                   // A,K kickers
		[]Card{C("H", 14), C("D", 11)},                                   // A,J kickers
		p1, ThreeOfAKind, "Trips Q; A,K beats A,J")

	// 7) Flush lexicographic compare: A,K,9,7,4 vs A,Q,9,7,4
	checkCompare(t,
		"Flush: compare flushRanks lexicographically",
		[]Card{C("S", 14), C("S", 9), C("S", 7), C("S", 4), C("D", 2)}, // 4 spades on board
		[]Card{C("S", 13), C("D", 12)},                                 // K♠ + offsuit -> A,K,9,7,4
		[]Card{C("S", 12), C("S", 10)},                                 // Q♠ T♠ -> A,Q,9,7,4
		p1, Flush, "K in second position beats Q")

	// 8) Straight tie from board (everyone plays the board)
	checkCompare(t,
		"Straight tie (board straight)",
		[]Card{C("C", 5), C("D", 6), C("S", 7), C("H", 8), C("D", 9)},
		[]Card{C("S", 14), C("C", 2)},
		[]Card{C("H", 3), C("D", 4)},
		tie, Straight, "Both 9-high straight from board")

	// 9) Wheel (A-2-3-4-5) vs no straight
	checkCompare(t,
		"Wheel vs no straight",
		[]Card{C("D", 2), C("C", 3), C("H", 4), C("D", 9), C("C", 13)},
		[]Card{C("S", 14), C("C", 5)},  // A,5 -> wheel
		[]Card{C("S", 14), C("D", 12)}, // A,Q -> no straight
		p1, Straight, "5-high straight")

	// 10) Full house tiebreak (trips equal; pair decides)
	// NOTE: With your current isHandBetter, this may FAIL because FullHouse comparison
	//       doesn't compare the pair (you store it in kicker, but isHandBetter
	//       doesn't check kicker for FullHouse). That's intentional: to expose the gap.
	checkCompare(t,
		"FullHouse compare pair when trips equal (EXPECTED TO FAIL until fixed)",
		[]Card{C("C", 13), C("D", 13), C("S", 9), C("H", 9), C("D", 2)}, // K K 9 9 2
		[]Card{C("S", 13), C("C", 4)},                                   // K + x  => KKK99
		[]Card{C("H", 13), C("D", 2)},                                   // K + 2  => KKK22
		tie, FullHouse, "KKK99 beats KKK22")
}
