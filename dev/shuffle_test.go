package main

import (
	"fmt"
	"math"
	"testing"
	"time"
)

// Verifies that the top card after calling shuffleDeck is ~uniform over all 52 cards.
// IMPORTANT: This test calls shuffleDeck(deck) directly (not newHand) so changes to the
// shuffler are covered by the test.
func TestShuffleDistribution_UsesShuffleDeck(t *testing.T) {
	const trials = 200 // independent experiments (averaged)
	const runs = 1000  // shuffles per experiment
	const positions = 52

	df := float64(positions - 1)              // 51
	exp := float64(runs) / float64(positions) // expected count per card

	sum, sumsq := 0.0, 0.0

	// Reference ordering EXACTLY like newHand builds the deck
	suits := []string{"S", "H", "D", "C"}
	ranks := []string{"14", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12", "13"}

	makeOrderedDeck := func() []Card {
		d := make([]Card, 0, 52)
		for _, s := range suits {
			for _, r := range ranks {
				d = append(d, Card{Suit: s, Rank: r})
			}
		}
		return d
	}

	for trial := 0; trial < trials; trial++ {
		// counts keyed like your deck: "S14", "H2", ...
		counts := make(map[string]int, positions)

		deck := makeOrderedDeck()
		for i := 0; i < runs; i++ {
			// Shuffle IN PLACE using your function
			shuffleDeck(deck)

			// Take the top card and count it
			top := deck[0]
			counts[top.Suit+top.Rank]++

			// Because shuffleDeck reseeds with time.Now().UnixNano() on each call,
			// add a tiny sleep occasionally to avoid repeated seeds in a tight loop.
			if i&7 == 0 {
				time.Sleep(time.Microsecond)
			}
		}

		// Compute chi-square across the exact 52 categories your deck uses
		var chi float64
		for _, s := range suits {
			for _, r := range ranks {
				obs := float64(counts[s+r])
				d := obs - exp
				chi += d * d / exp
			}
		}

		sum += chi
		sumsq += chi * chi
	}

	mean := sum / float64(trials)
	sem := math.Sqrt(2 * df / float64(trials)) // standard error of the mean chi-square

	fmt.Printf("avg chi² over %d trials (runs=%d, df=%d): %.4f (SEM=%.4f)\n",
		trials, runs, int(df), mean, sem)

	// Mean should be close to df within ~3*SEM
	if math.Abs(mean-df) > 3*sem {
		t.Fatalf("mean chi² %.3f deviates > 3*SEM (%.3f) from df=%.0f", mean, 3*sem, df)
	}
}
