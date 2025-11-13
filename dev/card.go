package main

import (
	"sort"
	"strconv"
)

type Card struct {
	Suit string `json:"suit"`
	Rank string `json:"rank"`
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
	flushRanks           []int
	flushSuit            string
	//only for comparing high card and pair
	extraHighCards []int
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

func getCardFrequencies(h *Hand, p *Player) []CardFrequency {
	freqs := make(map[string]int)

	// count board cards
	for _, c := range h.board {
		freqs[c.Rank]++
	}
	// count player cards
	for _, c := range p.Hand {
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

func getSuitFrequencies(h *Hand, p *Player) ([]SuitFrequency, map[string]int) {
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
	for _, c := range p.Hand {
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
func isFlush(freqs []SuitFrequency, highestRanks map[string]int) (bool, int, string) {
	for _, s := range freqs {
		if s.Count >= 5 {
			return true, highestRanks[s.Suit], s.Suit
		}
	}
	return false, 0, ""
}

// isStraight returns true if the hand is a straight and returns the rank of the highest card in the straight
func isStraight(freqs []CardFrequency) (bool, int) {
	length := 0
	highestCardInStraight := rankToInt(freqs[0].Rank)
	for i := 0; i < len(freqs)-1; i++ {
		//check duplicate ad check if next card is 1 higher
		if rankToInt(freqs[i].Rank) == rankToInt(freqs[i+1].Rank) {
			continue
		} else if rankToInt(freqs[i].Rank) != rankToInt(freqs[i+1].Rank)+1 {
			length = 0
		}
		// if length is 0 and next card is 1 higher then we set that as the highest card of new possible straight
		if rankToInt(freqs[i].Rank) == rankToInt(freqs[i+1].Rank)+1 {
			if length == 0 {
				highestCardInStraight = rankToInt(freqs[i].Rank)
			}
			length++
		}
		if length == 4 {
			return true, highestCardInStraight
		}
	}
	length = 0
	highestCardInStraight = rankToInt(freqs[0].Rank)
	// re do the check but with ace set as 1
	//make a copy of freqs
	tmp := append([]CardFrequency(nil), freqs...)
	if freqs[0].Rank == "14" {
		tmp = append(tmp[1:], CardFrequency{Rank: "1", Count: tmp[0].Count})
		sort.Slice(tmp, func(i, j int) bool {
			return rankToInt(tmp[i].Rank) > rankToInt(tmp[j].Rank)
		})
		for i := 0; i < len(tmp)-1; i++ {
			if rankToInt(tmp[i].Rank) == rankToInt(tmp[i+1].Rank) {
				continue
			} else if rankToInt(tmp[i].Rank) != rankToInt(tmp[i+1].Rank)+1 {
				length = 0
			}

			if rankToInt(tmp[i].Rank) == rankToInt(tmp[i+1].Rank)+1 {
				if length == 0 {
					highestCardInStraight = rankToInt(tmp[i].Rank)
				}
				length++
			}
			if length == 4 {
				return true, highestCardInStraight
			}
		}

	}

	return false, 0
}

func getPlayerBestHand(h *Hand, p *Player) BestHand {
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
	flush, highestCardInFlush, flushSuit := isFlush(getSuitFrequencies(h, p))
	if flush {
		BestHand.Type = Flush
		BestHand.rank = highestCardInFlush
		BestHand.flushSuit = flushSuit
		// get ranks of all of the flush cards
		for _, card := range h.board {
			if card.Suit == flushSuit {
				BestHand.flushRanks = append(BestHand.flushRanks, rankToInt(card.Rank))
			}
		}
		for _, card := range p.Hand {
			if card.Suit == flushSuit {
				BestHand.flushRanks = append(BestHand.flushRanks, rankToInt(card.Rank))
			}
		}
		//sort and keep only 5 highest ranks
		sort.Ints(BestHand.flushRanks)
		if len(BestHand.flushRanks) > 5 {
			BestHand.flushRanks = BestHand.flushRanks[len(BestHand.flushRanks)-5:]
		}
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
			// kicker = 2 highest cards other than trips
			for _, kicker := range freqs {
				if kicker.Rank != rc.Rank {
					BestHand.extraHighCards = append(BestHand.extraHighCards, rankToInt(kicker.Rank))
					if len(BestHand.extraHighCards) == 2 {
						break
					}
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
			// get 3 highest cards other than pair(freqs is sorted high to low)
			for _, kicker := range freqs {
				if kicker.Rank != rc.Rank {
					BestHand.extraHighCards = append(BestHand.extraHighCards, rankToInt(kicker.Rank))
					if len(BestHand.extraHighCards) == 3 {
						break
					}
				}
			}
			return BestHand
		}
	}
	//high card
	BestHand.Type = HighCard
	BestHand.rank = rankToInt(freqs[0].Rank)
	//high card kickers(get 4 highest cards other than high card)
	for _, kicker := range freqs {
		if kicker.Rank != freqs[0].Rank {
			BestHand.extraHighCards = append(BestHand.extraHighCards, rankToInt(kicker.Rank))
			if len(BestHand.extraHighCards) == 4 {
				break
			}
		}
	}

	return BestHand
}

func handRank(ht HandType) int {
	switch ht {
	case HighCard:
		return 1
	case Pair:
		return 2
	case TwoPair:
		return 3
	case ThreeOfAKind:
		return 4
	case Straight:
		return 5
	case Flush:
		return 6
	case FullHouse:
		return 7
	case Quads:
		return 8
	case StraightFlush:
		return 9
	default:
		return 0
	}
}

// returns true if h1 is better than h2, false if h2 is better
// used in a sort.slicetable function to say yes this hand is higher order
func isHandBetter(h1, h2 BestHand) bool {
	//if hand type is different then easy comparison
	r1, r2 := handRank(h1.Type), handRank(h2.Type)
	if r1 != r2 {
		return r1 > r2
	}
	// if hand type is same compare the rank of the type
	if h1.rank != h2.rank {
		return h1.rank > h2.rank
	}
	// special case: Two Pair needs the second pair comparison if the ranks are the same
	if h1.Type == TwoPair {
		if h1.twoPairSecondaryRank != h2.twoPairSecondaryRank {
			return h1.twoPairSecondaryRank > h2.twoPairSecondaryRank
		}
	}
	//same with flush if the rank was the same(highest card in flush) then we need to compare the flush
	if h1.Type == Flush {
		//need to go throguh each card in the flush to find where one has a higher card
		for i := 0; i < len(h1.flushRanks); i++ {
			if h1.flushRanks[i] != h2.flushRanks[i] {
				return h1.flushRanks[i] > h2.flushRanks[i]
			}
		}

	}
	// for trips compare the 2 cards in extra high cards
	if h1.Type == ThreeOfAKind {
		for i := 0; i < len(h1.extraHighCards); i++ {
			if h1.extraHighCards[i] != h2.extraHighCards[i] {
				return h1.extraHighCards[i] > h2.extraHighCards[i]
			}
		}
	}
	//for pair compare the 3 cards in extra high cards
	if h1.Type == Pair {
		for i := 0; i < len(h1.extraHighCards); i++ {
			if h1.extraHighCards[i] != h2.extraHighCards[i] {
				return h1.extraHighCards[i] > h2.extraHighCards[i]
			}
		}
	}
	// final tiebreaker for high cards
	if h1.Type == HighCard {
		for i := 0; i < len(h1.extraHighCards); i++ {
			if h1.extraHighCards[i] != h2.extraHighCards[i] {
				return h1.extraHighCards[i] > h2.extraHighCards[i]
			}
		}
	}
	//for full house, quads
	return h1.kicker > h2.kicker

}

// for determining if there is a chop
func handsEqual(a, b BestHand) bool {
	return !isHandBetter(a, b) && !isHandBetter(b, a)
}

type playerHand struct {
	playerId string
	hand     BestHand
}

// pass in array of best hands and players corresponding ids to get showdown best hand, returns id and best hand
func getShowdownBestHand(playerHands []playerHand) []playerHand {
	if len(playerHands) == 0 {
		return []playerHand{}
	}

	//get best hand
	best := playerHands[0]
	for i := 1; i < len(playerHands); i++ {
		if isHandBetter(playerHands[i].hand, best.hand) {
			best = playerHands[i]
		}
	}

	//append everything equal to the best hand to account for chops(appends just the winner if no chop)
	winners := make([]playerHand, 0, len(playerHands))
	for i := 0; i < len(playerHands); i++ {
		if handsEqual(playerHands[i].hand, best.hand) {
			winners = append(winners, playerHands[i])
		}
	}
	return winners
}
