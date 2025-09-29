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
