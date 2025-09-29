package main

import (
	"fmt"
	"math"
	"testing"
)

// do a chi-squared test
func TestShuffleDistribution(t *testing.T) {
	const trials = 1000
	const runs = 1000
	const positions = 52

	df := float64(positions - 1)
	exp := float64(runs) / float64(positions)

	sum, sumsq := 0.0, 0.0

	for trial := 0; trial < trials; trial++ {
		players := []Player{{Name: "P1"}, {Name: "P2"}}
		counts := make(map[string]int, positions)

		for i := 0; i < runs; i++ {
			h := newHand(players, 0)
			card := h.deck[0] // position 0
			counts[card.Suit+card.Rank]++
		}

		var chi float64
		// loop all 52 categories
		suits := []string{"Spades", "Hearts", "Diamonds", "Clubs"}
		ranks := []string{"Ace", "2", "3", "4", "5", "6", "7", "8", "9", "10", "Jack", "Queen", "King"}
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
	variance := (sumsq / float64(trials)) - mean*mean
	if variance < 0 {
		variance = 0
	}

	sem := math.Sqrt(2*df) / math.Sqrt(float64(trials))
	fmt.Printf("avg chi² over %d trials (runs=%d, df=51): %.4f (std=%.4f)\n", trials, runs, mean, sem)

	// Expect mean ≈ df with tolerance ~3*SEM
	if math.Abs(mean-df) > 3*sem {
		t.Fatalf("mean chi² %.3f deviates > 3*SEM (%.3f) from df=%.0f", mean, 3*sem, df)
	}
}
