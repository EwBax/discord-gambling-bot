package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Whether the bot is currently playing a game with someone
var GAME_ON = false

type Card struct {
	Rank string
	Suit string
}

func (c Card) String() string {
	return c.Rank + " of " + c.Suit
}

var ranks = map[string]int{"Ace": 1, "Two": 2, "Three": 3, "Four": 4, "Five": 5, "Six": 6, "Seven": 7, "Eight": 8, "Nine": 9, "Ten": 10, "Jack": 10, "Queen": 10, "King": 10}

type Deck []Card

func NewDeck() Deck {
	var deck Deck

	suits := []string{"Hearts", "Clubs", "Diamonds", "Spades"}

	for _, suit := range suits {
		for rank := range ranks {
			deck = append(deck, Card{rank, suit})
		}
	}

	return deck

}

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

func (d Deck) Shuffle() {

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := range d {
		j := rng.Intn(len(d))
		d[i], d[j] = d[j], d[i]
	}

}

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

type Hand []Card

func (h Hand) Value() int {

	value := 0
	numAces := 0

	// Getting the value and the number of aces in the deck
	for i := range h {
		value += ranks[h[i].Rank]
		if h[i].Rank == "Ace" {
			numAces++
		}
	}

	// If there are more aces and using one as a 10 would not cause the hand to bust, increase value by 10
	for numAces > 0 && value+10 <= 21 {
		value += 10
		numAces--
	}

	return value

}

func (h Hand) String() string {

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

type Blackjack struct {
	PlayerName    string
	CardDeck      Deck
	PlayerHand    Hand
	DealerHand    Hand
	IsPlayersTurn bool
}

func NewBlackjack(playerName string) Blackjack {

	newGame := Blackjack{playerName, NewDeck(), make(Hand, 0), make(Hand, 0), true}

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

func (b Blackjack) GetPlayerHand() string {

	message := fmt.Sprintf("%s your hand is:\n\n%s", BlackjackGame.PlayerName, BlackjackGame.PlayerHand)

	return message
}

func (b Blackjack) GetDealerHand() string {

	message := fmt.Sprintf("The dealer's hand is:\n\n%s", BlackjackGame.DealerHand)

	return message
}

func (b *Blackjack) Hit(h *Hand) {
	*h = append(*h, b.CardDeck.DealCard())
}

var BlackjackGame Blackjack

func main() {

	// Creating a new Discord session using the bot token
	sess, err := discordgo.New("Bot MTE1NzAzOTk4NTgxNjUxNDcwMQ.GOVQIL.KyV-XFZB0f1ylsYC0PQqOwFH5yw5vwgwWVupBM")
	if err != nil {
		log.Fatal(err)
	}

	// Handles a message being created in discord
	sess.AddHandler(MessageRecieved)

	// Sets the intent for the session
	sess.Identify.Intents = discordgo.IntentsAllWithoutPrivileged

	err = sess.Open()

	if err != nil {
		log.Fatal(err)
	}

	defer sess.Close()

	fmt.Println("The bot is online.")

	// This makes the bot continue to run
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}

func MessageRecieved(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Checks that the author was a user other than the bot
	if m.Author.ID == s.State.User.ID {
		return
	}

	if GAME_ON && m.Author.Username == BlackjackGame.PlayerName {

		if BlackjackGame.IsPlayersTurn {
			if strings.ToLower(m.Content) == "!hit" {
				s.ChannelMessageSend(m.ChannelID, "You chose to hit!")
				BlackjackGame.Hit(&BlackjackGame.PlayerHand)
				RunPlayerTurn(s, m)
			} else if strings.ToLower(m.Content) == "!hold" {
				s.ChannelMessageSend(m.ChannelID, "You chose to hold!")
				BlackjackGame.IsPlayersTurn = false
				RunDealerTurn(s, m)
			}
		}

	} else {

		// If a user enters !blackjack we begin the game
		if strings.ToLower(m.Content) == "!blackjack" {
			GAME_ON = true
			BlackjackGame = NewBlackjack(m.Author.Username)
			s.ChannelMessageSend(m.ChannelID, "Starting a new game of blackjack with "+m.Author.Username+"!")
			RunPlayerTurn(s, m)
		}

	}

}

// Handles the next Player turn in the game
func RunPlayerTurn(s *discordgo.Session, m *discordgo.MessageCreate) {

	if BlackjackGame.PlayerHand.Value() < 21 {
		PromptPlayer(s, m)
		return
	} else if BlackjackGame.PlayerHand.Value() > 21 {
		PlayerBust(s, m)
	} else {
		s.ChannelMessageSend(m.ChannelID, BlackjackGame.GetPlayerHand())
		RunDealerTurn(s, m)
	}

	// To get to this point the player either busts or holds
	BlackjackGame.IsPlayersTurn = false
}

// Handles the Dealer turns in the game
func RunDealerTurn(s *discordgo.Session, m *discordgo.MessageCreate) {

	s.ChannelMessageSend(m.ChannelID, "It is the dealer's turn!")
	s.ChannelMessageSend(m.ChannelID, BlackjackGame.GetDealerHand())

	// Dealer hits on 16 or less
	for BlackjackGame.DealerHand.Value() < 17 {
		BlackjackGame.Hit(&BlackjackGame.DealerHand)
		s.ChannelMessageSend(m.ChannelID, "The dealer hits!\n"+BlackjackGame.GetDealerHand())
	}

	if BlackjackGame.DealerHand.Value() > 21 {
		s.ChannelMessageSend(m.ChannelID, "\nThe dealer busts, you win!\n")
	} else {
		s.ChannelMessageSend(m.ChannelID, "\nThe dealer holds!")

		DisplayResults(s, m)
	}

	GameOver(s, m)

}

func PromptPlayer(s *discordgo.Session, m *discordgo.MessageCreate) {

	message := BlackjackGame.GetPlayerHand()
	s.ChannelMessageSend(m.ChannelID, message)
	s.ChannelMessageSend(m.ChannelID, "Enter !hit to hit, !hold to hold.")

}

func PlayerBust(s *discordgo.Session, m *discordgo.MessageCreate) {

	message := BlackjackGame.GetPlayerHand()
	s.ChannelMessageSend(m.ChannelID, message)
	s.ChannelMessageSend(m.ChannelID, "You bust! The dealer wins.")
	GameOver(s, m)

}

func PlayerHold(s *discordgo.Session, m *discordgo.MessageCreate) {

	message := BlackjackGame.GetPlayerHand()
	message += "\nYou hold! It is now the dealer's turn."
	s.ChannelMessageSend(m.ChannelID, message)

}

func GameOver(s *discordgo.Session, m *discordgo.MessageCreate) {
	s.ChannelMessageSend(m.ChannelID, "Game over!")
	GAME_ON = false
}

func DisplayResults(s *discordgo.Session, m *discordgo.MessageCreate) {

	message := "=======================\n\t\t\t\tRESULTS\n=======================\n\n"

	message += BlackjackGame.GetPlayerHand() + "\n\n"
	message += BlackjackGame.GetDealerHand() + "\n\n"

	if BlackjackGame.DealerHand.Value() >= BlackjackGame.PlayerHand.Value() {
		s.ChannelMessageSend(m.ChannelID, message+"The dealer wins.")
	} else {
		s.ChannelMessageSend(m.ChannelID, message+BlackjackGame.PlayerName+" wins!")
	}
}
