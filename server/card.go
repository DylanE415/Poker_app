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

func getPlayerBestHand(h Hand, p Player) HandType {

}
