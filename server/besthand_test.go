package main

import (
	"strconv"
	"testing"
)

// helper to build a card with numeric rank (Ace = 14)
func C(suit string, rank int) Card {
	return Card{Suit: suit, Rank: strconv.Itoa(rank)}
}

func TestGetPlayerBestHand_AllTypes(t *testing.T) {
	type want struct {
		typ                  HandType
		rank                 int
		kicker               int
		twoPairSecondaryRank int
	}

	tests := []struct {
		name  string
		board []Card
		hand  []Card
		want  want
	}{
		{
			name:  "Quads Aces with K kicker",
			board: []Card{C("S", 14), C("H", 14), C("D", 14), C("C", 7), C("C", 2)},
			hand:  []Card{C("C", 14), C("D", 13)}, // A♣ K♦
			want:  want{typ: Quads, rank: 14, kicker: 13},
		},
		{
			name:  "Full House KKK99",
			board: []Card{C("S", 13), C("H", 13), C("D", 9), C("C", 2), C("C", 3)},
			hand:  []Card{C("D", 13), C("C", 9)},
			want:  want{typ: FullHouse, rank: 13, kicker: 9},
		},
		{
			name:  "Flush in spades J-high",
			board: []Card{C("S", 2), C("S", 7), C("S", 11), C("D", 3), C("H", 4)},
			hand:  []Card{C("S", 5), C("S", 9)},
			want:  want{typ: Flush, rank: 11},
		},
		{
			name:  "Straight 8-9-T-J-Q (Q high)",
			board: []Card{C("D", 9), C("C", 10), C("H", 11), C("D", 3), C("H", 2)},
			hand:  []Card{C("S", 12), C("C", 8)},
			want:  want{typ: Straight, rank: 12},
		},
		{
			name:  "Straight 4-5-6-7-8, with a duplicate 7 (8 high)",
			board: []Card{C("D", 7), C("C", 7), C("H", 4), C("D", 5), C("H", 6)},
			hand:  []Card{C("S", 13), C("C", 8)},
			want:  want{typ: Straight, rank: 8},
		},
		{
			name:  "Wheel straight A-2-3-4-5 (5 high under low-ace rule)",
			board: []Card{C("D", 2), C("C", 3), C("H", 4), C("D", 9), C("D", 11)},
			hand:  []Card{C("S", 14), C("C", 5)},
			want:  want{typ: Straight, rank: 5},
		},
		{
			name:  "Trips Queens with A kicker",
			board: []Card{C("D", 12), C("C", 12), C("H", 7), C("D", 2), C("C", 9)},
			hand:  []Card{C("H", 12), C("C", 14)},
			want:  want{typ: ThreeOfAKind, rank: 12, kicker: 14},
		},
		{
			name:  "Two Pair Jacks and Nines with A kicker",
			board: []Card{C("D", 11), C("C", 11), C("H", 4), C("D", 2), C("C", 9)},
			hand:  []Card{C("H", 9), C("C", 14)},
			// Your code sets rank to the higher pair first (freqs sorted high→low), and twoPairSecondaryRank to the lower pair.
			want: want{typ: TwoPair, rank: 11, twoPairSecondaryRank: 9, kicker: 14},
		},
		{
			name:  "Pair of Kings with A kicker",
			board: []Card{C("D", 13), C("C", 7), C("H", 4), C("D", 2), C("C", 9)},
			hand:  []Card{C("C", 13), C("C", 14)},
			want:  want{typ: Pair, rank: 13, kicker: 14},
		},
		// If you later add High Card handling, add a case here.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := Hand{}
			p := Player{Name: "P1"}
			h.board = append([]Card{}, tt.board...)
			p.hand = append([]Card{}, tt.hand...)

			got := getPlayerBestHand(&h, &p) // ✅ pass pointer to h now

			if got.Type != tt.want.typ {
				t.Fatalf("Type = %v, want %v", got.Type, tt.want.typ)
			}
			if got.rank != tt.want.rank {
				t.Fatalf("rank = %d, want %d", got.rank, tt.want.rank)
			}

			switch tt.want.typ {
			case Quads, ThreeOfAKind, TwoPair, Pair:
				if got.kicker != tt.want.kicker {
					t.Fatalf("kicker = %d, want %d", got.kicker, tt.want.kicker)
				}
			}
			if tt.want.typ == TwoPair {
				if got.twoPairSecondaryRank != tt.want.twoPairSecondaryRank {
					t.Fatalf("twoPairSecondaryRank = %d, want %d",
						got.twoPairSecondaryRank, tt.want.twoPairSecondaryRank)
				}
			}
		})
	}

}
