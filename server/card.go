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
	Type                 HandType
	rank                 int
	kicker               int
	twoPairSecondaryRank int
}

type CardFrequency struct {
	Rank  string
	Count int
}

type SuitFrequency struct {
	Suit        string
	highestRank string
	Count       int
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

func getSuitFrequencies(h Hand, p Player) ([]SuitFrequency, map[string]int) {
	freqs := make(map[string]int)
	highestRanks := map[string]int{
		"S": 0,
		"H": 0,
		"D": 0,
		"C": 0,
	}
	//add 1 to suit and if rank is higher than current highest rank
	update := func(c Card) {
		freqs[c.Suit]++

		rankVal, err := strconv.Atoi(c.Rank)
		if err != nil {
			return // ignore bad values
		}

		if rankVal > highestRanks[c.Suit] {
			highestRanks[c.Suit] = rankVal
		}
	}
	// count board cards
	for _, c := range h.board {
		update(c)
	}
	// count player cards
	for _, c := range p.hand {
		update(c)
	}
	// convert freqs into a slice
	result := make([]SuitFrequency, 0, len(freqs))
	for s, c := range freqs {
		result = append(result, SuitFrequency{Suit: s, Count: c})
	}
	return result, highestRanks
}

// isFlush returns true if the hand is a flush and returns the rank of the highest card
func isFlush(freqs []SuitFrequency, highestRanks map[string]int) (bool, int) {
	for _, s := range freqs {
		if s.Count == 5 {
			return true, highestRanks[s.Suit]
		}
	}
	return false, 0
}

// isStraight returns true if the hand is a straight and returns the rank of the highest card in the straight
func isStraight(freqs []CardFrequency) (bool, int) {
	biggestLength := 0
	length := 0
	highestCardInStraight := rankToInt(freqs[0].Rank)
	for i := 0; i < len(freqs)-1; i++ {
		// descending rank
		if rankToInt(freqs[i].Rank) != rankToInt(freqs[i+1].Rank)+1 {
			if length > biggestLength {
				biggestLength = length
			}
			length = 0
		}
		length++
	}

	if biggestLength > 4 {
		return true, highestCardInStraight
	} else {
		//re run with ace as 1
		if freqs[0].Rank == "14" {
			//remove ace from front and add 1 to the end
			freqs = append(freqs[1:], CardFrequency{Rank: "1", Count: freqs[0].Count})
			return isStraight(freqs)
		}

	}

	return false, 0
}

func getPlayerBestHand(h Hand, p Player) BestHand {

	BestHand := BestHand{}
	freqs := getCardFrequencies(h, p) // sorted high → low

	// QUADS: first rank with count == 4
	for _, rankCount := range freqs {
		if rankCount.Count == 4 {

			BestHand.Type = Quads
			BestHand.rank = rankToInt(rankCount.Rank)
			// kicker = highest card other than quads
			for _, kicker := range freqs {
				if kicker.Rank != rankCount.Rank {
					BestHand.kicker = rankToInt(kicker.Rank)
					break
				}
			}
			return BestHand
		}
	}

	//check full house(any card has 3 frequencies and any other has 2 frequencies)
	for _, rankCount := range freqs {
		if rankCount.Count == 3 {
			for _, rankCount2 := range freqs {
				if rankCount2.Count == 2 {
					BestHand.Type = FullHouse
					BestHand.rank = rankToInt(rankCount.Rank)
					BestHand.kicker = rankToInt(rankCount2.Rank)
					return BestHand
				}
			}
		}
	}

	// check flush(any suit has 5 frequencies)
	flush, highestCardInFlush := isFlush(getSuitFrequencies(h, p))
	if flush {
		BestHand.Type = Flush
		BestHand.rank = highestCardInFlush
		return BestHand
	}

	//check straight
	straight, highestCardInStraight := isStraight(freqs)
	if straight {
		BestHand.Type = Straight
		BestHand.rank = highestCardInStraight
		return BestHand
	}

	//check trips
	for _, rc := range freqs {
		if rc.Count == 3 {
			BestHand.Type = ThreeOfAKind
			BestHand.rank = rankToInt(rc.Rank)
			// kicker = highest card other than trips
			for _, kicker := range freqs {
				if kicker.Rank != rc.Rank {
					BestHand.kicker = rankToInt(kicker.Rank)
					break
				}
			}
			return BestHand
		}
	}

	//check twopair, loop till pair, then another loop starting at next index to find another pair, then find highest kicker
	for i := 0; i < len(freqs)-1; i++ {
		if freqs[i].Count == 2 {
			for j := i + 1; j < len(freqs); j++ {
				if freqs[j].Count == 2 {
					BestHand.Type = TwoPair
					BestHand.rank = rankToInt(freqs[i].Rank)
					BestHand.twoPairSecondaryRank = rankToInt(freqs[j].Rank)
					for _, kicker := range freqs {
						if kicker.Rank != freqs[i].Rank && kicker.Rank != freqs[j].Rank {
							BestHand.kicker = rankToInt(kicker.Rank)
							break
						}
					}
					return BestHand
				}
			}
		}
	}

	//check pair
	for _, rc := range freqs {
		if rc.Count == 2 {
			BestHand.Type = Pair
			BestHand.rank = rankToInt(rc.Rank)
			// kicker = highest card other than pair
			for _, kicker := range freqs {
				if kicker.Rank != rc.Rank {
					BestHand.kicker = rankToInt(kicker.Rank)
					break
				}
			}
			return BestHand
		}
	}

	return BestHand
}
