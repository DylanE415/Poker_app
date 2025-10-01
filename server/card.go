package main

const (
	Spades   = "s"
	Hearts   = "h"
	Diamonds = "d"
	Clubs    = "c"
)

const (
	Ace   = "a"
	Two   = "2"
	Three = "3"
	Four  = "4"
	Five  = "5"
	Six   = "6"
	Seven = "7"
	Eight = "8"
	Nine  = "9"
	Ten   = "10"
	Jack  = "j"
	Queen = "q"
	King  = "k"
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
	FourOfAKind   HandType = "four of a kind"
	StraightFlush HandType = "straight flush"
)

func isStraightFlush(h Hand, p Player) bool {

}

func isQuads(h Hand, p Player) bool {
	// if either of the players cards has freq of 4 then return true
	if getCardFrequencies(h, p)[p.hand[0].Rank] == 4 || getCardFrequencies(h, p)[p.hand[1].Rank] == 4 {
		return true
	}
	return false
}

func isFullHouse(h Hand, p Player) bool {

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

func getCardFrequencies(h Hand, p Player) map[string]int {

	freqs := make(map[string]int)
	for i := range h.board {
		freqs[h.board[i].Rank]++
	}
	for i := range p.hand {
		freqs[p.hand[i].Rank]++
	}
	return freqs
}

func getSuitFrequencies(h Hand, p Player) map[string]int {
	freqs := make(map[string]int)
	for i := range h.board {
		freqs[h.board[i].Suit]++
	}
	for i := range p.hand {
		freqs[p.hand[i].Suit]++
	}
	return freqs
}

func getPlayerBestHand(h Hand, p Player) HandType {

}
