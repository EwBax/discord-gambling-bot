package main

import (
	"fmt"
	"strconv"

	"github.com/bwmarrin/discordgo"
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

// Implementing the stringer interface for blackjackhand
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
}

// NewBlackjack Initializes and returns a new game of blackjack. Creates and shuffles a new deck, then deals player and dealer hands.
func NewBlackjack(player Player, wager int) Blackjack {

	// Creating the new game
	newGame := Blackjack{player, wager, NewStandardDeck(), make(BlackjackHand, 0), make(BlackjackHand, 0), true}

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

	message := fmt.Sprintf("%s your hand is:\n\n%s", (*b).Player.Username, (*b).PlayerHand)

	return message
}

// GetDealerHand Returns a message with the dealer's hand
func (b *Blackjack) GetDealerHand() string {

	message := fmt.Sprintf("The dealer's hand is:\n\n%s", (*b).DealerHand)

	return message
}

// Hit deals a new card to the hand that is passed
func (b *Blackjack) Hit(h *BlackjackHand) {
	*h = append(*h, (*b).CardDeck.DealCard())
}

// RunPlayerTurn Handles the next Player turn in the game
func (b *Blackjack) RunPlayerTurn(s *discordgo.Session, m *discordgo.MessageCreate) {

	// If the player's hand is under 21, prompt them to hit or hold
	if (*b).PlayerHand.Value() < 21 {
		(*b).PromptPlayer(s, m)
		return
	} else if (*b).PlayerHand.Value() > 21 {
		// Player must have busted
		(*b).PlayerBust(s, m)
	} else {
		// The player's hand is 21, so they cannot hit anymore. Display their hand then move to dealer turn
		s.ChannelMessageSend(m.ChannelID, (*b).GetPlayerHand())
		(*b).RunDealerTurn(s, m)
	}

	// To get to this point the player either busts or holds
	(*b).IsPlayersTurn = false
}

// RunDealerTurn Handles the Dealer turns in the game
func (b *Blackjack) RunDealerTurn(s *discordgo.Session, m *discordgo.MessageCreate) {

	s.ChannelMessageSend(m.ChannelID, "It is the dealer's turn!")
	s.ChannelMessageSend(m.ChannelID, (*b).GetDealerHand())

	// Dealer hits on 16 or less
	for (*b).DealerHand.Value() < 17 {
		(*b).Hit(&(*b).DealerHand)
		s.ChannelMessageSend(m.ChannelID, "The dealer hits!\n"+(*b).GetDealerHand())
	}

	if (*b).DealerHand.Value() > 21 {
		s.ChannelMessageSend(m.ChannelID, "\nThe dealer busts, you win!\n")
	} else {
		s.ChannelMessageSend(m.ChannelID, "\nThe dealer holds!")

		// When the dealer's turn is over, display the results
		(*b).DisplayResults(s, m)
	}

	GameOver(s, m)

}

// PromptPlayer Prompts the player by displaying their hand then asking to hit or hold.
func (b *Blackjack) PromptPlayer(s *discordgo.Session, m *discordgo.MessageCreate) {

	message := b.GetPlayerHand()
	s.ChannelMessageSend(m.ChannelID, message)
	s.ChannelMessageSend(m.ChannelID, "Enter !hit to hit, !hold to hold.")

}

// PlayerBust Displays the player's hand and a message that they have busted
func (b *Blackjack) PlayerBust(s *discordgo.Session, m *discordgo.MessageCreate) {

	message := (*b).GetPlayerHand()
	s.ChannelMessageSend(m.ChannelID, message)
	s.ChannelMessageSend(m.ChannelID, "You bust! The dealer wins.")
	GameOver(s, m)

}

// PlayerHold Displays the player's hand and that they have chosen to hold.
func (b *Blackjack) PlayerHold(s *discordgo.Session, m *discordgo.MessageCreate) {

	message := (*b).GetPlayerHand()
	message += "\nYou hold! It is now the dealer's turn."
	s.ChannelMessageSend(m.ChannelID, message)

}

// DisplayResults Displays the final results of the game
func (b *Blackjack) DisplayResults(s *discordgo.Session, m *discordgo.MessageCreate) {

	message := "=======================\n\t\t\t\tRESULTS\n=======================\n\n"

	// Displays both hands
	message += (*b).GetPlayerHand() + "\n\n"
	message += (*b).GetDealerHand() + "\n\n"

	// Determining the winner. Dealer wins if their hand is >= player hand
	if (*b).DealerHand.Value() >= (*b).PlayerHand.Value() {
		s.ChannelMessageSend(m.ChannelID, message+"The dealer wins.")
		// If the player loses, they lose their wager. so we set it to negative here so when it is updated in game over,
		// the wager is subtracted
		b.Wager *= -1
	} else {
		s.ChannelMessageSend(m.ChannelID, message+(*b).Player.Username+" wins!")
	}
}
