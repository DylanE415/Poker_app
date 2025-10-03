package main

import (
	"sort"
	"strconv"
)

type Card struct {
	Suit string
	Rank string
}

// all hand combos (pairs, sets, etc)
type HandType string

const (
	HighCard      HandType = "high card"
	Pair          HandType = "pair"
	TwoPair       HandType = "two pair"
	ThreeOfAKind  HandType = "three of a kind"
	Straight      HandType = "straight"
	Flush         HandType = "flush"
	FullHouse     HandType = "full house"
	Quads         HandType = "four of a kind"
	StraightFlush HandType = "straight flush"
)

type BestHand struct {
	Type   HandType
	rank   int
	kicker int
}

type CardFrequency struct {
	Rank  string
	Count int
}

type SuitFrequency struct {
	Suit  string
	Count int
}

func rankToInt(rank string) int {
	v, err := strconv.Atoi(rank)
	if err != nil {
		return 0
	}
	return v
}

func getCardFrequencies(h Hand, p Player) []CardFrequency {
	freqs := make(map[string]int)

	// count board cards
	for _, c := range h.board {
		freqs[c.Rank]++
	}
	// count player cards
	for _, c := range p.hand {
		freqs[c.Rank]++
	}

	result := make([]CardFrequency, 0, len(freqs))
	for r, c := range freqs {
		result = append(result, CardFrequency{Rank: r, Count: c})
	}

	// sort by numeric rank descending
	sort.Slice(result, func(i, j int) bool {
		return rankToInt(result[i].Rank) > rankToInt(result[j].Rank)
	})

	return result
}

func getSuitFrequencies(h Hand, p Player) []SuitFrequency {
	freqs := make(map[string]int)

	// count board cards
	for _, c := range h.board {
		freqs[c.Suit]++
	}
	// count player cards
	for _, c := range p.hand {
		freqs[c.Suit]++
	}

	result := make([]SuitFrequency, 0, len(freqs))
	for s, c := range freqs {
		result = append(result, SuitFrequency{Suit: s, Count: c})
	}

	return result
}

func isStraight(freqs []CardFrequency) (bool, int) {
	biggestLength := 0
	length := 0
	highestCardInStraight := rankToInt(freqs[0].Rank)
	for i := 0; i < len(freqs)-1; i++ {
		// special case: A,2,3,4,5
		if rankToInt(freqs[i].Rank)+1 != rankToInt(freqs[i+1].Rank) || (freqs[i].Rank == "14" && freqs[i+1].Rank != "2") {
			if length > biggestLength {
				biggestLength = length
			}
			length = 0
		}
		length++
	}
	return (biggestLength >= 4), (highestCardInStraight)
}

func getPlayerBestHand(h Hand, p Player) BestHand {

	BestHand := BestHand{}
	freqs := getCardFrequencies(h, p) // sorted high → low

	// QUADS: first rank with count == 4
	for _, rc := range freqs {
		if rc.Count == 4 {

			BestHand.Type = Quads
			BestHand.rank = rankToInt(rc.Rank)
			// kicker = highest card other than quads
			for _, kicker := range freqs {
				if kicker.Rank != rc.Rank {
					BestHand.kicker = rankToInt(kicker.Rank)
					break
				}
			}
			return BestHand
		}
	}

	//check full house(any card has 3 frequencies and any other has 2 frequencies)
	for _, rc := range freqs {
		if rc.Count == 3 {
			for _, rc2 := range freqs {
				if rc2.Count == 2 {
					BestHand.Type = FullHouse
					BestHand.rank = rankToInt(rc.Rank)
					BestHand.kicker = rankToInt(rc2.Rank)
					return BestHand
				}
			}
		}
	}

	// check flush(any card has 5 frequencies)
	return BestHand
}
