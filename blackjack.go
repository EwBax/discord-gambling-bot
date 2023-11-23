package main

import (
	"fmt"
	"strconv"
)

// BlackjackHand Represents a player's hand in a game of blackjack.
type BlackjackHand []Card

// Value Returns the blackjack value of a player's hand.
func (h BlackjackHand) Value() int {

	value := 0
	numAces := 0

	// Getting the value and the number of aces in the deck
	for i := range h {
		value += ranks[h[i].Rank]
		if h[i].Rank == "Ace" {
			numAces++
		}
	}

	// If there are more aces and using one as an 11 would not cause the hand to bust, increase value by 10
	for numAces > 0 && value+10 <= 21 {
		value += 10
		numAces--
	}

	return value

}

// SoftSeventeen Returns whether the value of the hand is a soft 17. A soft is a value with an ace counting as an 11.
func (h BlackjackHand) SoftSeventeen() bool {

	value := 0
	numAces := 0

	// Getting the value and the number of aces in the deck
	for i := range h {
		value += ranks[h[i].Rank]
		if h[i].Rank == "Ace" {
			numAces++
		}
	}

	soft := false

	// If there are more aces and using one as an 11 would not cause the hand to bust, it is a soft value
	if numAces > 0 && value+10 <= 21 {
		value += 10
		numAces--
		soft = true
	}

	return value == 17 && soft

}

// Implementing the stringer interface for BlackjackHand
func (h BlackjackHand) String() string {

	message := ""

	// Making a message with the hand, adding each card to the message
	for i, card := range h {
		message += fmt.Sprintf("%s", card)
		if i < len(h)-1 {
			message += ","
		}
		message += "\n"
	}

	message += "\nThe value is: " + strconv.Itoa(h.Value())

	return message
}

// Blackjack Object representing a game of blackjack
type Blackjack struct {
	Player        Player
	Wager         int
	CardDeck      Deck
	PlayerHand    BlackjackHand
	DealerHand    BlackjackHand
	IsPlayersTurn bool
	ChannelID     string
}

// NewBlackjack Initializes and returns a new game of blackjack. Creates and shuffles a new deck, then deals player and dealer hands.
func NewBlackjack(player Player, wager int) Blackjack {

	// Creating the new game
	newGame := Blackjack{Player: player, Wager: wager, CardDeck: NewStandardDeck(), PlayerHand: make(BlackjackHand, 0),
		DealerHand: make(BlackjackHand, 0), IsPlayersTurn: true}

	// shuffling deck
	newGame.CardDeck.Shuffle()
	newGame.CardDeck.Shuffle()
	newGame.CardDeck.Shuffle()

	// Dealing player hand
	newGame.PlayerHand = append(newGame.PlayerHand, newGame.CardDeck.DealCard())
	newGame.PlayerHand = append(newGame.PlayerHand, newGame.CardDeck.DealCard())

	// Dealer hand
	newGame.DealerHand = append(newGame.DealerHand, newGame.CardDeck.DealCard())
	newGame.DealerHand = append(newGame.DealerHand, newGame.CardDeck.DealCard())

	return newGame

}

// GetPlayerHand Returns a message with the player's hand
func (b *Blackjack) GetPlayerHand() string {

	message := fmt.Sprintf("%s, your hand is:\n\n%s", (*b).Player.Username, (*b).PlayerHand)

	return message
}

// GetFullDealerHand Returns a message with the dealer's full hand
func (b *Blackjack) GetFullDealerHand() string {

	message := fmt.Sprintf("The dealer's hand is:\n\n%s", (*b).DealerHand)

	return message
}

// GetDealerHand Returns a message with the dealer's first card revealed, and the second card hidden. Only used at the start of the game, so player can see dealer's hand
// One card is hidden for the dealer in blackjack rules
func (b *Blackjack) GetDealerHand() string {

	message := fmt.Sprintf("The dealer's hand is:\n\n%s\n[Hidden Card]", (*b).DealerHand[0])

	return message

}

