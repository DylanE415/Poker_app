package main

import (
	"math/rand"
	"time"
)

type Hand struct {
	Players           []Player
	actionPlayerIndex int
	deck              []Card
}

func shuffleDeck(deck []Card) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := len(deck) - 1; i > 0; i-- {
		j := r.Intn(i + 1)
		deck[i], deck[j] = deck[j], deck[i]
	}
}
func newHand(players []Player, actionPlayerIndex int) Hand {

	suits := []string{"Spades", "Hearts", "Diamonds", "Clubs"}
	ranks := []string{"Ace", "2", "3", "4", "5", "6", "7", "8", "9", "10", "Jack", "Queen", "King"}
	deck := make([]Card, 0)

	for _, suit := range suits {
		for _, rank := range ranks {
			deck = append(deck, Card{Suit: suit, Rank: rank})
		}
	}

	shuffleDeck(deck)

	return Hand{
		Players:           players,
		actionPlayerIndex: actionPlayerIndex,
		deck:              deck,
	}
}
