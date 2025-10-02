package main


import (
	"fmt"
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
	Quads   HandType = "four of a kind"
	StraightFlush HandType = "straight flush"
)

func rankToInt(rank string) int {
	v, err := strconv.Atoi(rank)
	if err != nil {
		return 0
	}
	return v
}


func getPlayerBestHand(h Hand, p Player) map[HandType][]Card {

	freqs := getCardFrequencies(h, p)
	// check quads(any card has 4 frequencies)
	for rank, freq := range freqs {
		if freq == 4 {
			return Quads
		}
	}
	//check full house(any card has 3 frequencies and any other has 2 frequencies)
}

func isStraightFlush(h Hand, p Player) bool {

}


func isFullHouse(h Hand, p Player) bool {
	if getCardFrequencies()

}

func isFlush(h Hand, p Player) bool {

}

func isStraight(h Hand, p Player) bool {

}

func isThreeOfAKind(h Hand, p Player) bool {

}

func isTwoPair(h Hand, p Player) bool {

}

func isPair(h Hand, p Player) bool {

}

func getCardFrequencies(h Hand, p Player) []struct {Rank  string Count int} {
	freqs := make(map[string]int)

	// count board cards
	for _, c := range h.board {
		freqs[c.Rank]++
	}
	// count player cards
	for _, c := range p.hand {
		freqs[c.Rank]++
	}

	// flatten into slice
	type rankCount struct {
		Rank  string
		Count int
	}

	result := make([]rankCount, 0, len(freqs))
	for r, c := range freqs {
		result = append(result, rankCount{Rank: r, Count: c})
	}

	// sort by numeric rank descending
	sort.Slice(result, func(i, j int) bool {
		return rankToInt(result[i].Rank) > rankToInt(result[j].Rank)
	})

	return result
}

func getSuitFrequencies(h Hand, p Player) []struct {Suit  string Count int} {
	freqs := make(map[string]int)

	// count board cards
	for _, c := range h.board {
		freqs[c.Suit]++
	}
	// count player cards
	for _, c := range p.hand {
		freqs[c.Suit]++
	}

	// flatten into slice
	type rankCount struct {
		Suit  string
		Count int
	}

	result := make([]rankCount, 0, len(freqs))
	for r, c := range freqs {
		result = append(result, rankCount{Rank: r, Count: c})
	}

	return result
}
