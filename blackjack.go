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

// HasAce Returns whether the hand contains an Ace.
// TODO refactor this to find a soft 17
func (h BlackjackHand) HasAce() bool {

	for i := range h {
		if h[i].Rank == "Ace" {
			return true
		}
	}
	return false

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
func (b *Blackjack) RunPlayerTurn(s *discordgo.Session, m *discordgo.MessageCreate) {

	// If the player's hand is under 21, prompt them to hit or stand
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

	// To get to this point the player either busts or stands
	(*b).IsPlayersTurn = false
}

// RunDealerTurn Handles the Dealer turns in the game
func (b *Blackjack) RunDealerTurn(s *discordgo.Session, m *discordgo.MessageCreate) {

	s.ChannelMessageSend(m.ChannelID, "It is the dealer's turn!")

	// This is the logic for casino blackjack where the Dealer is playing against more than one player
	// The dealer hits on anything less than 17, and also hits on a soft 17 (has an ace)
	for (*b).DealerHand.Value() <= 17 || ((*b).DealerHand.Value() == 17 && (*b).DealerHand.HasAce()) {
		(*b).Hit(&(*b).DealerHand)
	}

	//// This is the logic for 1v1 blackjack. The Dealer can see the player's hand and will continue to hit until they either beat them or bust
	//for (*b).DealerHand.Value() <= (*b).PlayerHand.Value() && (*b).DealerHand.Value() < 21 {
	//	(*b).Hit(&(*b).DealerHand)
	//}

	(*b).DisplayResults(s, m)
	GameOver(s, m)

}

// PromptPlayer Prompts the player by displaying their hand then asking to hit or stand.
func (b *Blackjack) PromptPlayer(s *discordgo.Session, m *discordgo.MessageCreate) {

	message := b.GetPlayerHand()
	s.ChannelMessageSend(m.ChannelID, message)
	s.ChannelMessageSend(m.ChannelID, "Enter !hit to hit, !stand to stand.")

}

// PlayerBust Displays the player's hand and a message that they have busted
func (b *Blackjack) PlayerBust(s *discordgo.Session, m *discordgo.MessageCreate) {

	message := (*b).GetPlayerHand()
	s.ChannelMessageSend(m.ChannelID, message)
	s.ChannelMessageSend(m.ChannelID, "Uh oh, you bust!")
	b.DisplayResults(s, m)
	GameOver(s, m)

}

// PlayerStand Displays the player's hand and that they have chosen to stand.
func (b *Blackjack) PlayerStand(s *discordgo.Session, m *discordgo.MessageCreate) {

	message := (*b).GetPlayerHand()
	message += "\nYou stand! It is now the dealer's turn."
	s.ChannelMessageSend(m.ChannelID, message)

}

// DisplayResults Displays the final results of the game
func (b *Blackjack) DisplayResults(s *discordgo.Session, m *discordgo.MessageCreate) {

	message := "=======================\n\t\t\t\tRESULTS\n=======================\n\n"

	// Displays both hands
	message += (*b).GetPlayerHand() + "\n\n"
	message += (*b).GetFullDealerHand() + "\n\n"

	// Determining the winner. Dealer wins if their hand is >= player hand
	if (*b).PlayerHand.Value() > 21 || (*b).DealerHand.Value() > (*b).PlayerHand.Value() {
		s.ChannelMessageSend(m.ChannelID, message+"The dealer wins.")
		// If the player loses, they lose their wager. so we set it to negative here so when it is updated in game over,
		// the wager is subtracted
		(*b).Wager *= -1
	} else if (*b).DealerHand.Value() == (*b).PlayerHand.Value() {
		// Draw, set wager to 0, so they get their chips back
		s.ChannelMessageSend(m.ChannelID, message+"It's a draw!")
		(*b).Wager = 0
	} else {
		s.ChannelMessageSend(m.ChannelID, message+(*b).Player.Username+" wins!")
	}
}
