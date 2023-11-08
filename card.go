// Package card implements playing cards.
// This includes a single playing card with rank and suit, as well as a deck of cards.
package main

import (
	"math/rand"
	"time"
)

// Represents a single playing card
type Card struct {
	Rank string
	Suit string
}

// Implementing the stringer interface
func (c Card) String() string {
	return c.Rank + " of " + c.Suit
}

// Ranks for a typical deck of playing cards, mapped to their values
var ranks = map[string]int{"Ace": 1, "Two": 2, "Three": 3, "Four": 4, "Five": 5, "Six": 6, "Seven": 7, "Eight": 8, "Nine": 9, "Ten": 10, "Jack": 10, "Queen": 10, "King": 10}

// Represents a deck of playing cards
type Deck []Card

// Creates and returns a standard deck of 52 playing cards
func NewStandardDeck() Deck {
	var deck Deck

	suits := []string{"Hearts", "Clubs", "Diamonds", "Spades"}

	// Looping through suits and ranks and creating a card of each combination, then adding to deck
	for _, suit := range suits {
		for rank := range ranks {
			deck = append(deck, Card{rank, suit})
		}
	}

	return deck

}

// Implements the stringer function for a deck, listing out the contents of the deck on a new line.
func (d Deck) String() string {

	message := ""

	for i, c := range d {
		message += c.String()
		if i < len(d)-1 {
			message += ",\n"
		}
	}

	return message

}

// Shuffles a deck of cards
func (d Deck) Shuffle() {

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := range d {
		j := rng.Intn(len(d))
		d[i], d[j] = d[j], d[i]
	}

}

// Returns the top card on the deck (last in the slice) and removes it from the deck.
func (d *Deck) DealCard() Card {

	if len(*d) <= 0 {
		panic("Deck has no cards.")
	}

	// getting the last card
	card := (*d)[len(*d)-1]

	// slicing off the last card
	*d = (*d)[:len(*d)-1]

	return card

}