// Hit deals a new card to the hand that is passed
func (b *Blackjack) Hit(h *BlackjackHand) {
	*h = append(*h, (*b).CardDeck.DealCard())
}

// RunPlayerTurn Handles the next Player turn in the game
func (b *Blackjack) RunPlayerTurn() string {

	var message string

	// If the player's hand is under 21, prompt them to hit or stand
	if (*b).PlayerHand.Value() < 21 {
		message = fmt.Sprintln((*b).PromptPlayer())
	} else if (*b).PlayerHand.Value() > 21 {
		// Player must have busted
		message = fmt.Sprintln((*b).PlayerBust())
	} else {
		// The player's hand is 21, so they cannot hit anymore. Display their hand then move to dealer turn
		message = fmt.Sprintln((*b).GetPlayerHand())
		// To get to this point the player either busts or stands
		(*b).IsPlayersTurn = false
	}

	return message

}

// RunDealerTurn Handles the Dealer turns in the game
func (b *Blackjack) RunDealerTurn() {

	//// This is the logic for casino blackjack where the Dealer is playing against more than one player
	//// The dealer hits on anything less than 17, and also hits on a soft 17 (has an ace counting as 11)
	//for (*b).DealerHand.Value() < 17 || (*b).DealerHand.SoftSeventeen() {
	//	(*b).Hit(&(*b).DealerHand)
	//}

	// This is the logic for 1v1 blackjack. The Dealer can see the player's hand and will continue to hit until they either beat them or bust
	// If they are tied, the dealer will hit on a soft 17 or lower
	for ((*b).DealerHand.Value() < (*b).PlayerHand.Value() && (*b).DealerHand.Value() < 21) ||
		((*b).DealerHand.Value() == (*b).PlayerHand.Value() && ((*b).DealerHand.Value() < 17 || (*b).DealerHand.SoftSeventeen())) {
		(*b).Hit(&(*b).DealerHand)
	}

}

// PromptPlayer Prompts the player by displaying their hand then asking to hit or stand.
func (b *Blackjack) PromptPlayer() string {

	message := fmt.Sprintln((*b).GetPlayerHand())
	message += "Enter !hit to hit, !stand to stand."

	return message

}

// PlayerBust Displays the player's hand and a message that they have busted
func (b *Blackjack) PlayerBust() string {

	message := fmt.Sprintln((*b).GetPlayerHand())
	message += "\nUh oh, you bust!"

	(*b).IsPlayersTurn = false

	return message

}

// PlayerStand Displays the player's hand and that they have chosen to stand.
func (b *Blackjack) PlayerStand() string {

	(*b).IsPlayersTurn = false
	return "\nYou stand! It is now the dealer's turn."

}

// PlayerHit Displays feedback to the player and deals them a new card
func (b *Blackjack) PlayerHit() string {

	BlackjackGame.Hit(&BlackjackGame.PlayerHand)
	return "You chose to hit!"

}

// Results Displays the final results of the game
func (b *Blackjack) Results() string {

	message := "=======================\n\t\t\t\tRESULTS\n=======================\n\n"

	// Displays both hands
	message += (*b).GetPlayerHand() + "\n\n"
	message += (*b).GetFullDealerHand() + "\n\n"

	// Determining the winner. Dealer wins if their hand is > player hand, and not above 21
	if (*b).PlayerHand.Value() > 21 || ((*b).DealerHand.Value() > (*b).PlayerHand.Value() && (*b).DealerHand.Value() <= 21) {
		message += "The dealer wins."
		// If the player loses, they lose their wager. so we set it to negative here so when it is updated in game over,
		// the wager is subtracted
		(*b).Wager *= -1
		// Updating the player's losses stat
		(*b).Player.Losses++
	} else if (*b).DealerHand.Value() == (*b).PlayerHand.Value() {
		// Draw, set wager to 0, so they get their chips back
		message += "It's a draw!"
		(*b).Wager = 0
		// Updating the player's ties stat
		(*b).Player.Ties++
	} else {
		message += (*b).Player.Username + " wins!"
		// Updating the player's wins stat
		(*b).Player.Wins++
	}

	return message

}
